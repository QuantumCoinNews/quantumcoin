import {spawn} from 'node:child_process';
import fs from 'node:fs/promises';
import path from 'node:path';
import {root} from './config.js';
import {PublisherError} from './types.js';
import {saveJSON,readJSON} from './storage.js';

interface Envelope {version:'QC_DPAPI_V1';ciphertext:string}
export async function protect(action:'protect'|'unprotect',input:string):Promise<string>{
 if(process.platform!=='win32')throw new PublisherError('WINDOWS_SECRET_STORAGE_REQUIRED');
 if(Buffer.byteLength(input)>4*1024*1024)throw new PublisherError('SECRET_STORE_TOO_LARGE');
 return new Promise((resolve,reject)=>{
  const child=spawn('pwsh.exe',['-NoProfile','-NonInteractive','-File',path.join(root,'scripts/secret-storage.ps1'),'-Action',action],{stdio:['pipe','pipe','pipe'],windowsHide:true});
  const chunks:Buffer[]=[];let size=0;const timer=setTimeout(()=>child.kill(),30000);
  child.stderr.resume();child.stdout.on('data',(b:Buffer)=>{size+=b.length;if(size>6*1024*1024)child.kill();else chunks.push(b);});
  child.on('error',()=>{clearTimeout(timer);reject(new PublisherError('SECRET_STORAGE_UNAVAILABLE'));});
  child.on('exit',code=>{clearTimeout(timer);code===0?resolve(Buffer.concat(chunks).toString('utf8')):reject(new PublisherError('SECRET_STORAGE_FAILED'));});
  child.stdin.on('error',()=>{});child.stdin.end(input);
 });
}
export async function loadSecrets(file:string):Promise<NodeJS.ProcessEnv>{
 if(!path.isAbsolute(file))throw new PublisherError('ABSOLUTE_SECRET_PATH_REQUIRED');
 const value=await readJSON<Envelope>(file);
 if(value?.version!=='QC_DPAPI_V1'||typeof value.ciphertext!=='string')throw new PublisherError('SECRET_STORE_INVALID');
 let env:unknown;try{env=JSON.parse(await protect('unprotect',value.ciphertext));}catch{throw new PublisherError('SECRET_DECRYPTION_FAILED');}
 if(!env||typeof env!=='object'||Array.isArray(env)||Object.values(env).some(v=>typeof v!=='string'))throw new PublisherError('SECRET_STORE_INVALID');
 return env as NodeJS.ProcessEnv;
}
export async function saveSecrets(file:string,env:NodeJS.ProcessEnv):Promise<void>{
 if(!path.isAbsolute(file))throw new PublisherError('ABSOLUTE_SECRET_PATH_REQUIRED');
 const serializable=Object.fromEntries(Object.entries(env).filter(([,v])=>v!==undefined));
 const text=JSON.stringify(serializable);const ciphertext=await protect('protect',text);
 if(await protect('unprotect',ciphertext)!==text)throw new PublisherError('SECRET_ROUNDTRIP_FAILED');
 await saveJSON(file,{version:'QC_DPAPI_V1',ciphertext} satisfies Envelope);
}
export async function embedYouTube(env:NodeJS.ProcessEnv):Promise<NodeJS.ProcessEnv>{
 const result={...env};
 for(const [source,target] of [['YOUTUBE_CLIENT_SECRET_PATH','YOUTUBE_CLIENT_JSON'],['YOUTUBE_TOKEN_PATH','YOUTUBE_TOKEN_JSON']]){
  if(env[source]){const file=path.resolve(root,env[source]!);const bytes=await fs.readFile(file,'utf8');JSON.parse(bytes);result[target]=bytes;delete result[source];}
 }
 return result;
}
