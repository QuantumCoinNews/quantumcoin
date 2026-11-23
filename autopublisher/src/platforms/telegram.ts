import {DELIVERY_TIMEOUT_MS} from '../runtime/resilience.js';
import { readFile, stat } from 'node:fs/promises';
import { PublishInput, Receipt, PublisherError } from '../runtime/types.js';
export function telegramReadiness(env:NodeJS.ProcessEnv):string|undefined{return env.TELEGRAM_BOT_TOKEN && env.TELEGRAM_CHAT_ID?undefined:'MISSING_CREDENTIAL_CONFIGURATION';}
// Official HTTPS Bot API. Request URLs contain the token and must never be logged.
export async function publishTelegram(input:PublishInput,env:NodeJS.ProcessEnv,transport:typeof fetch=fetch):Promise<Receipt>{
 if(input.text.length>1024)throw new PublisherError('PROVIDER_CONFIGURATION_INVALID');
 if((await stat(input.video)).size>49*1024*1024)throw new PublisherError('PROVIDER_CONFIGURATION_INVALID');
 const body=new FormData();body.set('chat_id',env.TELEGRAM_CHAT_ID!);body.set('caption',input.text);body.set('supports_streaming','true');
 body.set('video',new Blob([new Uint8Array(await readFile(input.video))],{type:'video/mp4'}),'short.mp4');
 const response=await transport('https://api.telegram.org/bot'+env.TELEGRAM_BOT_TOKEN+'/sendVideo',{method:'POST',body,signal:AbortSignal.timeout(DELIVERY_TIMEOUT_MS)});
 const result=await response.json() as {ok?:boolean;error_code?:number;result?:{message_id?:number;chat?:{username?:string}}};
 if(!response.ok || !result.ok)throw {response:{status:result.error_code||response.status}};
 const id=result.result?.message_id;if(!id)throw new PublisherError('DELIVERY_UNCERTAIN');
 return {id:String(id),...(result.result?.chat?.username?{url:`https://t.me/${result.result.chat.username}/${id}`}:{})};
}
