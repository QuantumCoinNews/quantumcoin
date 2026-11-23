/* Legacy recovery only. Wallet creation and signing belong to the wallet app. */
(function () {
  "use strict";
  if (window.QCWalletRecovery) return;
  const LEGACY="qc_wallet_seed_v2", VAULT="qc_wallet_recovery_encrypted_v1";
  const VERSION="QC_GAME_RECOVERY_V1", ITERATIONS=600000, MAX_TEXT=8192;
  const enc=new TextEncoder(), dec=new TextDecoder("utf-8",{fatal:true});
  const b64=a=>btoa(String.fromCharCode(...a));
  function bytes(value) {
    if(typeof value!=="string" || value.length>20000) throw Error("Invalid recovery backup.");
    const result=Uint8Array.from(atob(value),c=>c.charCodeAt(0));
    if(b64(result)!==value) throw Error("Invalid recovery backup.");
    return result;
  }
  async function key(password,salt) {
    if(typeof password!=="string" || password.length<14 || password.length>1024) throw Error("Use a recovery passphrase of 14 to 1024 characters.");
    const material=await crypto.subtle.importKey("raw",enc.encode(password),"PBKDF2",false,["deriveKey"]);
    return crypto.subtle.deriveKey({name:"PBKDF2",salt,iterations:ITERATIONS,hash:"SHA-256"},material,
      {name:"AES-GCM",length:256},false,["encrypt","decrypt"]);
  }
  async function encrypt(text,password) {
    if(typeof text!=="string" || !text || enc.encode(text).length>MAX_TEXT) throw Error("Invalid legacy recovery size.");
    const salt=crypto.getRandomValues(new Uint8Array(16)),iv=crypto.getRandomValues(new Uint8Array(12));
    const cipher=await crypto.subtle.encrypt({name:"AES-GCM",iv,additionalData:enc.encode(VERSION)},await key(password,salt),enc.encode(text));
    return {version:VERSION,iterations:ITERATIONS,salt:b64(salt),iv:b64(iv),ciphertext:b64(new Uint8Array(cipher))};
  }
  async function decrypt(vault,password) {
    const fields=["version","iterations","salt","iv","ciphertext"];
    if(!vault || typeof vault!=="object" || Object.keys(vault).length!==fields.length || !fields.every(k=>Object.hasOwn(vault,k)) ||
      vault.version!==VERSION || vault.iterations!==ITERATIONS) throw Error("Invalid recovery backup.");
    const salt=bytes(vault.salt),iv=bytes(vault.iv),cipher=bytes(vault.ciphertext);
    if(salt.length!==16 || iv.length!==12 || cipher.length<17 || cipher.length>MAX_TEXT+16) throw Error("Invalid recovery backup.");
    const result=await crypto.subtle.decrypt({name:"AES-GCM",iv,additionalData:enc.encode(VERSION)},await key(password,salt),cipher);
    return dec.decode(result);
  }
  window.QCWalletRecovery=Object.freeze({encrypt,decrypt});
  function mount() {
    const host=document.getElementById("walletPanel");if(!host)return;
    ["btnGenerateSeed","btnToggleSeed","btnCopySeed"].forEach(id=>{const el=document.getElementById(id);if(el){el.disabled=true;el.hidden=true;}});
    const panel=document.createElement("section");
    panel.innerHTML='<h3>Wallet recovery</h3><p>Create and sign with the QuantumCoin wallet app. This game does not generate or store new recovery phrases. Encrypt older browser recovery here and keep its backup and passphrase offline.</p><label>Recovery passphrase <input type="password" autocomplete="new-password" minlength="14" maxlength="1024"></label> <button type="button" data-action="encrypt">Encrypt old browser recovery</button> <button type="button" data-action="download">Download encrypted backup</button> <button type="button" data-action="unlock">Unlock encrypted recovery</button> <button type="button" data-action="lock">Hide recovery</button><p><label>Restore encrypted backup <input type="file" accept="application/json,.json"></label></p><p><label><input type="checkbox"> I saved the encrypted backup and its passphrase safely.</label> <button type="button" data-action="remove">Remove old plaintext copy</button></p><textarea readonly hidden aria-label="Recovered legacy data"></textarea><p role="status"></p>';
    host.appendChild(panel);
    const password=panel.querySelector('input[type="password"]'),ack=panel.querySelector('input[type="checkbox"]');
    const status=panel.querySelector('[role="status"]'),output=panel.querySelector("textarea");
    let epoch=0,busy=false,verifiedVault=null,exportedVault=null,timer=null;
    function clear() {
      epoch++;password.value="";output.value="";output.hidden=true;ack.checked=false;
      verifiedVault=null;exportedVault=null;clearTimeout(timer);timer=null;
      status.textContent="Recovery hidden. Unlock again when needed.";
    }
    function guard(ticket) {
      if(ticket!==epoch || document.hidden || !window.isSecureContext || !crypto.subtle) throw Error("Recovery session locked. Use HTTPS or localhost and unlock again.");
    }
    async function exclusive(ticket,operation) {
      if(!navigator.locks || typeof navigator.locks.request!=="function") throw Error("A browser with secure storage locking is required. Existing data has been retained.");
      return navigator.locks.request("quantumcoin-wallet-recovery-v1",async()=>{guard(ticket);return operation();});
    }
    function download(raw) {
      const url=URL.createObjectURL(new Blob([raw],{type:"application/json"}));
      try {
        const link=document.createElement("a");link.href=url;link.download="quantumcoin-encrypted-recovery.json";
        link.click();exportedVault=raw;
      } finally { setTimeout(()=>URL.revokeObjectURL(url),1000); }
    }
    function show(text) {
      output.value=text;output.hidden=false;clearTimeout(timer);
      timer=setTimeout(()=>{clear();status.textContent="Recovery hidden. Unlock again when needed.";},60000);
    }
    try { status.textContent=localStorage.getItem(LEGACY)?"Old plaintext recovery exists. Encrypt it before using this device with others.":"No old plaintext recovery found. You can restore an encrypted backup below."; }
    catch(_){status.textContent="Browser recovery storage is unavailable.";}
    panel.addEventListener("click",async event=>{
      const action=event.target.dataset.action;if(!action)return;
      if(action==="lock"){clear();status.textContent="Recovery hidden.";return;}
      if(busy)return;
      busy=true;const ticket=epoch,pass=password.value;
      output.value="";output.hidden=true;
      try {
        guard(ticket);
        if(action==="encrypt") {
          await exclusive(ticket,async()=>{
            const old=localStorage.getItem(LEGACY);
            if(!old || localStorage.getItem(VAULT)) throw Error("No old recovery found, or an encrypted backup already exists. Existing backups will not be overwritten.");
            const vault=await encrypt(old,pass);
            if(await decrypt(vault,pass)!==old) throw Error("Backup verification failed.");
            guard(ticket);
            if(localStorage.getItem(LEGACY)!==old || localStorage.getItem(VAULT)) throw Error("Recovery storage changed. Unlock the current backup and retry.");
            const raw=JSON.stringify(vault);localStorage.setItem(VAULT,raw);
            if(localStorage.getItem(VAULT)!==raw) throw Error("Backup storage verification failed.");
            verifiedVault=raw;ack.checked=false;download(raw);
            status.textContent="Encrypted backup prepared. Save its passphrase and backup before removing the old plaintext copy.";
          });
        } else if(action==="download") {
          const raw=localStorage.getItem(VAULT);await decrypt(JSON.parse(raw),pass);guard(ticket);
          if(localStorage.getItem(VAULT)!==raw) throw Error("Recovery storage changed. Unlock again.");
          verifiedVault=raw;ack.checked=false;download(raw);
          status.textContent="Encrypted backup downloaded. Save its passphrase separately before removing the old plaintext copy.";
        } else if(action==="unlock") {
          const raw=localStorage.getItem(VAULT),text=await decrypt(JSON.parse(raw),pass);guard(ticket);
          if(localStorage.getItem(VAULT)!==raw) throw Error("Recovery storage changed. Unlock again.");
          verifiedVault=raw;show(text);
          status.textContent="Recovery unlocked on this page only for 60 seconds. Use your trusted wallet app privately.";
        } else if(action==="remove") {
          await exclusive(ticket,async()=>{
            const raw=localStorage.getItem(VAULT),old=localStorage.getItem(LEGACY);
            if(!ack.checked || !old || raw!==verifiedVault || raw!==exportedVault) throw Error("Download or restore the encrypted backup, verify it and confirm you saved it first.");
            const text=await decrypt(JSON.parse(raw),pass);guard(ticket);
            if(!ack.checked || text!==old || localStorage.getItem(VAULT)!==raw || localStorage.getItem(LEGACY)!==old) throw Error("Recovery storage changed. No plaintext was removed.");
            localStorage.removeItem(LEGACY);clear();
            status.textContent="Old plaintext copy removed. Your verified encrypted recovery backup remains.";
          });
        }
      } catch(error) {
        // Fixed validation messages only; never include recovery data or platform exceptions.
        if(ticket===epoch)status.textContent=error.message.startsWith("Recovery session") || error.message.startsWith("A browser") || error.message.startsWith("Download or restore") || error.message.startsWith("No old recovery") ? error.message : "Recovery step failed. Keep your backups; check the passphrase and storage, then retry. No unverified plaintext deletion was performed.";
      } finally {busy=false;}
    });
    panel.querySelector('input[type="file"]').addEventListener("change",async event=>{
      if(busy){event.target.value="";return;}
      busy=true;const ticket=epoch,pass=password.value;
      output.value="";output.hidden=true;
      try {
        guard(ticket);
        const file=event.target.files[0];if(!file || file.size>20000)throw Error();
        const vault=JSON.parse(await file.text());await decrypt(vault,pass);guard(ticket);
        await exclusive(ticket,async()=>{
          if(localStorage.getItem(VAULT))throw Error();
          const raw=JSON.stringify(vault);localStorage.setItem(VAULT,raw);
          if(localStorage.getItem(VAULT)!==raw)throw Error();
          verifiedVault=raw;exportedVault=raw;ack.checked=false;
          status.textContent="Encrypted backup restored and verified. Unlock explicitly to view recovery.";
        });
      } catch(_) {if(ticket===epoch)status.textContent="Backup import failed. Existing recovery data was not overwritten; check the file, passphrase and storage.";}
      finally{event.target.value="";busy=false;}
    });
    window.addEventListener("pagehide",clear);
    window.addEventListener("pageshow",clear);
    document.addEventListener("visibilitychange",()=>{if(document.hidden)clear();});
    window.addEventListener("storage",event=>{
      if(event.key===null || event.key===LEGACY || event.key===VAULT){clear();status.textContent="Recovery changed in another tab. Unlock the current backup again.";}
    });
  }
  if(document.readyState==="loading")document.addEventListener("DOMContentLoaded",mount,{once:true});else mount();
})();
