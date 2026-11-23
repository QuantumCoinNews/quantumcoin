import path from 'node:path';
import { Config, Job, Mode, Platform, Provider, PublisherError } from './types.js';
import { configHash, localDate, validDate } from './config.js';
import { readJSON, saveJSON } from './storage.js';
import { selectPool, hashFile } from './pool.js';
import { acquireLock, Lease } from './lock.js';
import { render } from './render.js';
import { prepareContent } from './ai.js';
import { validatePublication } from './contentPolicy.js';
import { reportPublisherJob } from './monitoring.js';
import { DeliveryGuard, deliveryDeadline } from './resilience.js';
import { providers, deliveryError, providerStatus } from '../platforms/registry.js';

interface Ledger {version:'QC_PUBLISHER_LEDGER_V1';startDate:string;nextIndex:number;activeJob?:string}
export interface Dependencies { providers:Record<Platform,Provider>; render:typeof render; lock:(state:string)=>Promise<Lease>; prepare:typeof prepareContent; now?:()=>number }
const live:Dependencies={providers,render,lock:acquireLock,prepare:prepareContent};
function ledgerPath(c:Config,mode:Mode):string{return path.join(c.stateDir,mode,'ledger.json');}
function jobPath(c:Config,mode:Mode,id:string):string{return path.join(c.stateDir,mode,'jobs',id+'.json');}
function validateLedger(ledger:Ledger):void {if(ledger.version!=='QC_PUBLISHER_LEDGER_V1' || !validDate(ledger.startDate) || !Number.isSafeInteger(ledger.nextIndex) || ledger.nextIndex<1 || ledger.activeJob && !/^\d{4}-\d{2}-\d{2}\.slot[12]$/.test(ledger.activeJob))throw new PublisherError('LEDGER_CORRUPT_PRESERVED');}
function validateJob(job:Job):void {
 if(job.version!=='QC_PUBLISH_JOB_V1' || !validDate(job.date) || ![1,2].includes(job.slot) || job.id!==`${job.date}.slot${job.slot}` || !['dry-run','render','publish'].includes(job.mode) || !job.pool || !Number.isSafeInteger(job.pool.idx) || !job.deliveries || !['planned','preparing','rendering','rendered','publishing','completed','failed','needs-review','abandoned'].includes(job.status))throw new PublisherError('JOB_CORRUPT_PRESERVED');
 for(const delivery of Object.values(job.deliveries)){if(!delivery || !['pending','submitting','confirmed','failed','uncertain'].includes(delivery.status) || !Number.isSafeInteger(delivery.attempts) || delivery.attempts<0 || delivery.status==='confirmed' && !delivery.receipt?.id)throw new PublisherError('RECEIPT_CORRUPT_PRESERVED');}
 if(job.status==='completed' && job.mode==='publish' && (Object.keys(job.deliveries).length===0 || Object.values(job.deliveries).some(d=>d?.status!=='confirmed')))throw new PublisherError('COMPLETION_WITHOUT_RECEIPTS_REJECTED');
}
export async function initialize(c:Config,startDate:string,nextIndex=1,deps:Dependencies=live):Promise<void>{
 if(!validDate(startDate))throw new PublisherError('INVALID_START_DATE');
 await selectPool(c,nextIndex);const lease=await deps.lock(c.stateDir);
 try{const existing=await readJSON<Ledger>(ledgerPath(c,'publish'));if(existing){validateLedger(existing);throw new PublisherError('CANONICAL_STATE_ALREADY_INITIALIZED');}
 lease.assert();await saveJSON(ledgerPath(c,'publish'),{version:'QC_PUBLISHER_LEDGER_V1',startDate,nextIndex} satisfies Ledger);
 }finally{await lease.release();}
}
export async function run(c:Config,options:{mode:Mode;slot:1|2;date?:string},deps:Dependencies=live):Promise<Job>{
 const date=options.date||localDate(new Date(),c.timezone);if(!validDate(date)||![1,2].includes(options.slot))throw new PublisherError('INVALID_JOB_DATE_OR_SLOT');
 const mode=options.mode,id=`${date}.slot${options.slot}`;const lease=await deps.lock(mode==='dry-run'?path.join(c.stateDir,'dry-run'):c.stateDir);
 let durableReport: Job | undefined;
 try {
  let ledger=await readJSON<Ledger>(ledgerPath(c,mode));
  if(!ledger){if(mode==='publish')throw new PublisherError('LEGACY_CUTOVER_INITIALIZATION_REQUIRED');ledger={version:'QC_PUBLISHER_LEDGER_V1',startDate:date,nextIndex:1};}
  validateLedger(ledger);if(date<ledger.startDate)throw new PublisherError('JOB_BEFORE_CUTOVER');
  if(ledger.activeJob && ledger.activeJob!==id){
   const previous=await readJSON<Job>(jobPath(c,mode,ledger.activeJob));if(previous)validateJob(previous);
   if(previous?.status!=='completed' && previous?.status!=='abandoned')throw new PublisherError('PREVIOUS_JOB_INCOMPLETE');
   ledger.nextIndex=previous.pool.next;delete ledger.activeJob;await saveJSON(ledgerPath(c,mode),ledger);
  }
  let job=await readJSON<Job>(jobPath(c,mode,id));
  if(job){validateJob(job);if(job.mode!==mode)throw new PublisherError('JOB_CONFIGURATION_CHANGED');durableReport=structuredClone(job);}
  if(job?.status==='completed' || job?.status==='abandoned'){
   if(ledger.activeJob===id){ledger.nextIndex=job.pool.next;delete ledger.activeJob;await saveJSON(ledgerPath(c,mode),ledger);}
   return job;
  }
  if(job&&job.configHash!==configHash(c))throw new PublisherError('JOB_CONFIGURATION_CHANGED');
  const pool=await selectPool(c,job?.pool.idx||ledger.nextIndex).catch(error=>{if(job&&error instanceof PublisherError&&error.code==='MEDIA_PROVENANCE_MISMATCH')throw new PublisherError('POOL_CHANGED_DURING_JOB');throw error;});
  if(job)pool.next=job.pool.next;
  if(job && pool.fingerprint!==job.pool.fingerprint)throw new PublisherError('POOL_CHANGED_DURING_JOB');
  if(mode!=='dry-run' && c.env.QC_PUBLISHER_AI_ENABLED==='1' && !c.env.OPENAI_API_KEY)throw new PublisherError('AI_CREDENTIAL_CONFIGURATION_REQUIRED');
  if(mode==='publish')for(const platform of c.platforms){const reason=deps.providers[platform].readiness(c.env);if(reason)throw new PublisherError(`${platform.toUpperCase()}_${reason}`);}
  if(!job){job={version:'QC_PUBLISH_JOB_V1',id,date,slot:options.slot,mode,status:'planned',pool,configHash:configHash(c),deliveries:Object.fromEntries(c.platforms.map(platform=>[platform,{status:'pending',attempts:0}]))};
   if(mode!=='dry-run')ledger.activeJob=id;lease.assert();await saveJSON(ledgerPath(c,mode),ledger);await saveJSON(jobPath(c,mode,id),job);
  }
  durableReport=structuredClone(job);
  const persist=async()=>{lease.assert();await saveJSON(jobPath(c,mode,id),job);durableReport=structuredClone(job);};
  // Dry-run never renders, generates paid assets, or calls a provider. Its
  // namespace/receipts cannot consume a production slot or production cursor.
  if(mode==='dry-run'){return job;}
  for(const delivery of Object.values(job.deliveries))if(delivery?.status==='submitting'){delivery.status='uncertain';delivery.error='INTERRUPTED_AFTER_SUBMISSION';}
  if(Object.values(job.deliveries).some(d=>d?.status==='uncertain')){job.status='needs-review';await persist();return job;}
  if(c.env.QC_PUBLISHER_AI_ENABLED==='1' && !job.prepared){
   if((job.generationAttempts||0)>=2)throw new PublisherError('AI_RETRY_BUDGET_EXHAUSTED');
   job.status='preparing';job.generationAttempts=(job.generationAttempts||0)+1;await persist();
   try{job.prepared=await deps.prepare(c,pool,path.join(c.stateDir,mode,'artifacts',id),id);await persist();}catch{job.status='failed';job.error='AI_GENERATION_FAILED';await persist();return job;}
  }
  if(job.prepared && await hashFile(job.prepared.audio)!==job.prepared.audioHash)throw new PublisherError('PREPARED_AUDIO_CHANGED');
  if(job.prepared?.background && await hashFile(job.prepared.background)!==job.prepared.backgroundHash)throw new PublisherError('PREPARED_IMAGE_CHANGED');
  const preparedPool=job.prepared?{...pool,topic:job.prepared.topic,audio:job.prepared.audio,background:job.prepared.background||pool.background}:pool;
  try{validatePublication(preparedPool.topic);}
  catch(error){job.editorial='rejected';job.status='failed';await persist();throw error;}
  if(job.editorial!=='passed'){job.editorial='passed';await persist();}
  if(job.video){if(await hashFile(job.video)!==job.videoHash)throw new PublisherError('RENDERED_ARTIFACT_CHANGED');}
  else{job.status='rendering';await persist();try{job.video=await deps.render(preparedPool,path.join(c.stateDir,mode,'artifacts',id),()=>lease.assert());job.videoHash=await hashFile(job.video);}catch{job.status='failed';job.error='RENDER_FAILED';await persist();return job;}job.status='rendered';await persist();}
  if(mode==='render'){job.status='completed';await persist();ledger.nextIndex=pool.next;delete ledger.activeJob;await saveJSON(ledgerPath(c,mode),ledger);return job;}
  job.status='publishing';delete job.error;await persist();
  const guard=new DeliveryGuard(c.stateDir,deps.now);
  for(const platform of c.platforms){
   const delivery=job.deliveries[platform]!;if(delivery.status==='confirmed')continue;
   if(!await guard.allow(platform)){delivery.status='failed';delivery.error='CIRCUIT_OPEN_OR_BACKOFF';job.status='failed';await persist();continue;}
   if(delivery.attempts>=3){job.status='failed';job.error='RETRY_BUDGET_EXHAUSTED';await persist();continue;}
   delivery.status='submitting';delivery.attempts++;delete delivery.error;delete delivery.httpStatus;await persist();
   const topic=preparedPool.topic;const text=platform==='x'?(topic.x||topic.title):topic.telegram||topic.title;
   try{lease.assert();const receipt=await deliveryDeadline(()=>deps.providers[platform].publish({title:topic.title,description:topic.youtube_description||topic.script,text,video:job.video!,privacy:c.privacy},c.env));
    if(!receipt.id)throw new PublisherError('DELIVERY_UNCERTAIN');delivery.status='confirmed';delivery.receipt=receipt;
   }catch(error){const code=deliveryError(error);delivery.error=code;delivery.httpStatus=providerStatus(error);delivery.status=code==='DELIVERY_UNCERTAIN'?'uncertain':'failed';job.status=delivery.status==='uncertain'?'needs-review':'failed';await persist();await guard.failure(platform,delivery.httpStatus);if(delivery.status==='uncertain')return job;continue;}
   // Receipt persistence failure leaves a submitting marker, requiring manual
   // reconciliation rather than a blind duplicate upload after restart.
   await persist();
   await guard.success(platform);
  }
  if(Object.values(job.deliveries).some(d=>d?.status!=='confirmed')){job.status='failed';job.error||='PLATFORM_DELIVERY_FAILED';await persist();return job;}
  job.status='completed';await persist();ledger.nextIndex=pool.next;delete ledger.activeJob;await saveJSON(ledgerPath(c,mode),ledger);return job;
 }finally{
  await lease.release();
  if(durableReport)await reportPublisherJob(c.env,durableReport).catch(()=>console.warn('PUBLISHER_MONITOR_REPORT_UNAVAILABLE'));
 }
}

// Operator cancellation preserves receipts/unknown deliveries. It never turns
// an uncertain provider result into a successful publication.
export async function abandon(c:Config,id:string,deps:Dependencies=live):Promise<void>{
 if(!/^\d{4}-\d{2}-\d{2}\.slot[12]$/.test(id))throw new PublisherError('INVALID_JOB_ID');
 const lease=await deps.lock(c.stateDir);try{
  const ledger=await readJSON<Ledger>(ledgerPath(c,'publish'));if(!ledger)throw new PublisherError('CANONICAL_STATE_MISSING');validateLedger(ledger);
  const job=await readJSON<Job>(jobPath(c,'publish',id));if(!job)throw new PublisherError('JOB_NOT_FOUND');validateJob(job);
  if(ledger.activeJob!==id || job.status==='completed')throw new PublisherError('JOB_NOT_ACTIVE');
  job.status='abandoned';job.error='OPERATOR_ABANDONED_AFTER_REVIEW';lease.assert();await saveJSON(jobPath(c,'publish',id),job);
  ledger.nextIndex=job.pool.next;delete ledger.activeJob;await saveJSON(ledgerPath(c,'publish'),ledger);
 }finally{await lease.release();}
}
