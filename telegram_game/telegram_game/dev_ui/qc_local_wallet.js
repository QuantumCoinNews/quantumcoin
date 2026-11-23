/* QC only. Ciphertext persists; words/passwords never enter an API request. */
(function(){
  "use strict";
  const ACTIVE="qc_local_wallet_active_v1", PREFIX="qc_local_wallet_vault_v1:";
  let worker=null, serial=0, epoch=0, timer=null;
  const pending=new Map();
  function stop(){
    epoch++;clearTimeout(timer);worker?.terminate();worker=null;
    for(const {reject} of pending.values())reject(Error("Wallet locked"));pending.clear();
  }
  function call(request){
    if(!window.isSecureContext)return Promise.reject(Error("Use HTTPS or localhost for the local QC wallet"));
    if(!worker){
      worker=new Worker("/dev_ui/qc_wallet_worker.js");
      worker.onmessage=({data})=>{const p=pending.get(data.id);if(!p)return;pending.delete(data.id);data.result?.ok?p.resolve(data.result):p.reject(Error(data.result?.error||"Wallet operation failed"));};
      worker.onerror=()=>stop();
    }
    const id=++serial;
    return new Promise((resolve,reject)=>{pending.set(id,{resolve,reject});worker.postMessage({id,request});});
  }
  function current(){
    const key=localStorage.getItem(ACTIVE);if(!key)return null;
    if(!key.startsWith(PREFIX))throw Error("Saved wallet selection is invalid; restore words or an encrypted backup");
    const vault=localStorage.getItem(key);if(!vault)throw Error("Encrypted wallet missing; restore words or backup");
    const data=JSON.parse(vault);
    if(data.version!=="QC_ENCRYPTED_MNEMONIC_V1")throw Error("Unknown wallet backup format");
    return {key,vault,address:data.address};
  }
  async function persist(result){
    // Keep every old encrypted vault. Publish selection only after read-back.
    const key=PREFIX+crypto.randomUUID();
    localStorage.setItem(key,result.vault);
    if(localStorage.getItem(key)!==result.vault)throw Error("Encrypted wallet could not be saved");
    localStorage.setItem(ACTIVE,key);
    if(localStorage.getItem(ACTIVE)!==key)throw Error("Wallet could not be selected");
    return result.address;
  }
  function mount(){
    const host=document.getElementById("walletPanel");if(!host)return;
    const panel=document.createElement("section");panel.className="wallet-card";
    panel.innerHTML='<h3>Recoverable QC wallet</h3><p>Changed phone or reinstalled? Restore your QC recovery words here. Words stay on this device. A new wallet starts empty and cannot recover an old balance.</p><p data-field="address"></p><label>Recovery words <textarea data-field="words" autocomplete="off" spellcheck="false" maxlength="2048"></textarea></label><label>Old QC address, if known <input data-field="expected" autocomplete="off"></label><label>Local password (14+ characters) <input data-field="password" type="password" autocomplete="off" maxlength="1024"></label><label>Repeat password when creating/restoring <input data-field="confirm" type="password" autocomplete="off" maxlength="1024"></label><p><button data-action="restore" type="button">Restore QC wallet</button> <button data-action="create" type="button">Create NEW empty QC wallet</button></p><p><button data-action="unlock" type="button">Unlock</button> <button data-action="lock" type="button">Lock</button> <button data-action="reveal" type="button">Show recovery words</button> <button data-action="backup" type="button">Download encrypted backup</button> <button data-action="link" type="button">Link this wallet to Telegram</button></p><label>Import encrypted QC backup <input data-field="file" type="file" accept=".qcvault,application/json"></label><p role="status"></p>';
    host.appendChild(panel);
    const field=name=>panel.querySelector('[data-field="'+name+'"]');
    const status=panel.querySelector('[role="status"]');let busy=false;
    function clearSecrets(){field("words").value="";field("password").value="";field("confirm").value="";}
    function showAddress(){try{const saved=current();field("address").textContent=saved?"QC address: "+saved.address:"No local QC wallet. Restore words or explicitly create a new one.";}catch(e){status.textContent=e.message;}}
    function lock(){stop();clearSecrets();busy=false;status.textContent="QC wallet locked";}
    async function unlock(){
      const saved=current();if(!saved)throw Error("Restore or create a QC wallet first");
      const result=await call({action:"unlock",vault:saved.vault,password:field("password").value});
      clearTimeout(timer);timer=setTimeout(lock,300000);return result;
    }
    async function post(path,body){
      const response=await fetch(path,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(body)});
      const data=await response.json();if(!response.ok||!data.ok)throw Error(data.error||"Wallet link failed");return data;
    }
    panel.addEventListener("click",async event=>{
      const action=event.target.dataset.action;if(!action)return;
      if(action==="lock"){lock();return;}if(busy)return;busy=true;
      const run=epoch;
      try{
        if(action==="create"||action==="restore"){
          if(field("password").value!==field("confirm").value)throw Error("Passwords do not match");
          if(action==="create"&&!window.confirm("Create a NEW EMPTY QC wallet? Existing wallets are kept. This does not recover old funds."))return;
          const expected=field("expected").value.trim()||window.QCAccount?.snapshot?.wallet?.linked_qc_address||"";
          const result=await call({action,words:action==="restore"?field("words").value:"",address:action==="restore"?expected:"",password:field("password").value});
          if(run!==epoch)return;
          if(action==="create"){
            field("words").value=result.words;
            // Save only after the words have actually been presented on screen.
            await new Promise(resolve=>setTimeout(resolve,0));
            if(run!==epoch||!window.confirm("Record the 24 words shown in the recovery field securely, then confirm. They restore this exact QC address. Have you recorded them?"))return;
          }
          await persist(result);showAddress();status.textContent="QC wallet saved encrypted. Check the address. Node balances are separate; no reward or balance has been created.";
        }else if(action==="unlock"){
          await unlock();if(run!==epoch)return;status.textContent="QC wallet unlocked for five minutes";
        }else if(action==="reveal"){
          const saved=current();if(!saved)throw Error("No local wallet");
          const result=await call({action:"reveal",vault:saved.vault,password:field("password").value});
          if(run!==epoch)return;field("words").value=result.words;
          clearTimeout(timer);timer=setTimeout(lock,60000);status.textContent="Record words privately. They will be hidden after one minute.";return;
        }else if(action==="backup"){
          const saved=current();if(!saved)throw Error("No local wallet");
          const url=URL.createObjectURL(new Blob([saved.vault],{type:"application/json"}));
          const link=document.createElement("a");link.href=url;link.download="quantumcoin-"+saved.address+".qcvault";link.click();setTimeout(()=>URL.revokeObjectURL(url),1000);
          status.textContent="Encrypted QC backup downloaded. Keep its password separately.";
        }else if(action==="link"){
          await window.QCPlayer.ready;const playerId=window.QCPlayer.id();if(!playerId)throw Error("Verified Telegram session required");
          const saved=current();if(!saved)throw Error("No local wallet");
          await unlock();if(run!==epoch)return;
          const {challenge}=await post("/api/wallet/link-challenge",{playerId,address:saved.address});
          if(run!==epoch)return;
          if(!window.confirm("Link QC address "+saved.address+" to "+playerId+"? Network: "+challenge.networkID+" Genesis: "+challenge.genesisHex))return;
          const {proof}=await call({action:"link",playerId,challenge});if(run!==epoch)return;
          await post("/api/wallet/link-qc-address",proof);if(run!==epoch)return;
          status.textContent="QC wallet ownership verified and linked";window.QCAccount?.refresh().catch(()=>{});
        }
      }catch(error){if(run===epoch)status.textContent=action==="restore"?"QC wallet was NOT recovered. Existing data is unchanged. Check the words and old address, or choose Create NEW empty QC wallet. No old balance will be recovered.":error.message;}
      finally{if(run===epoch){busy=false;field("password").value="";field("confirm").value="";if(action!=="reveal")field("words").value="";}}
    });
    field("file").addEventListener("change",async event=>{
      if(busy)return;busy=true;const run=epoch;
      try{
        const file=event.target.files?.[0];if(!file)return;if(file.size>16384)throw Error("Backup too large");
        const vault=await file.text();if(run!==epoch)return;
        const result=await call({action:"unlock",vault,password:field("password").value});if(run!==epoch)return;
        if(!window.confirm("Select imported QC wallet "+result.address+"? Existing encrypted wallets will be kept."))return;
        await persist({vault,address:result.address});showAddress();status.textContent="Encrypted QC backup restored; wallet locked";
      }catch(error){if(run===epoch)status.textContent="Backup not imported: "+error.message;}
      finally{if(run===epoch){stop();busy=false;clearSecrets();}event.target.value="";}
    });
    window.addEventListener("pagehide",lock);
    window.addEventListener("storage",event=>{if(event.key===null||event.key===ACTIVE||event.key?.startsWith(PREFIX)){lock();showAddress();}});
    document.addEventListener("visibilitychange",()=>{if(document.hidden)lock();});
    showAddress();
  }
  if(document.readyState==="loading")document.addEventListener("DOMContentLoaded",mount,{once:true});else mount();
})();
