import fs from 'node:fs/promises';
import path from 'node:path';
import {randomUUID} from 'node:crypto';
import {readJSON,saveJSON} from './storage.js';
import {acquireLock} from './lock.js';
import {PublisherError} from './types.js';
interface Candidate {file:string;size:number;mtime:number}
export async function maintain(stateDir:string,apply=false,now=Date.now()){
 const lease=await acquireLock(stateDir);
 try{
  const root=await fs.realpath(stateDir),candidates:Candidate[]=[];
  for(const mode of ['publish','render']){
   const jobs=path.join(root,mode,'jobs');
   let names:string[];try{names=await fs.readdir(jobs);}catch(e){if((e as NodeJS.ErrnoException).code==='ENOENT')continue;throw e;}
   for(const name of names){
    if(!/^\d{4}-\d{2}-\d{2}\.slot[12]\.json$/.test(name))continue;
    const job=await readJSON<{version:string;id:string;status:string;deliveries?:Record<string,{status:string;receipt?:{id:string}}> }>(path.join(jobs,name));
    if(job?.version!=='QC_PUBLISH_JOB_V1'||job.id+'.json'!==name)throw new PublisherError('RETENTION_INVALID_JOB');
    if(!['completed','abandoned'].includes(job.status))continue;
    const deliveries=Object.values(job.deliveries||{});
    if(deliveries.some(d=>['uncertain','submitting'].includes(d.status)))continue;
    if(mode==='publish'&&job.status==='completed'&&(!deliveries.length||deliveries.some(d=>d.status!=='confirmed'||!d.receipt?.id)))throw new PublisherError('RETENTION_INVALID_RECEIPTS');
    const dir=path.join(root,mode,'artifacts',job.id);let entries:string[];
    try{if((await fs.lstat(dir)).isSymbolicLink())throw new PublisherError('RETENTION_PATH_ESCAPE');if(!(await fs.realpath(dir)).startsWith(root+path.sep))throw new PublisherError('RETENTION_PATH_ESCAPE');entries=await fs.readdir(dir);}catch(e){if((e as NodeJS.ErrnoException).code==='ENOENT')continue;throw e;}
    for(const entry of entries){
     if(!['video.mp4','english-narration.mp3','english-illustration.png','english-content.json'].includes(entry)&&!/^\.render-[a-f0-9-]+\.mp4$/.test(entry))continue;
     const file=path.join(dir,entry),stat=await fs.lstat(file);
     if(!stat.isFile()||stat.isSymbolicLink()||now-stat.mtimeMs<30*86400000)continue;
     candidates.push({file,size:stat.size,mtime:stat.mtimeMs});
    }
   }
  }
  const manifest={version:'QC_RETENTION_V1',createdAt:new Date(now).toISOString(),files:candidates,removed:[] as string[]};
  if(apply){const file=path.join(root,'maintenance',randomUUID()+'.json');lease.assert();await saveJSON(file,manifest);
   for(const c of candidates){lease.assert();const stat=await fs.lstat(c.file);if(stat.isSymbolicLink()||!stat.isFile()||stat.size!==c.size||stat.mtimeMs!==c.mtime)throw new PublisherError('RETENTION_FILE_CHANGED');await fs.unlink(c.file);manifest.removed.push(c.file);await saveJSON(file,manifest);}
  }
  return {mode:apply?'apply':'plan',candidateFiles:candidates.length,removedFiles:manifest.removed.length,bytes:candidates.reduce((n,c)=>n+c.size,0),receiptsPreserved:true,credentialsPreserved:true};
 }finally{await lease.release();}
}
