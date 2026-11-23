"use strict";
const assert=require("node:assert/strict"),fs=require("node:fs"),vm=require("node:vm"),path=require("node:path");
const {test}=require("node:test"),{webcrypto}=require("node:crypto");
const source=fs.readFileSync(path.join(__dirname,"../telegram_game/dev_ui/wallet_recovery.js"),"utf8");
const LEGACY="qc_wallet_seed_v2",VAULT="qc_wallet_recovery_encrypted_v1";
const plain="test-only legacy recovery words",pass="test-only-long-recovery-passphrase";
function lockManager(){let queue=Promise.resolve();return {request(name,fn){const result=queue.then(fn);queue=result.catch(()=>{});return result;}};}
function browser(store=new Map([[LEGACY,plain]]),options={}) {
  const handlers={},events={},timers=new Map(),writes=[],downloads=[];
  const password={value:pass},ack={checked:false},output={value:"",hidden:true},status={textContent:""};
  const fileInput={value:"",addEventListener(name,fn){handlers.file=fn;}};
  const elements={'input[type="password"]':password,'input[type="checkbox"]':ack,
    'textarea':output,'[role="status"]':status,'input[type="file"]':fileInput};
  const panel={querySelector(selector){return elements[selector];},addEventListener(name,fn){handlers[name]=fn;}};
  const listen=(name,fn)=>{(events[name] ||= []).push(fn);};
  const subtle=new Proxy(webcrypto.subtle,{get(target,name){
    const method=target[name];if(typeof method!=="function")return method;
    return async (...args)=>{const result=await method.apply(target,args);if(name==="decrypt"&&options.afterDecrypt)await options.afterDecrypt();return result;};
  }});
  const context={window:{isSecureContext:options.secure!==false,addEventListener:listen},
    document:{readyState:"complete",hidden:false,addEventListener:listen,
      getElementById(id){return id==="walletPanel"?{appendChild(){}}:null;},
      createElement(tag){return tag==="section"?panel:{click(){if(options.failDownload)throw Error("download failed");downloads.push(this.download);}};}},
    navigator:{locks:options.noLocks?undefined:(options.locks||lockManager())},
    localStorage:{getItem(k){if(options.failGet)throw Error("storage read failed");if(k===VAULT&&options.badRead&&store.has(k))return "wrong-read-back";return store.get(k)||null;},
      setItem(k,v){assert.equal(k,VAULT,"Never persist new plaintext recovery");if(options.failSet)throw Error("quota exceeded");writes.push([k,v]);store.set(k,v);},
      removeItem(k){if(options.failRemove)throw Error("storage removal denied");store.delete(k);}},
    URL:{createObjectURL(){return "blob:test-only";},revokeObjectURL(){}},Blob,
    setTimeout(fn,ms){const id={};timers.set(id,{fn,ms});return id;},clearTimeout(id){timers.delete(id);},
    crypto:{subtle,getRandomValues:webcrypto.getRandomValues.bind(webcrypto)},
    fetch(){throw Error("Recovery must never send HTTP");},TextEncoder,TextDecoder,Uint8Array,btoa,atob};
  vm.createContext(context);vm.runInContext(source,context);
  return {store,options,password,ack,output,status,writes,downloads,context,api:context.window.QCWalletRecovery,
    click(action){return handlers.click({target:{dataset:{action}}});},
    emit(name,data={}){for(const fn of events[name]||[])fn(data);},
    timeout(){for(const {fn,ms} of [...timers.values()])if(ms===60000)fn();},
    import(raw,extra={}){fileInput.files=[{size:Buffer.byteLength(raw),text:async()=>raw,...extra}];fileInput.value="chosen";return handlers.file({target:fileInput});}};
}
const cryptoOnly=browser(new Map());
const fixture=cryptoOnly.api.encrypt(plain,pass).then(v=>JSON.stringify(v));
test("existing V1 encrypted backups remain compatible",async()=>{
  // Independently reproduce the original V1 envelope using WebCrypto, without
  // the new encrypt helper. Preserve the existing KDF and authenticated domain.
  const salt=webcrypto.getRandomValues(new Uint8Array(16)),iv=webcrypto.getRandomValues(new Uint8Array(12));
  const material=await webcrypto.subtle.importKey("raw",new TextEncoder().encode(pass),"PBKDF2",false,["deriveKey"]);
  const key=await webcrypto.subtle.deriveKey({name:"PBKDF2",salt,iterations:600000,hash:"SHA-256"},material,{name:"AES-GCM",length:256},false,["encrypt"]);
  const encrypted=await webcrypto.subtle.encrypt({name:"AES-GCM",iv,additionalData:new TextEncoder().encode("QC_GAME_RECOVERY_V1")},key,new TextEncoder().encode(plain));
  const v1={version:"QC_GAME_RECOVERY_V1",iterations:600000,salt:Buffer.from(salt).toString("base64"),iv:Buffer.from(iv).toString("base64"),ciphertext:Buffer.from(encrypted).toString("base64")};
  assert.equal(await cryptoOnly.api.decrypt(v1,pass),plain);
});
async function seeded(options={}){return browser(new Map([[LEGACY,plain],[VAULT,await fixture]]),options);}
async function ready(b){await b.click("download");b.ack.checked=true;}
function pause(){let resume,arrived;const entered=new Promise(r=>arrived=r),pending=new Promise(r=>resume=r);return {entered,resume:()=>resume(),wait:async()=>{arrived();await pending;}};}

test("AES-GCM round trip, independent salt/IV, no plaintext or password in envelope",async()=>{
  const a=JSON.parse(await fixture),b=await cryptoOnly.api.encrypt(plain,pass);
  assert.notEqual(a.salt,b.salt);assert.notEqual(a.iv,b.iv);assert.notEqual(a.ciphertext,b.ciphertext);
  assert.equal(await cryptoOnly.api.decrypt(a,pass),plain);assert(!JSON.stringify(a).includes(plain));assert(!JSON.stringify(a).includes(pass));
});
test("wrong password and modified ciphertext fail authentication",async()=>{
  const a=JSON.parse(await fixture);await assert.rejects(cryptoOnly.api.decrypt(a,"wrong-but-long-passphrase"));
  await assert.rejects(cryptoOnly.api.decrypt({...a,ciphertext:a.ciphertext.slice(0,-8)+"AAAAAAAA"},pass));
});
test("strict schema, KDF, base64, size and password bounds",async()=>{
  const a=JSON.parse(await fixture);
  for(const bad of [{...a,iterations:1},{...a,version:"other"},{...a,seed:plain},{...a,salt:"bad"},{...a,iv:""},{...a,ciphertext:"A".repeat(20001)}])await assert.rejects(cryptoOnly.api.decrypt(bad,pass));
  for(const text of ["","a".repeat(8193)])await assert.rejects(cryptoOnly.api.encrypt(text,pass));
  for(const password of ["short","a".repeat(1025)])await assert.rejects(cryptoOnly.api.encrypt(plain,password));
});
test("ordinary page load never loads, displays or rewrites legacy seed",()=>{
  const b=browser();assert.equal(b.output.value,"");assert.equal(b.writes.length,0);assert.equal(b.store.get(LEGACY),plain);
});
test("encryption keeps plaintext until verified backup and acknowledgement",async()=>{
  const b=browser();await b.click("encrypt");const raw=b.store.get(VAULT);
  assert(raw&&!raw.includes(plain));assert.equal(b.downloads.length,1);assert.equal(b.store.get(LEGACY),plain);
  await b.click("remove");assert.equal(b.store.get(LEGACY),plain);
  b.ack.checked=true;await b.click("remove");assert(!b.store.has(LEGACY));assert.equal(b.store.get(VAULT),raw);
  assert.equal(await b.api.decrypt(JSON.parse(raw),pass),plain);
});
test("wrong removal password cannot delete plaintext",async()=>{
  const b=await seeded();await ready(b);b.password.value="wrong-but-long-passphrase";await b.click("remove");assert.equal(b.store.get(LEGACY),plain);
});
test("reload requires renewed backup verification and supports re-download",async()=>{
  const first=browser();await first.click("encrypt");const b=browser(first.store);
  await b.click("unlock");b.ack.checked=true;await b.click("remove");assert.equal(b.store.get(LEGACY),plain);
  await ready(b);await b.click("remove");assert(!b.store.has(LEGACY));assert.equal(b.downloads.length,1);
});
test("valid encrypted file restores on a fresh device without automatic disclosure",async()=>{
  const b=browser(new Map());await b.import(await fixture);assert(b.store.has(VAULT));assert.equal(b.output.value,"");
  await b.click("unlock");assert.equal(b.output.value,plain);assert(!b.store.has(LEGACY));
});
test("bad password, corrupted or oversized import cannot mutate storage",async()=>{
  for(const [raw,password,extra] of [[await fixture,"wrong-but-long-passphrase",{}],["{bad",pass,{}],[await fixture,pass,{size:20001}]]){
    const b=browser();b.password.value=password;await b.import(raw,extra);assert(!b.store.has(VAULT));assert.equal(b.store.get(LEGACY),plain);assert.equal(b.output.value,"");
  }
});
test("existing encrypted recovery is never overwritten by import",async()=>{
  const b=await seeded(),old=b.store.get(VAULT);await b.import(await fixture);assert.equal(b.store.get(VAULT),old);assert.equal(b.writes.length,0);
});
test("quota failure preserves legacy and permits retry",async()=>{
  const b=browser(undefined,{failSet:true});await b.click("encrypt");assert.equal(b.store.get(LEGACY),plain);assert(!b.store.has(VAULT));
  b.options.failSet=false;await b.click("encrypt");assert(b.store.has(VAULT));
});
test("failed remove preserves both recovery copies",async()=>{
  const b=await seeded({failRemove:true});await ready(b);await b.click("remove");assert.equal(b.store.get(LEGACY),plain);assert(b.store.has(VAULT));
});
test("failed storage read-back cannot authorize deletion",async()=>{
  const b=browser(undefined,{badRead:true});await b.click("encrypt");assert.equal(b.store.get(LEGACY),plain);b.ack.checked=true;await b.click("remove");assert.equal(b.store.get(LEGACY),plain);
});
test("unsupported storage locking and insecure contexts fail closed",async()=>{
  for(const options of [{noLocks:true},{secure:false},{failGet:true}]){
    const b=browser(undefined,options);await b.click("encrypt");assert.equal(b.store.get(LEGACY),plain);assert(!b.store.has(VAULT));
  }
});
test("visibility lock cancels in-flight unlock before disclosure",async()=>{
  const gate=pause(),b=await seeded({afterDecrypt:gate.wait});const op=b.click("unlock");await gate.entered;
  b.context.document.hidden=true;b.emit("visibilitychange");gate.resume();await op;
  assert.equal(b.output.value,"");assert.equal(b.password.value,"");assert.equal(b.output.hidden,true);
});
test("page hide cancels in-flight encryption before persistence",async()=>{
  const gate=pause(),b=browser(undefined,{afterDecrypt:gate.wait});const op=b.click("encrypt");await gate.entered;b.emit("pagehide");gate.resume();await op;
  assert(!b.store.has(VAULT));assert.equal(b.store.get(LEGACY),plain);
});
test("manual lock cancels in-flight deletion",async()=>{
  const b=await seeded();await ready(b);const gate=pause();b.options.afterDecrypt=gate.wait;
  const op=b.click("remove");await gate.entered;await b.click("lock");gate.resume();await op;assert.equal(b.store.get(LEGACY),plain);
});
test("legacy changed during decryption is never deleted, even without an event",async()=>{
  const b=await seeded();await ready(b);const gate=pause();b.options.afterDecrypt=gate.wait;
  const op=b.click("remove");await gate.entered;b.store.set(LEGACY,"test-only-new-recovery");gate.resume();await op;assert.equal(b.store.get(LEGACY),"test-only-new-recovery");
});
test("vault changed during decryption invalidates deletion",async()=>{
  const b=await seeded();await ready(b);const gate=pause();b.options.afterDecrypt=gate.wait;
  const op=b.click("remove");await gate.entered;b.store.set(VAULT,"changed-in-another-tab");gate.resume();await op;assert.equal(b.store.get(LEGACY),plain);
});
test("late import cannot overwrite a vault created during decryption",async()=>{
  const gate=pause(),b=browser(undefined,{afterDecrypt:gate.wait});const op=b.import(await fixture);await gate.entered;
  b.store.set(VAULT,"newer-vault");gate.resume();await op;assert.equal(b.store.get(VAULT),"newer-vault");assert.equal(b.store.get(LEGACY),plain);
});
test("two cooperating tabs serialize writes and cannot replace a different password",async()=>{
  const store=new Map([[LEGACY,plain]]),locks=lockManager(),a=browser(store,{locks}),b=browser(store,{locks});
  b.password.value="another-test-only-passphrase";await Promise.all([a.click("encrypt"),b.click("encrypt")]);
  assert.equal(a.writes.length+b.writes.length,1);assert.equal(await a.api.decrypt(JSON.parse(store.get(VAULT)),pass),plain);
  assert.equal(store.get(LEGACY),plain);
});
test("page hide, BFCache return, storage events and timeout clear visible secrets",async()=>{
  for(const name of ["pagehide","pageshow","storage","timeout"]){
    const b=await seeded();await b.click("unlock");assert.equal(b.output.value,plain);
    if(name==="timeout")b.timeout();else b.emit(name,{key:VAULT});
    assert.equal(b.output.value,"");assert.equal(b.password.value,"");assert.equal(b.ack.checked,false);assert.equal(b.output.hidden,true);
    assert.match(b.status.textContent,name==="storage"?/changed/:/hidden/);
  }
});
test("storage events cancel import and disclose nothing",async()=>{
  const gate=pause(),b=browser(undefined,{afterDecrypt:gate.wait});const op=b.import(await fixture);await gate.entered;b.emit("storage",{key:null});gate.resume();await op;
  assert(!b.store.has(VAULT));assert.equal(b.output.value,"");assert.equal(b.store.get(LEGACY),plain);
});
test("failed download keeps a recoverable vault and retry enables safe migration",async()=>{
  const b=browser(undefined,{failDownload:true});await b.click("encrypt");assert(b.store.has(VAULT));assert.equal(b.store.get(LEGACY),plain);
  b.ack.checked=true;await b.click("remove");assert.equal(b.store.get(LEGACY),plain);
  b.options.failDownload=false;await ready(b);await b.click("remove");assert(!b.store.has(LEGACY));
});
test("failed unlock clears previously visible recovery",async()=>{
  const b=await seeded();await b.click("unlock");assert.equal(b.output.value,plain);
  b.password.value="wrong-but-long-passphrase";await b.click("unlock");assert.equal(b.output.value,"");assert.equal(b.output.hidden,true);
});
test("legacy script blocks cannot auto-load/save seed and recovery is mounted once",()=>{
  const html=fs.readFileSync(path.join(__dirname,"../telegram_game/dev_ui/game.html"),"utf8");
  assert(!/localStorage\.(getItem|setItem|removeItem)\(WALLET_SEED_KEY/.test(html));
  assert(!/walletState\.seed\s*=\s*(raw|s\.trim)/.test(html));
  assert.equal((html.match(/src="\/dev_ui\/wallet_recovery.js"/g)||[]).length,1);
  assert(!source.includes("navigator.clipboard"));assert(!source.includes("fetch("));
});
