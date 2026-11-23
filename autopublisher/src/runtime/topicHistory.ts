import fs from 'node:fs/promises';
import path from 'node:path';
import {readJSON} from './storage.js';
export function titleKey(title:string):string{return title.normalize('NFC').toLowerCase().replace(/[^\p{L}\p{N}]+/gu,' ').trim();}
export async function recentPublishedTitles(stateDir:string,currentJob:string):Promise<string[]>{
 const directory=path.join(stateDir,'publish','jobs');let files:string[];
 try{files=await fs.readdir(directory);}catch(e){if((e as NodeJS.ErrnoException).code==='ENOENT')return [];throw e;}
 const titles:string[]=[];
 for(const file of files.filter(f=>/^\d{4}-\d{2}-\d{2}\.slot[12]\.json$/.test(f)&&f!==currentJob+'.json').sort().reverse().slice(0,200)){
  const j=await readJSON<{deliveries?:Record<string,{status:string}>;prepared?:{topic:{title:string}};pool?:{topic:{title:string}}}>(path.join(directory,file));
  if(!j||!Object.values(j.deliveries||{}).some(d=>d.status==='confirmed'))continue;
  const title=j.prepared?.topic.title||j.pool?.topic.title;if(typeof title==='string')titles.push(title.slice(0,100));
 }
 return titles;
}
