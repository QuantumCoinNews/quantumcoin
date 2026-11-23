import { spawn } from 'node:child_process';
import fs from 'node:fs/promises';
import path from 'node:path';
import { root } from './config.js';
import { PublisherError } from './types.js';

export interface Lease { assert(): void; release(): Promise<void> }
export async function acquireLock(stateDir: string): Promise<Lease> {
  await fs.mkdir(stateDir,{recursive:true,mode:0o700});
  const absolute = await fs.realpath(stateDir);
  const child = spawn(process.platform === 'win32' ? 'pwsh.exe' : 'pwsh', ['-NoProfile','-NonInteractive','-ExecutionPolicy','Bypass','-File',path.join(root,'scripts/hold-runtime-lock.ps1'),'-LockPath',path.join(absolute,'runtime.lock')],{stdio:['pipe','pipe','pipe'],windowsHide:true});
  let alive = true; child.on('exit',()=>{alive=false;}); child.stderr.resume();
  try {
    await new Promise<void>((resolve,reject)=>{
      const timer = setTimeout(()=>reject(new PublisherError('LOCK_HOST_TIMEOUT')),30000);
      let output='';
      const finish=(error?: Error) => {clearTimeout(timer);error ? reject(error) : resolve();};
      child.once('error',()=>finish(new PublisherError('LOCK_HOST_UNAVAILABLE')));
      child.once('exit',()=>{ if(!output.includes('LOCKED'))finish(new PublisherError(output.includes('BUSY')?'RUN_IN_PROGRESS':'LOCK_HOST_FAILED')); });
      child.stdout.on('data',(data:Buffer)=>{output+=data.toString();if(output.includes('LOCKED'))finish();else if(output.includes('BUSY'))finish(new PublisherError('RUN_IN_PROGRESS'));});
    });
  } catch(error) {child.stdin.end();child.kill();throw error;}
  return {assert(){if(!alive)throw new PublisherError('RUNTIME_LOCK_LOST');},async release(){
    child.stdin.end(); if(!alive)return;
    await new Promise<void>(resolve=>{const timer=setTimeout(()=>{child.kill();resolve();},5000);child.once('exit',()=>{clearTimeout(timer);resolve();});});
  }};
}
