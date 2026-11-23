import fs from 'node:fs/promises';
import path from 'node:path';
import {createHash} from 'node:crypto';
import {Config,Topic,PublisherError} from './types.js';
import {readJSON,saveJSON} from './storage.js';
import {acquireLock} from './lock.js';
export interface MediaCatalog {version:'QC_MEDIA_V1';approvalBasis:string;items:Record<string,{audioFile:string;audioHash:string;topicHash:string}>}
export function bindingHash(topic:Topic):string{return createHash('sha256').update(JSON.stringify([topic.id,topic.title,topic.script])).digest('hex');}
export async function mediaCatalog(c:Config):Promise<MediaCatalog>{
 const data=await readJSON<MediaCatalog>(path.join(c.stateDir,'media/catalog-manifest.json'));
 if(data?.version!=='QC_MEDIA_V1'||!data.items||typeof data.items!=='object')throw new PublisherError('MEDIA_REGISTRATION_REQUIRED');
 return data;
}
export async function registerMedia(c:Config){
 const lease=await acquireLock(c.stateDir);
 try{
  const file=path.join(c.stateDir,'media/catalog-manifest.json');
  if(await readJSON(file))throw new PublisherError('MEDIA_ALREADY_REGISTERED');
  const topics=JSON.parse((await fs.readFile(c.topics,'utf8')).replace(/^\uFEFF/,'')) as Topic[];
  const files=(await fs.readdir(c.audioDir)).filter(f=>/\.(mp3|wav|m4a)$/i.test(f));
  const items:MediaCatalog['items']=Object.create(null);
  for(let i=0;i<topics.length;i++){
   const topic=topics[i];if(!topic||typeof topic.id!=='string'||typeof topic.script!=='string'||typeof topic.title!=='string'||Object.hasOwn(items,topic.id))throw new PublisherError('POOL_TOPIC_INVALID');
   const matches=files.filter(f=>Number(f.match(/(\d+)(?=\.[^.]+$)/)?.[1])===i+1);
   if(!matches.length)continue;if(matches.length!==1)throw new PublisherError('POOL_AUDIO_MISSING_OR_AMBIGUOUS');
   const full=path.join(c.audioDir,matches[0]),stat=await fs.lstat(full);if(!stat.isFile()||stat.isSymbolicLink())throw new PublisherError('INVALID_MEDIA_FILE');
   items[topic.id]={audioFile:matches[0],audioHash:createHash('sha256').update(await fs.readFile(full)).digest('hex'),topicHash:bindingHash(topic)};
  }
  if(!Object.keys(items).length)throw new PublisherError('NO_MEDIA_TO_REGISTER');
  lease.assert();await saveJSON(file,{version:'QC_MEDIA_V1',approvalBasis:'operator-supplied baseline; no speech transcription asserted',items} satisfies MediaCatalog);
  return {registered:Object.keys(items).length,transcriptionVerified:false};
 }finally{await lease.release();}
}
