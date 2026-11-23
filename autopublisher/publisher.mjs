import fs from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {createHash} from 'node:crypto';
import {spawn} from 'node:child_process';
const root=path.resolve(fileURLToPath(new URL('.',import.meta.url)));
let child;
try{
 if(Number(process.versions.node.split('.')[0])<22)throw Error('NODE_22_REQUIRED');
 const lockDir=path.join(root,'.build');await fs.mkdir(lockDir,{recursive:true});
 child=spawn(process.platform==='win32'?'pwsh.exe':'pwsh',['-NoProfile','-NonInteractive','-File',path.join(root,'scripts/hold-build-lock.ps1'),'-LockPath',path.join(lockDir,'runtime.lock')],{windowsHide:true,stdio:['pipe','pipe','pipe']});
 child.stderr.resume();child.stdin.on('error',()=>{});
 await new Promise((resolve,reject)=>{const timer=setTimeout(()=>reject(Error('BUILD_LOCK_TIMEOUT')),30000);let out='';const done=e=>{clearTimeout(timer);e?reject(e):resolve();};child.once('error',()=>done(Error('BUILD_LOCK_UNAVAILABLE')));child.once('exit',()=>{if(!out.includes('LOCKED'))done(Error('BUILD_BUSY'));});child.stdout.on('data',b=>{out+=b.toString();if(out.includes('LOCKED'))done();else if(out.includes('BUSY'))done(Error('BUILD_BUSY'));});});
 const manifest=JSON.parse((await fs.readFile(path.join(root,'dist/build-manifest.json'),'utf8')).replace(/^\uFEFF/,''));
 if(manifest.version!=='QC_BUILD_V1'||!Array.isArray(manifest.source)||!Array.isArray(manifest.compiled)||!manifest.source.length||!manifest.compiled.some(x=>x.path==='dist/runtime/cli.js'))throw Error('BUILD_REQUIRED');
 const sharedSchema=path.resolve(root,'../config/runtime-environment.schema.json');
 if(typeof manifest.sharedEnvironmentSchemaHash!=='string')throw Error('BUILD_REQUIRED');
 if((await fs.lstat(sharedSchema)).isSymbolicLink()||createHash('sha256').update(await fs.readFile(sharedSchema)).digest('hex')!==manifest.sharedEnvironmentSchemaHash)throw Error('SOURCE_RUNTIME_DRIFT');
 for(const entry of [...manifest.source,...manifest.compiled]){
  if(typeof entry.path!=='string'||typeof entry.sha256!=='string')throw Error('BUILD_REQUIRED');
  const file=path.resolve(root,entry.path);if(!file.startsWith(root+path.sep)||(await fs.lstat(file)).isSymbolicLink())throw Error('BUILD_REQUIRED');
  if(createHash('sha256').update(await fs.readFile(file)).digest('hex')!==entry.sha256)throw Error('SOURCE_RUNTIME_DRIFT');
 }
 const {main}=await import('./dist/runtime/cli.js');process.exitCode=await main(process.argv.slice(2));
}catch(error){const code=['BUILD_BUSY','NODE_22_REQUIRED','BUILD_LOCK_TIMEOUT','BUILD_LOCK_UNAVAILABLE','SOURCE_RUNTIME_DRIFT'].includes(error?.message)?error.message:'BUILD_REQUIRED';console.error(JSON.stringify({status:'failed',code,published:false}));process.exitCode=code==='BUILD_BUSY'?73:1;}
finally{if(child){child.stdin.end();if(child.exitCode===null)await new Promise(resolve=>{const timer=setTimeout(()=>{child.kill();resolve();},5000);child.once('exit',()=>{clearTimeout(timer);resolve();});});}}
