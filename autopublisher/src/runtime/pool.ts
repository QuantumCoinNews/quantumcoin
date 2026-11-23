import {mediaCatalog,bindingHash} from './mediaCatalog.js';
import {titleKey} from './topicHistory.js';
import fs from 'node:fs/promises';
import path from 'node:path';
import { createHash } from 'node:crypto';
import { createReadStream } from 'node:fs';
import { Config, PoolAsset, PublisherError, Topic } from './types.js';

export async function hashFile(file: string): Promise<string> {
  const stat = await fs.lstat(file); if (!stat.isFile() || stat.isSymbolicLink() || stat.size === 0) throw new PublisherError('INVALID_MEDIA_FILE');
  const hash = createHash('sha256'); for await (const chunk of createReadStream(file)) hash.update(chunk); return hash.digest('hex');
}
export async function selectPool(c: Config, idx: number): Promise<PoolAsset> {
  let topics: unknown; try { const stat=await fs.lstat(c.topics);if(!stat.isFile()||stat.size>8*1024*1024)throw Error();topics = JSON.parse((await fs.readFile(c.topics, 'utf8')).replace(/^\uFEFF/,'')); } catch { throw new PublisherError('POOL_TOPICS_UNREADABLE'); }
  if (!Array.isArray(topics) || topics.length === 0 || topics.length > 10000 || !Number.isSafeInteger(idx) || idx < 1 || idx > topics.length) throw new PublisherError('POOL_INDEX_INVALID');
  const ids=new Set<string>(),titles=new Set<string>();
  for(const item of topics as Topic[]){
   if(!item||typeof item.id!=='string'||!item.id.trim()||typeof item.title!=='string'||!item.title.trim()||item.title.length>100||typeof item.script!=='string'||!item.script.trim())throw new PublisherError('POOL_TOPIC_INVALID');
   for(const key of ['x','telegram','youtube_description'] as const)if(item[key]!==undefined&&typeof item[key]!=='string')throw new PublisherError('POOL_TOPIC_INVALID');
   const key=titleKey(item.title),id=item.id.normalize('NFC').toLowerCase();
   if(ids.has(id)||titles.has(key))throw new PublisherError('POOL_CATALOG_DUPLICATE');ids.add(id);titles.add(key);
   if((item.x?.length||0)>280||(item.telegram?.length||0)>1024||(item.youtube_description?.length||0)>4500)throw new PublisherError('POOL_CAPTION_OUT_OF_BOUNDS');
  }
  const topic = topics[idx - 1] as Topic;
  if (!topic || typeof topic.id !== 'string' || typeof topic.title !== 'string' || !topic.title.trim() || typeof topic.script !== 'string' || !topic.script.trim()) throw new PublisherError('POOL_TOPIC_INVALID');
  const ai=c.env.QC_PUBLISHER_AI_ENABLED==='1';
  const audioFiles = (ai?[]:await fs.readdir(c.audioDir)).filter(n => /\.(mp3|wav|m4a)$/i.test(n));
  const catalog=ai?undefined:await mediaCatalog(c);
  const record=catalog&&Object.hasOwn(catalog.items,topic.id)?catalog.items[topic.id]:undefined;
  if(!ai&&(!record||record.topicHash!==bindingHash(topic)||path.basename(record.audioFile)!==record.audioFile))throw new PublisherError('MEDIA_PROVENANCE_MISMATCH');
  const matches=ai?[]:audioFiles.filter(n=>n===record!.audioFile);
  if (!ai && matches.length !== 1) throw new PublisherError('POOL_AUDIO_MISSING_OR_AMBIGUOUS');
  const generatedVisual=ai&&c.env.QC_PUBLISHER_AI_VISUALS==='1';
  const backgrounds = (generatedVisual?[]:await fs.readdir(c.backgroundDir)).filter(n => /\.(png|jpg|jpeg|mp4)$/i.test(n)).sort((a,b) => a.localeCompare(b,'en',{numeric:true}));
  if (!generatedVisual && backgrounds.length === 0) throw new PublisherError('POOL_BACKGROUND_MISSING');
  const audio = ai?'':path.join(c.audioDir, matches[0]); const background = generatedVisual?'':path.join(c.backgroundDir, backgrounds[(idx - 1) % backgrounds.length]);
  if(!ai&&await hashFile(audio)!==record!.audioHash)throw new PublisherError('MEDIA_PROVENANCE_MISMATCH');
  const fingerprint = createHash('sha256').update(JSON.stringify([topic,ai?'generated-english-audio':await hashFile(audio),generatedVisual?'generated-illustration':await hashFile(background)])).digest('hex');
  const preparedIndexes=(topics as Topic[]).map((t,i)=>({t,i})).filter(({t})=>catalog&&Object.hasOwn(catalog.items,t.id)&&audioFiles.includes(catalog.items[t.id].audioFile)).map(({i})=>i+1);
  const next=ai?(idx===topics.length?1:idx+1):(preparedIndexes.find(n=>n>idx)||preparedIndexes[0]);
  return {idx,next,topic,audio,background,fingerprint};
}
