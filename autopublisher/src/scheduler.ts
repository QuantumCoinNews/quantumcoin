import path from 'node:path';
import {acquireLock} from './runtime/lock.js';
import {dueSlot,scheduledTarget,requireCronAuthority} from './runtime/schedulePolicy.js';
import cron from 'node-cron';
import { Config, Mode, PublisherError } from './runtime/types.js';
import { run } from './runtime/engine.js';
import { localDate } from './runtime/config.js';

export async function scheduleDaily(c:Config,mode:Mode){
 requireCronAuthority(c.env);
 if(!cron.validate(c.cron))throw new PublisherError('INVALID_CRON');
 const authority=await acquireLock(path.join(c.stateDir,'scheduler-authority'));
 const task=cron.schedule(c.cron,()=>{
  const now=new Date();const hour=Number(new Intl.DateTimeFormat('en',{timeZone:c.timezone,hour:'2-digit',hourCycle:'h23'}).format(now));
  void Promise.resolve().then(async()=>{authority.assert();return run(c,{mode,...await scheduledTarget(c.stateDir,mode,now,c.timezone)});}).then(job=>console.log(JSON.stringify({event:'scheduled_run',jobId:job.id,status:job.status,mode}))).catch(error=>console.error(JSON.stringify({event:'scheduled_run_failed',code:error instanceof PublisherError?error.code:'RUN_FAILED'})));
 },{timezone:c.timezone});
 for(const signal of ['SIGINT','SIGTERM'] as const)process.once(signal,()=>{task.stop();void authority.release().finally(()=>{process.exitCode=0;});});
 return task;
}
