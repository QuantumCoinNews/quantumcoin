import path from 'node:path';
import { randomInt } from 'node:crypto';
import { readJSON, saveJSON } from './storage.js';
import { Platform, PublisherError } from './types.js';

interface Circuit { version:1; failures:number; nextAttemptAt:number; status?:number }
// Called while the production OS lease is held. No provider SDK retry loops:
// an ambiguous POST is never retried automatically, regardless of status age.
export class DeliveryGuard {
 constructor(private stateDir:string,private now:()=>number=Date.now){}
 private file(platform:Platform){return path.join(this.stateDir,'delivery-health',platform+'.json');}
 async state(platform:Platform):Promise<Circuit>{
  const v=await readJSON<Circuit>(this.file(platform));
  if(!v)return {version:1,failures:0,nextAttemptAt:0};
  if(v.version!==1||!Number.isSafeInteger(v.failures)||v.failures<0||!Number.isSafeInteger(v.nextAttemptAt)||v.nextAttemptAt<0)throw new PublisherError('CIRCUIT_STATE_CORRUPT');
  return v;
 }
 async allow(platform:Platform):Promise<boolean>{return (await this.state(platform)).nextAttemptAt<=this.now();}
 async success(platform:Platform):Promise<void>{await saveJSON(this.file(platform),{version:1,failures:0,nextAttemptAt:0} satisfies Circuit);}
 async failure(platform:Platform,status?:number):Promise<void>{
  const previous=await this.state(platform);const failures=Math.min(previous.failures+1,20);
  // Billing/auth failures require operator action rather than rapid retries.
  const terminal=status!==undefined&&[401,402,403].includes(status);
  const cooldown=terminal?3600000:Math.min(900000,2000*2**(failures-1))+randomInt(0,1001);
  await saveJSON(this.file(platform),{version:1,failures,nextAttemptAt:this.now()+cooldown,status} satisfies Circuit);
 }
}

export const DELIVERY_TIMEOUT_MS=120000;

// The SDKs also have request timeouts. This covers credential refresh and
// other work before their HTTP request starts. A deadline is ambiguous: the
// remote side may still deliver, so the engine must require reconciliation.
export async function deliveryDeadline<T>(operation:()=>Promise<T>,milliseconds=DELIVERY_TIMEOUT_MS):Promise<T>{
 let timer:ReturnType<typeof setTimeout>|undefined;
 try{return await Promise.race([Promise.resolve().then(operation),new Promise<never>((_resolve,reject)=>{timer=setTimeout(()=>reject(new PublisherError('DELIVERY_UNCERTAIN')),milliseconds);})]);}
 finally{if(timer)clearTimeout(timer);}
}
