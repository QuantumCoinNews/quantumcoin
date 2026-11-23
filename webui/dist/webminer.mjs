import {solveCurrentJob,waitForMiningRetry} from './mining-session.mjs';
const address=document.querySelector('#address'),token=document.querySelector('#token');
const status=document.querySelector('#status'),start=document.querySelector('#start'),stop=document.querySelector('#stop');
let active;
function idle(){start.disabled=false;stop.disabled=true;}
stop.onclick=()=>{active?.abort();status.textContent='Stopped. Submitted work may have committed; check the explorer.';idle();};
start.onclick=async()=>{
  if(!globalThis.isSecureContext||!globalThis.crypto?.subtle){status.textContent='HTTPS or localhost is required.';return;}
  const controller=new AbortController();active=controller;start.disabled=true;stop.disabled=false;
  const credential=token.value;token.value='';const miner=address.value.trim();
  async function api(path,body){
    const headers={'X-QC-API-KEY':credential};if(body)headers['Content-Type']='application/json';
    const response=await fetch(path,{method:body?'POST':'GET',headers,body:body?JSON.stringify(body):undefined,
      signal:controller.signal,credentials:'omit',redirect:'error',cache:'no-store'});
    const data=await response.json();
    if(!response.ok)throw Object.assign(Error(data.message||'node_request_failed'),{status:response.status,fatal:[401,403].includes(response.status)});
    return data;
  }
  try{
    const identity=await api('/api/mine/tip');
    while(!controller.signal.aborted){
      let job;
      try{job=await api('/api/mine/job?address='+encodeURIComponent(miner));}
      catch(error){if(![409,429,503].includes(error.status))throw error;status.textContent='Waiting for current node work…';await waitForMiningRetry(controller.signal);continue;}
      if(job.genesis_hash!==identity.genesis_hash)throw Error('mining_network_changed');
      status.textContent=`Solving block ${job.height}, ${job.difficulty} bits…`;
      const found=await solveCurrentJob({job,workerURL:'/canonical-pow-worker.mjs',signal:controller.signal,
        status:async()=>{const tip=await api('/api/mine/tip');if(tip.genesis_hash!==identity.genesis_hash)throw Object.assign(Error('mining_network_changed'),{fatal:true});return {current:tip.height+1===job.height&&tip.hash===job.prev_hash};},
        progress:p=>{status.textContent=p.connection?'Waiting for node…':`Block ${job.height}: ${p.hashes} hashes`;}});
      if(found.restart)continue;
      let receipt;
      try{receipt=await api('/api/mine/submit',{address:job.miner,job_id:job.job_id,nonce:found.solution.nonce});}
      catch(error){
        if(controller.signal.aborted)throw error;
        const saved=await api(`/api/mine/result?height=${job.height}&hash=${found.solution.hash}&address=${encodeURIComponent(miner)}`);
        if(saved.accepted&&saved.genesis_hash===identity.genesis_hash)receipt=saved;
        else if(error.status===409)continue;else throw error;
      }
      if(!receipt.accepted||receipt.hash!==found.solution.hash)throw Error('node_pow_receipt_mismatch');
      status.textContent=`Block ${receipt.height} accepted locally. Awaiting network confirmations and maturity. Continuing…`;
    }
  }catch(error){if(!controller.signal.aborted)status.textContent=error.message;}
  finally {controller.abort();if(active===controller)idle();}
};
