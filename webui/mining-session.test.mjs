import assert from 'node:assert/strict';
import test from 'node:test';
import {solveCurrentJob} from './dist/mining-session.mjs';

function setup(extra={}) {
 const worker={terminated:0,postMessage(){},terminate(){this.terminated++;}};
 const controller=new AbortController();
 const promise=solveCurrentJob({job:{expires:Date.now()/1000+60},signal:controller.signal,
   status:async()=>({current:true}),pollMillis:2,workerFactory:()=>worker,...extra});
 return {worker,controller,promise};
}
test('stale work restarts and releases the device worker',async()=>{
 const run=setup({status:async()=>({current:false})});
 assert.deepEqual(await run.promise,{restart:true});assert.equal(run.worker.terminated,1);
});
test('same-height hash changes are handled by the status contract',async()=>{
 const original={height:4,hash:'a'},reorg={height:4,hash:'b'};
 const run=setup({status:async()=>({current:original.height===reorg.height&&original.hash===reorg.hash})});
 assert.deepEqual(await run.promise,{restart:true});
});
test('stop cancels hashing and prevents a late result',async()=>{
 const run=setup();run.controller.abort();
 await assert.rejects(run.promise,/mining_stopped/);
 run.worker.onmessage({data:{nonce:'5'}});assert.equal(run.worker.terminated,1);
});
test('a late status response cannot discard a found nonce',async()=>{
 let release,started;const waiting=new Promise(r=>started=r);
 const run=setup({status:()=>{started();return new Promise(r=>release=r);}});
 await waiting;run.worker.onmessage({data:{nonce:'9007199254740993',hash:'hash'}});
 release({current:false});
 assert.deepEqual(await run.promise,{solution:{nonce:'9007199254740993',hash:'hash'}});
 assert.equal(run.worker.terminated,1);
});
test('expiry rotates work even when node status is unavailable',async()=>{
 const run=setup({job:{expires:0},status:async()=>{throw Error('offline');}});
 assert.deepEqual(await run.promise,{restart:true});
});
test('network mismatch stops instead of mining a second chain',async()=>{
 const run=setup({status:async()=>({ok:false,error:'mining_network_changed'})});
 await assert.rejects(run.promise,/mining_network_changed/);
});
