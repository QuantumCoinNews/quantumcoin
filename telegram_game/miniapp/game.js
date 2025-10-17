/* Quantum Mining – single file Phaser scene (no external assets) */
const STATE = {
  qc: 0,
  energy: 100,
  energyMax: 100,
  level: 1,
  shipName: "Basic Miner",
  power: 1.0,
  mining: false,
  t: 0,
  duration: 60,
  lastDaily: ""
};

const W = Math.min(1280, window.innerWidth);
const H = Math.min(720,  window.innerHeight);

const config = {
  type: Phaser.AUTO,
  parent: 'game',
  width: W,
  height: H,
  backgroundColor: '#0a0f1f',
  physics: { default: 'arcade' },
  scene: { preload, create, update }
};

new Phaser.Game(config);

let txtCenter, planet, ship, beam, stars, timerText;

function preload(){}

function create(){
  const c = this;

  // Starfield particles
  const g = new Phaser.Geom.Rectangle(0,0,W,H);
  stars = c.add.particles(0,0,'').createEmitter({
    x: { min: 0, max: W }, y: { min: 0, max: H },
    speedX: { min:-10, max:10 }, speedY:{min:-8,max:8},
    lifespan: 6000, quantity: 2, scale: { start:.8, end:0 },
    blendMode: 'ADD', tint: 0x8fbaff, emitZone:{ type:'random', source:g }
  });

  // Planet (vector draw)
  planet = c.add.circle(W*0.5, H*0.52, 120, 0x10244f).setStrokeStyle(4, 0x4bb6ff, 0.7);
  c.tweens.add({ targets: planet, angle: 360, duration: 40000, repeat: -1 });

  // Ship (triangle)
  const shipPath = new Phaser.Curves.Path(0,0);
  const shipGraphics = c.add.graphics({ x: W*0.38, y: H*0.38 });
  shipGraphics.fillStyle(0x7ad8ff, 1);
  shipGraphics.fillTriangle(0,30, 60,0, 60,60);
  ship = shipGraphics;

  // Beam
  beam = c.add.rectangle(W*0.44, H*0.48, 16, 180, 0x39e0ff, 0.55).setAngle(20);
  c.tweens.add({ targets: beam, alpha: { from:.25, to:.8 }, yoyo:true, duration:500, repeat:-1 });

  // Center text (countdown / prompts)
  txtCenter = c.add.text(W/2, 80, "Quantum Mining", {
    fontFamily:'Inter, Arial', fontSize: 26, color:'#eaf2ff'
  }).setOrigin(0.5,0.5);

  timerText = c.add.text(W/2, 110, "", {
    fontFamily:'Inter, Arial', fontSize: 18, color:'#9fc0ff'
  }).setOrigin(0.5,0.5);

  // UI buttons (from DOM)
  document.getElementById('startBtn').onclick = startMining;
  document.getElementById('adBtn').onclick    = () => gainEnergy(50);
  document.getElementById('dailyBtn').onclick = claimDaily;
  document.getElementById('tgwtBtn').onclick  = completeTGWT;
  document.getElementById('shopBtn').onclick  = openShop;

  // Social (fake tasks)
  document.getElementById('xFollow').onclick = (e)=>{ e.preventDefault(); addTGWT(1,"x"); };
  document.getElementById('tgJoin').onclick  = (e)=>{ e.preventDefault(); addTGWT(1,"telegram"); };
  document.getElementById('ytSub').onclick   = (e)=>{ e.preventDefault(); addTGWT(1,"youtube"); };

  // initial HUD
  refreshHUD();
}

function update(_, dtMs){
  const dt = dtMs/1000;
  if (STATE.mining){
    STATE.t += dt;
    const left = Math.max(0, Math.ceil(STATE.duration - STATE.t));
    timerText.setText(`⛏️ Mining… ${left}s`);
    // gentle float
    ship.y = (H*0.38) + Math.sin(STATE.t*2)*4;
    if (STATE.t >= STATE.duration){
      STATE.mining = false;
      const reward = +(rand(1,5) * STATE.power).toFixed(2);
      STATE.qc += reward;
      txtCenter.setText(`✅ +${reward} QC earned`);
      timerText.setText("Press ▶ to mine again");
      refreshHUD();
    }
  } else {
    ship.y = (H*0.38) + Math.sin(performance.now()/400)*2;
  }
}

function startMining(){
  if (STATE.mining) return;
  if (STATE.energy < 10){
    txtCenter.setText("⚠ Not enough energy. Watch ad for +50.");
    return;
  }
  STATE.energy = Math.max(0, STATE.energy - 10);
  STATE.t = 0;
  STATE.mining = true;
  txtCenter.setText("⛏️ Mining started");
  refreshHUD();
}

function gainEnergy(n){
  STATE.energy = Math.min(STATE.energyMax, STATE.energy + n);
  txtCenter.setText(`🎦 Watched ad: +${n} Energy`);
  refreshHUD();
}

function claimDaily(){
  const today = new Date().toISOString().slice(0,10);
  if (STATE.lastDaily === today){
    txtCenter.setText("🎁 Daily already claimed.");
    return;
  }
  STATE.lastDaily = today;
  STATE.qc += 15;
  STATE.energy = Math.min(STATE.energyMax, STATE.energy + 20);
  txtCenter.setText("🎉 Daily: +15 QC & +20 Energy");
  refreshHUD();
}

let tgwt = 0;
function addTGWT(n,label){
  tgwt += n;
  txtCenter.setText(`🪙 ${label} task: +${n} TGWT (total ${tgwt})`);
}

function completeTGWT(){
  txtCenter.setText(`🪙 TGWT total: ${tgwt}`);
}

function openShop(){
  // simple upgrade: buy power for 50 QC
  if (STATE.qc < 50){
    txtCenter.setText("🛒 Need 50 QC to upgrade ship.");
    return;
  }
  STATE.qc -= 50;
  STATE.power = +(STATE.power + 0.3).toFixed(1);
  STATE.shipName = `Upgraded Miner`;
  txtCenter.setText(`✅ Ship upgraded! Power x${STATE.power}`);
  refreshHUD();
}

function refreshHUD(){
  byId('qc').textContent     = `QC: ${STATE.qc.toFixed(2)}`;
  byId('energy').textContent = `Energy: ${STATE.energy}/${STATE.energyMax}`;
  byId('level').textContent  = `Level: ${STATE.level}`;
  byId('ship').textContent   = `Ship: ${STATE.shipName} (x${STATE.power})`;
}

function byId(id){ return document.getElementById(id); }
function rand(min,max){ return Math.random()*(max-min)+min; }
