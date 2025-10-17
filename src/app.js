// src/app.js
// Basit statik oyun iskeleti: Canvas harita, ödül motoru, TGWT, Sosyal görevler (TikTok dahil), MetaMask stub.

const $ = s => document.querySelector(s);
const $$ = s => document.querySelectorAll(s);

const state = loadState();
let economy = null;
let rafId = 0;
let sessionTimer = null;

// ---------- Bootstrap ----------
(async function boot(){
  economy = await fetch('/config/economy.json').then(r=>r.json());
  initUI();
  resizeCanvas();
  window.addEventListener('resize', resizeCanvas);
  startRenderLoop();
  buildShop();
  updateRef();
  render();
})();

// ---------- STATE ----------
function loadState(){
  const def = {
    qc: 0, tgwt: 0, level: 1,
    energy: 100, energyMax: 100,
    shipName: 'Basic Miner', shipMult: 1,
    running: false, timeLeft: 0,
    mm: null, // metamask
    inv: { nfts: [] }, // inventory
    counts: { sessions: 0, ref: 0 },
    cooldowns: { tgwtLast: 0, daily: 0, pityCounter: 0 }
  };
  try{
    const s = JSON.parse(localStorage.getItem('qc_state')||'{}');
    return Object.assign(def, s);
  }catch{ return def; }
}
function saveState(){ localStorage.setItem('qc_state', JSON.stringify(state)); }

// ---------- UI INITIALIZE ----------
function initUI(){
  // Tab geçiş
  $$('.tabs button').forEach(btn=>{
    btn.addEventListener('click', ()=>{
      $('.tabs button.active')?.classList.remove('active');
      btn.classList.add('active');
      $$('.tab').forEach(p=>p.classList.remove('active'));
      $('#tab-'+btn.dataset.tab).classList.add('active');
    });
  });
  // Start/Stop
  $('#btnStart').onclick = toggleMining;
  // Enerji +50
  $('#btnEnergy').onclick = ()=>{ state.energy = Math.min(state.energyMax, state.energy+50); toast('⚡ +50 enerji'); render(); saveState(); };
  // Daily
  $('#btnDaily').onclick = openDailyModal;
  $('#btnDailyQuest').onclick = claimDaily;
  // TGWT claim
  $('#btnTGWT').onclick = claimTGWT;
  // Store
  $('#btnStore').onclick = ()=> openStore();
  $('#btnMore').onclick = ()=> openModal('<h3>Daha Fazla</h3><p>TGWT çekim geçmişi, ayarlar vb.</p>');
  // Wallet actions
  $('#btnConnect').onclick = connectMM;
  $('#btnReceive').onclick = ()=> openModal(`<h3>QC Al</h3><p>Cüzdan adresin:</p><code>${state.mm||'Bağlanmadı'}</code>`);
  $('#btnSend').onclick = openSendModal;
  $('#btnWithdrawTGWT').onclick = withdrawTGWT;
  // Modal close
  $('#modalClose').onclick = closeModal;
  $('#modal').addEventListener('click', e=>{ if(e.target.id==='modal') closeModal(); });

  // Sosyal görevler global fonksiyon
  window.claimSocial = claimSocial;

  // Canvas click handler (kazım noktaları)
  $('#mapCanvas').addEventListener('mousedown', onCanvasClick);
}

// ---------- RENDER ----------
function render(){
  $('#qc').textContent = state.qc.toFixed(2);
  $('#tgwt').textContent = state.tgwt;
  $('#energy').textContent = state.energy;
  $('#energyMax').textContent = state.energyMax;
  $('#energyNow').textContent = state.energy;
  $('#energyMax2').textContent = state.energyMax;
  $('#level').textContent = state.level;
  $('#shipName').textContent = state.shipName;
  $('#mult').textContent = 'x'+state.shipMult;
  $('#status').textContent = state.running ? 'MINING' : 'READY';
  $('#timer').textContent = fmt(state.timeLeft);
}
function fmt(s){ const m=String(Math.floor(s/60)).padStart(2,'0'); const r=String(s%60).padStart(2,'0'); return `${m}:${r}`}

// ---------- CANVAS / MAP ----------
const map = { planets: [], mouse: {x:0,y:0} };

function resizeCanvas(){
  const c = $('#mapCanvas');
  const dpr = window.devicePixelRatio || 1;
  const rect = $('#gameArea').getBoundingClientRect();
  c.width = Math.floor(rect.width * dpr);
  c.height = Math.floor(rect.height * dpr);
  c.style.width = rect.width+'px';
  c.style.height = rect.height+'px';
  buildPlanets();
}

function buildPlanets(){
  map.planets = [];
  const c = $('#mapCanvas'), W = c.width, H = c.height, ctx = c.getContext('2d');
  const cols = economy.mining.planets;
  const spacing = W/(cols+1), y = H*0.45;
  for(let i=0;i<cols;i++){
    const x = spacing*(i+1);
    const r = 90 * (window.devicePixelRatio||1);
    const spots = [];
    const RS = 150 * (window.devicePixelRatio||1);
    for(let s=0;s<economy.mining.spotsPerPlanet;s++){
      const angle = (Math.PI*2/economy.mining.spotsPerPlanet)*s;
      spots.push({ x: x + Math.cos(angle)*RS, y: y + Math.sin(angle)*RS, a: angle, pulse: Math.random() });
    }
    map.planets.push({ x, y, r, hue: 130+ i*30, spots, rot: Math.random()*Math.PI*2 });
  }
  // Pre-draw grid bg into offscreen if wanted (şimdi animasyonlu yıldız alanı kullanacağız)
}

function startRenderLoop(){
  const c = $('#mapCanvas'), ctx = c.getContext('2d');
  const dpr = window.devicePixelRatio||1;
  let t0 = performance.now();
  function loop(t){
    const dt = (t - t0)/1000; t0 = t;
    ctx.clearRect(0,0,c.width,c.height);

    // Starfield
    drawStars(ctx, c.width, c.height, t);

    // Planets & nodes
    for(const p of map.planets){
      p.rot += dt*0.1;
      drawPlanet(ctx, p);
      drawOrbit(ctx, p);
      for(const s of p.spots){
        s.pulse += dt;
        drawNode(ctx, s);
      }
    }
    rafId = requestAnimationFrame(loop);
  }
  rafId = requestAnimationFrame(loop);
}

function drawStars(ctx,W,H,t){
  ctx.save();
  const count = 250;
  for(let i=0;i<count;i++){
    const x = (i*9973 % W); // deterministik dağılım
    const y = (i*7919 % H);
    const tw = (Math.sin((t/500)+(i*13)) + 1.8)/2.8;
    ctx.fillStyle = `rgba(0,230,118,${0.15*tw})`;
    ctx.fillRect(x, y, 2, 2);
  }
  ctx.restore();
}
function drawPlanet(ctx,p){
  const g = ctx.createRadialGradient(p.x, p.y, p.r*0.2, p.x, p.y, p.r);
  g.addColorStop(0, `hsla(${p.hue},90%,60%,.9)`);
  g.addColorStop(1, `hsla(${p.hue},90%,30%,.1)`);
  ctx.fillStyle = g;
  ctx.beginPath(); ctx.arc(p.x, p.y, p.r, 0, Math.PI*2); ctx.fill();
  // Glow
  ctx.strokeStyle = `hsla(${p.hue},100%,60%,.35)`;
  ctx.lineWidth = 4; ctx.beginPath(); ctx.arc(p.x, p.y, p.r+8, 0, Math.PI*2); ctx.stroke();
}
function drawOrbit(ctx,p){
  ctx.save();
  ctx.strokeStyle = 'rgba(0,230,118,.18)';
  ctx.setLineDash([8,8]); ctx.lineWidth = 1.5;
  const R = 150 * (window.devicePixelRatio||1);
  ctx.beginPath(); ctx.arc(p.x, p.y, R, 0, Math.PI*2); ctx.stroke();
  ctx.restore();
}
function drawNode(ctx,s){
  const amp = 1 + Math.sin(s.pulse*3)*0.3;
  ctx.beginPath();
  ctx.fillStyle = `rgba(0,230,118,0.9)`;
  ctx.arc(s.x, s.y, 6*amp*(window.devicePixelRatio||1), 0, Math.PI*2);
  ctx.fill();
  // outer glow
  ctx.beginPath();
  ctx.fillStyle = `rgba(0,230,118,0.18)`;
  ctx.arc(s.x, s.y, 12*amp*(window.devicePixelRatio||1), 0, Math.PI*2);
  ctx.fill();
}

function onCanvasClick(e){
  const rect = e.target.getBoundingClientRect();
  const dpr = window.devicePixelRatio||1;
  const mx = (e.clientX - rect.left) * dpr;
  const my = (e.clientY - rect.top) * dpr;

  for(const p of map.planets){
    for(const s of p.spots){
      const dist2 = (mx-s.x)*(mx-s.x)+(my-s.y)*(my-s.y);
      const r = 12*(window.devicePixelRatio||1);
      if(dist2 < r*r){
        startMiningAt(s);
        return;
      }
    }
  }
}

// ---------- MINING ----------
function toggleMining(){
  if(state.running){ stopSession(); return; }
  // boşta start: yakın bir spot seç
  const flat = map.planets.flatMap(p=>p.spots);
  if(!flat.length) return;
  startMiningAt(flat[Math.floor(Math.random()*flat.length)]);
}
function startMiningAt(node){
  if(state.running) return;
  if(state.energy < economy.mining.energyPerSession) return toast('⚠️ Enerji yetersiz');
  state.energy -= economy.mining.energyPerSession;
  state.running = true;
  state.timeLeft = economy.mining.sessionSeconds;
  $('#btnStart').textContent = '⏸️ Stop';
  render(); saveState();

  sessionTimer = setInterval(()=>{
    state.timeLeft = Math.max(0, state.timeLeft-1);
    if(state.timeLeft===0){ completeSession(); }
    render();
  }, 1000);

  // küçük “spark” efekti
  spark(node.x, node.y);
}
function stopSession(){
  state.running = false;
  clearInterval(sessionTimer);
  $('#btnStart').textContent = `▶️ Start (${economy.mining.sessionSeconds}s)`;
  render(); saveState();
}
function completeSession(){
  clearInterval(sessionTimer);
  state.running = false;
  $('#btnStart').textContent = `▶️ Start (${economy.mining.sessionSeconds}s)`;
  // Ödül hesap (demo – gerçek sunucuda)
  const base = economy.mining.baseYield * (state.shipMult||1);
  const qcGain = +(base + Math.random()*0.5).toFixed(2);
  state.qc += qcGain;
  state.counts.sessions++;
  // NFT şansı
  const pity = state.cooldowns.pityCounter || 0;
  const r = Math.random()*100;
  let nft = null;
  if(r < economy.rarity.nftTegetDropPct*100) nft = 'teget';
  else if(r < economy.rarity.nftSpecialDropPct*100 + Math.min(pity*0.01, 1)) nft = 'special';
  if(nft){
    state.inv.nfts.push({ id: 'nft_'+Date.now(), kind: nft });
    state.cooldowns.pityCounter = 0;
    toast(`🎉 ${nft==='teget'?'TEGET ENDER':'Özel'} NFT kazandın!`);
  }else{
    state.cooldowns.pityCounter = (pity+1) % (economy.rarity.pityAfterSessions||200);
  }
  toast(`⛏️ +${qcGain.toFixed(2)} QC`);
  render(); saveState();
}

// basit görsel kıvılcım
function spark(x,y){
  const c = $('#mapCanvas'), ctx = c.getContext('2d');
  let a = 1, r = 10*(window.devicePixelRatio||1);
  const id = setInterval(()=>{
    ctx.save();
    ctx.globalAlpha = a; a -= 0.08;
    ctx.beginPath(); ctx.strokeStyle = 'rgba(0,230,118,1)'; ctx.lineWidth = 2;
    ctx.arc(x,y,r,0,Math.PI*2); ctx.stroke(); r+=6;
    ctx.restore();
    if(a<=0) clearInterval(id);
  }, 30);
}

// ---------- TGWT ----------
function claimTGWT(){
  const now = Date.now();
  const last = state.cooldowns.tgwtLast || 0;
  const cdMs = economy.tgwt.cooldownHours * 3600 * 1000;
  const remain = last + cdMs - now;
  if(remain > 0){
    const h = Math.ceil(remain/3600000);
    return toast(`⏍ TGWT için ${h} saat sonra tekrar dene`);
  }
  state.tgwt += economy.tgwt.perClaim;
  state.cooldowns.tgwtLast = now;
  toast('⏍ +1 TGWT');
  render(); saveState();
}

async function withdrawTGWT(){
  if(!state.mm){ await connectMM(); if(!state.mm) return; }
  if(state.tgwt < economy.tgwt.withdrawMin) return toast(`Min ${economy.tgwt.withdrawMin} TGWT gerekir`);
  // Sunucu isteği burada olmalı
  toast(`⏍ ${state.tgwt} TGWT çekim talebi alındı → ${state.mm}`);
  state.tgwt = 0; render(); saveState();
}

// ---------- SOSYAL GÖREVLER (TikTok dahil) ----------
function claimSocial(platform){
  const key = 'social_'+platform;
  if(localStorage.getItem(key)==='1') return toast('Bu görev zaten tamamlandı');
  state.tgwt += (economy.tgwt.socialFollowOneTime||1);
  localStorage.setItem(key,'1');
  toast(`📣 ${platform} görevi tamamlandı! +1 TGWT`);
  render(); saveState();
}

// ---------- GÜNLÜK ----------
function openDailyModal(){
  const last = state.cooldowns.daily || 0;
  const can = !isSameDay(last, Date.now());
  openModal(`<h3>Günlük Görev</h3>
    <p>Her gün girişe +${economy.daily.loginRewardQc} QC.</p>
    <button class="btn" ${can?'':'disabled'} id="doDaily">${can?'Al':'Bugün alındı'}</button>`);
  $('#doDaily')?.addEventListener('click', ()=>{ claimDaily(); closeModal(); });
}
function claimDaily(){
  const last = state.cooldowns.daily || 0;
  if(isSameDay(last, Date.now())) return toast('Bugünün ödülü zaten alındı');
  state.cooldowns.daily = Date.now();
  state.qc += economy.daily.loginRewardQc;
  toast(`🗓️ +${economy.daily.loginRewardQc} QC (günlük)`);
  render(); saveState();
}
function isSameDay(a,b){ const da=new Date(a), db=new Date(b); return da.getFullYear()===db.getFullYear() && da.getMonth()===db.getMonth() && da.getDate()===db.getDate(); }

// ---------- STORE / GEMİLER ----------
function buildShop(){
  const wrap = $('#shopList'); wrap.innerHTML = '';
  economy.shop.forEach(it=>{
    const card = document.createElement('div'); card.className='card';
    card.innerHTML = `<h4>${it.title}</h4>
      <p>Çarpan: x${it.mult}</p>
      <button class="btn buy" data-id="${it.id}">${it.priceQc} QC</button>`;
    wrap.appendChild(card);
  });
  wrap.querySelectorAll('.buy').forEach(b=> b.onclick = ()=> buyShip(b.dataset.id));
}
function openStore(){ $('.tabs button[data-tab="ships"]').click(); }
function buyShip(id){
  const it = economy.shop.find(x=>x.id===id);
  if(!it) return;
  if(state.qc < it.priceQc) return toast('QC yetersiz');
  state.qc -= it.priceQc;
  state.shipMult = it.mult;
  state.shipName = it.title;
  toast(`🚀 ${it.title} satın alındı! x${it.mult}`);
  render(); saveState();
}

// ---------- REF / ARKADAŞ ----------
function updateRef(){
  const uid = getUserId();
  $('#refLink').textContent = `${location.origin}/?ref=${uid}`;
  $('#refCount').textContent = state.counts.ref;
}
function getUserId(){
  let id = localStorage.getItem('user_id');
  if(!id){ id = 'u_'+Math.random().toString(36).slice(2,10); localStorage.setItem('user_id', id); }
  return id;
}

// ---------- WALLET (MetaMask stub) ----------
async function connectMM(){
  if(!window.ethereum){ toast('MetaMask gerekli'); return; }
  try{
    const [acc] = await window.ethereum.request({ method:'eth_requestAccounts' });
    state.mm = acc; toast('🦊 MetaMask bağlandı'); saveState();
  }catch(e){ toast('MetaMask reddedildi'); }
}
function openSendModal(){
  openModal(`<h3>QC Gönder</h3>
    <label>Alıcı Adres</label><input id="toAddr" placeholder="0x..." />
    <label>Miktar (QC)</label><input id="amount" type="number" min="0" step="0.01" />
    <button class="btn" id="sendDo">Gönder</button>`);
  $('#sendDo').onclick = ()=>{
    const amt = parseFloat($('#amount').value||'0');
    if(amt<=0) return toast('Miktar gir');
    if(state.qc < amt) return toast('QC yetersiz');
    state.qc -= amt;
    toast(`⬆️ ${amt.toFixed(2)} QC gönderildi (demo)`);
    closeModal(); render(); saveState();
  };
}

// ---------- MODAL/TOAST ----------
function openModal(html){ $('#modalBody').innerHTML = html; $('#modal').classList.remove('hidden'); }
function closeModal(){ $('#modal').classList.add('hidden'); }
function toast(msg){
  const t = $('#toast'); t.textContent = msg; t.classList.add('show');
  clearTimeout(t._h); t._h = setTimeout(()=> t.classList.remove('show'), 2200);
}

// ---------- HELPERS ----------
function fmtMs(ms){ const s = Math.ceil(ms/1000); return fmt(s); }

// stop session güvenliği
window.addEventListener('beforeunload', ()=>{ if(state.running) stopSession(); saveState(); });
