const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const root = process.env.QC_PROJECT_ROOT || path.resolve(__dirname, '../..');
const source = fs.readFileSync(path.join(root, 'telegram_game/telegram_game/dev_ui/player_session.js'), 'utf8');

function browser({initData = 'test-signed-data', loginOk = true, expired = false, deferred = false} = {}) {
  const calls = [];
  const notices = [];
  let release;
  const pending = deferred ? new Promise(resolve => { release = resolve; }) : Promise.resolve();
  const window = {
    location: {href: 'https://game.test/game', origin: 'https://game.test'},
    Telegram: {WebApp: {initData}},
    fetch: async (input, options = {}) => {
      calls.push({input, options});
      if (String(input).includes('/identity/verify')) {
        await pending;
        return new Response(JSON.stringify(loginOk ? {ok:true, accessToken:'test-bearer-only-in-memory', playerId:'tg_9', expiresAt:42} : {ok:false}), {status:loginOk?200:401});
      }
      return new Response('{}', {status:expired?401:200});
    },
  };
  const document = {
    body: {appendChild: element => notices.push(element)},
    getElementById: id => notices.find(element => element.id === id),
    createElement: () => ({setAttribute(){},style:{}}), addEventListener(){},
  };
  const context = {window, document, URL, Request, Headers, Response,
    localStorage: {getItem(){throw Error('identity must never use localStorage');},setItem(){throw Error('credential must never be persisted');}}};
  vm.runInNewContext(source, context);
  return {window, calls, notices, release};
}

async function main() {
  let count = 0;
  for (const settings of [{initData:''}, {loginOk:false}]) {
    const b = browser(settings);
    await b.window.QC_TELEGRAM_IDENTITY_READY;
    assert.equal(b.window.QCGetPlayerId(), '');
    assert.equal(b.notices.length, 1);
    await assert.rejects(b.window.fetch('/api/wallet?playerId=dev_user'), /telegram_session_required/);
    assert.equal(b.calls.length, settings.initData === '' ? 0 : 1);
    count++;
  }

  const b = browser({deferred:true});
  const exactBody = '{"playerId":"tg_9","transaction":"{\\"Outputs\\":[{\\"Amount\\":9007199254740993}]}"}';
  const submission = b.window.fetch('/api/wallet/send-qc', {method:'POST',body:exactBody,headers:{'Content-Type':'application/json'}});
  await Promise.resolve();
  assert.equal(b.calls.length, 1, 'private call must wait for login');
  b.release();
  await submission;
  assert.equal(b.calls[1].options.body, exactBody, 'signed JSON bytes were rewritten');
  assert.equal(b.calls[1].options.headers.get('Authorization'), 'Bearer test-bearer-only-in-memory');
  assert.equal(b.calls[1].options.redirect, 'error');
  assert.equal(b.window.QCGetPlayerId(), 'tg_9');
  assert.ok(!JSON.stringify(b.window.QC_TG_IDENTITY).includes('test-bearer'));
  count++;

  await assert.rejects(b.window.fetch('https://attacker.test/api/wallet'), /game_api_origin_mismatch/);
  assert.equal(b.calls.length, 2, 'cross-origin API request must not be sent');
  await b.window.fetch('https://cdn.test/image.png');
  assert.equal(b.calls[2].options.headers, undefined, 'game bearer leaked to external asset');
  count++;

  const request = new Request('https://game.test/api/settings', {method:'POST',body:'{"playerId":"tg_9"}',headers:{'X-Test':'request'}});
  await b.window.fetch(request, {headers:{'X-Extra':'options'}});
  assert.equal(b.calls[3].input, request);
  assert.equal(b.calls[3].options.headers.get('X-Test'), 'request');
  assert.equal(b.calls[3].options.headers.get('X-Extra'), 'options');
  count++;

  const expired = browser({expired:true});
  await expired.window.QC_TELEGRAM_IDENTITY_READY;
  await expired.window.fetch('/api/wallet?playerId=tg_9');
  assert.equal(expired.window.QCGetPlayerId(), '');
  assert.equal(expired.notices.length, 1);
  await assert.rejects(expired.window.fetch('/api/wallet?playerId=tg_9'), /telegram_session_required/);
  count++;

  const html = fs.readFileSync(path.join(root,'telegram_game/telegram_game/dev_ui/game.html'),'utf8');
  const scripts = [];
  for (const match of html.matchAll(/<script\b([^>]*)>([\s\S]*?)<\/script>/gi)) {
    assert.equal(match[2].trim(), '', 'No executable inline scripts');
    const src=match[1].match(/src="([^"]+)"/)[1];
    if (!src.startsWith('/dev_ui/')) continue;
    new vm.Script(fs.readFileSync(path.join(root,'telegram_game/telegram_game',src.replace(/^\//,'')), 'utf8'),{filename:src});
    scripts.push(src);
  }
  assert.equal(new Set(scripts).size,scripts.length,'Duplicate controller loaded');
  for(const required of ['/dev_ui/player_session.js','/dev_ui/verified_actions.js',
    '/dev_ui/account_wallet.js','/dev_ui/qc_local_wallet.js','/dev_ui/wallet_recovery.js','/dev_ui/wallet_link.js']) {
    assert(scripts.includes(required),'Required wallet/identity controller missing: '+required);
  }
  assert(Object.isFrozen(b.window.QCPlayer));
  try {b.window.getPlayerId=()=> 'forged';b.window.QC_TG_IDENTITY.playerId='forged';} catch(_) {}
  assert.equal(b.window.getPlayerId(),'tg_9');
  console.log('PASS: '+count+' browser identity contracts; '+scripts.length+' controller scripts parsed');
}
main().catch(error => { console.error(error); process.exitCode = 1; });
