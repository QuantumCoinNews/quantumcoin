import fs from 'node:fs';
import path from 'node:path';
import {auth} from 'googleapis/build/src/apis/youtube/index.js';
import {root} from './config.js';
import {PublisherError} from './types.js';
import {loadSecrets,saveSecrets} from './secrets.js';

export function youtubeConfigured(env:NodeJS.ProcessEnv):boolean {
 return Boolean((env.YOUTUBE_CLIENT_JSON&&env.YOUTUBE_TOKEN_JSON)||(env.YOUTUBE_CLIENT_SECRET_PATH&&env.YOUTUBE_TOKEN_PATH&&fs.existsSync(path.resolve(root,env.YOUTUBE_CLIENT_SECRET_PATH))&&fs.existsSync(path.resolve(root,env.YOUTUBE_TOKEN_PATH))));
}
export function youtubeCredentials(env:NodeJS.ProcessEnv){
 try{
  const raw=JSON.parse(env.YOUTUBE_CLIENT_JSON||fs.readFileSync(path.resolve(root,env.YOUTUBE_CLIENT_SECRET_PATH!),'utf8'));
  const client=raw.installed||raw.web;
  if(!client?.client_id||!client?.client_secret)throw Error();
  const tokens=JSON.parse(env.YOUTUBE_TOKEN_JSON||fs.readFileSync(path.resolve(root,env.YOUTUBE_TOKEN_PATH!),'utf8'));
  const oauth=new auth.OAuth2(client.client_id,client.client_secret,client.redirect_uris?.[0]);oauth.setCredentials(tokens);
  let pending=Promise.resolve();let failed=false;
  oauth.on('tokens',fresh=>{
   Object.assign(tokens,fresh);env.YOUTUBE_TOKEN_JSON=JSON.stringify(tokens);
   // Serialize all token updates; keep refresh_token when Google omits it.
   if(env.QC_PUBLISHER_SECRET_FILE)pending=pending.then(async()=>{const stored=await loadSecrets(env.QC_PUBLISHER_SECRET_FILE!);stored.YOUTUBE_TOKEN_JSON=JSON.stringify(tokens);await saveSecrets(env.QC_PUBLISHER_SECRET_FILE!,stored);}).catch(()=>{failed=true;});
  });
  return {oauth,async flush(){await pending;if(failed)throw new PublisherError('CREDENTIAL_PERSISTENCE_FAILED');}};
 }catch{throw new PublisherError('PROVIDER_CONFIGURATION_INVALID');}
}
export function revokedCredential(error:unknown):boolean{
 const value=error as {response?:{data?:{error?:string}}};
 return ['invalid_grant','invalid_client','unauthorized_client'].includes(value?.response?.data?.error||'');
}
