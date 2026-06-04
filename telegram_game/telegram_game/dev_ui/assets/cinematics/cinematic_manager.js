/*
  QuantumCoin Space Miner - External Cinematic Manager
  Safe mode:
  - Injects its own overlay.
  - Does not modify existing HTML structure.
  - Skips empty placeholder videos by checking file size first.
  - Falls back to the game without breaking.
*/

(function () {
  "use strict";

  if (window.QC_CINEMATIC_MANAGER) return;

  const CONFIG = {
    introSeenKey: "qc_cinematic_intro_seen_v2",
    logo: "assets/cinematics/logo/quantumcoin_logo.png",
    intro: [
      {
        id: "intro_logo",
        src: "assets/videos/intro/intro_logo.mp4",
        caption: "QuantumCoin systems online...",
        fallbackMs: 900
      },
      {
        id: "intro_ship",
        src: "assets/videos/intro/intro_ship.mp4",
        caption: "Deep space approach initiated...",
        fallbackMs: 900
      },
      {
        id: "intro_cockpit",
        src: "assets/videos/intro/intro_cockpit.mp4",
        caption: "Cockpit link established. Mining coordinates confirmed.",
        fallbackMs: 8500
      },
      {
        id: "intro_moon_overview",
        src: "assets/videos/intro/intro_moon_overview.mp4",
        caption: "Lunar mining zones detected. Select a zone to begin.",
        fallbackMs: 900
      }
    ],
    zones: {
      "LUNA-01": {
        src: "assets/videos/zones/zone_01.mp4",
        caption: "Approaching LUNA-01 mining zone...",
        fallbackMs: 700
      },
      "LUNA-02": {
        src: "assets/videos/zones/zone_02.mp4",
        caption: "Approaching LUNA-02 mining zone...",
        fallbackMs: 700
      },
      "LUNA-03": {
        src: "assets/videos/zones/zone_03.mp4",
        caption: "Approaching LUNA-03 mining zone...",
        fallbackMs: 700
      },
      "LUNA-04": {
        src: "assets/videos/zones/zone_04.mp4",
        caption: "Approaching LUNA-04 mining zone...",
        fallbackMs: 700
      },
      "LUNA-05": {
        src: "assets/videos/zones/zone_05.mp4",
        caption: "Approaching LUNA-05 mining zone...",
        fallbackMs: 700
      }
    }
  };

  let currentResolve = null;
  let originalEnterMiningSite = null;
  let zoneLock = false;

  function byId(id) {
    return document.getElementById(id);
  }

  function ensureLayer() {
    let layer = byId("qcCinematicLayer");
    if (layer) return layer;

    layer = document.createElement("div");
    layer.id = "qcCinematicLayer";
    layer.className = "qc-cinematic-hidden";
    layer.setAttribute("aria-hidden", "true");

    layer.innerHTML = [
      '<video id="qcCinematicVideo" class="qc-cinematic-video" playsinline preload="metadata"></video>',
      '<div id="qcCinematicCard" class="qc-cinematic-card">',
      '  <img id="qcCinematicLogo" class="qc-cinematic-logo" alt="QuantumCoin Logo">',
      '  <div class="qc-cinematic-kicker">QUANTUMCOIN SPACE MINER</div>',
      '  <div class="qc-cinematic-title">Cinematic Launch Sequence</div>',
      '  <div class="qc-cinematic-text">Tap to begin the mission. Sound and cockpit dialogue will start after your touch.</div>',
      '  <button id="qcCinematicStartBtn" class="qc-cinematic-start-btn" type="button">TAP TO BEGIN</button>',
      '  <button id="qcCinematicEnterBtn" class="qc-cinematic-enter-btn" type="button">ENTER GAME</button>',
      '</div>',
      '<div id="qcCinematicCaption" class="qc-cinematic-caption">Preparing QuantumCoin mission...</div>',
      '<button id="qcCinematicSkipBtn" class="qc-cinematic-skip-btn" type="button">Skip</button>'
    ].join("");

    document.body.appendChild(layer);

    const logo = byId("qcCinematicLogo");
    if (logo) logo.src = CONFIG.logo;

    bindButtons();

    return layer;
  }

  function videoEl() { return byId("qcCinematicVideo"); }
  function cardEl() { return byId("qcCinematicCard"); }
  function captionEl() { return byId("qcCinematicCaption"); }
  function skipEl() { return byId("qcCinematicSkipBtn"); }

  function showLayer(mode) {
    const layer = ensureLayer();

    layer.classList.remove("qc-cinematic-hidden");
    layer.setAttribute("aria-hidden", "false");
    document.body.classList.add("qc-cinematic-open");

    const v = videoEl();
    const card = cardEl();
    const cap = captionEl();
    const skip = skipEl();

    if (mode === "start") {
      if (card) card.style.display = "block";
      if (cap) cap.style.display = "none";
      if (skip) skip.style.display = "none";
      if (v) v.style.display = "none";
    } else {
      if (card) card.style.display = "none";
      if (cap) cap.style.display = "block";
      if (skip) skip.style.display = "block";
      if (v) v.style.display = "block";
    }
  }

  function hideLayer() {
    const layer = byId("qcCinematicLayer");
    const v = videoEl();

    if (v) {
      try {
        v.pause();
        v.removeAttribute("src");
        v.load();
      } catch (_) {}
    }

    if (layer) {
      layer.classList.add("qc-cinematic-hidden");
      layer.setAttribute("aria-hidden", "true");
    }

    document.body.classList.remove("qc-cinematic-open");
    currentResolve = null;
  }

  async function isPlayableFile(src) {
    try {
      const response = await fetch(src + "?check=" + Date.now(), {
        method: "HEAD",
        cache: "no-store"
      });

      if (!response.ok) return false;

      const lenText = response.headers.get("content-length");
      const len = lenText ? parseInt(lenText, 10) : 0;

      return Number.isFinite(len) && len > 1024;
    } catch (_) {
      return false;
    }
  }

  function finishClip() {
    if (currentResolve) {
      const done = currentResolve;
      currentResolve = null;
      done();
    }
  }

  async function playClip(item) {
    if (!item || !item.src) return;

    const playable = await isPlayableFile(item.src);

    if (!playable) {
      return;
    }

    return new Promise(function (resolve) {
      const v = videoEl();
      const cap = captionEl();

      if (!v) {
        resolve();
        return;
      }

      showLayer("video");

      if (cap) cap.textContent = item.caption || "Loading cinematic...";

      let finished = false;
      let timer = null;

      function cleanup() {
        if (finished) return;
        finished = true;

        try {
          v.onended = null;
          v.onerror = null;
          v.onabort = null;
          v.onstalled = null;
          v.pause();
          // Keep the last frame visible between intro clips.
          // Do not clear src/load here; hideLayer() will clean the video after the full sequence.
        } catch (_) {}

        if (timer) clearTimeout(timer);

        currentResolve = null;
        resolve();
      }

      currentResolve = cleanup;

      v.onended = cleanup;
      v.onerror = cleanup;
      v.onabort = cleanup;
      v.onstalled = null; // do not close early on local-server buffering

      timer = setTimeout(cleanup, Math.max(8500, item.fallbackMs || 8500));

      try {
        v.src = item.src + "?v=" + Date.now();
        // Force browser to load the newly assigned clip while keeping the cinematic layer visible.
        v.load();
        v.currentTime = 0;
        v.muted = false;

        const p = v.play();

        if (p && typeof p.catch === "function") {
          p.catch(function () {
            try {
              v.muted = true;
              v.play().catch(cleanup);
            } catch (_) {
              cleanup();
            }
          });
        }
      } catch (_) {
        cleanup();
      }
    });
  }

  async function playIntro() {
    try {
      sessionStorage.setItem(CONFIG.introSeenKey, "1");
    } catch (_) {}

    showLayer("video");

    for (const clip of CONFIG.intro) {
      await playClip(clip);
    }

    hideLayer();
  }

  function showIntroGate() {
    let alreadySeen = false;

    try {
      alreadySeen = sessionStorage.getItem(CONFIG.introSeenKey) === "1";
    } catch (_) {}

    if (alreadySeen) return;

    showLayer("start");
  }

  function normalizeZoneKey(siteId) {
    const raw = String(siteId || "").trim().toUpperCase();

    if (/^LUNA-\d+$/.test(raw)) {
      return "LUNA-" + raw.replace("LUNA-", "").padStart(2, "0");
    }

    const found = raw.match(/(?:LUNA|ZONE|SITE|MINE)?[-_ ]?(\d+)/);
    if (found) {
      return "LUNA-" + found[1].padStart(2, "0");
    }

    return raw;
  }

  async function playZone(siteId) {
    const key = normalizeZoneKey(siteId);
    const clip = CONFIG.zones[key];

    if (!clip) return;

    showLayer("video");
    await playClip(clip);
    hideLayer();
  }

  function hookEnterMiningSite() {
    if (typeof window.enterMiningSite !== "function") {
      setTimeout(hookEnterMiningSite, 300);
      return;
    }

    if (window.enterMiningSite.__qcCinematicWrapped) return;

    originalEnterMiningSite = window.enterMiningSite;

    const wrapped = async function (siteId) {
      if (zoneLock) {
        return originalEnterMiningSite.apply(this, arguments);
      }

      zoneLock = true;

      try {
        await playZone(siteId);
      } catch (_) {
        hideLayer();
      }

      zoneLock = false;
      return originalEnterMiningSite.apply(this, arguments);
    };

    wrapped.__qcCinematicWrapped = true;
    window.enterMiningSite = wrapped;
  }

  function bindButtons() {
    const startBtn = byId("qcCinematicStartBtn");
    const enterBtn = byId("qcCinematicEnterBtn");
    const skipBtn = byId("qcCinematicSkipBtn");

    if (startBtn && !startBtn.__qcBound) {
      startBtn.__qcBound = true;
      startBtn.addEventListener("click", function () {
        playIntro().catch(hideLayer);
      });
    }

    if (enterBtn && !enterBtn.__qcBound) {
      enterBtn.__qcBound = true;
      enterBtn.addEventListener("click", function () {
        try {
          sessionStorage.setItem(CONFIG.introSeenKey, "1");
        } catch (_) {}
        hideLayer();
      });
    }

    if (skipBtn && !skipBtn.__qcBound) {
      skipBtn.__qcBound = true;
      skipBtn.addEventListener("click", function () {
        finishClip();
        hideLayer();
      });
    }
  }

  window.QC_CINEMATIC_MANAGER = {
    config: CONFIG,
    showIntroGate,
    playIntro,
    playZone,
    hideLayer
  };

  window.addEventListener("DOMContentLoaded", function () {
    ensureLayer();
    hookEnterMiningSite();
    setTimeout(showIntroGate, 500);
  });
})();



