/* Actual device PoW and provider-approved rewards. No local money creation. */
(function () {
  "use strict";
  let active = null;
  async function post(path,body,signal){
    const response=await fetch(path,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(body),signal});
    const data=await response.json();
    if(!response.ok)throw Object.assign(Error(data.error||"request_failed"),{fatal:[401,403].includes(response.status)});
    return data;
  }
  window.QCStopVerifiedPoW=function(){if(active){active.controller.abort();active.reject(Error('mining_stopped'));active=null;}};
  window.QCStartVerifiedPoW=function(session){
    if(active?.sessionId===session.sessionId)return active.first;
    window.QCStopVerifiedPoW();
    if(!crypto.subtle)return Promise.reject(Error("https_required_for_device_mining"));
    let resolveFirst,rejectFirst;
    const current={sessionId:session.sessionId,controller:new AbortController(),receipt:null};
    current.first=new Promise((resolve,reject)=>{resolveFirst=resolve;rejectFirst=reject;});active=current;
    current.reject=rejectFirst;
    const body={playerId:session.playerId,sessionId:session.sessionId};
    (async()=>{
      const {solveCurrentJob,waitForMiningRetry}=await import('/mining/mining-session.mjs');
      let nextBlock=false;
      while(!current.controller.signal.aborted&&active===current){
        let data;
        try{data=await post('/api/mine/work',{...body,nextBlock},current.controller.signal);}
        catch(error){if(error.fatal||current.controller.signal.aborted)throw error;await waitForMiningRetry(current.controller.signal);continue;}
        if(!data.ok){
          if(['canonical_node_work_unavailable','work_already_submitted_or_pending','work_reconciliation_pending','node_network_or_tip_unavailable'].includes(data.error)){
            window.QC_CLIENT_POW_STATUS={status:'waiting',sessionId:session.sessionId,error:data.error,receipt:current.receipt};
            await waitForMiningRetry(current.controller.signal);continue;
          }
          throw Error(data.error||'node_work_unavailable');
        }
        let receipt;
        if(data.alreadyAccepted)receipt=data.receipt;
        else {
          const job=data.job;
          window.QC_CLIENT_POW_STATUS={status:'solving',sessionId:session.sessionId,height:job.height,receipt:current.receipt};
          const found=await solveCurrentJob({job,workerURL:'/mining/canonical-pow-worker.mjs',signal:current.controller.signal,
            status:()=>post('/api/mine/work/status',body,current.controller.signal),progress:p=>{window.QC_CLIENT_POW_STATUS={status:'solving',sessionId:session.sessionId,height:job.height,hashes:p.hashes,receipt:current.receipt};}});
          if(found.restart)continue;
          if(found.receipt)receipt=found.receipt;
          else {
            let accepted;
            try{accepted=await post('/api/mine/work/submit',{...body,jobId:job.job_id,nonce:found.solution.nonce},current.controller.signal);}
            catch(error){accepted={ok:false,error:error.message};}
            if(!accepted.ok){
              // Reconcile by reading the canonical chain, never resubmit a nonce.
              if(current.controller.signal.aborted)throw Error('mining_stopped');
              accepted=await post('/api/mine/work/status',body,current.controller.signal);
              if(accepted.ok&&!accepted.alreadyAccepted&&!accepted.current)continue;
              if(!accepted.alreadyAccepted)throw Error(accepted.error||'work_reconciliation_pending');
            }
            receipt=accepted.receipt;
            if(receipt.hash!==found.solution.hash)throw Error('node_pow_receipt_mismatch');
          }
        }
        current.receipt=receipt;nextBlock=true;
        window.QC_CLIENT_POW_STATUS={status:'accepted',sessionId:session.sessionId,receipt};
        resolveFirst({ok:true,receipt});
      }
    })().catch(error=>{
      if(active===current){window.QC_CLIENT_POW_STATUS={status:'stopped',sessionId:session.sessionId,error:error.message,receipt:current.receipt};active=null;}
      rejectFirst(error);
    });
    return current.first;
  };
  window.QCSubmitVerifiedTaskReward = async function (campaignId, proof) {
    await window.QC_TELEGRAM_IDENTITY_READY;
    const playerId = window.QCGetPlayerId();
    if (!playerId) throw new Error("telegram_session_required");
    const result = await post("/api/tgwt/rewards/claim", {playerId,campaignId,proof});
    if (result.ok !== true) throw Error(result.error || "reward_not_verified");
    if (typeof window.QC_refreshWalletFromBackend === "function") await window.QC_refreshWalletFromBackend();
    window.dispatchEvent(new CustomEvent("qc:verified-reward", {detail:result}));
    return result;
  };
})();
