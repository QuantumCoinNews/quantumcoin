/* One account controller. Server balances are observations, never local credits. */
(function () {
  "use strict";
  if (window.QCAccount) return;
  let pending = null, last = null, withdrawal = null;
  const $ = id => document.getElementById(id);
  const text = (id, value) => { const node=$(id); if(node)node.textContent=String(value ?? "—"); };
  const esc = value => String(value ?? "").replace(/[&<>"']/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]));
  function units(value) {
    const raw=String(value ?? "0");
    if(!/^\d{1,19}$/.test(raw))throw Error("invalid_integer_balance");
    const n=BigInt(raw), fraction=String(n%1000000n).padStart(6,"0").replace(/0+$/,"");
    return String(n/1000000n)+(fraction?"."+fraction:"");
  }
  async function request(path, body) {
    await window.QCPlayer.ready;
    if(!window.QCPlayer.id())throw Error("telegram_session_required");
    const response=await fetch(path,body===undefined?{cache:"no-store"}:{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(body)});
    const data=await response.json();
    if(!response.ok||data.ok===false)throw Error(data.error||"account_request_failed");
    return data;
  }
  function mount() {
    const host=$("walletPanel");if(!host||$("qcRealSendCard"))return;
    const card=document.createElement("section");card.id="qcRealSendCard";card.className="wallet-card top-card";
    card.innerHTML='<h3>QuantumCoin wallet</h3><p>Wallet balances come from the node. Sign transfers in your wallet; import only public signed transaction JSON.</p><p>Address: <span id="qcReceiveAddress">—</span></p><p>QC: <span id="qcReceiveBalance">—</span> · Spendable: <span id="qcReceiveSpendable">—</span></p><p id="qcReceiveStatus" role="status"></p><button class="wallet-btn" id="btnRefreshQCWallet" type="button">Refresh</button><button class="wallet-btn" id="btnCopyQCAddress" type="button">Copy address</button><label>Public signed transaction JSON<textarea id="qcSendSignedTx" rows="5" maxlength="1048576" autocomplete="off" spellcheck="false"></textarea></label><button class="wallet-btn" id="btnSendQC" type="button">Submit signed transaction</button><p id="qcSendStatus" role="status"></p>';
    host.appendChild(card);
    $("btnRefreshQCWallet").addEventListener("click",()=>refresh().catch(showError));
    $("btnCopyQCAddress").addEventListener("click",async()=>{try{const address=last?.wallet.linked_qc_address;if(!address)throw Error("wallet_not_linked");await navigator.clipboard.writeText(address);}catch(error){showError(error);}});
    $("btnSendQC").addEventListener("click",send);
  }
  function renderLedger(ledger, history) {
    const host=$("tgwtRewardsMount");if(!host)return;
    const b=ledger.balances||{};
    host.innerHTML='<p>Verified task rewards. Withdrawals reserve funds; BSC payment verification is deferred to mainnet. Users pay BNB gas.</p><div class="tgwt-ledger-grid">'+
      [["Earned","earned_u"],["Pending","pending_u"],["Verified","verified_u"],["Reserved","reserved_u"],["Paid","withdrawn_u"]].map(([title,key])=>'<div class="tgwt-ledger-item"><div class="label">'+title+'</div><div class="value" id="account-'+key+'">'+esc(units(b[key]))+'</div></div>').join("")+
      '</div><p>Pool: '+esc(ledger.pool?.total)+' TGWT · Distributed: '+esc(ledger.pool?.distributed)+'</p><div class="account-actions"><label>BSC address<input id="tgwtApiWithdrawTo" placeholder="0x..." autocomplete="off"></label><label>Amount (TGWT)<input id="tgwtApiWithdrawAmount" placeholder="1.000001" inputmode="decimal" autocomplete="off"></label><button class="wallet-btn" id="tgwtApiWithdrawBtn" type="button">Reserve withdrawal</button></div><p id="tgwtApiStatus" role="status"></p><h4>Ledger history</h4>'+
      (ledger.events||[]).map(ev=>'<p>'+esc(ev.kind)+' · '+esc(units(ev.amount_u))+' TGWT</p>').join("")+'<h4>Withdrawal requests</h4>'+
      (history.items||[]).map(item=>'<p>'+esc(item.status)+' · '+esc(item.amount)+' TGWT · '+esc(item.txHash||"No on-chain payment")+'</p>').join("");
    $("tgwtApiWithdrawBtn").addEventListener("click",reserve);
  }
  function showError(error) {text("qcReceiveStatus",error.message||"Account unavailable");}
  async function refresh() {
    if(pending)return pending;
    pending=(async()=>{
      const id=encodeURIComponent(window.QCPlayer.id()||(await window.QCPlayer.ready).playerId||"");
      const [wallet,ledger,history]=await Promise.all([request("/api/wallet?playerId="+id),request("/api/v1/tgwt/ledger?user_id="+id+"&limit=20"),request("/api/v1/withdraw/list?user_id="+id+"&limit=10")]);
      last=Object.freeze({wallet,ledger});mount();window.QCScene?.applyAccount(wallet,ledger);
      text("walletQCValue",wallet.qc_balance);text("walletTGWTValue",units(ledger.balances?.verified_u));text("walletLevelValue",wallet.level);
      text("hudTGWT",units(ledger.balances?.verified_u));
      text("qcReceiveAddress",wallet.linked_qc_address||"Not linked");text("qcReceiveBalance",wallet.real_qc_balance);text("qcReceiveSpendable",wallet.real_qc_spendable);
      text("qcReceiveStatus",wallet.real_qc_ok?"Node balance verified":"Node balance unavailable"+(wallet.legacyNumericReviewRequired?" · Legacy totals require review":""));
      // Keep user-entered withdrawal intent intact during periodic refresh.
      if(!$("tgwtApiWithdrawBtn"))renderLedger(ledger,history);
      else for(const key of ["earned_u","pending_u","verified_u","reserved_u","withdrawn_u"])text("account-"+key,units(ledger.balances?.[key]));
      return wallet;
    })().finally(()=>{pending=null;});return pending;
  }
  function hasSecret(value) {
    return value&&typeof value==="object"&&Object.entries(value).some(([key,child])=>["priv_hex","privhex","privatekey","private_key","mnemonic","seed"].includes(key.toLowerCase())||hasSecret(child));
  }
  async function send() {
    const button=$("btnSendQC");if(button.disabled)return;button.disabled=true;
    try {
      const raw=$("qcSendSignedTx").value;
      const envelope=JSON.parse(raw);if(!envelope.transaction||hasSecret(envelope))throw Error("public_signed_transaction_required");
      const result=await request("/api/wallet/send-qc",{playerId:window.QCPlayer.id(),initData:String(window.Telegram?.WebApp?.initData||""),transaction:raw});
      text("qcSendStatus","Submitted: "+result.txid+" · Awaiting node confirmation");$("qcSendSignedTx").value="";await refresh();
    }catch(error){text("qcSendStatus",error instanceof SyntaxError?"Import public signed JSON only":error.message);}finally{button.disabled=false;}
  }
  async function reserve() {
    const button=$("tgwtApiWithdrawBtn");if(button.disabled)return;button.disabled=true;
    try {
      const amount=$("tgwtApiWithdrawAmount").value.trim(),toAddress=$("tgwtApiWithdrawTo").value.trim();
      if(!/^\d{1,7}(?:\.\d{1,6})?$/.test(amount)||!/^0x[0-9a-fA-F]{40}$/.test(toAddress))throw Error("Enter a BSC address and a TGWT amount with at most 6 decimal places");
      const fingerprint=toAddress+":"+amount;
      if(withdrawal&&withdrawal.fingerprint!==fingerprint)throw Error("Review the previous uncertain request before changing its amount or destination");
      withdrawal ||= {fingerprint,nonce:crypto.randomUUID()};
      const result=await request("/api/v1/withdraw/request",{userId:window.QCPlayer.id(),asset:"TGWT",amount,toAddress,nonce:withdrawal.nonce});
      text("tgwtApiStatus","Reserved: "+result.requestId+" · "+result.status+" · No BSC payment sent");withdrawal=null;await refresh();
    }catch(error){text("tgwtApiStatus",error.message);}finally{button.disabled=false;}
  }
  Object.defineProperty(window,"QCAccount",{value:Object.freeze({refresh,units,get snapshot(){return last;}})});
  // Compatibility names are stable aliases, never wrapper chains.
  for(const name of ["qcSyncWalletFromBackend","QC_refreshWalletFromBackend","qcRefreshReceiveSendWallet","refreshTGWTApiPanel"])
    Object.defineProperty(window,name,{value:refresh});
  document.addEventListener("click",event=>{if(event.target.closest?.('[data-nav="wallet"],[data-nav="tgwt"]'))refresh().catch(showError);});
  window.QCPlayer.ready.then(identity=>{if(identity.ok){mount();refresh().catch(showError);setInterval(()=>{if(!document.hidden)refresh().catch(showError);},15000);}});
})();
