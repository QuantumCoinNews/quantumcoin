const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { pathToFileURL } = require('node:url');
const source = fs.readFileSync(path.resolve(__dirname,'../telegram_game/dev_ui/verified_actions.js'),'utf8');
const tick = () => new Promise(resolve => setImmediate(resolve));
let miningHelpers;
// Resolve the browser module boundary into the real shared helper. The VM gets
// its own Worker factory; no experimental VM modules or simulated PoW helper.
const browserImport="import('/mining/mining-session.mjs')";
assert.equal(source.split(browserImport).length,2,'Review the browser module import contract');
const executableSource=source.replace(browserImport,"loadMiningSession('/mining/mining-session.mjs')");

function browser({deferred=false,mismatch=false,denyReward=false,recovered=false}={}) {
  const calls=[],workers=[],events=[];
  let release;
  const wait=deferred?new Promise(resolve=>{release=resolve;}):Promise.resolve();
  class Worker {
    constructor(url,options){this.url=url;this.options=options;workers.push(this);}
    postMessage(job){this.job=job;}
    terminate(){this.stopped=true;}
  }
  let refreshes=0;
  const window={QC_TELEGRAM_IDENTITY_READY:Promise.resolve(),QCGetPlayerId:()=> 'tg_9',
    QC_refreshWalletFromBackend:async()=>{refreshes++;},dispatchEvent:event=>events.push(event)};
  const fetch=async(url,options)=>{
    const body=JSON.parse(options.body);calls.push({url,body});
    let data;
    if(url==='/api/mine/work') {
      await wait;
      data=recovered?{ok:true,alreadyAccepted:true,receipt:{hash:'ab'.repeat(32)}}:
        {ok:true,job:{protocol:'QC_WEB_BLOCK_POW_V1',header:'canonical-node-header',job_id:'test-job',height:42,expires:Date.now()/1000+60}};
    } else if(url==='/api/mine/work/submit') data={ok:true,receipt:{hash:(mismatch?'cd':'ab').repeat(32),height:42}};
    else data=denyReward?{ok:false,error:'provider_evidence_invalid'}:{ok:true,credited_u:1000000,onChainPaid:false};
    return new Response(JSON.stringify(data),{status:200});
  };
  const loadMiningSession=async specifier=>{
    assert.equal(specifier,'/mining/mining-session.mjs');
    return {...miningHelpers,solveCurrentJob:options=>miningHelpers.solveCurrentJob({...options,
      workerFactory:()=>new Worker(options.workerURL,{type:'module'})})};
  };
  vm.runInNewContext(executableSource,{window,fetch,Worker,loadMiningSession,AbortController,crypto:{subtle:{}},CustomEvent:class{constructor(type,options){this.type=type;this.detail=options.detail;}},Response});
  return {window,calls,workers,events,release,refreshes:()=>refreshes};
}

async function main(){
  miningHelpers=await import(pathToFileURL(path.resolve(__dirname,'../../webui/dist/mining-session.mjs')));
  let count=0;
  const session={playerId:'tg_9',sessionId:'server-owned-session'};
  const b=browser();
  const solved=b.window.QCStartVerifiedPoW(session);
  await tick();
  assert.equal(b.workers[0].url,'/mining/canonical-pow-worker.mjs');
  assert.equal(b.workers[0].job.header,'canonical-node-header');
  await b.workers[0].onmessage({data:{nonce:'9007199254740993',hash:'ab'.repeat(32)}});
  await solved;
  assert.equal(b.calls[1].body.nonce,'9007199254740993');
  assert.equal(b.calls[1].body.sessionId,session.sessionId);
  assert.equal(b.window.QC_CLIENT_POW_STATUS.status,'accepted');
  assert(b.calls.some(call=>call.url==='/api/mine/work'&&call.body.nextBlock===true),'Accepted miner must request the canonical next job');
  b.window.QCStopVerifiedPoW();await tick();
  assert(b.workers.every(worker=>worker.stopped),'Stop must release all device workers');
  assert.ok(b.calls.every(call=>call.url!=='/api/mine'&&call.body.amount===undefined));
  count++;

  const stopped=browser({deferred:true});
  const pending=stopped.window.QCStartVerifiedPoW(session);
  const rejected=assert.rejects(pending,/mining_stopped/);
  stopped.window.QCStopVerifiedPoW();stopped.release();await rejected;
  assert.equal(stopped.workers.length,0,'stopped request must not launch a late worker');
  count++;

  const bad=browser({mismatch:true});
  const work=bad.window.QCStartVerifiedPoW(session);
  const badResult=assert.rejects(work,/node_pow_receipt_mismatch/);
  await tick();await bad.workers[0].onmessage({data:{nonce:'1',hash:'ab'.repeat(32)}});
  await badResult;
  assert.equal(bad.window.QC_CLIENT_POW_STATUS.status,'stopped');
  count++;

  const recover=browser({recovered:true});
  const result=await recover.window.QCStartVerifiedPoW(session);
  assert.equal(result.receipt.hash,'ab'.repeat(32));
  recover.window.QCStopVerifiedPoW();await tick();
  assert.equal(recover.workers.length,0);
  count++;

  const proof={statement:{test:'provider-only'},signature:'test-only-signature'};
  await b.window.QCSubmitVerifiedTaskReward('test-campaign',proof);
  assert.deepEqual(JSON.parse(JSON.stringify(b.calls.find(call=>call.url==='/api/tgwt/rewards/claim').body)),{playerId:'tg_9',campaignId:'test-campaign',proof});
  assert.equal(b.refreshes(),1);assert.equal(b.events[0].type,'qc:verified-reward');
  assert.equal(b.events[0].detail.onChainPaid,false);
  count++;

  const denied=browser({denyReward:true});
  await assert.rejects(denied.window.QCSubmitVerifiedTaskReward('task',proof),/provider_evidence_invalid/);
  assert.equal(denied.refreshes(),0);assert.equal(denied.events.length,0);
  count++;
  console.log('PASS: '+count+' device-PoW and verified reward UI contracts');
}
main().catch(error=>{console.error(error);process.exitCode=1;});
