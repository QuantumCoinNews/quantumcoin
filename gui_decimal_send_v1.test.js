const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const html = fs.readFileSync(process.argv[2] || 'webui/dist/index.html', 'utf8');
const script = html.match(/<script>\s*([\s\S]*?)<\/script>/i)?.[1];
assert.ok(script, 'wallet script missing');
assert.ok(!html.includes('id="qc-send-priv"'), 'browser must not collect a private key');

async function submit(raw) {
  const controls = new Map(), requests = [];
  const document = {getElementById(id) {
    if (!controls.has(id)) controls.set(id, {value:'', textContent:'', addEventListener(event,fn){this[event]=fn;}});
    return controls.get(id);
  }};
  const fetch = async (url, options) => {requests.push({url,options});return {ok:true,json:async()=>({success:true})};};
  vm.runInNewContext(script,{document,fetch,JSON,encodeURIComponent});
  document.getElementById('qc-send-signed').value = raw;
  document.getElementById('qc-api-token').value = 'test-token';
  await controls.get('qc-btn-send').click();
  return requests;
}
(async()=>{
  // Raw JSON is forwarded unchanged: parsing/re-encoding this integer in JS
  // would invalidate a perfectly valid wallet signature.
  const raw='{"transaction":{"Outputs":[{"Amount":9007199254740993,"PubKeyHash":"public"}],"Sender":"owner"}}';
  const requests=await submit(raw);
  assert.equal(requests.length,1);
  assert.equal(requests[0].url,'/api/tx/send');
  assert.equal(requests[0].options.body,raw);
  assert.equal(requests[0].options.headers['X-QC-API-KEY'],'test-token');
  for(const invalid of ['{}','not json','{"transaction":{},"priv_hex":"secret"}','{"transaction":{"privateKey":"secret"}}']) assert.equal((await submit(invalid)).length,0);
  console.log('Signed wallet envelope, exact numeric bytes, secret rejection and header authentication: PASS');
})().catch(error=>{console.error(error);process.exitCode=1;});
