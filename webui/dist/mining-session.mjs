// Shared client lifecycle for node-operator and Telegram device PoW.
// Status polling never submits a solution or creates a reward.
export function waitForMiningRetry(signal, millis=2000) {
  return new Promise((resolve,reject)=>{
    if(signal.aborted){reject(Error('mining_stopped'));return;}
    const abort=()=>{clearTimeout(timer);reject(Error('mining_stopped'));};
    const timer=setTimeout(()=>{signal.removeEventListener('abort',abort);resolve();},millis);
    signal.addEventListener('abort',abort,{once:true});
  });
}
export function solveCurrentJob({job, workerURL, signal, status, progress=()=>{}, pollMillis=2000, workerFactory}) {
  return new Promise((resolve,reject)=>{
    if (signal?.aborted) { reject(Error('mining_stopped'));return; }
    const worker=workerFactory?workerFactory():new Worker(workerURL,{type:'module'});
    let done=false,checking=false;
    const stop=()=>finish(null,Error('mining_stopped'));
    const timer=setInterval(check,pollMillis);
    function finish(result,error){
      if(done)return;done=true;clearInterval(timer);signal?.removeEventListener('abort',stop);worker.terminate();
      if(error)reject(error);else resolve(result);
    }
    async function check(){
      if(done||checking)return;
      if(Date.now()>=job.expires*1000){finish({restart:true});return;}
      checking=true;
      try {
        const state=await status(job);
        if(done)return;
        if(state.alreadyAccepted){finish({receipt:state.receipt});return;}
        if(state.ok===false){
          if(['mining_network_changed','telegram_session_required'].includes(state.error)) throw Object.assign(Error(state.error),{fatal:true});
          progress({connection:'waiting'});return;
        }
        if(state.current===false)finish({restart:true});
      }catch(error){if(!done){if(error.fatal)finish(null,error);else progress({connection:'waiting'});}}
      finally {checking=false;}
    }
    signal?.addEventListener('abort',stop,{once:true});
    worker.onerror=()=>finish(null,Error('device_mining_worker_failed'));
    worker.onmessage=({data})=>{
      if(done)return;
      if(data.error){if(Date.now()>=job.expires*1000)finish({restart:true});else finish(null,Error(data.error));return;}
      if(data.nonce!==undefined)finish({solution:data});else progress(data);
    };
    worker.postMessage(job);
  });
}
