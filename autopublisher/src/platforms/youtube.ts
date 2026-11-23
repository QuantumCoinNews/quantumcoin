import {DELIVERY_TIMEOUT_MS} from '../runtime/resilience.js';
import {youtubeConfigured,youtubeCredentials,revokedCredential} from '../runtime/oauth.js';
import fs from 'node:fs';
import path from 'node:path';
import { youtube, auth } from 'googleapis/build/src/apis/youtube/index.js';
import { root } from '../runtime/config.js';
import { PublishInput, Receipt, PublisherError } from '../runtime/types.js';

export function youtubeReadiness(env:NodeJS.ProcessEnv):string|undefined {return youtubeConfigured(env)?undefined:'MISSING_CREDENTIAL_CONFIGURATION';}
export async function publishYouTube(input:PublishInput,env:NodeJS.ProcessEnv,factory:(oauth:InstanceType<typeof auth.OAuth2>)=>Pick<ReturnType<typeof youtube>,'videos'>=oauth=>youtube({version:'v3',auth:oauth})):Promise<Receipt> {
 if(!input.title.trim()||input.title.length>100||input.description.length>5000)throw new PublisherError('PROVIDER_CONFIGURATION_INVALID');
 const {oauth,flush}=youtubeCredentials(env);
 try {
 const result=await factory(oauth).videos.insert({part:['snippet','status'],requestBody:{snippet:{title:input.title.slice(0,100),description:input.description.slice(0,5000),defaultLanguage:'en',defaultAudioLanguage:'en'},status:{privacyStatus:input.privacy,selfDeclaredMadeForKids:false}},media:{body:fs.createReadStream(input.video)}},{timeout:DELIVERY_TIMEOUT_MS,retry:false});
 if(!result.data.id)throw new PublisherError('DELIVERY_UNCERTAIN');
 await flush();return {id:result.data.id,url:`https://www.youtube.com/watch?v=${result.data.id}`};
 }catch(error){await flush();if(revokedCredential(error))throw new PublisherError('PROVIDER_CREDENTIAL_REVOKED');throw error;}
}
