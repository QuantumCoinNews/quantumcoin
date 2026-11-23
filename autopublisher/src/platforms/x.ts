import {DELIVERY_TIMEOUT_MS} from '../runtime/resilience.js';
import { TwitterApi } from 'twitter-api-v2';
import { PublishInput, Receipt, PublisherError } from '../runtime/types.js';
export function xCredentials(env:NodeJS.ProcessEnv){return {appKey:env.TW_CONSUMER_KEY||env.X_APP_KEY||env.X_CONSUMER_KEY||env.TWITTER_API_KEY||'',appSecret:env.TW_CONSUMER_SECRET||env.X_APP_SECRET||env.X_CONSUMER_SECRET||env.TWITTER_API_SECRET||'',accessToken:env.TW_ACCESS_TOKEN||env.X_ACCESS_TOKEN||env.TWITTER_ACCESS_TOKEN||'',accessSecret:env.TW_ACCESS_SECRET||env.X_ACCESS_SECRET||env.X_ACCESS_TOKEN_SECRET||env.TWITTER_ACCESS_SECRET||''};}
export function xReadiness(env:NodeJS.ProcessEnv):string|undefined{return Object.values(xCredentials(env)).every(Boolean)?undefined:'MISSING_CREDENTIAL_CONFIGURATION';}
export async function publishX(input:PublishInput,env:NodeJS.ProcessEnv,factory:(c:ReturnType<typeof xCredentials>)=>{create(text:string):Promise<{data:{id?:string}}>} = c=>({create:text=>new TwitterApi(c).v2.post('tweets',{text},{timeout:DELIVERY_TIMEOUT_MS})})):Promise<Receipt>{
 if(!input.text.trim()||input.text.length>280)throw new PublisherError('PROVIDER_CONFIGURATION_INVALID');
 const result=await factory(xCredentials(env)).create(input.text);
 if(!result.data.id)throw new PublisherError('DELIVERY_UNCERTAIN');
 return {id:result.data.id,url:`https://x.com/i/status/${result.data.id}`};
}
