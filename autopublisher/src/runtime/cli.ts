import {registerMedia} from './mediaCatalog.js';
import {maintain} from './maintenance.js';
import {dueSlot,scheduledTarget} from './schedulePolicy.js';
import {acquireLock} from './lock.js';
import {credentialHealth} from './credentialHealth.js';
import {loadSecrets,saveSecrets,embedYouTube} from './secrets.js';
import { parseArgs } from 'node:util';
import { readFile } from 'node:fs/promises';
import { parse as parseEnv } from 'dotenv';
import { configuration, localDate } from './config.js';
import { initialize, run, abandon } from './engine.js';
import { selectPool } from './pool.js';
import { providers } from '../platforms/registry.js';
import { PublisherError, Mode } from './types.js';
import { scheduleDaily } from '../scheduler.js';

export async function main(args:string[]):Promise<number>{
 try{
  const {values,positionals}=parseArgs({args,allowPositionals:true,options:{scheduler:{type:'string'},apply:{type:'boolean'},publish:{type:'boolean'},'render-only':{type:'boolean'},slot:{type:'string'},date:{type:'string'},'state-dir':{type:'string'},'env-file':{type:'string'},'secret-file':{type:'string'},'settings-file':{type:'string'},platforms:{type:'string'},'start-date':{type:'string'},'next-index':{type:'string'},job:{type:'string'},help:{type:'boolean'}}});
  if(values.help){console.log('autopublisher doctor | run [--slot 1|2] [--render-only | --publish] | schedule [--publish] | init --start-date YYYY-MM-DD | abandon --job YYYY-MM-DD.slot1\nUse --env-file for explicit credentials, --state-dir for an absolute state path. Default run is dry-run; no automatic .env load.');return 0;}
  const command=positionals[0]||'doctor';if(positionals.length>1 || !['doctor','run','schedule','init','abandon','secrets-import','secrets-update','auth-health','maintenance','media-register'].includes(command))throw new PublisherError('INVALID_COMMAND');
  if(values.publish && values['render-only'])throw new PublisherError('CONFLICTING_RUN_MODES');
  if(!['secrets-import','secrets-update'].includes(command)&&values['env-file']&&values['secret-file'])throw new PublisherError('CONFLICTING_CREDENTIAL_SOURCES');
  if(values.publish&&(values['env-file']||process.execArgv.some(arg=>arg==='--env-file'||arg.startsWith('--env-file='))))throw new PublisherError('PLAINTEXT_PUBLICATION_CREDENTIALS_RETIRED');
  const env={...process.env};if(values['env-file'])Object.assign(env,parseEnv(await readFile(values['env-file'])));
  if(command==='secrets-import'){
   if(!values['env-file']||!values['secret-file'])throw new PublisherError('EXPLICIT_IMPORT_PATHS_REQUIRED');
   const imported=await embedYouTube(parseEnv(await readFile(values['env-file'])));
   try{await readFile(values['secret-file']);throw new PublisherError('SECRET_STORE_ALREADY_EXISTS');}catch(error){if((error as NodeJS.ErrnoException).code!=='ENOENT')throw error;}
   await saveSecrets(values['secret-file'],imported);console.log(JSON.stringify({status:'encrypted',published:false,rotationPerformed:false,originalsRemoved:false}));return 0;
  }
  if(values['secret-file']){Object.assign(env,await loadSecrets(values['secret-file']));env.QC_PUBLISHER_SECRET_FILE=values['secret-file'];}
  if(values['settings-file']){
   const settings=parseEnv(await readFile(values['settings-file']));
   const allowed=new Set(['CONTENT_LANG','QC_PUBLISHER_AI_ENABLED','QC_PUBLISHER_AI_VISUALS','OPENAI_IMAGE_MODEL','QC_PUBLISHER_TIMEZONE','QC_PUBLISHER_CRON','QC_PUBLISHER_SCHEDULER','QC_PUBLISHER_PLATFORMS','QC_PUBLISHER_MONITOR_ORIGIN','YOUTUBE_PRIVACY','OPENAI_MODEL','OPENAI_TTS_MODEL','OPENAI_TTS_VOICE']);
   if(Object.keys(settings).some(key=>!allowed.has(key)))throw new PublisherError('INVALID_PUBLIC_SETTINGS');
   Object.assign(env,settings);
  }
  if(values['state-dir'])env.QC_PUBLISHER_STATE_DIR=values['state-dir'];if(values.platforms)env.QC_PUBLISHER_PLATFORMS=values.platforms;
  const c=configuration(env);const mode:Mode=values.publish?'publish':values['render-only']?'render':'dry-run';
  if(command==='secrets-update'){
   if(!values['env-file']||!values['secret-file'])throw new PublisherError('EXPLICIT_IMPORT_PATHS_REQUIRED');
   const lease=await acquireLock(c.stateDir);
   try{const existing=await loadSecrets(values['secret-file']);const changes=await embedYouTube(parseEnv(await readFile(values['env-file'])));lease.assert();await saveSecrets(values['secret-file'],{...existing,...changes});}finally{await lease.release();}
   console.log(JSON.stringify({status:'encrypted-update',published:false,rotationPerformed:false}));return 0;
  }
  if(command==='media-register'){console.log(JSON.stringify(await registerMedia(c)));return 0;}
  if(command==='maintenance'){console.log(JSON.stringify(await maintain(c.stateDir,Boolean(values.apply))));return 0;}
  if(command==='auth-health'){const lease=await acquireLock(c.stateDir);try{console.log(JSON.stringify(await credentialHealth(env)));return 0;}finally{await lease.release();}}
  if(command==='doctor'){
   let poolReady=true;let poolError:string|undefined;try{await selectPool(c,Number(values['next-index']||1));}catch(error){poolReady=false;poolError=error instanceof PublisherError?error.code:'POOL_UNAVAILABLE';}
   console.log(JSON.stringify({runtime:'typescript-canonical-v1',defaultMode:'dry-run',timezone:c.timezone,cron:c.cron,poolReady,poolError,platforms:Object.fromEntries(c.platforms.map(p=>[p,{implemented:!['instagram','tiktok'].includes(p),configuration:providers[p].readiness(c.env)||'present',liveCheckPerformed:false}])),liveAcceptance:'not-assessed-by-doctor; inspect production receipts',cutover:'init --start-date is required before publish'}));return 0;
  }
  if(command==='init'){if(!values['start-date'])throw new PublisherError('START_DATE_REQUIRED');await initialize(c,values['start-date'],Number(values['next-index']||1));console.log(JSON.stringify({status:'initialized',published:false}));return 0;}
  if(command==='abandon'){if(!values.job)throw new PublisherError('JOB_ID_REQUIRED');await abandon(c,values.job);console.log(JSON.stringify({status:'abandoned',published:false}));return 0;}
  if(command==='schedule'){if(values.date || values.slot)throw new PublisherError('SCHEDULE_USES_CONFIGURED_LOCAL_TIME');await scheduleDaily(c,mode);console.log(JSON.stringify({status:'scheduled',mode,timezone:c.timezone,cron:c.cron}));return 0;}
  const hour=Number(new Intl.DateTimeFormat('en',{timeZone:c.timezone,hour:'2-digit',hourCycle:'h23'}).format(new Date()));
  if(values.date&&!values.slot)throw new PublisherError('EXPLICIT_DATE_REQUIRES_SLOT');
  const due=dueSlot(new Date(),c.timezone);
  const slot=Number(values.slot||due.slot);if(slot!==1&&slot!==2)throw new PublisherError('INVALID_SLOT');
  if(values.scheduler&&(values.scheduler!=='windows-task'||c.env.QC_PUBLISHER_SCHEDULER!==values.scheduler))throw new PublisherError('SCHEDULER_AUTHORITY_MISMATCH');
  const authority=values.scheduler?await acquireLock(c.stateDir+'/scheduler-authority'):undefined;
  let job;
  try{const target:{date:string;slot:1|2}=values.scheduler?await scheduledTarget(c.stateDir,mode,new Date(),c.timezone):{slot,date:values.date||(values.slot?localDate(new Date(),c.timezone):due.date)};job=await run(c,{mode,...target});}finally{await authority?.release();}
  console.log(JSON.stringify({jobId:job.id,mode:job.mode,status:job.status,poolIndex:job.pool.idx,published:job.status==='completed'&&mode==='publish',error:job.error,platforms:Object.fromEntries(Object.entries(job.deliveries).map(([name,d])=>[name,{status:d?.status,attempts:d?.attempts}]))}));
  return mode==='dry-run'||job.status==='completed'?0:1;
 }catch(error){const code=error instanceof PublisherError?error.code:'COMMAND_FAILED';console.error(JSON.stringify({status:'failed',code,published:false}));return code==='RUN_IN_PROGRESS'?73:1;}
}
