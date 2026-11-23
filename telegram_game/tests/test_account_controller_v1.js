const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path'),vm=require('node:vm');
const source=fs.readFileSync(path.join(__dirname,'../telegram_game/dev_ui/account_wallet.js'),'utf8');
function browser(){
  const nodes=new Map(),calls=[];let failWithdraw=false;
  class Element{
    constructor(id=''){this.id=id;this.value='';this.textContent='';this.disabled=false;this.handlers={};}
    set innerHTML(value){this.html=value;for(const match of value.matchAll(/id="([^"]+)"/g))nodes.set(match[1],new Element(match[1]));}
    get innerHTML(){return this.html||'';}
    appendChild(node){nodes.set(node.id,node);}
    addEventListener(event,handler){this.handlers[event]=handler;}
  }
  for(const id of ['walletPanel','tgwtRewardsMount','walletQCValue','walletTGWTValue','walletLevelValue','hudTGWT'])nodes.set(id,new Element(id));
  const window={QCPlayer:{id:()=> 'tg_9',ready:Promise.resolve({ok:true,playerId:'tg_9'})},Telegram:{WebApp:{initData:'test-only-init'}},QCScene:{applyAccount(){} }};
  const document={hidden:false,getElementById:id=>nodes.get(id),createElement:()=>new Element(),addEventListener(){}};
  const fetch=async(url,options={})=>{
    calls.push({url,options});
    if(url==='/api/v1/withdraw/request'&&failWithdraw)throw Error('uncertain_transport');
    let data={ok:true};
    if(url.startsWith('/api/wallet?'))data={playerId:'tg_9',qc_balance:'0',xpTotal:0,level:1,real_qc_balance:'0.000001',real_qc_spendable:'0.000001',real_qc_ok:true,linked_qc_address:'public-test-address'};
    else if(url.startsWith('/api/v1/tgwt/ledger'))data={ok:true,balances:{earned_u:2000001,verified_u:2000001,pending_u:0,reserved_u:0,withdrawn_u:0},events:[],pool:{total:'3000000',distributed:'2.000001'}};
    else if(url.startsWith('/api/v1/withdraw/list'))data={ok:true,items:[]};
    else if(url==='/api/v1/withdraw/request')data={ok:true,requestId:'test-reservation',status:'ADMIN_REVIEW'};
    else if(url==='/api/wallet/send-qc')data={ok:true,txid:'test-public-tx'};
    return {ok:true,json:async()=>data};
  };
  const context={window,document,fetch,crypto:{randomUUID:()=> 'test-only-request-12345678'},navigator:{clipboard:{writeText:async()=>{}}},setInterval(){},console};
  vm.runInNewContext(source,context);
  return {window,nodes,calls,context,setFail:value=>{failWithdraw=value;}};
}
(async()=>{
  const b=browser();await b.window.QCAccount.refresh();
  assert.equal(b.nodes.get('qcReceiveBalance').textContent,'0.000001');
  assert.equal(b.nodes.get('walletTGWTValue').textContent,'2.000001');
  assert(Object.isFrozen(b.window.QCAccount));
  const identity=b.window.QCAccount;vm.runInNewContext(source,b.context);assert.equal(b.window.QCAccount,identity);
  const signed='{"transaction":{"Sender":"public-test-address","Outputs":[{"Amount":9007199254740993}],"Inputs":[]}}';
  b.nodes.get('qcSendSignedTx').value=signed;await b.nodes.get('btnSendQC').handlers.click();
  const send=b.calls.find(c=>c.url==='/api/wallet/send-qc');assert.equal(JSON.parse(send.options.body).transaction,signed,'Signed int64 JSON must remain byte-for-byte inside its envelope');
  b.calls.length=0;b.nodes.get('qcSendSignedTx').value='{"transaction":{"seed":"must-not-send"}}';await b.nodes.get('btnSendQC').handlers.click();assert(!b.calls.some(c=>c.url==='/api/wallet/send-qc'));
  b.nodes.get('tgwtApiWithdrawTo').value='0x'+'12'.repeat(20);b.nodes.get('tgwtApiWithdrawAmount').value='1.000001';b.setFail(true);
  await b.nodes.get('tgwtApiWithdrawBtn').handlers.click();
  b.setFail(false);await b.nodes.get('tgwtApiWithdrawBtn').handlers.click();
  const attempts=b.calls.filter(c=>c.url==='/api/v1/withdraw/request').map(c=>JSON.parse(c.options.body));
  assert.equal(attempts.length,2);assert.equal(attempts[0].amount,'1.000001');assert.equal(attempts[0].nonce,attempts[1].nonce);
  assert.equal(b.nodes.get('tgwtApiWithdrawAmount').value,'1.000001','Refresh must preserve entered decimal text');
  b.calls.length=0;b.nodes.get('tgwtApiWithdrawAmount').value='1.0000001';await b.nodes.get('tgwtApiWithdrawBtn').handlers.click();assert(!b.calls.some(c=>c.url==='/api/v1/withdraw/request'));
  assert.equal(b.window.QCAccount.units('9007199254740993'),'9007199254.740993');
  const scene=fs.readFileSync(path.join(__dirname,'../telegram_game/dev_ui/game_scene.js'),'utf8');
  const frame=scene.match(/function __loop\(ts\) \{[\s\S]*?\n\}/)[0];
  const drawn=[];
  const sceneContext={window:{__safeCall(fn,label,...args){if(typeof fn==='function')return fn(...args);}},
    isMineSceneOpen:()=>false,currentUiMode:'start',__lastTs:0,__rafId:0,miningRAF:0,requestAnimationFrame:()=>1};
  for(const name of ['updateXP','updateStars','updateMeteors','updateParticles','updateShip','updateAutoMine','drawScene','refreshHUDFromState'])sceneContext[name]=()=>drawn.push(name);
  vm.runInNewContext(frame+';__loop(1000);',sceneContext);
  assert(drawn.includes('drawScene'),'Private scene functions must render without window exports');
  assert(drawn.includes('updateShip'));assert(drawn.includes('refreshHUDFromState'));
  console.log('PASS: exact account rendering, immutable owner, signed int64 transport, secret rejection, decimal withdrawal and stable retry nonce');
})().catch(error=>{console.error(error);process.exitCode=1;});
