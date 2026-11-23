/* Address linkage accepts public proofs from a separate wallet, never keys. */
(function(){
  "use strict";
  async function post(path,body){
    const response=await fetch(path,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(body)});
    const data=await response.json();if(!response.ok||!data.ok)throw Error(data.error||"wallet_link_failed");return data;
  }
  function mount(){
    const host=document.getElementById("walletPanel");if(!host)return;
    const panel=document.createElement("section");
    panel.innerHTML='<h3>Link a QuantumCoin wallet</h3><p>Prove control with your wallet. Only the public signature is sent to the game.</p><label>QC address <input data-field="address" autocomplete="off"></label> <button data-action="challenge" type="button">Create ownership challenge</button><p><label>Public signed proof JSON <textarea data-field="proof" autocomplete="off" spellcheck="false"></textarea></label> <button data-action="prove" type="button">Verify and link</button></p><p role="status"></p>';
    host.appendChild(panel);let busy=false;
    panel.addEventListener("click",async event=>{
      const action=event.target.dataset.action;if(!action||busy)return;busy=true;
      const status=panel.querySelector('[role="status"]');
      try{
        const playerId=window.QCGetPlayerId();if(!playerId)throw Error("telegram_session_required");
        const address=panel.querySelector('[data-field="address"]').value.trim();
        if(action==="challenge"){
          const data=await post("/api/wallet/link-challenge",{playerId,address});
          const url=URL.createObjectURL(new Blob([JSON.stringify(data.challenge,null,2)],{type:"application/json"}));
          const link=document.createElement("a");link.href=url;link.download="quantumcoin-wallet-link-challenge.json";link.click();setTimeout(()=>URL.revokeObjectURL(url),1000);
          status.textContent="Challenge saved. Sign it with the offline wallet tool, review the network and account, then paste the public proof here. The challenge expires in five minutes.";
        }else{
          const proof=JSON.parse(panel.querySelector('[data-field="proof"]').value);
          const keys=["playerId","address","nonce","publicKey","signature"];
          if(!proof || Object.keys(proof).length!==keys.length || !keys.every(k=>typeof proof[k]==="string") || proof.playerId!==playerId || proof.address!==address)throw Error("public_wallet_proof_required");
          const result=await post("/api/wallet/link-qc-address",proof);
          status.textContent="Wallet ownership verified: "+result.qcAddress;
          panel.querySelector('[data-field="proof"]').value="";
        }
      }catch(error){status.textContent=error instanceof SyntaxError?"Paste only the public signed proof JSON. Never paste a private key or recovery phrase.":String(error.message);}
      finally{busy=false;}
    });
  }
  if(document.readyState==="loading")document.addEventListener("DOMContentLoaded",mount,{once:true});else mount();
})();
