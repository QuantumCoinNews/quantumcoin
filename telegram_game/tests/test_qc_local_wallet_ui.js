"use strict";
const {test}=require("node:test"),assert=require("node:assert/strict"),vm=require("node:vm"),fs=require("node:fs"),path=require("node:path");
const source=fs.readFileSync(path.join(__dirname,"../telegram_game/dev_ui/qc_local_wallet.js"),"utf8");
const ACTIVE="qc_local_wallet_active_v1",PREFIX="qc_local_wallet_vault_v1:";
const address="1FLm5EcKtdBGNYg42Q21tfoXFwA5KYZy8W",words="test-only recovery phrase",password="test-only-long-password";
const vault=JSON.stringify({version:"QC_ENCRYPTED_MNEMONIC_V1",address,ciphertext:"test-only-ciphertext"});
function browser(options={}){
 const fields=Object.fromEntries(["words","expected","password","confirm","address","file"].map(k=>[k,{value:"",textContent:"",addEventListener(n,fn){this.handler=fn;}}]));
 fields.password.value=fields.confirm.value=password;
 const status={textContent:""},events={},timers=new Map(),requests=[],messages=[],store=new Map(options.store||[]);
 let handler,worker,lastId=0;
 const panel={querySelector(s){return s==='[role="status"]'?status:fields[s.match(/data-field="([^"]+)"/)[1]];},addEventListener(n,fn){handler=fn;}};
 class Worker{
  constructor(){worker=this;this.terminated=false;}
  postMessage({id,request}){messages.push(request);if(options.hold){this.held={id,request};return;}
   let result={ok:true};if(["restore","create"].includes(request.action))result=options.failRestore&&request.action==="restore"?{ok:false,error:"invalid words"}:{ok:true,address,vault,...(request.action==="create"?{words}:{} )};
   else if(request.action==="reveal")result={ok:true,address,words};
   else if(request.action==="unlock")result={ok:true,address};
   else if(request.action==="link")result={ok:true,proof:{playerId:"tg_42",address,nonce:"2".repeat(64),publicKey:"public-only",signature:"public-only"}};
   Promise.resolve().then(()=>this.onmessage?.({data:{id,result}}));
  }
  terminate(){this.terminated=true;}
 }
 const listen=(n,fn)=>{(events[n]||=[]).push(fn);};
 const context={window:{isSecureContext:true,confirm:()=>options.cancel!==true,addEventListener:listen,QCPlayer:{ready:Promise.resolve({ok:true}),id:()=>"tg_42"},QCAccount:{snapshot:null,refresh:async()=>{}}},
  document:{readyState:"complete",hidden:false,getElementById:()=>({appendChild(){}}),createElement:tag=>tag==="section"?panel:{click(){}},addEventListener:listen},
  Worker,localStorage:{getItem:k=>store.get(k)||null,setItem(k,v){if(options.storageFails)throw Error("storage unavailable");store.set(k,v);}},
  crypto:{randomUUID:()=>String(++lastId)},URL:{createObjectURL:()=>"blob:test",revokeObjectURL(){}},Blob,
  setTimeout(fn,ms){if(ms===0){queueMicrotask(fn);return 0;}const id=++lastId;timers.set(id,fn);return id;},clearTimeout:id=>timers.delete(id),
  fetch:async(url,init)=>{requests.push({url,body:JSON.parse(init.body)});return {ok:true,json:async()=>url.endsWith("challenge")?{ok:true,challenge:{networkID:"testnet",genesisHex:"1".repeat(64)}}:{ok:true}};},console};
 vm.runInNewContext(source,context);
 return {fields,status,requests,messages,store,events,timers,context,get worker(){return worker;},click:action=>handler({target:{dataset:{action}}})};
}
test("restore persists only ciphertext and preserves all old recovery data",async()=>{
 const b=browser({store:[["qc_wallet_seed_v2","old-legacy-migration-data"],[PREFIX+"old","old-encrypted-vault"],[ACTIVE,PREFIX+"old"]]});
 b.fields.words.value=words;await b.click("restore");
 assert.equal(b.store.get("qc_wallet_seed_v2"),"old-legacy-migration-data");assert.equal(b.store.get(PREFIX+"old"),"old-encrypted-vault");
 assert.equal(b.store.get(b.store.get(ACTIVE)),vault);assert.equal(b.requests.length,0);
 assert.equal(b.fields.words.value,"");assert.equal(b.fields.password.value,"");
 assert(![...b.store.values()].includes(words));assert(![...b.store.values()].includes(password));
});
test("failed recovery and cancelled new wallet never replace selection",async()=>{
 const prior=[[ACTIVE,PREFIX+"old"],[PREFIX+"old",vault]];
 const b=browser({store:prior,failRestore:true});b.fields.words.value="wrong words";await b.click("restore");
 assert.equal(b.store.get(ACTIVE),PREFIX+"old");assert.match(b.status.textContent,/NOT recovered/);assert.equal(b.messages.filter(r=>r.action==="create").length,0);
 const c=browser({store:prior,cancel:true});await c.click("create");assert.equal(c.messages.length,0);assert.equal(c.store.get(ACTIVE),PREFIX+"old");
});
test("storage failure preserves old selection; reveal erases password and lock clears words",async()=>{
 const b=browser({store:[[ACTIVE,PREFIX+"old"],[PREFIX+"old",vault]],storageFails:true});await b.click("restore");assert.equal(b.store.get(ACTIVE),PREFIX+"old");
 b.fields.password.value=password;await b.click("reveal");assert.equal(b.fields.words.value,words);assert.equal(b.fields.password.value,"");
 await b.click("lock");assert.equal(b.fields.words.value,"");assert.equal(b.worker.terminated,true);
});
test("locking cancels pending restore and ignores late worker responses",async()=>{
 const b=browser({hold:true});const task=b.click("restore");const w=b.worker;await b.click("lock");await task;
 w.onmessage({data:{id:w.held.id,result:{ok:true,address,vault}}});assert.equal(b.store.size,0);assert.equal(b.fields.words.value,"");
});
test("Telegram linking sends only public challenge request and signed proof",async()=>{
 const b=browser({store:[[ACTIVE,PREFIX+"old"],[PREFIX+"old",vault]]});b.fields.words.value=words;await b.click("link");
 assert.equal(b.requests.length,2);const wire=JSON.stringify(b.requests);assert(!wire.includes(words));assert(!wire.includes(password));assert(!wire.includes("ciphertext"));
 assert.deepEqual(Object.keys(b.requests[1].body).sort(),["address","nonce","playerId","publicKey","signature"]);
});
test("another tab's wallet selection change cancels unlock and hides words",async()=>{
 const b=browser({store:[[ACTIVE,PREFIX+"old"],[PREFIX+"old",vault]]});await b.click("reveal");assert.equal(b.fields.words.value,words);
 b.events.storage[0]({key:ACTIVE});assert.equal(b.fields.words.value,"");assert.equal(b.worker.terminated,true);
});
test("encrypted backup import preserves previous ciphertext and clears its unlocked worker",async()=>{
 const b=browser({store:[[ACTIVE,PREFIX+"old"],[PREFIX+"old",vault]]});
 await b.fields.file.handler({target:{files:[{size:vault.length,text:async()=>vault}],value:"file"}});
 assert.equal(b.store.get(PREFIX+"old"),vault);assert.notEqual(b.store.get(ACTIVE),PREFIX+"old");assert.equal(b.store.get(b.store.get(ACTIVE)),vault);
 assert.equal(b.worker.terminated,true);assert.equal(b.fields.password.value,"");assert.equal(b.requests.length,0);
});
