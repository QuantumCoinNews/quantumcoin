// Minimal canvas scene with ship + particle mining, API-integrated
const W = window, D = document, $ = sel => D.querySelector(sel);

// state
const S = {
    user: null,
    sectors: [],
    ships: [],
    selectedSector: null,
    mining: { active: false, endsAt: 0, duration: 0 },
    nodes: [], // 5 positions
    ship: { x: 200, y: 360, targetX: 200, targetY: 360 },
    particles: []
};

// canvas
const canvas = $("#game"), ctx = canvas.getContext("2d");
function fitCanvas() { canvas.width = W.innerWidth; canvas.height = W.innerHeight; }
fitCanvas(); W.addEventListener("resize", fitCanvas);

// helpers
function lerp(a, b, t) { return a + (b - a) * t; }
function rand(a, b) { return a + Math.random() * (b - a); }

// mountain generator
function drawMountains() {
    const h = canvas.height, w = canvas.width;
    ctx.fillStyle = "#0b0f1d";
    ctx.beginPath();
    ctx.moveTo(0, h * 0.75);
    for (let x = 0; x <= w; x += 60) {
        const y = h * 0.75 - 50 * Math.sin(x * 0.01) - 30 * Math.cos(x * 0.02);
        ctx.lineTo(x, y);
    }
    ctx.lineTo(w, h); ctx.lineTo(0, h); ctx.closePath(); ctx.fill();

    ctx.fillStyle = "#0e1426";
    ctx.beginPath();
    ctx.moveTo(0, h * 0.82);
    for (let x = 0; x <= w; x += 70) {
        const y = h * 0.82 - 40 * Math.sin(x * 0.012) - 20 * Math.cos(x * 0.017);
        ctx.lineTo(x, y);
    }
    ctx.lineTo(w, h); ctx.lineTo(0, h); ctx.closePath(); ctx.fill();
}

// moon ground + craters
function drawMoon() {
    const h = canvas.height, w = canvas.width;
    const gY = h * 0.86;
    ctx.fillStyle = "#1a1f37";
    ctx.fillRect(0, gY, w, h - gY);
    // craters
    ctx.fillStyle = "#12162a";
    for (let i = 0; i < 8; i++) {
        const x = (i + 0.5) * (w / 8);
        const r = 18 + 10 * Math.sin(i * 1.3);
        ctx.beginPath(); ctx.ellipse(x, gY + 20, r * 1.8, r, 0, 0, Math.PI * 2); ctx.fill();
    }
}

// mining nodes (5)
function ensureNodes() {
    if (S.nodes.length) return;
    const h = canvas.height, w = canvas.width, gY = h * 0.86;
    const xs = [0.15, 0.33, 0.5, 0.67, 0.85];
    S.nodes = xs.map((t, i) => ({ x: Math.floor(w * t), y: gY - 8, r: 18, name: `Sector ${i + 1}` }));
}
function drawNodes() {
    S.nodes.forEach(n => {
        ctx.beginPath();
        ctx.arc(n.x, n.y, n.r, 0, Math.PI * 2);
        ctx.fillStyle = (S.selectedSector === n.name) ? "#ffd54a" : "#2a3158";
        ctx.fill();
        ctx.strokeStyle = "#39406a"; ctx.stroke();
        // label
        ctx.fillStyle = "#8b93a7"; ctx.font = "12px Segoe UI";
        ctx.textAlign = "center"; ctx.fillText(n.name, n.x, n.y - 26);
    });
}

// ship
function drawShip() {
    const s = S.ship;
    // move
    s.x = lerp(s.x, s.targetX, 0.05);
    s.y = lerp(s.y, s.targetY, 0.05);

    // body
    ctx.save();
    ctx.translate(s.x, s.y);
    ctx.beginPath();
    ctx.moveTo(0, -14); ctx.lineTo(16, 6); ctx.lineTo(-16, 6); ctx.closePath();
    ctx.fillStyle = "#7aa2ff"; ctx.fill();
    ctx.strokeStyle = "#8ec5ff"; ctx.stroke();

    // engine glow
    ctx.beginPath();
    ctx.arc(0, 6, 6, 0, Math.PI * 2); ctx.fillStyle = "#2b3560"; ctx.fill();

    // thruster
    if (Math.hypot(s.x - s.targetX, s.y - s.targetY) > 2) {
        ctx.beginPath(); ctx.moveTo(-6, 10); ctx.lineTo(0, 18); ctx.lineTo(6, 10); ctx.closePath();
        ctx.fillStyle = "#66e0a8"; ctx.fill();
    }

    ctx.restore();
}

// particles (gold)
function spawnParticles(x, y) {
    for (let i = 0; i < 12; i++) {
        S.particles.push({
            x, y, vx: rand(-0.8, 0.8), vy: rand(-1.6, -0.6), life: rand(0.8, 1.5)
        });
    }
}
function updateParticles(dt) {
    const g = 1.6 * dt;
    S.particles.forEach(p => {
        p.x += p.vx * 60 * dt;
        p.y += p.vy * 60 * dt;
        p.vy += g * 0.5;
        p.life -= dt;
    });
    S.particles = S.particles.filter(p => p.life > 0);
}
function drawParticles() {
    ctx.fillStyle = "#ffd54a";
    S.particles.forEach(p => {
        ctx.globalAlpha = Math.max(0, p.life);
        ctx.beginPath(); ctx.arc(p.x, p.y, 2.2, 0, Math.PI * 2); ctx.fill();
    });
    ctx.globalAlpha = 1;
}

// mining visuals
function miningTick() {
    if (!S.mining.active) return;
    const now = performance.now();
    if (now >= S.mining.endsAt) {
        S.mining.active = false;
        finishMining();
        $("#session").innerText = "-";
        return;
    }
    // spawn particles at node
    const node = S.nodes.find(n => n.name === S.selectedSector);
    if (node && Math.random() < 0.4) {
        spawnParticles(node.x, node.y - 10);
    }
    const remain = Math.ceil((S.mining.endsAt - now) / 1000);
    $("#session").innerText = `Mining... ${remain}s`;
}

// click select node
canvas.addEventListener("click", (e) => {
    const r = canvas.getBoundingClientRect();
    const x = e.clientX - r.left, y = e.clientY - r.top;
    const hit = S.nodes.find(n => (x - n.x) ** 2 + (y - n.y) ** 2 <= n.r ** 2);
    if (hit) {
        S.selectedSector = hit.name;
        $("#sector").innerText = hit.name;
        // move ship
        S.ship.targetX = hit.x;
        S.ship.targetY = hit.y - 38;
    }
});

// main loop
let lastTs = performance.now();
function frame(ts) {
    const dt = Math.min(0.05, (ts - lastTs) / 1000); lastTs = ts;

    // bg
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    drawMountains(); drawMoon(); ensureNodes(); drawNodes(); drawParticles(); drawShip();

    updateParticles(dt); miningTick();
    requestAnimationFrame(frame);
}
requestAnimationFrame(frame);

// ---- UI wiring ----
$("#btnMine").addEventListener("click", async () => {
    if (!S.selectedSector) { alert("Select a sector (click a node)"); return; }
    if (S.mining.active) { alert("Already mining"); return; }
    try {
        const res = await fetch("/api/start_mining", { method: "POST", headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ sector: S.selectedSector }) });
        const js = await res.json();
        if (!res.ok) throw new Error(js.detail || "start failed");
        S.mining.active = true;
        S.mining.duration = js.duration;
        S.mining.endsAt = performance.now() + js.duration * 1000;
        $("#energy").innerText = `${js.energy}/100`;
        $("#session").innerText = `Mining... ${js.duration}s`;
    } catch (e) { alert(e.message); }
});

async function finishMining() {
    try {
        const res = await fetch("/api/finish_mining", { method: "POST" });
        const js = await res.json();
        if (!res.ok) throw new Error(js.detail || "finish failed");
        $("#qc").innerText = Number(js.qc).toFixed(3);
        $("#energy").innerText = `${js.energy}/100`;
        alert(`+${js.reward_qc} QC`);
    } catch (e) { alert(e.message); }
}

// modal helpers
function openModal(title, html) {
    $("#modal-title").innerText = title;
    $("#modal-body").innerHTML = html;
    $("#modal").classList.remove("hidden");
}
$("#modal-close").addEventListener("click", () => $("#modal").classList.add("hidden"));

// shop
$("#btnShop").addEventListener("click", () => {
    const items = S.ships.map(s => `
    <div class="card">
      <img src="${s.img}" alt="${s.name}"/>
      <div class="name">${s.name}</div>
      <div class="price">${s.price_qc} QC • x${s.mul.toFixed(2)}</div>
      <button data-ship="${s.id}">Buy</button>
    </div>`).join("");
    openModal("Buy Ship", `<div class="card-grid">${items}</div>`);
    $("#modal-body").addEventListener("click", async (e) => {
        const id = e.target?.getAttribute?.("data-ship");
        if (!id) return;
        try {
            const res = await fetch("/api/buy_ship", { method: "POST", headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ship_id: Number(id) }) });
            const js = await res.json();
            if (!res.ok) throw new Error(js.detail || "buy failed");
            $("#qc").innerText = Number(js.qc).toFixed(3);
            $("#ship").innerText = `${js.ship} (x${Number(js.ship_mul).toFixed(2)})`;
            alert("Purchased!");
        } catch (err) { alert(err.message); }
    }, { once: true });
});

// social
$("#btnSocial").addEventListener("click", () => {
    const links = [
        { name: "Follow X", href: "https://x.com" },
        { name: "Follow YouTube", href: "https://youtube.com" },
        { name: "Follow TikTok", href: "https://tiktok.com" },
    ];
    const buttons = links.map(l => `<div class="field"><a class="link" target="_blank" href="${l.href}">🔗 ${l.name}</a></div>`).join("");
    const tasks = (S.social_tasks || []).map(t => `<button data-task="${t.task}">Claim ${t.task} (+${t.amount} TGWT)</button>`).join("");
    openModal("Social Tasks", `<div class="row"><div>${buttons}</div><div>${tasks}</div></div>`);
    $("#modal-body").addEventListener("click", async (e) => {
        const task = e.target?.getAttribute?.("data-task");
        if (!task) return;
        const res = await fetch("/api/claim_social", { method: "POST", headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ task }) });
        const js = await res.json();
        if (!res.ok) { alert(js.detail || "claim failed"); return; }
        $("#tgwt").innerText = js.tgwt;
        alert(`+${(S.social_tasks.find(x => x.task === task) || {}).amount || 0} TGWT`);
    }, { once: true });
});

// friends (placeholder)
$("#btnFriends").addEventListener("click", () => {
    openModal("Friends", `<p>Referral coming soon. Invite friends to earn QC bonus.</p>`);
});

// ad (placeholder)
$("#btnAd").addEventListener("click", () => {
    openModal("Watch Ad", `<p>Ad integration will appear here. For dev, +10 energy.</p><button id="gainEnergy">Gain +10</button>`);
    $("#gainEnergy").addEventListener("click", () => {
        const el = $("#energy");
        const parts = el.innerText.split("/"); let cur = Number(parts[0] || 0);
        cur = Math.min(100, cur + 10);
        el.innerText = `${cur}/100`;
        alert("+10 energy");
    }, { once: true });
});

// wallet
$("#btnWallet").addEventListener("click", async () => {
    const w = await (await fetch("/api/wallet")).json();
    const html = `
    <div class="field"><b>QC wallet:</b> <span>${w.qc_addr}</span></div>
    <div class="field"><b>BNB wallet:</b> <span>${w.bnb_addr || "-"}</span></div>
    <div class="sep"></div>
    <button id="mmConnect">Connect MetaMask (BNB)</button>
    <div class="field"><input id="manualAddr" class="input" placeholder="0x..."/></div>
    <button id="linkManual">Link Address</button>
  `;
    openModal("Wallet", html);

    $("#mmConnect").addEventListener("click", async () => {
        try {
            if (!window.ethereum) { alert("MetaMask not found"); return; }
            const accts = await window.ethereum.request({ method: "eth_requestAccounts" });
            const addr = accts[0];
            const res = await fetch("/api/link_wallet", { method: "POST", headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ bnb_address: addr }) });
            const js = await res.json();
            if (!res.ok) throw new Error(js.detail || "link failed");
            alert("Wallet linked: " + addr);
            $("#modal").classList.add("hidden");
        } catch (e) { alert(e.message); }
    });

    $("#linkManual").addEventListener("click", async () => {
        const addr = $("#manualAddr").value.trim();
        const res = await fetch("/api/link_wallet", { method: "POST", headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ bnb_address: addr }) });
        const js = await res.json();
        if (!res.ok) { alert(js.detail || "link failed"); return; }
        alert("Wallet linked: " + js.bnb_addr);
        $("#modal").classList.add("hidden");
    });
});

// boot
(async function () {
    const boot = await (await fetch("/api/boot")).json();
    S.user = boot.user; S.sectors = boot.sectors; S.ships = boot.ships; S.social_tasks = boot.social_tasks;

    $("#qc").innerText = Number(S.user.qc).toFixed(3);
    $("#tgwt").innerText = S.user.tgwt;
    $("#energy").innerText = `${S.user.energy}/100`;
    $("#level").innerText = S.user.level;
    $("#ship").innerText = `${S.user.ship} (x${Number(S.user.ship_mul).toFixed(2)})`;

    // preselect middle sector
    S.selectedSector = "Sector 3"; $("#sector").innerText = S.selectedSector;
    const m = S.nodes[2]; if (m) { S.ship.x = m.x; S.ship.y = m.y - 40; S.ship.targetX = m.x; S.ship.targetY = m.y - 40; }
})();
