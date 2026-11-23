import {youtube} from 'googleapis/build/src/apis/youtube/index.js';
import {TwitterApi} from 'twitter-api-v2';
import {youtubeConfigured,youtubeCredentials,revokedCredential} from './oauth.js';
import {xCredentials,xReadiness} from '../platforms/x.js';
import {ensureXOAuth2AccessToken,xOAuth2Configured,xOAuth2Lifecycle} from './xOAuth2.js';
import {providerStatus} from '../platforms/registry.js';
export function credentialError(error:unknown):string{
 if(revokedCredential(error)||providerStatus(error)===401)return 'reauthorization-required';
 const v=error as {response?:{data?:{error?:{errors?:{reason?:string}[]}}}};
 if(v?.response?.data?.error?.errors?.some(e=>['quotaExceeded','dailyLimitExceeded','userRateLimitExceeded'].includes(e.reason||'')))return 'quota-exceeded';
 if(providerStatus(error)===402)return 'billing-required';
 if(providerStatus(error)===429)return 'rate-limited';
 return 'check-failed';
}
// Explicit read-only account probes. Authentication is not upload acceptance.
export async function credentialHealth(env:NodeJS.ProcessEnv){
 const report:Record<string,string>={};
 if(!youtubeConfigured(env))report.youtube='not-configured';else try{const {oauth,flush}=youtubeCredentials(env);try{await youtube({version:'v3',auth:oauth}).channels.list({mine:true,part:['id']},{timeout:20000,retry:false});report.youtube='authenticated';}finally{await flush();}}catch(e){report.youtube=credentialError(e);}
 if(xOAuth2Configured(env))try{const token=await ensureXOAuth2AccessToken(env);const r=await fetch('https://api.x.com/2/users/me',{headers:{authorization:`Bearer ${token}`},signal:AbortSignal.timeout(20000)});report.x=r.ok?'authenticated':r.status===401?'reauthorization-required':r.status===429?'rate-limited':'check-failed';}catch(e){report.x=credentialError(e);}
 else if(xReadiness(env))report.x='not-configured';else try{await new TwitterApi(xCredentials(env)).v2.get('users/me',undefined,{timeout:20000});report.x='authenticated';}catch(e){report.x=credentialError(e);}
 if(!env.TELEGRAM_BOT_TOKEN)report.telegram='not-configured';else try{const r=await fetch('https://api.telegram.org/bot'+env.TELEGRAM_BOT_TOKEN+'/getMe',{signal:AbortSignal.timeout(20000)});const b=await r.json() as {ok?:boolean};report.telegram=r.ok&&b.ok?'authenticated':'check-failed';}catch{report.telegram='check-failed';}
 return {purpose:'read-only-credential-health',publicationCapability:'not-tested',accounts:report,xTokenLifecycle:xOAuth2Lifecycle(env)};
}
