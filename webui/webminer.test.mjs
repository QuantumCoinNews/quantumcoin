import assert from 'node:assert/strict';
import test from 'node:test';
test('browser miner keeps credentials in memory and continues on the canonical next tip',async t=>{
 const fields=new Map([['#address',{value:'test wallet'}],['#token',{value:'fixture-token'}],['#status',{}],['#start',{}],['#stop',{}]]);
 const requests=[];let jobs=0,workers=0,ready;
 const continued=new Promise(r=>ready=r),genesis='ab'.repeat(32);
 const base={protocol:'QC_WEB_BLOCK_POW_V1',miner:'test wallet',difficulty:4,genesis_hash:genesis,expires:Date.now()/1000+60};
 globalThis.document={querySelector:s=>fields.get(s)};globalThis.isSecureContext=false;
 globalThis.Worker=class {
   constructor(path){assert.equal(path,'/canonical-pow-worker.mjs');this.number=++workers;}
   postMessage(){if(this.number===1)queueMicrotask(()=>this.onmessage({data:{nonce:'9007199254740993',hash:'solution'}}));else ready();}
   terminate(){}
 };
 t.after(()=>{delete globalThis.document;delete globalThis.isSecureContext;delete globalThis.Worker;});
 t.mock.method(globalThis,'fetch',async(path,options)=>{
   requests.push({path,options});let data;
   if(path==='/api/mine/tip')data={genesis_hash:genesis,height:11,hash:'tip'};
   else if(path.startsWith('/api/mine/job'))data={...base,job_id:'job'+(++jobs),height:11+jobs,prev_hash:jobs===1?'tip':'solution'};
   else if(path==='/api/mine/submit')data={accepted:true,height:12,hash:'solution'};
   else assert.fail('unexpected request '+path);
   return {ok:true,json:async()=>data};
 });
 await import('./dist/webminer.mjs?continuous-test');
 await fields.get('#start').onclick();assert.equal(requests.length,0);
 globalThis.isSecureContext=true;const running=fields.get('#start').onclick();
 await continued;assert.equal(fields.get('#token').value,'');assert.equal(jobs,2);
 fields.get('#stop').onclick();await running;
 assert.equal(requests.filter(x=>x.path==='/api/mine/submit').length,1);
 for(const {options} of requests){assert.equal(options.headers['X-QC-API-KEY'],'fixture-token');assert.equal(options.redirect,'error');assert.equal(options.credentials,'omit');}
 assert.deepEqual(JSON.parse(requests.find(x=>x.path==='/api/mine/submit').options.body),{address:'test wallet',job_id:'job1',nonce:'9007199254740993'});
 assert.equal(fields.get('#start').disabled,false);
});
