
(function(){
  const QC_API_BASE = window.QC_API_BASE || "";
  const QC_PLAYER_ID_KEY = "qc_player_id_v1";

  let activeMiningSession = null;

  const getPlayerId=window.QCGetPlayerId;

  function zoneCodeFromId(id){
    const n = Number(id || 1);
    return "LUNA-" + String(n).padStart(2, "0");
  }

  function getActiveShipId(){
    try {
      if (typeof window.getActiveShipConfig === "function") {
        const cfg = window.getActiveShipConfig();
        if (cfg && cfg.id) return String(cfg.id);
      }
    } catch(e) {}
    return "starter";
  }

  async function qcApiPost(path, body){
    const res = await fetch(QC_API_BASE + path, {
      method: "POST",
      headers: {
        "Accept": "application/json",
        "Content-Type": "application/json"
      },
      body: JSON.stringify(body || {})
    });

    if (!res.ok) {
      const text = await res.text().catch(()=> "");
      throw new Error("API error " + res.status + " " + text);
    }

    return await res.json();
  }

  async function qcStartMiningSession(siteId){
    if (window.QC_TELEGRAM_IDENTITY_READY) await window.QC_TELEGRAM_IDENTITY_READY;
    if (!window.QC_TG_IDENTITY || !window.QC_TG_IDENTITY.ok) throw new Error("telegram_session_required");
    const playerId = getPlayerId();
    const zoneId = zoneCodeFromId(siteId);
    const shipId = getActiveShipId();
    let nftSessionId = "";
    let nftInitData = "";

    try {
      const tg = window.Telegram && window.Telegram.WebApp;
      const identity = window.QC_TG_IDENTITY;
      if (tg && tg.initData && identity && identity.ok &&
          String(identity.playerId) === String(playerId)) {
        nftInitData = String(tg.initData);
        const opened = await qcApiPost("/api/v1/nft/monthly/session/start", {
          playerId,
          initData: nftInitData
        });
        if (opened && opened.ok && opened.sessionId) {
          nftSessionId = String(opened.sessionId);
        }
      }
    } catch (err) {
      console.warn("QC NFT session start unavailable:", err);
    }

    const data = await qcApiPost("/api/mine/start", {
      playerId,
      zoneId,
      shipId,
      siteId,
      initData: nftInitData,
      nftSessionId
    });

    if (!data || !data.ok || !data.sessionId) throw new Error((data && data.error) || "mining_start_failed");
    activeMiningSession = {
      sessionId: data.sessionId,
      playerId,
      zoneId,
      shipId,
      siteId,
      startedAt: Date.now(),
      startedResponse: data,
      nftSessionId,
      nftInitData
    };

    if (data.miningMode === "REAL_CHAIN" && window.QCStartVerifiedPoW) {
      const startedSession=activeMiningSession;
      window.QCStartVerifiedPoW(startedSession).catch(function(error){
        if(activeMiningSession===startedSession) window.QC_CLIENT_POW_STATUS = {status:"stopped",sessionId:startedSession.sessionId,error:error.message};
      });
    }
    window.QC_BACKEND_MINING_LAST = {
      ok: true,
      status: "started",
      playerId,
      zoneId,
      shipId,
      siteId,
      response: data,
      ts: Date.now()
    };

    console.log("QC mining backend session started:", window.QC_BACKEND_MINING_LAST);
    return data;
  }

  async function qcFinishMiningSession(reason){
    if (!activeMiningSession) return null;

    const s = activeMiningSession;
    if (reason === "exitMineScene" && window.QCStopVerifiedPoW) window.QCStopVerifiedPoW();

    const durationSec = Math.max(1, Math.round((Date.now() - s.startedAt) / 1000));

    let data;
    try {
      data = await qcApiPost("/api/mine/finish", {
        playerId: s.playerId,
        sessionId: s.sessionId,
        zoneId: s.zoneId,
        shipId: s.shipId,
        siteId: s.siteId,
        durationSec
      });
    } finally {
      if (s.nftSessionId && s.nftInitData) {
        try {
          const nftDone = await qcApiPost(
            "/api/v1/nft/monthly/session/complete",
            {
              playerId: s.playerId,
              initData: s.nftInitData,
              sessionId: s.nftSessionId
            }
          );
          window.QC_NFT_MONTHLY_LAST = nftDone;
        } catch (err) {
          console.warn("QC NFT session completion unavailable:", err);
        }
      }
    }

    if (data && data.ok) { window.QCStopVerifiedPoW?.(); activeMiningSession = null; }
    window.QC_BACKEND_MINING_LAST = {
      ok: !!(data && data.ok),
      status: data && data.ok ? "finished" : "not_completed",
      reason: reason || "unknown",
      playerId: s.playerId,
      zoneId: s.zoneId,
      shipId: s.shipId,
      siteId: s.siteId,
      durationSec,
      response: data,
      ts: Date.now()
    };

    console.log("QC mining backend session finished:", window.QC_BACKEND_MINING_LAST);

    if (typeof window.qcSyncWalletFromBackend === "function") {
      window.qcSyncWalletFromBackend().catch(function(err){
        console.warn("QC wallet sync after mining failed:", err);
      });
    }

    return data;
  }

  window.qcSolveCurrentMiningWork = function(){
    if (!activeMiningSession) return Promise.reject(new Error("server_session_required"));
    const current = window.QC_CLIENT_POW_STATUS;
    if (current && current.receipt && current.sessionId === activeMiningSession.sessionId) return Promise.resolve(current);
    return window.QCStartVerifiedPoW(activeMiningSession);
  };

  window.qcStartMiningSession=qcStartMiningSession;
  window.qcFinishMiningSession=qcFinishMiningSession;
  let opening=null;
  window.addEventListener("qc:mine-enter",event=>{opening=qcStartMiningSession(event.detail.siteId);opening.catch(()=>{});});
  window.addEventListener("qc:mine-exit",()=>{Promise.resolve(opening).then(()=>qcFinishMiningSession("exitMineScene")).catch(()=>{});});

})();


/* MINING_REAL_BLOCK_UX_V1
   This does NOT mint fake QC.
   It only calls the existing backend mining finish bridge and then syncs wallet from backend.
*/
(function MINING_REAL_BLOCK_UX_V1(){
  if (window.__miningRealBlockUxV1Applied) return;
  window.__miningRealBlockUxV1Applied = true;

  let currentMineSiteId = null;
  let busy = false;

  function $(id){ return document.getElementById(id); }

  function esc(s){
    return String(s ?? "")
      .replaceAll("&","&amp;")
      .replaceAll("<","&lt;")
      .replaceAll(">","&gt;")
      .replaceAll('"',"&quot;")
      .replaceAll("'","&#39;");
  }

  function ensureMiningUxBox(){
    const mineHud = $("mineHud");
    if (!mineHud) return null;

    let box = $("mineRealUxBox");
    if (box) return box;

    box = document.createElement("div");
    box.id = "mineRealUxBox";
    box.innerHTML = `
      <div id="mineRealUxStatus">
        Mining session ready. QC balance will update only after a real block is found and confirmed by the QuantumCoin chain.
      </div>
      <button id="mineRealBlockBtn" type="button">Find Real Block</button>
      <div id="mineRealUxResult"></div>
    `;

    mineHud.appendChild(box);

    const btn = $("mineRealBlockBtn");
    if (btn && !btn.__mineRealBlockBound) {
      btn.__mineRealBlockBound = true;
      btn.addEventListener("click", runRealBlockMining);
    }

    return box;
  }

  function setMiningStatus(msg, cls){
    ensureMiningUxBox();
    const el = $("mineRealUxStatus");
    if (el) el.textContent = msg;
  }

  function setMiningResult(html){
    ensureMiningUxBox();
    const el = $("mineRealUxResult");
    if (el) el.innerHTML = html || "";
  }

  function pickBlockInfo(data){
    const src = data || {};
    const candidates = [
      src,
      src.realBlock,
      src.qcMineResult,
      src.qc_mine_result,
      src.block,
      src.result
    ].filter(Boolean);

    const out = {};
    for (const c of candidates) {
      out.height = out.height ?? c.height ?? c.blockHeight ?? c.chainHeight;
      out.reward = out.reward ?? c.reward ?? c.blockReward ?? c.amount;
      out.hash = out.hash ?? c.block_hash ?? c.blockHash ?? c.hash;
      out.success = out.success ?? c.success ?? c.ok;
    }
    return out;
  }

  async function runRealBlockMining(){
    if (busy) return;

    const btn = $("mineRealBlockBtn");
    busy = true;
    if (btn) {
      btn.disabled = true;
      btn.textContent = "Searching block...";
    }

    setMiningStatus("This device must solve real block work. A click or elapsed game time cannot produce a block reward.");
    setMiningResult("");

    try {
      if (typeof window.qcFinishMiningSession !== "function") {
        throw new Error("qcFinishMiningSession is not available.");
      }

      if (window.qcSolveCurrentMiningWork) await window.qcSolveCurrentMiningWork();
      const data = await window.qcFinishMiningSession("manual_real_block_button");

      if (!data) {
        setMiningStatus("Mining session is not ready yet. Wait 1 second after entering the zone, then try again.");
        return;
      }

      const info = pickBlockInfo(data);

      if (data.ok === false || data.success === false) {
        setMiningStatus("Mining failed or QC node did not return a confirmed block.");
        setMiningResult(`<b>Error:</b> ${esc(data.error || data.message || "unknown")}`);
        return;
      }

      setMiningStatus("Real block result received. Syncing wallet balance from backend...");

      if (typeof window.qcSyncWalletFromBackend === "function") {
        try { await window.qcSyncWalletFromBackend(); } catch(e) {}
      }

      const lines = [
        `<b>Block accepted locally; subject to confirmations and maturity.</b>`,
        `Height: ${esc(info.height || "-")}`,
        `Reward: ${esc(info.reward ?? "See chain receipt")} QC`,
        `Hash: ${esc(info.hash || "-")}`,
        `Balance source: backend / QuantumCoin chain`
      ];

      setMiningResult(lines.join("<br>"));
      setMiningStatus("Reward processed by real mining flow. Balance is refreshed from the chain/backend, not from fake click logic.");
    } catch(e) {
      setMiningStatus("Mining UX error.");
      setMiningResult(`<b>Error:</b> ${esc(e && e.message ? e.message : e)}`);
    } finally {
      busy = false;
      if (btn) {
        btn.disabled = false;
        btn.textContent = "Find Real Block";
      }
    }
  }

  window.addEventListener("qc:mine-enter",event=>{currentMineSiteId=event.detail.siteId;ensureMiningUxBox();setMiningStatus("Device work is required; server verification decides the result.");setMiningResult("");});
  window.ensureMiningUxBox = ensureMiningUxBox;
  window.runRealBlockMining = runRealBlockMining;

  document.addEventListener("DOMContentLoaded", () => {
    setTimeout(ensureMiningUxBox, 1200);
  });
})();
