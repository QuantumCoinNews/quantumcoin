import {localDate} from './config.js';
import {PublisherError} from './types.js';
import path from 'node:path';
import {readJSON} from './storage.js';
import {Mode} from './types.js';
export const CANONICAL_CRON='0 18,21 * * *';
export function dueSlot(now:Date,timezone:string):{date:string;slot:1|2}{
 const date=localDate(now,timezone);
 const hour=Number(new Intl.DateTimeFormat('en',{timeZone:timezone,hour:'2-digit',hourCycle:'h23'}).format(now));
 if(hour>=21)return {date,slot:2};
 if(hour>=18)return {date,slot:1};
 const previous=new Date(date+'T00:00:00Z');previous.setUTCDate(previous.getUTCDate()-1);
 return {date:previous.toISOString().slice(0,10),slot:2};
}
export function requireCronAuthority(env:NodeJS.ProcessEnv):void{
 if(env.QC_PUBLISHER_SCHEDULER!=='cron')throw new PublisherError('CRON_NOT_SCHEDULING_AUTHORITY');
}
export async function scheduledTarget(stateDir:string,mode:Mode,now:Date,timezone:string){
 const due=dueSlot(now,timezone);
 const ledger=await readJSON<{version:string;activeJob?:string}>(path.join(stateDir,mode,'ledger.json'));
 if(!ledger?.activeJob)return due;
 const match=/^(\d{4}-\d{2}-\d{2})\.slot([12])$/.exec(ledger.activeJob);
 if(ledger.version!=='QC_PUBLISHER_LEDGER_V1'||!match)throw new PublisherError('LEDGER_CORRUPT_PRESERVED');
 const previous=await readJSON<{status:string}>(path.join(stateDir,mode,'jobs',ledger.activeJob+'.json'));
 if(previous&&['completed','abandoned'].includes(previous.status))return due;
 return {date:match[1],slot:Number(match[2]) as 1|2};
}
