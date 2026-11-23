/* Scene and cosmetic views; account, identity and recovery have separate owners. */
(function(){

"use strict";

/* ======================
   I18N (DROP-IN)
====================== */
window.I18N = {

  en: {
    // HUD
    hudLevel: "Level",
    hudQC: "QC Balance",
    hudTGWT: "TGWT Balance",
    hudXP: "XP",
    hint: "← → move, ↑ thrust, 1–5 select mining spot, SPACE toggles auto mine",

    // Bottom nav
    navStart: "Start",
    navWallet: "Wallet",
    navTop: "Top",
    navStore: "Ship Store",
    navTGWT: "TGWT",
    navSocial: "QC Follow",
    navWatch: "Watch & Earn",
    navSettings: "Settings",

    // Panel titles/bodies (generic)
    panelTitles: {
      wallet: "Wallet",
      top: "Top Players",
      store: "Ship Store",
      tgwt: "TGWT Rewards",
      social: "QC Social",
      watch: "Watch & Earn",
      settings: "Settings",
    },
    panelBodies: {
      wallet: "View balances, connect MetaMask, and manage your QC seed.",
      top: "Leaderboard and global stats for Space Miner (demo until backend is connected).",
      store: "Everyone starts with a free ship. Buy and equip advanced ships using QC.",
      tgwt: "Ledger-backed TGWT balances, pool status, ledger history and withdrawal requests are shown here.",
      social: "Follow QuantumCoin on official channels. Verified follows may grant TGWT later.",
      watch: "Watch selected videos to earn demo rewards.",
      settings: "Language and game settings will be managed here.",
    },

    // Mine overlay (supports both nested & flat keys)
    mine: {
      subtitle: "QuantumCoin mining sector.",
      backToMoon: "← Back to Moon",
    },
    "mine.subtitle": "QuantumCoin mining sector.",
    "mine.backToMoon": "← Back to Moon",

    // TOP UI (ids: topStatusTitle, topCard..., topHeader..., topFooterNoteText)
    topUI: {
      sectionStatusTitle: "GLOBAL STATS",
      sectionStatusSubtitle: "Summary statistics of all players inside Space Miner.",
      cardTotalPlayersTitle: "TOTAL PLAYERS",
      cardTotalPlayersDesc: "Unique accounts joined so far.",
      cardOnlineTitle: "ONLINE NOW",
      cardOnlineDesc: "Players active in the last minutes.",
      cardRankTitle: "YOUR RANK",
      cardRankDesc: "Your position by total QC earnings.",
      cardTotalEarnedTitle: "TOTAL QC EARNED",
      cardTotalEarnedDesc: "Total QC earned since you started.",
      sectionLeaderboardTitle: "TOP QC EARNERS",
      sectionLeaderboardSubtitle: "Top players sorted by QC earned.",
      colRank: "#",
      colPlayer: "PLAYER",
      colQC: "QC",
      colSessions: "SESSIONS",
      colLastSeen: "LAST SEEN",
      footerNote: "Real on-chain statistics will appear here once backend is connected. Demo data for now.",
    },

    // WALLET UI (ids: walletBalancesTitle, walletQCLabel, walletConnectionSub, btnGenerateSeed, etc.)
    walletUI: {
      headingBalances: "Balances",
      labelGameQC: "In-game QC balance",
      labelGameTGWT: "TGWT balance",
      labelLevel: "Level",

      headingConnection: "Wallet & MetaMask",
      connectionSub: "Manage MetaMask connection and QC seed in one place.",

      metaTitle: "MetaMask connection",
      btnConnect: "Connect MetaMask",

      pillNotConnected: "Not connected",
      pillConnectedPrefix: "Connected:",

      statusDisconnected: "Not connected.",
      statusNoMeta: "MetaMask was not found in this browser. Please install the extension.",
      statusConnectedPrefix: "Connected:",

      // Seed section MUST stay English in all languages (your rule)
      seedTitle: "QC wallet and recovery",
      seedNotGenerated: "Create and sign with the QuantumCoin wallet app.",
      btnGenerateSeed: "Generate new seed",
      btnShowSeed: "Show seed",
      btnHideSeed: "Hide seed",
      btnCopySeed: "Copy seed",
      seedWarning:
        "Use the wallet app for new wallets. Encrypt any old browser recovery in the recovery panel below; keep its backup and passphrase offline.",
      copyOk: "Seed copied to clipboard.",
      copyFail: "Copy failed. Please copy manually.",
    },

    // STORE UI
    storeUI: {
      heading: "Ship Store",
      intro: "Everyone starts with a free starter ship. Buy advanced ships with QC.",
      priceLabel: "Price",
      freeLabel: "Free",
      tierLabel: "Tier",
      ownedBadge: "Owned",
      activeBadge: "Active",
      buyButton: "Buy ship",
      equipButton: "Set active ship",
      activeButton: "Active ship",
      notEnoughQC: "Not enough QC to buy this ship.",
    },

    // SOCIAL UI
    socialUI: {
      panelTitle: "QC Social",
      panelIntro:
        "Follow QuantumCoin on official channels. Verified TGWT campaigns are planned for BSC mainnet only.",
      cardTitle: "Follow QuantumCoin",
      cardIntro: "TGWT reward campaigns are disabled on testnet and remain off until explicitly enabled for BSC mainnet.",
      linkWebsite: "Website – qcnetwork.ai",
      linkX: "X (Twitter) – @QuantumCoinQC",
      linkTelegram: "Telegram – QuantumCoin Community",
      linkTikTok: "TikTok – @quantumcoin21",
      linkYouTube: "YouTube – @QuantumCoinHQ",
      linkTGWTWebsite: "https://thegreenworldtechnology.com/",
      linkXTGWT: "https://x.com/TGWTToken1",
      rewardNote: "Plan: 1 TGWT per verified platform per user. Rules will be finalized later.",
    },

    // WATCH UI
    watch: {
      title: "WATCH & EARN",
      desc: "DEV TEST: Watch flow preview. Final Watch & Earn will use verified Google Rewarded Ads and TGWT reward after valid ad completion.",
      statusLabel: "Status",
      requiredTimeLabel: "Required time",
      rewardLabel: "Reward (DEV TEST)",
      statusIdle: "Not started yet",
      statusRunning: "Watching in progress...",
      statusCompleted: "Time completed — reward ready (demo)!",
      hint: "DEV TEST: timer simulates rewarded-ad completion. Final mode will require Google rewarded-ad verification.",
      videoListTitle: "Videos",
      durationLabel: "Duration",
      startBtn: "Start Watching",
      videoIntroTitle: "What is QuantumCoin? (Short Intro)",
      videoSpaceMinerTitle: "Space Miner Gameplay (Demo)",
    },
  },

  tr: {
    // HUD
    hudLevel: "Seviye",
    hudQC: "QC Bakiyesi",
    hudTGWT: "TGWT Bakiyesi",
    hudXP: "XP",
    hint: "← → yön, ↑ itme, 1–5 maden alanını seç, SPACE otomatik kazımı aç/kapat",

    // Bottom nav
    navStart: "Başla",
    navWallet: "Cüzdan",
    navTop: "Lider",
    navStore: "Gemi Mağazası",
    navTGWT: "TGWT",
    navSocial: "QC Takip",
    navWatch: "İzle Kazan",
    navSettings: "Ayarlar",

    // Panel titles/bodies
    panelTitles: {
      wallet: "Cüzdan",
      top: "Liderler",
      store: "Gemi Mağazası",
      tgwt: "TGWT Ödülleri",
      social: "QC Sosyal",
      watch: "İzle Kazan",
      settings: "Ayarlar",
    },
    panelBodies: {
      wallet: "Bakiyeleri gör, MetaMask’e bağlan ve QC seed yönetimini yap.",
      top: "Liderlik tablosu ve global istatistikler (backend bağlanınca gerçek veriler).",
      store: "Herkes ücretsiz başlangıç gemisiyle başlar. QC ile gemi alıp aktif edebilirsin.",
      tgwt: "Ledger destekli TGWT bakiyeleri, havuz durumu, işlem geçmişi ve çekim istekleri burada gösterilir.",
      social: "QuantumCoin’i resmi kanallardan takip et. Doğrulanan takipler ileride TGWT kazandırabilir.",
      watch: "Seçili videoları izleyerek demo ödül kazan.",
      settings: "Dil ve oyun ayarları burada yönetilecek.",
    },

    // Mine overlay (flat + nested)
    mine: {
      subtitle: "QuantumCoin mining sector.",
      backToMoon: "← Ay'a Dön",
    },
    "mine.subtitle": "QuantumCoin mining sector.",
    "mine.backToMoon": "← Ay'a Dön",

    // TOP UI
    topUI: {
      sectionStatusTitle: "GENEL DURUM",
      sectionStatusSubtitle: "Space Miner içindeki tüm oyuncuların özet istatistikleri.",
      cardTotalPlayersTitle: "TOPLAM OYUNCU",
      cardTotalPlayersDesc: "Şimdiye kadar oyuna giren benzersiz hesap sayısı.",
      cardOnlineTitle: "ŞU AN ÇEVRİMİÇİ",
      cardOnlineDesc: "Son dakikalarda aktif olan oyuncular.",
      cardRankTitle: "SENİN SIRALAMAN",
      cardRankDesc: "Toplam QC kazancına göre sıralamadaki yerin.",
      cardTotalEarnedTitle: "TOPLAM QC KAZANCI",
      cardTotalEarnedDesc: "Oyuna başladığından beri kazandığın toplam QC.",
      sectionLeaderboardTitle: "EN ÇOK QC KAZANANLAR",
      sectionLeaderboardSubtitle: "QC kazancına göre sıralama.",
      colRank: "#",
      colPlayer: "OYUNCU",
      colQC: "QC",
      colSessions: "SEANS",
      colLastSeen: "SON GÖRÜLME",
      footerNote: "Gerçek zincir istatistikleri backend bağlanınca görünecek. Şimdilik demo veri.",
    },

    // WALLET UI (seed bölümü İngilizce kalır)
    walletUI: {
      headingBalances: "Bakiyeler",
      labelGameQC: "Oyun içi QC bakiyesi",
      labelGameTGWT: "TGWT bakiyesi",
      labelLevel: "Seviye",

      headingConnection: "Cüzdan & MetaMask",
      connectionSub: "MetaMask bağlantısı ve QC seed yönetimi tek yerde.",

      metaTitle: "MetaMask bağlantısı",
      btnConnect: "MetaMask'e Bağlan",

      pillNotConnected: "Bağlı değil",
      pillConnectedPrefix: "Bağlı:",

      statusDisconnected: "Bağlı değil.",
      statusNoMeta: "Bu tarayıcıda MetaMask bulunamadı. Lütfen eklentiyi kur.",
      statusConnectedPrefix: "Bağlı:",

      // Seed MUST stay English
      seedTitle: "QC wallet and recovery",
      seedNotGenerated: "Create and sign with the QuantumCoin wallet app.",
      btnGenerateSeed: "Generate new seed",
      btnShowSeed: "Show seed",
      btnHideSeed: "Hide seed",
      btnCopySeed: "Copy seed",
      seedWarning:
        "Use the wallet app for new wallets. Encrypt any old browser recovery in the recovery panel below; keep its backup and passphrase offline.",
      copyOk: "Seed copied to clipboard.",
      copyFail: "Copy failed. Please copy manually.",
    },

    // STORE UI
    storeUI: {
      heading: "Gemi Mağazası",
      intro: "Herkes ücretsiz başlangıç gemisiyle başlar. QC ile gelişmiş gemiler satın al.",
      priceLabel: "Fiyat",
      freeLabel: "Ücretsiz",
      tierLabel: "Seviye",
      ownedBadge: "Sende",
      activeBadge: "Aktif",
      buyButton: "Gemi satın al",
      equipButton: "Aktif yap",
      activeButton: "Aktif gemi",
      notEnoughQC: "Bu gemiyi almak için yeterli QC yok.",
    },

    // SOCIAL UI
    socialUI: {
      panelTitle: "QC Sosyal",
      panelIntro: "QuantumCoin’i resmi kanallardan takip et. Doğrulanmış TGWT kampanyaları yalnızca BSC mainnet için planlanıyor.",
      cardTitle: "QuantumCoin’i takip et",
      cardIntro: "TGWT ödül kampanyaları testnet’te kapalıdır; BSC mainnet için açıkça etkinleştirilene kadar ödül verilmez.",
      linkWebsite: "Website – qcnetwork.ai",
      linkX: "X (Twitter) – @QuantumCoinQC",
      linkTelegram: "Telegram – QuantumCoin Topluluğu",
      linkTikTok: "TikTok – @quantumcoin21",
      linkYouTube: "YouTube – @QuantumCoinHQ",
      linkTGWTWebsite: "https://thegreenworldtechnology.com/",
      linkXTGWT: "https://x.com/TGWTToken1",
      rewardNote: "Plan: doğrulanan her platform için kullanıcı başına 1 TGWT. Detaylar sonra.",
    },

    // WATCH UI
    watch: {
      title: "İZLE KAZAN",
      desc: "QuantumCoin ile ilgili kısa videoları izleyerek demo QC ve TGWT kazan.",
      statusLabel: "Durum",
      requiredTimeLabel: "Gerekli süre",
      rewardLabel: "Ödül (demo)",
      statusIdle: "Henüz başlatılmadı",
      statusRunning: "İzleme devam ediyor...",
      statusCompleted: "Süre tamamlandı — ödül hazır (demo)!",
      hint: "Demo: süre dolunca sistem “Ödül Hazır” olarak işaretler.",
      videoListTitle: "Videolar",
      durationLabel: "Süre",
      startBtn: "İzlemeye Başla",
      videoIntroTitle: "QuantumCoin Nedir? (Kısa Tanıtım)",
      videoSpaceMinerTitle: "Space Miner Oynanış (Demo)",
    },
  },

  es: {
    // HUD
    hudLevel: "Nivel",
    hudQC: "Saldo QC",
    hudTGWT: "Saldo TGWT",
    hudXP: "XP",
    hint: "← → moverse, ↑ impulso, 1–5 elegir mina, SPACE activa/desactiva auto-mina",

    // Bottom nav
    navStart: "Inicio",
    navWallet: "Billetera",
    navTop: "Top",
    navStore: "Tienda de Naves",
    navTGWT: "TGWT",
    navSocial: "Seguir QC",
    navWatch: "Ver y Ganar",
    navSettings: "Ajustes",

    panelTitles: {
      wallet: "Billetera",
      top: "Top Jugadores",
      store: "Tienda de Naves",
      tgwt: "Recompensas TGWT",
      social: "QC Social",
      watch: "Ver y Ganar",
      settings: "Ajustes",
    },
    panelBodies: {
      wallet: "Ver saldos, conectar MetaMask y gestionar tu seed de QC.",
      top: "Clasificación y estadísticas globales (demo hasta conectar backend).",
      store: "Todos comienzan con una nave gratis. Compra y equipa naves con QC.",
      tgwt: "Saldos TGWT respaldados por ledger, estado del pool, historial y solicitudes de retiro.",
      social: "Sigue QuantumCoin en canales oficiales. Verificaciones pueden dar TGWT luego.",
      watch: "Mira videos seleccionados para ganar recompensas demo.",
      settings: "Idioma y configuración del juego.",
    },

    mine: {
      subtitle: "QuantumCoin mining sector.",
      backToMoon: "← Volver a la Luna",
    },
    "mine.subtitle": "QuantumCoin mining sector.",
    "mine.backToMoon": "← Volver a la Luna",

    topUI: {
      sectionStatusTitle: "ESTADÍSTICAS GLOBALES",
      sectionStatusSubtitle: "Resumen de estadísticas de todos los jugadores.",
      cardTotalPlayersTitle: "JUGADORES TOTALES",
      cardTotalPlayersDesc: "Cuentas únicas registradas.",
      cardOnlineTitle: "EN LÍNEA",
      cardOnlineDesc: "Activos en los últimos minutos.",
      cardRankTitle: "TU RANGO",
      cardRankDesc: "Tu posición por ganancias de QC.",
      cardTotalEarnedTitle: "QC TOTAL GANADO",
      cardTotalEarnedDesc: "QC total desde que empezaste.",
      sectionLeaderboardTitle: "TOP GANADORES QC",
      sectionLeaderboardSubtitle: "Jugadores ordenados por QC ganado.",
      colRank: "#",
      colPlayer: "JUGADOR",
      colQC: "QC",
      colSessions: "SESIONES",
      colLastSeen: "ÚLTIMA VEZ",
      footerNote: "Datos demo por ahora. Estadísticas reales cuando se conecte el backend.",
    },

    walletUI: {
      headingBalances: "Saldos",
      labelGameQC: "Saldo QC en el juego",
      labelGameTGWT: "Saldo TGWT",
      labelLevel: "Nivel",

      headingConnection: "Billetera & MetaMask",
      connectionSub: "Gestiona MetaMask y el seed de QC en un solo lugar.",

      metaTitle: "Conexión MetaMask",
      btnConnect: "Conectar MetaMask",

      pillNotConnected: "No conectado",
      pillConnectedPrefix: "Conectado:",

      statusDisconnected: "No conectado.",
      statusNoMeta: "No se encontró MetaMask en este navegador. Instala la extensión.",
      statusConnectedPrefix: "Conectado:",

      // Seed MUST stay English
      seedTitle: "QC wallet and recovery",
      seedNotGenerated: "Create and sign with the QuantumCoin wallet app.",
      btnGenerateSeed: "Generate new seed",
      btnShowSeed: "Show seed",
      btnHideSeed: "Hide seed",
      btnCopySeed: "Copy seed",
      seedWarning:
        "Use the wallet app for new wallets. Encrypt any old browser recovery in the recovery panel below; keep its backup and passphrase offline.",
      copyOk: "Seed copied to clipboard.",
      copyFail: "Copy failed. Please copy manually.",
    },

    storeUI: {
      heading: "Tienda de Naves",
      intro: "Todos empiezan con una nave gratis. Compra naves avanzadas con QC.",
      priceLabel: "Precio",
      freeLabel: "Gratis",
      tierLabel: "Nivel",
      ownedBadge: "Tuya",
      activeBadge: "Activa",
      buyButton: "Comprar nave",
      equipButton: "Activar nave",
      activeButton: "Nave activa",
      notEnoughQC: "No tienes suficiente QC.",
    },

    socialUI: {
      panelTitle: "QC Social",
      panelIntro: "Sigue QuantumCoin en canales oficiales. Las campañas TGWT verificadas se preparan solo para BSC mainnet.",
      cardTitle: "Sigue QuantumCoin",
      cardIntro: "Las campañas de recompensa TGWT están desactivadas en testnet y seguirán apagadas hasta habilitarse explícitamente en BSC mainnet.",
      linkWebsite: "Sitio web – qcnetwork.ai",
      linkX: "X (Twitter) – @QuantumCoinQC",
      linkTelegram: "Telegram – Comunidad QuantumCoin",
      linkTikTok: "TikTok – @quantumcoin21",
      linkYouTube: "YouTube – @QuantumCoinHQ",
      linkTGWTWebsite: "https://thegreenworldtechnology.com/",
      linkXTGWT: "https://x.com/TGWTToken1",
      rewardNote: "Plan: 1 TGWT por plataforma verificada por usuario. Reglas luego.",
    },

    watch: {
      title: "VER Y GANAR",
      desc: "Mira videos cortos sobre QuantumCoin para ganar recompensas demo.",
      statusLabel: "Estado",
      requiredTimeLabel: "Tiempo requerido",
      rewardLabel: "Recompensa (demo)",
      statusIdle: "Aún no iniciado",
      statusRunning: "Viendo en progreso...",
      statusCompleted: "Tiempo completo — ¡recompensa lista (demo)!",
      hint: "Demo: al completar el tiempo, se marca “Recompensa lista”.",
      videoListTitle: "Videos",
      durationLabel: "Duración",
      startBtn: "Empezar",
      videoIntroTitle: "¿Qué es QuantumCoin? (Intro corta)",
      videoSpaceMinerTitle: "Space Miner Gameplay (Demo)",
    },
  },

  zh: {
    // HUD
    hudLevel: "等级",
    hudQC: "QC 余额",
    hudTGWT: "TGWT 余额",
    hudXP: "经验",
    hint: "← → 移动, ↑ 推进, 1–5 选择矿点, 空格 开/关 自动挖矿",

    // Bottom nav
    navStart: "开始",
    navWallet: "钱包",
    navTop: "排行",
    navStore: "飞船商店",
    navTGWT: "TGWT",
    navSocial: "关注QC",
    navWatch: "观看赚取",
    navSettings: "设置",

    panelTitles: {
      wallet: "钱包",
      top: "排行榜",
      store: "飞船商店",
      tgwt: "TGWT 奖励",
      social: "QC 社交",
      watch: "观看赚取",
      settings: "设置",
    },
    panelBodies: {
      wallet: "查看余额、连接 MetaMask，并管理 QC seed。",
      top: "排行榜与全局统计（连接后端后显示真实数据）。",
      store: "每位玩家都有免费初始飞船，可用 QC 购买并装备高级飞船。",
      tgwt: "显示由账本支持的 TGWT 余额、奖池状态、历史记录和提取请求。",
      social: "关注 QuantumCoin 官方渠道，验证后未来可能获得 TGWT。",
      watch: "观看指定视频获得演示奖励。",
      settings: "语言与游戏设置。",
    },

    mine: {
      subtitle: "QuantumCoin mining sector.",
      backToMoon: "← 返回月面",
    },
    "mine.subtitle": "QuantumCoin mining sector.",
    "mine.backToMoon": "← 返回月面",

    topUI: {
      sectionStatusTitle: "全局统计",
      sectionStatusSubtitle: "Space Miner 玩家总体概览。",
      cardTotalPlayersTitle: "玩家总数",
      cardTotalPlayersDesc: "已加入的唯一账户数。",
      cardOnlineTitle: "当前在线",
      cardOnlineDesc: "最近几分钟活跃的玩家。",
      cardRankTitle: "你的排名",
      cardRankDesc: "按 QC 收益的排名位置。",
      cardTotalEarnedTitle: "累计 QC",
      cardTotalEarnedDesc: "自开始以来的总 QC 收益。",
      sectionLeaderboardTitle: "QC 收益排行",
      sectionLeaderboardSubtitle: "按 QC 收益排序的玩家。",
      colRank: "#",
      colPlayer: "玩家",
      colQC: "QC",
      colSessions: "场次",
      colLastSeen: "最近上线",
      footerNote: "当前为演示数据。连接后端后显示真实链上统计。",
    },

    walletUI: {
      headingBalances: "余额",
      labelGameQC: "游戏内 QC 余额",
      labelGameTGWT: "TGWT 余额",
      labelLevel: "等级",

      headingConnection: "钱包 & MetaMask",
      connectionSub: "在一个界面中管理 MetaMask 与 QC seed。",

      metaTitle: "MetaMask 连接",
      btnConnect: "连接 MetaMask",

      pillNotConnected: "未连接",
      pillConnectedPrefix: "已连接：",

      statusDisconnected: "未连接。",
      statusNoMeta: "此浏览器未检测到 MetaMask，请安装扩展。",
      statusConnectedPrefix: "已连接：",

      // Seed MUST stay English
      seedTitle: "QC wallet and recovery",
      seedNotGenerated: "Create and sign with the QuantumCoin wallet app.",
      btnGenerateSeed: "Generate new seed",
      btnShowSeed: "Show seed",
      btnHideSeed: "Hide seed",
      btnCopySeed: "Copy seed",
      seedWarning:
        "Use the wallet app for new wallets. Encrypt any old browser recovery in the recovery panel below; keep its backup and passphrase offline.",
      copyOk: "Seed copied to clipboard.",
      copyFail: "Copy failed. Please copy manually.",
    },

    storeUI: {
      heading: "飞船商店",
      intro: "每位玩家都有免费初始飞船。使用 QC 购买高级飞船。",
      priceLabel: "价格",
      freeLabel: "免费",
      tierLabel: "等级",
      ownedBadge: "已拥有",
      activeBadge: "已装备",
      buyButton: "购买飞船",
      equipButton: "设为当前",
      activeButton: "当前飞船",
      notEnoughQC: "QC 不足。",
    },

    socialUI: {
      panelTitle: "QC 社交",
      panelIntro: "关注 QuantumCoin 官方渠道。已验证的 TGWT 活动仅计划在 BSC 主网上运行。",
      cardTitle: "关注 QuantumCoin",
      cardIntro: "TGWT 奖励活动在测试网上关闭；只有在 BSC 主网上明确启用后才会发放奖励。",
      linkWebsite: "官网 – qcnetwork.ai",
      linkX: "X (Twitter) – @QuantumCoinQC",
      linkTelegram: "Telegram – QuantumCoin 社区",
      linkTikTok: "TikTok – @quantumcoin21",
      linkYouTube: "YouTube – @QuantumCoinHQ",
      linkTGWTWebsite: "https://thegreenworldtechnology.com/",
      linkXTGWT: "https://x.com/TGWTToken1",
      rewardNote: "计划：每个平台验证后每位用户 1 TGWT（规则后续完善）。",
    },

    watch: {
      title: "观看赚取",
      desc: "观看 QuantumCoin 短视频，获得演示奖励。",
      statusLabel: "状态",
      requiredTimeLabel: "需要时间",
      rewardLabel: "奖励（演示）",
      statusIdle: "尚未开始",
      statusRunning: "观看中…",
      statusCompleted: "时间完成 — 奖励已就绪（演示）！",
      hint: "演示：计时结束后标记“奖励就绪”。",
      videoListTitle: "视频",
      durationLabel: "时长",
      startBtn: "开始观看",
      videoIntroTitle: "什么是 QuantumCoin？（短介绍）",
      videoSpaceMinerTitle: "Space Miner 玩法演示（Demo）",
    },
  },
};
// ================== TGWT Ledger i18n patch (0) ==================
(function patchTGWTI18N(){
  const I = window.I18N || (window.I18N = {});
  const merge = (lang, obj) => {
    I[lang] = I[lang] || {};
    Object.assign(I[lang], obj);
  };

  merge("en", {
    tgwt_ledger_title: "TGWT Ledger",
    tgwt_earned: "Earned",
    tgwt_pending: "Pending",
    tgwt_verified: "Verified",
    tgwt_withdrawn: "Withdrawn",
    tgwt_withdraw_title: "Withdraw request",
    tgwt_withdraw_to: "BSC address (0x...)",
    tgwt_withdraw_amount: "Amount (TGWT)",
    tgwt_withdraw_send: "Send Withdraw Request",
    tgwt_status: "Status",
    tgwt_user: "user"
  });

  merge("tr", {
    tgwt_ledger_title: "TGWT Defteri",
    tgwt_earned: "Kazanılan",
    tgwt_pending: "Bekleyen",
    tgwt_verified: "Doğrulanmış",
    tgwt_withdrawn: "Çekilmiş",
    tgwt_withdraw_title: "Çekim isteği",
    tgwt_withdraw_to: "BSC adresi (0x...)",
    tgwt_withdraw_amount: "Miktar (TGWT)",
    tgwt_withdraw_send: "Çekim İsteği Gönder",
    tgwt_status: "Durum",
    tgwt_user: "kullanıcı"
  });

  merge("es", {
    tgwt_ledger_title: "Libro TGWT",
    tgwt_earned: "Ganado",
    tgwt_pending: "Pendiente",
    tgwt_verified: "Verificado",
    tgwt_withdrawn: "Retirado",
    tgwt_withdraw_title: "Solicitud de retiro",
    tgwt_withdraw_to: "Dirección BSC (0x...)",
    tgwt_withdraw_amount: "Cantidad (TGWT)",
    tgwt_withdraw_send: "Enviar solicitud",
    tgwt_status: "Estado",
    tgwt_user: "usuario"
  });

  // Dikkat: sende "zh" mi "cn" mi kullanılıyor? Buton "中文" ama kod genelde "zh".
  merge("zh", {
    tgwt_ledger_title: "TGWT 账本",
    tgwt_earned: "已获得",
    tgwt_pending: "待处理",
    tgwt_verified: "已验证",
    tgwt_withdrawn: "已提取",
    tgwt_withdraw_title: "提取请求",
    tgwt_withdraw_to: "BSC 地址 (0x...)",
    tgwt_withdraw_amount: "数量 (TGWT)",
    tgwt_withdraw_send: "发送请求",
    tgwt_status: "状态",
    tgwt_user: "用户"
  });
})();
// ================================================================


/* =========================================================
   SAFE DOM HELPERS
========================================================= */
const $ = (id) => document.getElementById(id);
const setText = (id, t) => { const el = $(id); if (el) el.textContent = String(t); };
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));

/* =========================================================
   STORAGE KEYS
========================================================= */
const LANG_STORAGE_KEY    = "qc_lang_v2";
const CORE_STORAGE_KEY    = "qc_core_state_v2";
const SHIP_STORAGE_KEY    = "qc_ship_state_v2";
const WALLET_SEED_KEY     = "qc_wallet_seed_v2";
const WALLET_METAMASK_KEY = "qc_wallet_metamask_v2";
const WATCH_STATE_KEY     = "qc_watch_state_v2";

/* =========================================================
   I18N helpers
========================================================= */
function getPath(obj, path) {
  const parts = String(path || "").split(".");
  let cur = obj;
  for (const p of parts) { if (!cur) return undefined; cur = cur[p]; }
  return cur;
}

function T(key, forcedLang) {
  const useLang = forcedLang || (typeof currentLang === "string" ? currentLang : "en");
  const dictAll = (window.I18N && typeof window.I18N === "object") ? window.I18N : { en: {} };
  const dict = dictAll[useLang] || dictAll.en || {};
  const fb = dictAll.en || {};
  const v = getPath(dict, key);
  if (typeof v === "string") return v;
  const f = getPath(fb, key);
  if (typeof f === "string") return f;
  return String(key || "");
}

/* =========================================================
   CANVAS & SIZE
========================================================= */
const canvas = $("gameCanvas");
if (!canvas) throw new Error("gameCanvas not found");
const ctx = canvas.getContext("2d", { alpha: true });
if (!ctx) throw new Error("2d context not available");

/* =========================================================
   SCENE STATE (✅ single loop - clean)
========================================================= */
let currentScene = "map";   // legacy
let miningActive = false;   // legacy

let miningRAF = 0;          // requestAnimationFrame id (legacy uyum)
let miningTimer = 0;        // setInterval id (kullanırsan; kullanmıyorsan 0 kalır)

let __rafId = 0;
let __lastTs = 0;

// Sekmeden dönünce dt şişmesin: timestamp reset
document.addEventListener("visibilitychange", () => {
  if (!document.hidden) __lastTs = 0;
});

/** TEK gerçek kaynak: body.mine-open + mineScene show/hidden */
function isMineSceneOpen() {
  const el = $("mineScene");
  if (document.body && document.body.classList.contains("mine-open")) return true;
  if (!el) return false;
  return el.classList.contains("show") && !el.classList.contains("hidden");
}

/* =========================================================
   SAFE CALL (anti-spam + never breaks loop)
========================================================= */
(function ensureSafeCall() {
  // global binder: hem window._safeCall hem window.__safeCall
  if (typeof window._safeCall === "function" && typeof window.__safeCall === "function") return;

  let lastKey = "";
  let lastAt = 0;

  function safeCall(fn, label, ...args) {
    try {
      if (!fn) return undefined;

      // string name support: safeCall("updateXP", "updateXP", dt)
      if (typeof fn === "string") {
        const name = fn;
        fn = window[name];
        label = label || name;
      }

      if (typeof fn !== "function") return undefined;
      return fn(...args);
    } catch (e) {
      const msg = (e && e.message) ? e.message : String(e);
      const key = (label || "loop") + ":" + msg;
      const now = (typeof performance !== "undefined" && performance.now) ? performance.now() : Date.now();

      // throttle: aynı hatayı saniyede 1 kez göster
      if (key !== lastKey || (now - lastAt) > 1000) {
        lastKey = key;
        lastAt = now;
        console.error(`loop fn error${label ? " (" + label + ")" : ""}:`, e);
      }
      return undefined;
    }
  }

  window._safeCall = safeCall;
  window.__safeCall = safeCall;

  // local aliases (bu blok içinde direkt kullanacağız)
  // eslint-disable-next-line no-unused-vars
  var _safeCall = safeCall;
  // eslint-disable-next-line no-unused-vars
  var __safeCall = safeCall;
})();

/* =========================================================
   UI globals (bu blok içindeyse burada kalsın)
========================================================= */
let currentLang = "tr";
let currentUiMode = "start";
const LANG_BUTTON_LABELS = Object.freeze({ tr: "TR", en: "EN", es: "ES", zh: "中文" });

// ✅ WATCH STATE must exist before applyLang() calls refreshWatchUI()
let watchState = { running: false, elapsed: 0, required: 30, rewardGranted: false };

/* =========================================================
   LOOP CONTROL
========================================================= */
function startMainLoop() {
  stopMainLoop();
  __lastTs = 0;
  __rafId = requestAnimationFrame(__loop);
  miningRAF = __rafId;
}

function stopMainLoop() {
  if (__rafId) cancelAnimationFrame(__rafId);
  __rafId = 0;
  __lastTs = 0;
  miningRAF = 0;
}

/* ======================================================
   XP HOTFIX (define once, globally safe)
========================================================= */
// XP_DAILY_CAP tanımsızsa loop patlatıyordu. Burada güvenli şekilde tanımlıyoruz.
// "XP konusu kapansın" için cap'i pratikte devre dışı bırakıyoruz.
if (typeof XP_DAILY_CAP === "undefined") {
  window.XP_DAILY_CAP = Number.POSITIVE_INFINITY;
}

/* =========================================================
   MAIN LOOP (single RAF)
========================================================= */
function __loop(ts) {
  if (!__lastTs) __lastTs = ts;

  // dt hesapla (ms) + clamp: max 50ms
  const dtMs = Math.min(Math.max(ts - __lastTs, 0), 50);
  __lastTs = ts;

  // saniye isteyenler için:
  const dt = dtMs / 1000;

  const mineOpen = isMineSceneOpen();
  const uiMode = (typeof currentUiMode === "string") ? currentUiMode : "start";

  // XP (updateXP ms bekliyorsa dtMs ver; değilse kendi içinde dt'ye çevirirsin)
  window.__safeCall(updateXP, "updateXP", dtMs);

  // UPDATE (ambience)
  window.__safeCall(updateStars, "updateStars", dt);
  window.__safeCall(updateMeteors, "updateMeteors", dt);
  window.__safeCall(updateParticles, "updateParticles", dt);

  // Ship sadece map/start ve mine kapalı iken
  if (uiMode === "start" && !mineOpen) {
    window.__safeCall(updateShip, "updateShip", dt);
  }

  // Auto-mine (kendi içinde koşul kontrol ediyorsa her zaman çağırmak güvenli)
  window.__safeCall(updateAutoMine, "updateAutoMine", dt);

  // Mining FX sadece mine overlay açıkken
  if (mineOpen) {
    window.__safeCall(window.tickMineFX, "tickMineFX", dt);
  }

  // Watch timer
  window.__safeCall(window.updateWatch, "updateWatch", dt);

  // DRAW (always)
  window.__safeCall(drawScene, "drawScene", dt);

  // HUD refresh (varsa)
  window.__safeCall(refreshHUDFromState, "refreshHUDFromState");

  __rafId = requestAnimationFrame(__loop);
  miningRAF = __rafId;
}

/* =========================================================
   START (DOM hazır olunca)
========================================================= */
window.addEventListener("load", () => {
  // ✅ ilk frame gelmeden canvas ölçülerini kur
  resize();

  // (opsiyonel) mine spark canvas da hazır olsun
  ensureMineSparkCanvasSize();

  // ✅ sonra tek RAF loop
  startMainLoop();
}, { once: true });


/* =========================================================
   DPI / LAYOUT
========================================================= */
let dpr = window.devicePixelRatio || 1;
let width = 0, height = 0;
let firstLayoutDone = false;

/* =========================================================
   ASSETS
========================================================= */
const ASSET_VER = { bg:"4", moon:"1", mine:"1" };
function withV(path, v){
  const sep = path.includes("?") ? "&" : "?";
  return `${path}${sep}v=${encodeURIComponent(v)}`;
}
function makeImg(src, label){
  const img = new Image();
  img.decoding = "async";
  img.onload  = () => { img.__loaded = true; };
  img.onerror = () => console.warn(label + " missing:", src);
  img.src = src;
  return img;
}
const bgImg   = makeImg(withV("assets/nebula_bg_v4.png", ASSET_VER.bg), "Background");
const moonImg = makeImg(withV("assets/moon_cc0.png", ASSET_VER.moon), "Moon");

function drawImageCover(ctx2, img, dx, dy, dw, dh, alpha = 1) {
  if (!img || !img.complete || img.naturalWidth === 0) return false;
  const iw = img.naturalWidth, ih = img.naturalHeight;
  const ir = iw / ih, dr = dw / dh;
  let sx = 0, sy = 0, sw = iw, sh = ih;
  if (ir > dr) { sw = ih * dr; sx = (iw - sw) / 2; }
  else         { sh = iw / dr; sy = (ih - sh) / 2; }
  ctx2.save();
  ctx2.globalAlpha = alpha;
  ctx2.drawImage(img, sx, sy, sw, sh, dx, dy, dw, dh);
  ctx2.restore();
  return true;
}

/* =========================================================
   CORE STATE (localStorage)
========================================================= */
function loadCoreStateFromStorage() {return {qc:0,tgwt:0,xp:0,level:1,totalEarnedQC:0};}
function saveCoreStateToStorage() {/* Scene never stores account data. */}
let coreState = loadCoreStateFromStorage();
// Günlük sayaç (localStorage ile)






// dt: ms veya sec olabilir, aşağıda güvenli hale getiriyoruz
function updateXP(){ /* Server snapshot owns XP. */ }

// ====================== XP (FAST + STABLE DEMO) ======================
// Single source of truth
const XP_PER_LEVEL       = 100;   // HUD 120k çizgisine uyumlu
const XP_RATE_PER_SECOND = 0.5;      // eski demoya yakın hız (gözle görülür)
window.XP_DAILY_CAP      = Number.POSITIVE_INFINITY; // cap kapalı (demo)

// core save throttle (disk spam olmasın)
let __saveCoreT = 0;
function scheduleSaveCoreState(){
  clearTimeout(__saveCoreT);
  __saveCoreT = setTimeout(() => {
    try{
      syncStateToCore();
      saveCoreStateToStorage();
    }catch{}
  }, 250);
}

// dtAny: ms veya sec gelebilir



/* =========================================================
   SHIP CONFIG
========================================================= */
const SHIP_CONFIG = window.SHIP_CONFIG || [
  { id:"starter", sprite:"assets/ships/ship_starter.png", tier:0, priceQC:0, speedMultiplier:1.0, baseMiningBonus:0.0,
    name:{tr:"Başlangıç Gemisi",en:"Starter Ship",es:"Nave Inicial",zh:"初始飞船"},
    description:{tr:"Herkeste ücretsiz. Dengeli hız ve kazım.",en:"Free for everyone. Balanced speed and mining.",es:"Ligera y ágil. Pequeño bonus de minería.",zh:"所有人免费。速度与挖矿均衡。"} },
  { id:"ship_01", sprite:"assets/ships/ship_01.png", tier:1, priceQC:10000, speedMultiplier:1.08, baseMiningBonus:0.05,
    name:{tr:"Kuyruk Yıldızı",en:"Comet",es:"Cometa",zh:"彗星"},
    description:{tr:"Hafif, çevik. Az mining bonus.",en:"Light and agile. Small mining bonus.",es:"Ligera y ágil. Pequeño bonus de minería.",zh:"轻巧灵活，小幅挖矿加成。"} },
  { id:"ship_02", sprite:"assets/ships/ship_02.png", tier:2, priceQC:20000, speedMultiplier:1.12, baseMiningBonus:0.10,
    name:{tr:"Madenci",en:"Miner",es:"Minera",zh:"矿工"},
    description:{tr:"Daha iyi kazım verimi.",en:"Better mining efficiency.",es:"Mejor eficiencia de minería.",zh:"更高挖矿效率。"} },
  { id:"ship_03", sprite:"assets/ships/ship_03.png", tier:3, priceQC:40000, speedMultiplier:1.18, baseMiningBonus:0.16,
    name:{tr:"Spektral",en:"Spectral",es:"Espectral",zh:"光谱"},
    description:{tr:"Hız + kazım dengesi.",en:"Balanced speed and mining.",es:"Equilibrio entre velocidad y minería.",zh:"速度与挖矿更均衡。"} },
  { id:"ship_04", sprite:"assets/ships/ship_04.png", tier:4, priceQC:60000, speedMultiplier:1.25, baseMiningBonus:0.22,
    name:{tr:"Nova",en:"Nova",es:"Nova",zh:"新星"},
    description:{tr:"Yüksek performans.",en:"High performance.",es:"Alto rendimiento.",zh:"高性能。"} },
  { id:"ship_05", sprite:"assets/ships/ship_05.png", tier:5, priceQC:80000, speedMultiplier:1.33, baseMiningBonus:0.30,
    name:{tr:"Kuantum",en:"Quantum",es:"Cuántica",zh:"量子"},
    description:{tr:"En iyi gemi. Maksimum bonus.",en:"Best ship. Maximum bonus.",es:"La mejor nave. Bonus máximo.",zh:"最佳飞船，最高加成。"} },
];

const shipSprites = {};
SHIP_CONFIG.forEach(cfg=>{
  const img = new Image();
  img.decoding = "async";
  img.src = cfg.sprite;
  img.onerror = () => console.warn("Ship sprite missing:", cfg.sprite);
  shipSprites[cfg.id] = img;
});

let shipState = { owned: { starter: true }, activeId: "starter" };
function loadShipStateFromStorage() {
  try {
    const raw = localStorage.getItem(SHIP_STORAGE_KEY);
    if (!raw) return;
    const p = JSON.parse(raw);
    if (p && typeof p === "object") {
      if (p.owned && typeof p.owned === "object") shipState.owned = p.owned;
      if (typeof p.activeId === "string") shipState.activeId = p.activeId;
    }
  } catch {}
  shipState.owned.starter = true;
  if (!shipState.activeId) shipState.activeId = "starter";
}
function saveShipStateToStorage() { try { localStorage.setItem(SHIP_STORAGE_KEY, JSON.stringify(shipState)); } catch {} }
function isShipOwned(id){ return !!shipState.owned[id]; }
function getActiveShipConfig(){
  let cfg = SHIP_CONFIG.find(s=>s.id===shipState.activeId);
  if (!cfg) cfg = SHIP_CONFIG[0];
  return cfg;
}
function getCurrentShipStats(){
  const cfg = getActiveShipConfig();
  const speed = cfg.speedMultiplier || 1.0;
  const miningBonus = cfg.baseMiningBonus || 0.0;
  return { speed, mining: 1 + miningBonus, handling: 1.0 };
}
loadShipStateFromStorage();

/* =========================================================
   MINING SITES (5)
========================================================= */
const MINING_SITES = window.MINING_SITES || [
  { id: 1, code:"LUNA-01", name:"VEGA CRATER",  bg:"assets/mines/mine_01.png",
    map:{ px:-0.42, py:-0.12, r: 9 }, target:{ x:0.50, y:0.78 }, color:"#22c55e" },
  { id: 2, code:"LUNA-02", name:"ORION RIDGE",  bg:"assets/mines/mine_02.png",
    map:{ px: 0.34, py:-0.22, r: 9 }, target:{ x:0.62, y:0.76 }, color:"#38bdf8" },
  { id: 3, code:"LUNA-03", name:"NEBULA VAULT", bg:"assets/mines/mine_03.png",
    map:{ px: 0.10, py: 0.10, r: 9 }, target:{ x:0.58, y:0.72 }, color:"#a78bfa" },
  { id: 4, code:"LUNA-04", name:"NOVA TRENCH",  bg:"assets/mines/mine_04.png",
    map:{ px:-0.30, py: 0.32, r: 9 }, target:{ x:0.60, y:0.80 }, color:"#fb7185" },
  { id: 5, code:"LUNA-05", name:"PULSAR FIELD", bg:"assets/mines/mine_05.png",
    map:{ px: 0.42, py: 0.34, r: 9 }, target:{ x:0.64, y:0.74 }, color:"#f59e0b" },
];

const miningSites = MINING_SITES.map(s => ({
  id: s.id,
  px: Number(s.map?.px ?? 0),
  py: Number(s.map?.py ?? 0),
  rBase: Number(s.map?.r ?? 9),
  color: s.color || "#ffffff",
  code: s.code || ("LUNA-0" + s.id),
  name: s.name || "UNKNOWN SITE",
  bg: s.bg || "",
  target: s.target || { x: 0.62, y: 0.78 },
}));

// ------------------- STATE / GLOBAL FLAGS -------------------
let isMineOpen = false;

/* =========================================================
   SOCIAL LINKS
========================================================= */
const SOCIAL_LINKS = window.SOCIAL_LINKS || {
  website: { url:"https://qcnetwork.ai/", labelKey:"socialUI.linkWebsite" },
  x:       { url:"https://x.com/QuantumCoinQC", labelKey:"socialUI.linkX" },

  // NEW
  x_tgwt:  { url:"https://x.com/TGWTToken1", labelKey:"socialUI.linkXTGWT" },
  tgwt:    { url:"https://thegreenworldtechnology.com/", labelKey:"socialUI.linkTGWTWebsite" },

  telegram:{ url:"https://web.telegram.org/a/#-1002870924021", labelKey:"socialUI.linkTelegram" },
  tiktok:  { url:"https://www.tiktok.com/@quantumcoin21", labelKey:"socialUI.linkTikTok" },
  youtube: { url:"https://www.youtube.com/@QuantumCoinHQ", labelKey:"socialUI.linkYouTube" },
};
const SETTINGS_README_EN = `QuantumCoin Space Miner — Prototype (EN)

Quick start
- Run: python -m http.server 8000
- Open: http://127.0.0.1:8000/space_miner_prototype.html?v=1

Gameplay
- On Moon map: choose a mining site (click or keys 1–5)
- Mining happens in the Mining Scene (site overlay), not on the main map.
- SPACE: toggle auto-mine (demo)

Data
- Progress is stored locally in your browser (localStorage).
- Seed phrase text is always English by design.

Links
- Website: qcnetwork.ai
- X: @QuantumCoinQC
- X (TGWT): @TGWTToken1
- TGWT Website: thegreenworldtechnology.com
`;

/* =========================================================
   SEED WORDS (EN only)
========================================================= */
const WORDS = [
  "orbit","lunar","asteroid","galaxy","quantum","miner","rocket","nebula",
  "comet","plasma","cosmic","stellar",
  "gravity","horizon","zenith","nova","pulse","photon","ion","saturn",
  "apollo","drift","signal","vault",
  "matrix","vector","cipher","module","anchor","crystal","echo","ember",
  "flux","halo","mirage","zen"
];


/* =========================================================
   GLOBAL STATE
========================================================= */

const walletState = {
  seed: null,
  seedVisible: false,
  metamaskAddress: null,
  metamaskConnected: false
};

const state = {
  ship: { x:0, y:0, vx:0, vy:0, angle:-Math.PI/2, radius:18, fuel:100 },
  stars: [],
  meteors: [],
  particles: [],
  keys: {},
  scene: { zoom: 1, zoomTarget: 1 },

  resources: Number.isFinite(coreState.qc) ? coreState.qc : 0,
  totalEarnedQC: Number.isFinite(coreState.totalEarnedQC) ? coreState.totalEarnedQC : 0,
  xpTotal: Number.isFinite(coreState.xp) ? coreState.xp : 0,
  level: Number.isFinite(coreState.level) ? coreState.level : 1,
  tgwt: Number.isFinite(coreState.tgwt) ? coreState.tgwt : 0,

  autoMineEnabled: true,
  mining: { activeIndex: 0, progress: 0, progressTarget: 6.0 },

  ui: { pointer:{x:0,y:0,inside:false}, hoverSiteId:null }
};

function syncStateToCore(){
  coreState.qc = state.resources;
  coreState.totalEarnedQC = state.totalEarnedQC;
  coreState.tgwt = state.tgwt;
  coreState.xp = state.xpTotal;
  coreState.level = state.level;
}

/* =========================================================
   HUD
========================================================= */
function refreshHUDFromState(){
  setText("hudLevel", state.level);
  setText("hudQC",   Number(state.resources).toFixed(0));
  setText("hudTGWT", state.tgwtDisplay || "0");

  const hudXpInner = $("hudXpInner");
  const hudXpText  = $("hudXpText");
  const xpInLevel = Number(state.xpTotal) % XP_PER_LEVEL;
  const xpPct = (xpInLevel / XP_PER_LEVEL) * 100;
  if(hudXpInner) hudXpInner.style.width = xpPct + "%";

  const xpK = (xpInLevel / 1000).toFixed(1);
  const xpReqK = (XP_PER_LEVEL / 1000).toFixed(1);
  if(hudXpText) hudXpText.textContent = `${xpInLevel} / ${XP_PER_LEVEL} XP`;
}

/* =========================================================
   RESIZE
========================================================= */
function resize(){
  dpr = window.devicePixelRatio || 1;
  width = window.innerWidth;
  height = window.innerHeight;

  canvas.style.width = width + "px";
  canvas.style.height = height + "px";
  canvas.width = Math.floor(width * dpr);
  canvas.height = Math.floor(height * dpr);
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

  if (!firstLayoutDone) {
    state.ship.x = width / 2;
    state.ship.y = height * 0.78;
    firstLayoutDone = true;
  }

  initStars();
  initMeteors();
  ensureMineSparkCanvasSize();
}
window.addEventListener("resize", resize);

function ensureMineSparkCanvasSize(){
  const sc = $("mineSparkCanvas");
  if(!sc) return;
  const dpr2 = Math.max(1, window.devicePixelRatio || 1);
  sc.width = Math.floor(width * dpr2);
  sc.height = Math.floor(height * dpr2);
  sc.style.width = width + "px";
  sc.style.height = height + "px";
  const sctx = sc.getContext("2d");
  if(sctx) sctx.setTransform(dpr2,0,0,dpr2,0,0);
}

/* =========================================================
   STARS & METEORS
========================================================= */
function randRange(min,max){ return min + Math.random()*(max-min); }

function initStars(){
  state.stars = [];
  const count = 220;
  for(let i=0;i<count;i++){
    state.stars.push({
      x:Math.random()*width,
      y:Math.random()*height,
      size:Math.random()*1.6+0.4,
      base:Math.random()*0.6+0.2,
      speed:Math.random()*0.02+0.01
    });
  }
}

function initMeteors(){
  state.meteors = [];
  const count = 25;
  for(let i=0;i<count;i++) state.meteors.push(makeMeteor());
}

function makeMeteor(){
  const m = {
    x:0,y:0,vx:0,vy:0,
    speed:260, len:220, thick:1.6, headR:2.2, alpha:0.75, age:0,
    front: (Math.random() < 0.35) // %30 AYIN ÖNÜNDEN
  };
  resetMeteor(m);
  m.age = Math.random() * 2;
  return m;
}

function resetMeteor(a){
  const margin = 120;
  const side = Math.floor(Math.random() * 4);

  // ara sıra yeniden katman seçimi (tam random değil; istikrar için hafif düşük oran)
  if (typeof a.front !== "boolean") a.front = (Math.random() < 0.30);
  else if (Math.random() < 0.12) a.front = (Math.random() < 0.30);

  a.speed = randRange(240, 360);
  a.len   = randRange(180, 320);
  a.thick = randRange(1.1, 2.0);
  a.headR = randRange(1.8, 2.8);
  a.alpha = randRange(0.55, 0.82);
  a.age   = 0;

  const tilt = randRange(-0.25, 0.25);
  if(side === 0){
    a.x = Math.random() * width; a.y = -margin;
    const ang = (Math.PI/2) + tilt; a.vx = Math.cos(ang); a.vy = Math.sin(ang);
  } else if(side === 1){
    a.x = width + margin; a.y = Math.random() * height;
    const ang = Math.PI + tilt; a.vx = Math.cos(ang); a.vy = Math.sin(ang);
  } else if(side === 2){
    a.x = Math.random() * width; a.y = height + margin;
    const ang = (-Math.PI/2) + tilt; a.vx = Math.cos(ang); a.vy = Math.sin(ang);
  } else {
    a.x = -margin; a.y = Math.random() * height;
    const ang = 0 + tilt; a.vx = Math.cos(ang); a.vy = Math.sin(ang);
  }

  // depth tuning (front = daha yakın, daha hızlı, daha kalın)
  if (a.front) {
    a.speed *= 1.35;
    a.len   *= 1.15;
    a.thick *= 1.55;
    a.headR *= 1.25;
    a.alpha = Math.min(0.95, a.alpha + 0.12);
  } else {
    a.alpha = Math.min(0.82, a.alpha);
  }

  const mag = Math.hypot(a.vx, a.vy) || 1;
  a.vx /= mag; a.vy /= mag;
}

function updateMeteors(dt){
  for(const m of state.meteors){
    m.age += dt;
    m.x += m.vx * m.speed * dt;
    m.y += m.vy * m.speed * dt;
    const margin = 240;
    if (m.x < -margin || m.x > width + margin || m.y < -margin || m.y > height + margin) resetMeteor(m);
  }
}

function drawMeteor(m){
  const fadeIn = clamp(m.age / 0.35, 0, 1);
  const a = m.alpha * fadeIn;

  const hx = m.x, hy = m.y;
  const tx = hx - m.vx * m.len;
  const ty = hy - m.vy * m.len;

  ctx.save();
  ctx.globalAlpha = a;
  ctx.lineCap = "round";

  const grad = ctx.createLinearGradient(tx, ty, hx, hy);
  grad.addColorStop(0, "rgba(255,255,255,0)");
  grad.addColorStop(1, "rgba(220,240,255,1)");

  ctx.strokeStyle = grad;
  ctx.lineWidth = m.thick;
  ctx.beginPath();
  ctx.moveTo(tx, ty);
  ctx.lineTo(hx, hy);
  ctx.stroke();

  ctx.fillStyle = "rgba(255,255,255,1)";
  ctx.beginPath();
  ctx.arc(hx, hy, m.headR, 0, Math.PI*2);
  ctx.fill();
  ctx.restore();
}


/* =========================================================
   MOON + SITES
========================================================= */
function getMoonGeom(){
  const baseR = Math.min(width,height)*0.26;
  const zoom = state.scene.zoom || 1;
  const r = baseR * zoom * 1.70;
  const cx = width/2;
  const cy = height*0.52;
  return {cx,cy,r};
}
function drawMoon(){
  const { cx, cy, r } = getMoonGeom();
  ctx.save();
  ctx.beginPath();
  ctx.arc(cx, cy, r, 0, Math.PI * 2);
  ctx.closePath();
  ctx.clip();

  if (moonImg && moonImg.complete && moonImg.naturalWidth > 0) {
    const k = 1.26;
    ctx.save();
ctx.beginPath();
ctx.arc(cx, cy, r * k, 0, Math.PI * 2);
ctx.closePath();
ctx.clip();
drawImageCover(ctx, moonImg, cx - r * k, cy - r * k, r * 2 * k, r * 2 * k, 1);
ctx.restore();
  } else {
    const g = ctx.createRadialGradient(cx, cy, r * 0.2, cx, cy, r);
    g.addColorStop(0, "#ffffff");
    g.addColorStop(1, "#555b72");
    ctx.fillStyle = g;
    ctx.fillRect(cx - r, cy - r, r * 2, r * 2);
  }
  ctx.restore();
}
function drawRoundedRect(ctx2, x, y, w, h, r) {
  const rr = Math.min(r, w / 2, h / 2);
  ctx2.beginPath();
  ctx2.moveTo(x + rr, y);
  ctx2.arcTo(x + w, y, x + w, y + h, rr);
  ctx2.arcTo(x + w, y + h, x, y + h, rr);
  ctx2.arcTo(x, y + h, x, y, rr);
  ctx2.arcTo(x, y, x + w, y, rr);
  ctx2.closePath();
}
function drawMiningSite(ctx2, x, y, site, rOverride) {
  const hoverId = state.ui?.hoverSiteId ?? null;
  const isHover = hoverId === site.id;

  const baseR = (rOverride ?? site.rBase ?? 9);
  const t = performance.now() * 0.005;
  const pulse = isHover ? (1 + 0.06 * Math.sin(t)) : 1;
  const r = Math.max(6, baseR * (isHover ? 1.35 : 1) * pulse);

  ctx2.save();
  ctx2.shadowBlur = isHover ? 28 : 18;
  ctx2.shadowColor = site.color;

  ctx2.beginPath();
  ctx2.fillStyle = site.color;
  ctx2.globalAlpha = 0.95;
  ctx2.arc(x, y, r, 0, Math.PI * 2);
  ctx2.fill();

  ctx2.shadowBlur = 0;
  ctx2.globalAlpha = 1;

  ctx2.beginPath();
  ctx2.fillStyle = "rgba(255,255,255,0.55)";
  ctx2.arc(x, y, Math.max(2, r * 0.35), 0, Math.PI * 2);
  ctx2.fill();

  ctx2.beginPath();
  ctx2.lineWidth = isHover ? 3 : 2;
  ctx2.strokeStyle = "rgba(255,255,255,0.75)";
  ctx2.arc(x, y, r + (isHover ? 6 : 4), 0, Math.PI * 2);
  ctx2.stroke();

  const label = `${site.code} · ${site.name}`;
  ctx2.textAlign = "left";
  ctx2.textBaseline = "middle";
  ctx2.font = isHover
    ? "700 14px system-ui, -apple-system, Segoe UI, Roboto, Arial"
    : "600 10px system-ui, -apple-system, Segoe UI, Roboto, Arial";

  const padX = isHover ? 14 : 8;
  const metrics = ctx2.measureText(label);
  const w = Math.ceil(metrics.width) + padX * 2;
  const h = isHover ? 28 : 14;

  let lx = x + r + 10;
  if (lx + w + 12 > width) lx = x - r - 10 - w;
  const ly = y;
  const boxX = lx;
  const boxY = ly - h / 2;

  ctx2.globalAlpha = isHover ? 0.96 : 0.70;
  ctx2.fillStyle = isHover ? "rgba(10,18,36,0.78)" : "rgba(0,0,0,0.45)";
  drawRoundedRect(ctx2, boxX, boxY, w, h, 10);
  ctx2.fill();

  ctx2.globalAlpha = isHover ? 0.95 : 0.55;
  ctx2.lineWidth = 1;
  ctx2.strokeStyle = isHover ? "rgba(255,255,255,0.35)" : "rgba(255,255,255,0.22)";
  drawRoundedRect(ctx2, boxX, boxY, w, h, 10);
  ctx2.stroke();

  ctx2.globalAlpha = 1;
  ctx2.fillStyle = "rgba(255,255,255,0.92)";
  ctx2.fillText(label, boxX + padX, ly);

  ctx2.restore();
}
function drawMiningSites(){
  const {cx,cy,r} = getMoonGeom();
  for(const site of miningSites){
    const sx = cx + site.px * r * 0.9;
    const sy = cy + site.py * r * 0.9;
    drawMiningSite(ctx, sx, sy, site, site.rBase);
  }
}
function hitTestMiningSites(px, py){
  const {cx,cy,r} = getMoonGeom();
  let best = null;
  let bestD = 1e9;
  for(const site of miningSites){
    const sx = cx + site.px * r * 0.9;
    const sy = cy + site.py * r * 0.9;
    const dx = px - sx;
    const dy = py - sy;
    const d = Math.hypot(dx, dy);
    const rad = Math.max(10, (site.rBase || 9) * 2.0);
    if (d <= rad && d < bestD){ bestD = d; best = site.id; }
  }
  return best;
}

/* =========================================================
   INPUT + POINTER HITTEST (bind once)
========================================================= */
let canvasPointerBound = false;

function bindCanvasPointerOnce() {
  if (canvasPointerBound) return;
  canvasPointerBound = true;

  canvas.addEventListener("mousemove", (e) => {
    const r = canvas.getBoundingClientRect();
    const x = e.clientX - r.left;
    const y = e.clientY - r.top;

    state.ui.pointer.x = x;
    state.ui.pointer.y = y;
    state.ui.pointer.inside = true;

    if (currentUiMode === "start" && !isMineSceneOpen()) {
      const hit = hitTestMiningSites(x, y);
      state.ui.hoverSiteId = hit;
      canvas.style.cursor = hit ? "pointer" : "default";
    } else {
      state.ui.hoverSiteId = null;
      canvas.style.cursor = "default";
    }
  });

  canvas.addEventListener("mouseleave", () => {
    state.ui.pointer.inside = false;
    state.ui.hoverSiteId = null;
    canvas.style.cursor = "default";
  });

  canvas.addEventListener("click", (e) => {
    if (isMineSceneOpen()) return;
    if (currentUiMode !== "start") return;
    const r = canvas.getBoundingClientRect();
    const x = e.clientX - r.left;
    const y = e.clientY - r.top;
    const id = hitTestMiningSites(x, y);
    if (id) enterMiningSite(id);
    // enterMiningSite içinde (mine açılırken)
miningActive = true;
  });
}
bindCanvasPointerOnce();

window.addEventListener("keydown", (e) => {
  state.keys[e.key] = true;

  // 1-5 spot select (map üstünde)
  if (!isMineSceneOpen() && currentUiMode === "start") {
    const k = e.key;
    if (k >= "1" && k <= "5") {
      state.mining.activeIndex = clamp((Number(k) - 1) | 0, 0, miningSites.length - 1);
      const site = miningSites[state.mining.activeIndex];

      // küçük bir feedback (particle varsa)
      if (typeof window.addParticles === "function") {
        const { cx, cy, r } = getMoonGeom();
        const sx = cx + site.px * r * 0.9;
        const sy = cy + site.py * r * 0.9;
        window.addParticles(sx, sy, site.color);
      }
    }
  }

  // SPACE toggles auto mine (demo) — tek basışta
  if (e.code === "Space" && !e.repeat) {
    e.preventDefault();
    state.autoMineEnabled = !state.autoMineEnabled;
  }

  // ESC close panels / mine
  if (e.key === "Escape") {
    if (isMineSceneOpen()) {
      if (typeof window.exitMineScene === "function") window.exitMineScene();
    } else {
      if (typeof window.closeAllPanels === "function") window.closeAllPanels();
      // exitMineScene içinde (mine kapanırken)
miningActive = false;
    }
  }
});

window.addEventListener("keyup", (e) => {
  state.keys[e.key] = false;
  if (e.code === "Space") e.preventDefault();
});

/* =========================================================
   SHIP (MAP)
========================================================= */
function updateShip(dt){
  const { speed } = getCurrentShipStats();
  const rot = 3.2 * dt;
  if (state.keys["ArrowLeft"] || state.keys["a"]) state.ship.angle -= rot;
  if (state.keys["ArrowRight"] || state.keys["d"]) state.ship.angle += rot;

  if (state.keys["ArrowUp"] || state.keys["w"]) {
    const thrust = 260 * speed;
    state.ship.vx += Math.cos(state.ship.angle) * thrust * dt;
    state.ship.vy += Math.sin(state.ship.angle) * thrust * dt;
  }

  state.ship.vx *= Math.pow(0.985, dt * 60);
  state.ship.vy *= Math.pow(0.985, dt * 60);

  state.ship.x += state.ship.vx * dt;
  state.ship.y += state.ship.vy * dt;

  if (state.ship.x < -40) state.ship.x = width + 40;
  if (state.ship.x > width + 40) state.ship.x = -40;
  if (state.ship.y < -40) state.ship.y = height + 40;
  if (state.ship.y > height + 40) state.ship.y = -40;
}
function drawShip(){
  const cfg = getActiveShipConfig();
  const img = shipSprites[cfg.id];

  ctx.save();
  ctx.translate(state.ship.x, state.ship.y);
  ctx.rotate(state.ship.angle + Math.PI/2);

  if (img && img.complete && img.naturalWidth > 0) {
    const w = 54, h = 54;
    ctx.drawImage(img, -w*0.9, -h*0.9, w*1.8, h*1.8);
  } else {
    ctx.fillStyle = "rgba(255,255,255,0.92)";
    ctx.beginPath();
    ctx.moveTo(0, -18);
    ctx.lineTo(12, 16);
    ctx.lineTo(-12, 16);
    ctx.closePath();
    ctx.fill();
  }
  ctx.restore();
}

/* =========================================================
   MAP MINING (DEMO)
========================================================= */
function drawMiningVeinEffect(){
  if (!state.autoMineEnabled) return;
  const idx = clamp(state.mining.activeIndex|0, 0, miningSites.length-1);
  const site = miningSites[idx];
  const {cx,cy,r} = getMoonGeom();
  const sx = cx + site.px * r * 0.9;
  const sy = cy + site.py * r * 0.9;

  const dx = sx - state.ship.x;
  const dy = sy - state.ship.y;
  const dist = Math.hypot(dx, dy);
  if (dist > 260) return;

  ctx.save();
  ctx.globalAlpha = 0.45;
  ctx.lineWidth = 2;
  const grad = ctx.createLinearGradient(state.ship.x, state.ship.y, sx, sy);
  grad.addColorStop(0, "rgba(56,189,248,0)");
  grad.addColorStop(1, "rgba(34,197,94,1)");
  ctx.strokeStyle = grad;
  ctx.beginPath();
  ctx.moveTo(state.ship.x, state.ship.y);
  ctx.lineTo(sx, sy);
  ctx.stroke();
  ctx.restore();
}

/* =========================================================
   PARTICLES
========================================================= */
function addParticles(x,y,color){
  for(let i=0;i<14;i++){
    state.particles.push({
      x,y,
      vx:(Math.random()-0.5)*220,
      vy:(Math.random()-0.5)*220,
      life:1,
      color
    });
  }
}
function updateParticles(dt){
  const next = [];
  for(const p of state.particles){
    p.life -= dt * 1.4;
    p.x += p.vx * dt;
    p.y += p.vy * dt;
    p.vx *= Math.pow(0.96, dt * 60);
    p.vy *= Math.pow(0.96, dt * 60);
    if (p.life > 0) next.push(p);
  }
  state.particles = next;
}
function drawParticles(){
  ctx.save();
  for(const p of state.particles){
    ctx.globalAlpha = clamp(p.life, 0, 1);
    ctx.fillStyle = p.color || "rgba(255,255,255,0.8)";
    ctx.beginPath();
    ctx.arc(p.x, p.y, 2.2, 0, Math.PI*2);
    ctx.fill();
  }
  ctx.restore();
}

/* =========================================================
   MINE SCENE (overlay) — enter/exit + FX (UPDATED)
========================================================= */
let mineFxTimer = 0;
let mineEventsBound = false;

const mineSpark = { parts: [] };

function clearMineFX() {
  mineFxTimer = 0;
  mineSpark.parts.length = 0;

  const fx = $("mineFXLayer");
  if (fx) fx.querySelectorAll(".mine-beam,.mine-hit").forEach((n) => n.remove());

  const sc = $("mineSparkCanvas");
  if (sc) {
    const c = sc.getContext("2d");
    if (c) {
      const cssW = sc.__cssW || sc.clientWidth || 0;
      const cssH = sc.__cssH || sc.clientHeight || 0;
      c.setTransform(sc.__dpr || 1, 0, 0, sc.__dpr || 1, 0, 0);
      c.clearRect(0, 0, cssW, cssH);
    }
  }
}

// Canvas DPI + size (mineFXLayer baz alınır)
function ensureMineSparkCanvasSize() {
  const sc = $("mineSparkCanvas");
  const fx = $("mineFXLayer");
  if (!sc || !fx) return;

  const r = fx.getBoundingClientRect();
  const cssW = Math.max(1, Math.floor(r.width));
  const cssH = Math.max(1, Math.floor(r.height));
  const dpr = Math.max(1, Math.min(3, window.devicePixelRatio || 1));

  // sadece değiştiyse set et (flicker yok)
  if (sc.__cssW === cssW && sc.__cssH === cssH && sc.__dpr === dpr) return;

  sc.__cssW = cssW;
  sc.__cssH = cssH;
  sc.__dpr = dpr;

  sc.style.width = cssW + "px";
  sc.style.height = cssH + "px";
  sc.width = Math.floor(cssW * dpr);
  sc.height = Math.floor(cssH * dpr);

  const c = sc.getContext("2d");
  if (c) c.setTransform(dpr, 0, 0, dpr, 0, 0);
}

function bindMineEventsOnce() {
  if (mineEventsBound) return;
  mineEventsBound = true;

  const btn = $("mineCloseBtn");
  if (btn) btn.addEventListener("click", exitMineScene);

  window.addEventListener("resize", () => {
    if (!isMineSceneOpen()) return;
    ensureMineSparkCanvasSize();
    clearMineFX();
  }, { passive: true });
}
bindMineEventsOnce();

function spawnMineSpark(x, y) {
  // x/y: mineFXLayer local coords (CSS px)
  // basit limit: patlama olmasın
  if (mineSpark.parts.length > 420) mineSpark.parts.splice(0, mineSpark.parts.length - 420);

  for (let i = 0; i < 10; i++) {
    mineSpark.parts.push({
      x, y,
      vx: (Math.random() - 0.5) * 220,
      vy: (Math.random() - 0.9) * 260,
      life: 0.55 + Math.random() * 0.35,
      r: 1.2 + Math.random() * 1.8,
    });
  }
}

function spawnMineBeam() {
  if (!isMineSceneOpen()) return;

  const fx = $("mineFXLayer");
  const ship = $("mineShip");
  if (!fx || !ship) return;

  const baseRect = fx.getBoundingClientRect();
  if (baseRect.width < 20 || baseRect.height < 20) return;

  // canvas size sync
  ensureMineSparkCanvasSize();

  const shipRect = ship.getBoundingClientRect();
  const startX = (shipRect.left - baseRect.left) + shipRect.width * 0.52 + (Math.random() * 6 - 3);
  const startY = (shipRect.top  - baseRect.top ) + shipRect.height * 0.66 + (Math.random() * 6 - 3);

  const rawEndX = startX + (Math.random() * 180 - 90);
  const rawEndY = startY + (140 + Math.random() * 120);
  const endX = clamp(rawEndX, 12, baseRect.width  - 12);
  const endY = clamp(rawEndY, 12, baseRect.height - 12);

  const dx = endX - startX;
  const dy = endY - startY;
  const len = Math.max(30, Math.hypot(dx, dy));
  const ang = Math.atan2(dy, dx) * (180 / Math.PI);

  const beam = document.createElement("div");
  beam.className = "mine-beam";
  beam.style.left = startX + "px";
  beam.style.top = startY + "px";
  beam.style.width = len + "px";
  beam.style.setProperty("--ang", ang + "deg");
  fx.appendChild(beam);

  const hit = document.createElement("div");
  hit.className = "mine-hit";
  hit.style.left = (endX - 8) + "px";
  hit.style.top  = (endY - 8) + "px";
  fx.appendChild(hit);

  // kısa ömür
  setTimeout(() => { beam.remove(); hit.remove(); }, 520);

  // spark canvas'a kıvılcım
  spawnMineSpark(endX, endY);
}

function tickMineFX(dt) {
  if (!isMineSceneOpen()) return;
  if (!Number.isFinite(dt) || dt <= 0) return;

  // tab geri gelince dt devasa olabiliyor → sınırlayalım
  dt = Math.min(dt, 0.08);

  // canvas sync
  ensureMineSparkCanvasSize();

  // beam ritmi (dt büyükse aşırı beam basmasın)
  mineFxTimer += dt;
  const period = 0.13;
  let guard = 0;
  while (mineFxTimer >= period && guard < 6) {
    mineFxTimer -= period;
    spawnMineBeam();
    guard++;
  }
  if (guard >= 6) mineFxTimer = 0;

  // spark update/draw
  const sc = $("mineSparkCanvas");
  if (!sc) return;
  const c = sc.getContext("2d");
  if (!c) return;

  const w = sc.__cssW || sc.clientWidth || 0;
  const h = sc.__cssH || sc.clientHeight || 0;

  c.setTransform(sc.__dpr || 1, 0, 0, sc.__dpr || 1, 0, 0);
  c.clearRect(0, 0, w, h);

  const next = [];
  for (const p of mineSpark.parts) {
    p.life -= dt * 1.6;
    p.x += p.vx * dt;
    p.y += p.vy * dt;
    p.vx *= Math.pow(0.92, dt * 60);
    p.vy *= Math.pow(0.92, dt * 60);

    if (p.life > 0) {
      next.push(p);

      const a = clamp(p.life, 0, 1);
      c.save();
      c.globalAlpha = a;
      c.fillStyle = "rgba(255,255,255,1)";
      c.beginPath();
      c.arc(p.x, p.y, p.r, 0, Math.PI * 2);
      c.fill();
      c.restore();
    }
  }
  mineSpark.parts = next;
}


/* =========================================================
   ENTER / EXIT MINE
========================================================= */
function enterMiningSite(id) {
  const site = miningSites.find(s => s.id === id) || miningSites[0];
  const mineScene = $("mineScene");
  const mineBg = $("mineBg");
  const mineTitle = $("mineTitle");
  const mineSub = $("mineSub");
  const mineShip = $("mineShip");

  if (!mineScene || !mineBg || !mineTitle || !mineSub || !mineShip) return;

  // UI state
  currentScene = "mine";
  isMineOpen = true;
  state.mining.activeIndex = clamp((site.id - 1) | 0, 0, miningSites.length - 1);

  // bg + labels
  mineBg.style.backgroundImage = `url("${site.bg}")`;
  mineTitle.textContent = `${site.code} · ${site.name}`;
  mineSub.textContent = T("mine.subtitle");

  // ship sprite (aktif gemi)
  const cfg = getActiveShipConfig();
  mineShip.src = cfg.sprite || "assets/ships/ship_starter.png";

  // show overlay
  mineScene.classList.remove("hidden");
  mineScene.classList.add("show");
  mineScene.setAttribute("aria-hidden", "false");
  document.body.classList.add("mine-open");

  // FX reset
  clearMineFX();
  ensureMineSparkCanvasSize();

  // ship animasyon sınıfı (bir frame sonra)
  mineScene.classList.remove("show-ship");
  requestAnimationFrame(() => {
    mineScene.classList.add("show-ship");
  });

  // mining demo burada aktif
  miningActive = true;

window.dispatchEvent(new CustomEvent("qc:mine-enter",{detail:{siteId:id}}));
}

function exitMineScene() {
  const mineScene = $("mineScene");
  if (!mineScene) return;

  clearMineFX();

  mineScene.classList.remove("show-ship");
  mineScene.classList.remove("show");
  mineScene.classList.add("hidden");
  mineScene.setAttribute("aria-hidden", "true");

  document.body.classList.remove("mine-open");

  currentScene = "map";
  isMineOpen = false;
  miningActive = false;

window.dispatchEvent(new CustomEvent("qc:mine-exit"));
}

/* =========================================================
   AUTO MINE (only inside mine overlay)
========================================================= */
// XP/Level persist throttling (storage'ı her frame yazmamak için)
let _lastXpPersistAt = 0;

/* =========================================================
   XP + LEVEL (single source of truth)
========================================================= */


function updateAutoMine(dt) {if(!isMineSceneOpen()||!miningActive)return;state.mining.progress+=Math.min(0.1,Math.max(0,Number(dt)||0));if(state.mining.progress>=state.mining.progressTarget){state.mining.progress=0;addParticles(width*0.5,height*0.7,"#22c55e");}}

/* =========================================================
   STARS UPDATE + DRAW SCENE
========================================================= */
function updateStars(dt){
  for (const s of state.stars) {
    s.y += s.speed * 22 * dt;
    if (s.y > height + 10) { s.y = -10; s.x = Math.random() * width; }
  }
}

function drawStars(){
  const t = performance.now() * 0.001;
  ctx.save();
  for (const s of state.stars) {
    const tw = 0.18 * Math.sin(t * (0.8 + s.speed * 20) + s.x * 0.01);
    const a = clamp(s.base + tw, 0.08, 0.9);
    ctx.globalAlpha = a;
    ctx.fillStyle = "rgba(255,255,255,1)";
    ctx.beginPath();
    ctx.arc(s.x, s.y, s.size, 0, Math.PI * 2);
    ctx.fill();
  }
  ctx.restore();
}


// =========================================================
// QC BLUE BEACON MINING MARKERS PATCH
// - Keeps original mining site coordinates and click logic.
// - Replaces the visual marker with a blue/cyan animated beacon:
//   base glow + rising laser beam + pulse rings.
// =========================================================
drawMiningSite = function drawMiningSite(ctx2, x, y, site, rOverride) {
  const hoverId = state.ui?.hoverSiteId ?? null;
  const isHover = hoverId === site.id;

  const baseR = (rOverride ?? site.rBase ?? 9);
  const now = performance.now() * 0.001;

  const cyan = "rgba(56, 189, 248, ";
  const cyanStrong = "rgba(34, 211, 238, ";
  const white = "rgba(255, 255, 255, ";
  const siteColor = site.color || "#38bdf8";

  const pulseA = (Math.sin(now * 3.2 + site.id * 0.9) + 1) * 0.5;
  const pulseB = (Math.sin(now * 4.4 + site.id * 1.7) + 1) * 0.5;

  const r = Math.max(7, baseR * (isHover ? 1.25 : 1));
  const beamH = r * (isHover ? 6.8 : 5.4);
  const beamW = r * (isHover ? 0.72 : 0.54);

  ctx2.save();
  ctx2.globalCompositeOperation = "lighter";

  // 1) rising vertical blue laser beam
  const beamGrad = ctx2.createLinearGradient(x, y + r * 1.4, x, y - beamH);
  beamGrad.addColorStop(0.00, "rgba(34, 211, 238, 0.00)");
  beamGrad.addColorStop(0.18, "rgba(34, 211, 238, 0.20)");
  beamGrad.addColorStop(0.55, "rgba(125, 211, 252, 0.30)");
  beamGrad.addColorStop(1.00, "rgba(255, 255, 255, 0.00)");

  ctx2.strokeStyle = beamGrad;
  ctx2.lineWidth = beamW;
  ctx2.shadowBlur = isHover ? 26 : 18;
  ctx2.shadowColor = "rgba(56,189,248,0.95)";
  ctx2.beginPath();
  ctx2.moveTo(x, y + r * 1.1);
  ctx2.lineTo(x, y - beamH);
  ctx2.stroke();

  // 2) thin core beam
  ctx2.strokeStyle = white + (isHover ? "0.72)" : "0.42)");
  ctx2.lineWidth = Math.max(1.2, r * 0.16);
  ctx2.beginPath();
  ctx2.moveTo(x, y + r * 0.55);
  ctx2.lineTo(x, y - beamH * 0.92);
  ctx2.stroke();

  // 3) base glow
  const glow = ctx2.createRadialGradient(x, y, 0, x, y, r * 4.1);
  glow.addColorStop(0.00, "rgba(34, 211, 238, 0.58)");
  glow.addColorStop(0.35, "rgba(56, 189, 248, 0.22)");
  glow.addColorStop(1.00, "rgba(56, 189, 248, 0.00)");
  ctx2.fillStyle = glow;
  ctx2.beginPath();
  ctx2.arc(x, y, r * 4.1, 0, Math.PI * 2);
  ctx2.fill();

  // 4) animated ground rings
  for (let i = 0; i < 3; i++) {
    const phase = (now * 0.95 + site.id * 0.22 + i * 0.34) % 1;
    const ringR = r * (1.35 + phase * 2.55);
    const alpha = (1 - phase) * (isHover ? 0.48 : 0.34);

    ctx2.strokeStyle = cyan + alpha.toFixed(3) + ")";
    ctx2.lineWidth = Math.max(1, r * 0.10 * (1 - phase * 0.45));
    ctx2.beginPath();
    ctx2.ellipse(x, y + r * 0.16, ringR * 1.42, ringR * 0.42, 0, 0, Math.PI * 2);
    ctx2.stroke();
  }

  // 5) outer marker shell
  ctx2.shadowBlur = isHover ? 28 : 18;
  ctx2.shadowColor = "rgba(56,189,248,0.95)";
  ctx2.strokeStyle = cyanStrong + (isHover ? "0.95)" : "0.72)");
  ctx2.lineWidth = isHover ? 3.2 : 2.4;
  ctx2.beginPath();
  ctx2.arc(x, y, r * (1.45 + pulseA * 0.10), 0, Math.PI * 2);
  ctx2.stroke();

  // 6) inner disk
  const core = ctx2.createRadialGradient(x - r * 0.28, y - r * 0.28, 0, x, y, r * 1.05);
  core.addColorStop(0.0, "rgba(255,255,255,0.98)");
  core.addColorStop(0.25, "rgba(125,211,252,0.92)");
  core.addColorStop(1.0, "rgba(14,165,233,0.82)");

  ctx2.fillStyle = core;
  ctx2.beginPath();
  ctx2.arc(x, y, r * 0.92, 0, Math.PI * 2);
  ctx2.fill();

  // 7) tiny original color identity dot
  ctx2.globalCompositeOperation = "source-over";
  ctx2.fillStyle = siteColor;
  ctx2.globalAlpha = 0.78;
  ctx2.beginPath();
  ctx2.arc(x, y, Math.max(2.4, r * 0.28), 0, Math.PI * 2);
  ctx2.fill();
  ctx2.globalAlpha = 1;

  // 8) label pill
  const label = `${site.code} · ${site.name}`;
  ctx2.font = `700 ${Math.max(11, r * 0.92)}px system-ui, -apple-system, Segoe UI, sans-serif`;
  const textW = ctx2.measureText(label).width;
  const padX = 9;
  const pillH = Math.max(20, r * 1.75);
  const pillX = x + r * 2.0;
  const pillY = y - pillH / 2;

  ctx2.fillStyle = "rgba(6, 10, 24, 0.58)";
  ctx2.strokeStyle = "rgba(125, 211, 252, 0.36)";
  ctx2.lineWidth = 1;

  if (typeof ctx2.roundRect === "function") {
    ctx2.beginPath();
    ctx2.roundRect(pillX, pillY, textW + padX * 2, pillH, 8);
    ctx2.fill();
    ctx2.stroke();
  } else {
    ctx2.fillRect(pillX, pillY, textW + padX * 2, pillH);
    ctx2.strokeRect(pillX, pillY, textW + padX * 2, pillH);
  }

  ctx2.fillStyle = "rgba(255,255,255,0.92)";
  ctx2.shadowBlur = 8;
  ctx2.shadowColor = "rgba(56,189,248,0.35)";
  ctx2.fillText(label, pillX + padX, y + r * 0.35);

  ctx2.restore();
};

function drawScene(dt){
  // bg
  if (!drawImageCover(ctx, bgImg, 0, 0, width, height, 1)) {
    ctx.clearRect(0,0,width,height);
    ctx.fillStyle = "#000";
    ctx.fillRect(0,0,width,height);
  }

  drawStars();

  // meteors back
  for(const m of state.meteors){ if (!m.front) drawMeteor(m); }

  // moon + sites + ship (map only)
  if (!isMineSceneOpen()) {
    drawMoon();

    if (currentUiMode === "start") {
      drawMiningSites();
      drawMiningVeinEffect();
      drawShip();
    }

    drawParticles();

    // meteors front
    for(const m of state.meteors){ if (m.front) drawMeteor(m); }
  } else {
    // mine açıkken de arka planda ambience kalsın (istersen kapatırız)
    for(const m of state.meteors){ if (m.front) drawMeteor(m); }
    drawParticles();
  }
}

/* =========================================================
   UI PANELS (modal + watch)
========================================================= */
function setNavActive(mode){
  document.querySelectorAll(".nav-btn").forEach(btn=>{
    btn.classList.toggle("active", btn.dataset.nav === mode);
  });
}

function showUiPanel(mode){
  const uiPanel = $("uiPanel");
  if (!uiPanel) return;

  // watch ayrı panel
  hideWatchPanel();

  currentUiMode = mode;

  const title = $("uiPanelTitle");
  const body  = $("uiPanelBody");

  const titles = (I18N[currentLang] || I18N.en).panelTitles || (I18N.en.panelTitles);
  const bodies = (I18N[currentLang] || I18N.en).panelBodies  || (I18N.en.panelBodies);

  if (title) title.textContent = (titles && titles[mode]) ? titles[mode] : String(mode);
  if (body)  body.textContent  = (bodies && bodies[mode]) ? bodies[mode] : "";

  // panels
  ["walletPanel","topPanel","storePanel","tgwtPanel","socialPanel","settingsPanel"].forEach(id=>{
    const el = $(id);
    if (el) el.hidden = true;
  });

  const map = {
    wallet: "walletPanel",
    top: "topPanel",
    store: "storePanel",
    tgwt: "tgwtPanel",
    social: "socialPanel",
    settings: "settingsPanel",
  };
  const showId = map[mode];
  const el = showId ? $(showId) : null;
  if (el) el.hidden = false;

  uiPanel.hidden = false;

  // mode-specific renders
  if (mode === "store") renderStoreGrid();
  if (mode === "social") renderSocialLinks();
  if (mode === "wallet") refreshWalletUI();
  if (mode === "tgwt") renderTGWTRewardsUI();
  if (mode === "settings") renderSettingsUI();
}

function hideUiPanel(){
  const uiPanel = $("uiPanel");
  if (uiPanel) uiPanel.hidden = true;
}

function showWatchPanel(){
  hideUiPanel();
  currentUiMode = "watch";
  const w = $("panel-watch");
  if (w) {
    w.classList.add("show");
    w.setAttribute("aria-hidden","false");
  }
  refreshWatchUI();
}
function hideWatchPanel(){
  const w = $("panel-watch");
  if (w) {
    w.classList.remove("show");
    w.setAttribute("aria-hidden","true");
  }
}

function closeAllPanels(){
  hideUiPanel();
  hideWatchPanel();
  setNavActive("start");
  currentUiMode = "start";
}

/* =========================================================
   LANGUAGE APPLY
========================================================= */
function applyLang(lang){
  if (!I18N[lang]) lang = "en";
  currentLang = lang;

  try { localStorage.setItem(LANG_STORAGE_KEY, currentLang); } catch {}

  // lang btn active
  document.querySelectorAll(".lang-btn").forEach(btn=>{
    btn.classList.toggle("active", btn.dataset.lang === currentLang);
    const code = btn.dataset.lang;
    if (LANG_BUTTON_LABELS[code]) btn.textContent = LANG_BUTTON_LABELS[code];
  });

  // HUD labels
  setText("hudLabelLevel", T("hudLevel"));
  setText("hudLabelQC",    T("hudQC"));
  setText("hudLabelTGWT",  T("hudTGWT"));
  setText("hudLabelXP",    T("hudXP"));
  setText("hintText",      T("hint"));

  // nav labels
  setText("navStartText",    T("navStart"));
  setText("navWalletText",   T("navWallet"));
  setText("navTopText",      T("navTop"));
  setText("navStoreText",    T("navStore"));
  setText("navTGWTText",     T("navTGWT"));
  setText("navSocialText",   T("navSocial"));
  setText("navWatchText",    T("navWatch"));
  setText("navSettingsText", T("navSettings"));

  // Mine overlay
  setText("mineSub", T("mine.subtitle"));
  setText("mineCloseBtn", T("mine.backToMoon"));

  // Top UI
  const top = (I18N[currentLang] || I18N.en).topUI || I18N.en.topUI;
  if (top) {
    setText("topStatusTitle", top.sectionStatusTitle);
    setText("topStatusSubtitle", top.sectionStatusSubtitle);
    setText("topCardTotalPlayersTitle", top.cardTotalPlayersTitle);
    setText("topCardTotalPlayersDesc", top.cardTotalPlayersDesc);
    setText("topCardOnlineTitle", top.cardOnlineTitle);
    setText("topCardOnlineDesc", top.cardOnlineDesc);
    setText("topCardRankTitle", top.cardRankTitle);
    setText("topCardRankDesc", top.cardRankDesc);
    setText("topCardTotalEarnedTitle", top.cardTotalEarnedTitle);
    setText("topCardTotalEarnedDesc", top.cardTotalEarnedDesc);

    setText("topLeaderboardTitle", top.sectionLeaderboardTitle);
    setText("topLeaderboardSubtitle", top.sectionLeaderboardSubtitle);

    setText("topHeaderRank", top.colRank);
    setText("topHeaderPlayer", top.colPlayer);
    setText("topHeaderQC", top.colQC);
    setText("topHeaderSessions", top.colSessions);
    setText("topHeaderLast", top.colLastSeen);
    setText("topFooterNoteText", top.footerNote);
  }

  // Wallet UI (seed always EN)
  const wui = (I18N[currentLang] || I18N.en).walletUI || I18N.en.walletUI;
  setText("walletBalancesTitle", wui.headingBalances);
  setText("walletQCLabel", wui.labelGameQC);
  setText("walletTGWTLabel", wui.labelGameTGWT);
  setText("walletLevelLabel", wui.labelLevel);

  setText("walletConnectionTitle", wui.headingConnection);
  setText("walletConnectionSub", wui.connectionSub);
  setText("walletMetaTitle", wui.metaTitle);
  setText("walletMetaConnectLabel", wui.btnConnect);

  // seed area forced EN
  const seedEN = (I18N.en.walletUI || {});
  setText("walletSeedTitle", seedEN.seedTitle);
  setText("btnGenerateSeed", seedEN.btnGenerateSeed);
  setText("btnCopySeed", seedEN.btnCopySeed);
  setText("walletSeedNote", seedEN.seedWarning);

  // Store
  const sui = (I18N[currentLang] || I18N.en).storeUI || I18N.en.storeUI;
  setText("storePanelTitle", sui.heading);
  setText("storePanelIntro", sui.intro);

  // Social
  const soi = (I18N[currentLang] || I18N.en).socialUI || I18N.en.socialUI;
  setText("socialPanelTitle", soi.panelTitle);
  setText("socialPanelIntro", soi.panelIntro);
  setText("socialCardTitle", soi.cardTitle);
  setText("socialCardIntro", soi.cardIntro);
  setText("socialRewardNote", soi.rewardNote);

  // Watch
  refreshWatchUI();

  // TGWT UI (panel title already from panelTitles)
  if (currentUiMode === "tgwt") renderTGWTRewardsUI();
}

function initLangSelector(){
  try {
    const stored = localStorage.getItem(LANG_STORAGE_KEY);
    if (stored && I18N[stored]) currentLang = stored;
  } catch {}
  applyLang(currentLang);

  document.querySelectorAll(".lang-btn").forEach(btn=>{
    btn.addEventListener("click", ()=>{
      const code = btn.dataset.lang;
      applyLang(code);
    });
  });
}

/* =========================================================
   WALLET (seed + metamask)
========================================================= */



function refreshWalletUI(){window.QCAccount?.refresh().catch(()=>{});}

/* =========================================================
   STORE
========================================================= */
function renderStoreGrid(){
  const grid = $("storeGrid");
  if (!grid) return;

  grid.innerHTML = "";

  const sui = (I18N[currentLang] || I18N.en).storeUI || I18N.en.storeUI;

  for (const cfg of SHIP_CONFIG) {
    const card = document.createElement("div");
    card.className = "store-card";

    const owned = isShipOwned(cfg.id);
    const active = shipState.activeId === cfg.id;

    const badge = document.createElement("div");
    badge.className = "store-badge";
    badge.textContent = owned ? sui.ownedBadge : `${sui.priceLabel}: ${cfg.priceQC ? cfg.priceQC.toLocaleString() : sui.freeLabel}`;
    card.appendChild(badge);

    if (active) {
      const b2 = document.createElement("div");
      b2.className = "store-badge store-badge--active";
      b2.textContent = sui.activeBadge;
      card.appendChild(b2);
    }

    const img = document.createElement("img");
    img.src = cfg.sprite;
    img.alt = cfg.name?.[currentLang] || cfg.name?.en || cfg.id;
    card.appendChild(img);

    const title = document.createElement("div");
    title.className = "store-card-title";
    title.textContent = cfg.name?.[currentLang] || cfg.name?.en || cfg.id;
    card.appendChild(title);

    const desc = document.createElement("div");
    desc.className = "store-card-desc";
    desc.textContent = cfg.description?.[currentLang] || cfg.description?.en || "";
    card.appendChild(desc);

    const meta = document.createElement("div");
    meta.className = "store-card-meta";
    meta.innerHTML = `<span>${sui.tierLabel}: ${cfg.tier}</span><span>${sui.priceLabel}: ${cfg.priceQC ? cfg.priceQC.toLocaleString() : sui.freeLabel}</span>`;
    card.appendChild(meta);

    const btn = document.createElement("button");
    btn.className = "store-card-btn";

    if (!owned) {
      btn.textContent = sui.buyButton;
      btn.disabled = (state.resources < cfg.priceQC);
      btn.addEventListener("click", ()=>{
        if (state.resources < cfg.priceQC) { alert(sui.notEnoughQC); return; }
        state.resources -= cfg.priceQC;
        shipState.owned[cfg.id] = true;
        shipState.activeId = cfg.id;
        saveShipStateToStorage();

        syncStateToCore();
        saveCoreStateToStorage();

        refreshHUDFromState();
        renderStoreGrid();
      });
    } else if (!active) {
      btn.textContent = sui.equipButton;
      btn.addEventListener("click", ()=>{
        shipState.activeId = cfg.id;
        saveShipStateToStorage();
        renderStoreGrid();
      });
    } else {
      btn.textContent = sui.activeButton;
      btn.disabled = true;
    }

    card.appendChild(btn);
    grid.appendChild(card);
  }
}

/* =========================================================
   SOCIAL
========================================================= */
function renderSocialLinks(){
  const ul = $("socialLinksList");
  if (!ul) return;
  ul.innerHTML = "";

  const soi = (I18N[currentLang] || I18N.en).socialUI || I18N.en.socialUI;

  const items = [
    { url: SOCIAL_LINKS.website.url,  label: soi.linkWebsite },
    { url: SOCIAL_LINKS.x.url,        label: soi.linkX },
    { url: SOCIAL_LINKS.telegram.url, label: soi.linkTelegram },
    { url: SOCIAL_LINKS.tiktok.url,   label: soi.linkTikTok },
    { url: SOCIAL_LINKS.youtube.url,  label: soi.linkYouTube },
    { url: SOCIAL_LINKS.tgwt.url,     label: soi.linkTGWTWebsite },
    { url: SOCIAL_LINKS.x_tgwt.url,   label: soi.linkXTGWT },
  ];

  for (const it of items) {
    const li = document.createElement("li");
    const a = document.createElement("a");
    a.href = it.url;
    a.target = "_blank";
    a.rel = "noopener";
    a.textContent = it.label;
    li.appendChild(a);
    ul.appendChild(li);
  }
}

/* =========================================================
   WATCH
========================================================= */
function loadWatchState(){
  try {
    const raw = localStorage.getItem(WATCH_STATE_KEY);
    if (!raw) return;
    const p = JSON.parse(raw);
    if (!p || typeof p !== "object") return;
    watchState.running = !!p.running;
    watchState.elapsed = Number(p.elapsed) || 0;
    watchState.required = Number(p.required) || 30;
    watchState.rewardGranted = !!p.rewardGranted;
  } catch {}
}
function saveWatchState(){
  try { localStorage.setItem(WATCH_STATE_KEY, JSON.stringify(watchState)); } catch {}
}

function refreshWatchUI(){
  setText("watchTitle", T("watch.title"));
  setText("watchDesc", T("watch.desc"));
  setText("watchStatusLabel", T("watch.statusLabel"));
  setText("watchRequiredLabel", T("watch.requiredTimeLabel"));
  setText("watchRewardLabel", T("watch.rewardLabel"));
  setText("watchHintText", T("watch.hint"));
  setText("watchVideoListTitle", T("watch.videoListTitle"));
  setText("watchVideoListDesc", T("panelBodies.watch"));
  setText("watchDurationLabel", T("watch.durationLabel"));
  setText("watchDurationLabelB", T("watch.durationLabel"));
  setText("watchDurationLabel2", T("watch.durationLabel"));
  setText("watchStartBtn", T("watch.startBtn"));
  setText("watchVideoIntroTitle", T("watch.videoIntroTitle"));
  setText("watchVideoSpaceMinerTitle", T("watch.videoSpaceMinerTitle"));

  // time labels (static demo)
  const req = watchState.required || 30;
  setText("watchRequiredTime", `00:${String(req).padStart(2,"0")}`);

  // status text
  const st = $("watchStatusText");
  if (st) {
    if (watchState.rewardGranted) st.textContent = T("watch.statusCompleted");
    else if (watchState.running) st.textContent = T("watch.statusRunning");
    else st.textContent = T("watch.statusIdle");
  }

  // elapsed
  const el = Math.floor(watchState.elapsed || 0);
  setText("watchElapsed", `00:${String(clamp(el,0,99)).padStart(2,"0")}`);

  // progress
  const fill = $("watch-progress-fill");
  if (fill) {
    const pct = req > 0 ? clamp((watchState.elapsed / req) * 100, 0, 100) : 0;
    fill.style.width = pct + "%";
  }
}

function updateWatch(dt){/* Watch controller owns visual progress; provider evidence owns rewards. */}

function startWatching(){
  if (watchState.rewardGranted) {
    // tekrar demo reset istersen burada açarız
    watchState.rewardGranted = false;
    watchState.elapsed = 0;
  }
  watchState.running = true;
  saveWatchState();
  refreshWatchUI();
}

/* =========================================================
   TGWT LEDGER (demo)
========================================================= */









function renderTGWTRewardsUI(){window.QCAccount?.refresh().catch(()=>{});}

/* =========================================================
   SETTINGS (simple)
========================================================= */
function renderSettingsUI(){
  const body = $("settingsBody");
  if (body) body.textContent = SETTINGS_README_EN;
}

/* =========================================================
   NAV + CLOSE BUTTONS
========================================================= */
function initNavAndPanels(){
  const uiClose = $("uiCloseBtn");
  if (uiClose) uiClose.addEventListener("click", closeAllPanels);

  const watchClose = $("watchCloseBtn");
  if (watchClose) watchClose.addEventListener("click", closeAllPanels);

  const watchStart = $("watchStartBtn");
  if (watchStart) watchStart.addEventListener("click", startWatching);

  document.querySelectorAll(".nav-btn").forEach(btn=>{
    btn.addEventListener("click", ()=>{
      const mode = btn.dataset.nav;
      if (!mode) return;

      if (mode === "start") {
        closeAllPanels();
        return;
      }
      if (mode === "watch") {
        setNavActive("watch");
        showWatchPanel();
        return;
      }

      setNavActive(mode);
      showUiPanel(mode);
    });
  });
}

/* =========================================================
   BOOT
========================================================= */
function boot(){
  // close mine with button text
  const mineClose = $("mineCloseBtn");
  if (mineClose) mineClose.addEventListener("click", exitMineScene);

  resize();
  initLangSelector();

  loadWatchState();



  initNavAndPanels();
  refreshHUDFromState();

  // default
  setNavActive("start");
  closeAllPanels();

  // start loop
  startMainLoop();
}

(function QC_PATCH(){
  window.__qc_patch_v1 = window.__qc_patch_v1 || { applied:false };
  if (window.__qc_patch_v1.applied) return;
  window.__qc_patch_v1.applied = true;

  // ---------- tiny helpers ----------
  const fmt0 = (n) => (Number(n)||0).toLocaleString(undefined,{maximumFractionDigits:0});
  const nowISO = () => new Date().toISOString();
  const isBscAddr = (s) => /^0x[a-fA-F0-9]{40}$/.test(String(s||"").trim());

  // ---------- make sure drawParticles exists (in case cut/removed) ----------
  if (typeof window.drawParticles !== "function") {
    window.drawParticles = function drawParticles(){
      if (!state || !ctx) return;
      ctx.save();
      for(const p of state.particles || []){
        const a = clamp(p.life, 0, 1);
        ctx.globalAlpha = a;
        ctx.fillStyle = p.color || "rgba(56,189,248,0.9)";
        ctx.beginPath();
        ctx.arc(p.x, p.y, 2.2, 0, Math.PI*2);
        ctx.fill();
      }
      ctx.restore();
    };
  }

  // ---------- optional updateStars (loop calls it) ----------
  if (typeof window.updateStars !== "function") {
    window.updateStars = function updateStars(dt){
      if (!state || !state.stars) return;
      for(const s of state.stars){
        s.tw = (s.tw || Math.random()*Math.PI*2) + (s.speed||0.02) * dt * 6;
      }
    };
  }

  // =========================================================
  // TGWT LEDGER (LOCAL DEMO)
  // =========================================================










  function renderTGWTPanel(){window.QCAccount?.refresh().catch(()=>{});}

  function refreshTGWTPanelIfOpen(){
    if (currentUiMode === "tgwt") renderTGWTPanel();
  }
  window.refreshTGWTPanelIfOpen = refreshTGWTPanelIfOpen;

  // =========================================================
  // WATCH PANEL (DEMO TIMER + REWARD)
  // =========================================================
  function showWatchPanel(){
    const sec = $("panel-watch");
    if(!sec) return;
    // UI panel kapat
    if ($("uiPanel")) $("uiPanel").hidden = true;
    sec.classList.add("show");
    sec.setAttribute("aria-hidden","false");
    currentUiMode = "watch";
    setActiveNav("watch");
    refreshWatchUI();
  }
  function hideWatchPanel(toStart=true){
    const sec = $("panel-watch");
    if(!sec) return;
    sec.classList.remove("show");
    sec.setAttribute("aria-hidden","true");
    if (toStart){
      currentUiMode = "start";
      setActiveNav("start");
    }
  }
  window.showWatchPanel = showWatchPanel;
  window.hideWatchPanel = hideWatchPanel;

  function mmss(s){
    s = Math.max(0, Math.floor(Number(s)||0));
    const m = String(Math.floor(s/60)).padStart(2,"0");
    const r = String(s%60).padStart(2,"0");
    return `${m}:${r}`;
  }

  function refreshWatchUI(){
    setText("watchTitle", T("watch.title"));
    setText("watchDesc", T("watch.desc"));
    setText("watchStatusLabel", T("watch.statusLabel"));
    setText("watchRequiredLabel", T("watch.requiredTimeLabel"));
    setText("watchRewardLabel", T("watch.rewardLabel"));
    setText("watchHintText", T("watch.hint"));
    setText("watchVideoListTitle", T("watch.videoListTitle"));
    setText("watchVideoListDesc", T("watch.videoListDesc") || $("watchVideoListDesc")?.textContent || "");
    setText("watchDurationLabel", T("watch.durationLabel"));
    setText("watchDurationLabelB", T("watch.durationLabel"));
    setText("watchDurationLabel2", T("watch.durationLabel"));
    setText("watchVideoIntroTitle", T("watch.videoIntroTitle"));
    setText("watchVideoSpaceMinerTitle", T("watch.videoSpaceMinerTitle"));
    setText("watchStartBtn", T("watch.startBtn"));

    setText("watchRequiredTime", mmss(watchState.required || 30));
    setText("watchElapsed", mmss(watchState.elapsed || 0));

    const fill = $("watch-progress-fill");
    if(fill){
      const pct = clamp((watchState.elapsed||0) / (watchState.required||30), 0, 1) * 100;
      fill.style.width = pct + "%";
    }

    if(watchState.running){
      setText("watchStatusText", T("watch.statusRunning"));
    } else if(watchState.rewardGranted){
      setText("watchStatusText", T("watch.statusCompleted"));
    } else {
      setText("watchStatusText", T("watch.statusIdle"));
    }
  }
  window.refreshWatchUI = refreshWatchUI;

  function grantWatchRewardOnce(){
    watchState.rewardGranted = false;
    const line = document.getElementById("watchStatusText");
    if (line) line.textContent = "Provider verification required; a timer is not reward proof.";
  }

  function updateWatch(dt){/* Watch controller owns visual progress; provider evidence owns rewards. */}
  window.updateWatch = updateWatch;

  // =========================================================
  // UI NAV + PANELS
  // =========================================================
  function setActiveNav(key){
    document.querySelectorAll(".nav-btn").forEach(btn=>{
      btn.classList.toggle("active", btn.dataset.nav === key);
    });
  }
  window.setActiveNav = setActiveNav;

  function closeAllPanels(){
    const ui = $("uiPanel");
    if (ui) ui.hidden = true;
    ["walletPanel","topPanel","storePanel","tgwtPanel","socialPanel","settingsPanel"].forEach(id=>{
      const el = $(id); if(el) el.hidden = true;
    });
    hideWatchPanel(false);
    currentUiMode = "start";
    setActiveNav("start");
  }
  window.closeAllPanels = closeAllPanels;

  function openPanel(key){
    if (isMineSceneOpen()) return; // mine açıkken panel yok
    hideWatchPanel(false);

    const ui = $("uiPanel");
    if(!ui) return;

    ui.hidden = false;

    const map = {
      wallet:"walletPanel",
      top:"topPanel",
      store:"storePanel",
      tgwt:"tgwtPanel",
      social:"socialPanel",
      settings:"settingsPanel",
    };

    Object.keys(map).forEach(k=>{
      const el = $(map[k]);
      if(el) el.hidden = (k !== key);
    });

    currentUiMode = key;
    setActiveNav(key);

    setText("uiPanelTitle", T(`panelTitles.${key}`));
    setText("uiPanelBody",  T(`panelBodies.${key}`));

    if(key === "store") renderStore();
    if(key === "wallet") refreshWalletPanelIfOpen();
    if(key === "social") renderSocialLinks();
    if(key === "tgwt") renderTGWTPanel();
  }
  window.openPanel = openPanel;

  // =========================================================
  // WALLET: seed + metamask wiring
  // =========================================================



  function loadMeta(){
    try{
      const raw = localStorage.getItem(WALLET_METAMASK_KEY);
      if(!raw) return;
      const p = JSON.parse(raw);
      if(p && typeof p === "object"){
        walletState.metamaskAddress = p.addr || null;
        walletState.metamaskConnected = !!p.connected;
      }
    }catch{}
  }
  function saveMeta(){
    try{
      localStorage.setItem(WALLET_METAMASK_KEY, JSON.stringify({
        addr: walletState.metamaskAddress,
        connected: walletState.metamaskConnected
      }));
    }catch{}
  }




  function refreshWalletPanelIfOpen(){
    if(currentUiMode !== "wallet") return;
    // Labels (seed kısmı EN forced)
    setText("walletBalancesTitle", T("walletUI.headingBalances"));
    setText("walletQCLabel",      T("walletUI.labelGameQC"));
    setText("walletTGWTLabel",    T("walletUI.labelGameTGWT"));
    setText("walletLevelLabel",   T("walletUI.labelLevel"));

    setText("walletConnectionTitle", T("walletUI.headingConnection"));
    setText("walletConnectionSub",   T("walletUI.connectionSub"));

    setText("walletMetaTitle",       T("walletUI.metaTitle"));
    setText("walletMetaConnectLabel",T("walletUI.btnConnect"));

    // Seed section EN only
    setText("walletSeedTitle",       T("walletUI.seedTitle","en"));
    setText("btnGenerateSeed",       T("walletUI.btnGenerateSeed","en"));
    setText("btnCopySeed",           T("walletUI.btnCopySeed","en"));
    setText("walletSeedNote",        T("walletUI.seedWarning","en"));

    // Values
    setText("walletQCValue",   fmt0(state.resources));
    setText("walletTGWTValue", state.tgwtDisplay || "0");
    setText("walletLevelValue",fmt0(state.level));

    // Meta status pill
    const pillText = $("walletMetaStatusPillText");
    const dot = $("walletMetaDot");
    const metaLine = $("walletMetaStatus");
    const connected = !!walletState.metamaskConnected && !!walletState.metamaskAddress;

    if(dot){
      dot.style.background = connected ? "#22c55e" : "#f97316";
      dot.style.boxShadow = connected ? "0 0 8px rgba(34,197,94,.9)" : "0 0 8px rgba(249,115,22,.9)";
    }

    if(connected){
      if(pillText) pillText.textContent = `${T("walletUI.pillConnectedPrefix")} ${walletState.metamaskAddress.slice(0,6)}...${walletState.metamaskAddress.slice(-4)}`;
      if(metaLine){ metaLine.className="wallet-status ok"; metaLine.textContent = `${T("walletUI.statusConnectedPrefix")} ${walletState.metamaskAddress}`; }
    } else {
      if(pillText) pillText.textContent = T("walletUI.pillNotConnected");
      if(metaLine){
        const hasMeta = !!window.ethereum;
        metaLine.className = "wallet-status warn";
        metaLine.textContent = hasMeta ? T("walletUI.statusDisconnected") : T("walletUI.statusNoMeta");
      }
    }

    // Seed box
    const box = $("walletSeedBox");
    if(box){
      if(!walletState.seed){
        box.classList.add("seed-phrase-empty");
        box.textContent = T("walletUI.seedNotGenerated","en");
      } else {
        box.classList.remove("seed-phrase-empty");
        box.textContent = walletState.seedVisible ? walletState.seed : "•••• •••• •••• •••• •••• ••••";
      }
    }

    const toggleBtn = $("btnToggleSeed");
    if(toggleBtn){
      toggleBtn.textContent = walletState.seedVisible ? T("walletUI.btnHideSeed","en") : T("walletUI.btnShowSeed","en");
    }
  }
  window.refreshWalletPanelIfOpen = refreshWalletPanelIfOpen;

  async function connectMetaMask(){
    const line = $("walletMetaStatus");
    if(!window.ethereum){
      if(line){ line.className="wallet-status warn"; line.textContent = T("walletUI.statusNoMeta"); }
      return;
    }
    try{
      const accounts = await window.ethereum.request({ method:"eth_requestAccounts" });
      const addr = accounts && accounts[0] ? String(accounts[0]) : null;
      walletState.metamaskAddress = addr;
      walletState.metamaskConnected = !!addr;
      saveMeta();
      refreshWalletPanelIfOpen();
    }catch(err){
      if(line){ line.className="wallet-status warn"; line.textContent = `MetaMask error: ${err?.message || err}`; }
    }
  }





  // =========================================================
  // STORE RENDER (buy/equip)
  // =========================================================
  function renderStore(){
    const grid = $("storeGrid");
    if(!grid) return;

    setText("storePanelTitle", T("storeUI.heading"));
    setText("storePanelIntro", T("storeUI.intro"));

    const frag = document.createDocumentFragment();

    SHIP_CONFIG.forEach(cfg=>{
      const owned = isShipOwned(cfg.id);
      const active = shipState.activeId === cfg.id;

      const name = (cfg.name && (cfg.name[currentLang] || cfg.name.en)) ? (cfg.name[currentLang] || cfg.name.en) : (cfg.id);
      const desc = (cfg.description && (cfg.description[currentLang] || cfg.description.en)) ? (cfg.description[currentLang] || cfg.description.en) : "";

      const card = document.createElement("div");
      card.className = "store-card";

      const img = document.createElement("img");
      img.src = cfg.sprite;
      img.alt = name;

      const title = document.createElement("div");
      title.className = "store-card-title";
      title.textContent = name;

      const d = document.createElement("div");
      d.className = "store-card-desc";
      d.textContent = desc;

      const meta = document.createElement("div");
      meta.className = "store-card-meta";
      meta.innerHTML = `
        <span>${T("storeUI.tierLabel")}: <b>${cfg.tier||0}</b></span>
        <span>${T("storeUI.priceLabel")}: <b>${cfg.priceQC ? fmt0(cfg.priceQC) : T("storeUI.freeLabel")}</b></span>
      `;

      const btn = document.createElement("button");
      btn.className = "store-card-btn";

      if(owned){
        btn.textContent = active ? T("storeUI.activeButton") : T("storeUI.equipButton");
        btn.disabled = active;
        btn.addEventListener("click", ()=>{
          shipState.activeId = cfg.id;
          saveShipStateToStorage();
          renderStore();
          // Mine açıkken gemi görseli güncellensin
          const ms = $("mineShip");
          if(ms && isMineSceneOpen()) ms.src = cfg.sprite;
        });
      } else {
        btn.textContent = T("storeUI.buyButton");
        btn.addEventListener("click", ()=>{
          const price = Number(cfg.priceQC)||0;
          if(state.resources < price){
            alert(T("storeUI.notEnoughQC"));
            return;
          }
          state.resources -= price;
          shipState.owned[cfg.id] = true;
          shipState.activeId = cfg.id;
          saveShipStateToStorage();
          syncStateToCore();
          saveCoreStateToStorage();
          refreshHUDFromState();
          renderStore();
        });
      }

      if(owned){
        const badge = document.createElement("div");
        badge.className = "store-badge";
        badge.textContent = T("storeUI.ownedBadge");
        card.appendChild(badge);
      }
      if(active){
        const badge2 = document.createElement("div");
        badge2.className = "store-badge store-badge--active";
        badge2.textContent = T("storeUI.activeBadge");
        card.appendChild(badge2);
      }

      card.appendChild(img);
      card.appendChild(title);
      card.appendChild(d);
      card.appendChild(meta);
      card.appendChild(btn);

      frag.appendChild(card);
    });

    grid.innerHTML = "";
    grid.appendChild(frag);
  }
  window.renderStore = renderStore;

  // =========================================================
  // SOCIAL LINKS RENDER
  // =========================================================
  function renderSocialLinks(){
    setText("socialPanelTitle", T("socialUI.panelTitle"));
    setText("socialPanelIntro", T("socialUI.panelIntro"));
    setText("socialCardTitle",  T("socialUI.cardTitle"));
    setText("socialCardIntro",  T("socialUI.cardIntro"));
    setText("socialRewardNote", T("socialUI.rewardNote"));

    const ul = $("socialLinksList");
    if(!ul) return;
    ul.innerHTML = "";

    const order = ["website","x","x_tgwt","tgwt","telegram","tiktok","youtube"];
    order.forEach(k=>{
      const it = SOCIAL_LINKS[k];
      if(!it) return;
      const li = document.createElement("li");
      const a = document.createElement("a");
      a.href = it.url;
      a.target = "_blank";
      a.rel = "noopener";
      a.textContent = T(it.labelKey);
      li.appendChild(a);
      ul.appendChild(li);
    });
  }
  window.renderSocialLinks = renderSocialLinks;

// =========================================================
// MINE OVERLAY enter/exit + fallback FX (if missing)
// =========================================================

// Robust site list resolver (miningSites veya MINING_SITES hangisi varsa)
const __mineSites =
  (typeof miningSites !== "undefined" && Array.isArray(miningSites) && miningSites.length)
    ? miningSites
    : (typeof MINING_SITES !== "undefined" && Array.isArray(MINING_SITES) && MINING_SITES.length)
      ? MINING_SITES
      : [];

const __defaultMineSite = {
  id: "LUNA-01",
  code: "LUNA-01",
  name: "Mining Site",
  bg: "",
  target: { x: 0.62, y: 0.78 }
};

let __currentMineSite = __mineSites[0] || __defaultMineSite;

// Canvas size helper (kendi fallback'ı)
if (typeof window.ensureMineSparkCanvasSize !== "function") {
  window.ensureMineSparkCanvasSize = function ensureMineSparkCanvasSize() {
    const sc = $("mineSparkCanvas");
    const layer = $("mineFXLayer");
    if (!sc || !layer) return { w: 0, h: 0 };

    const r = layer.getBoundingClientRect();
    const dpr = window.devicePixelRatio || 1;

    sc.width = Math.max(1, Math.floor(r.width * dpr));
    sc.height = Math.max(1, Math.floor(r.height * dpr));
    sc.style.width = r.width + "px";
    sc.style.height = r.height + "px";

    const ctx = sc.getContext("2d");
    if (ctx) ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

    return { w: r.width, h: r.height };
  };
}

window.enterMiningSite=enterMiningSite;
window.exitMineScene=exitMineScene;
if (typeof window.enterMiningSite !== "function") {
  window.enterMiningSite = function enterMiningSite(id) {
    const site = (__mineSites.find(s => s.id === id) || __mineSites[0] || __defaultMineSite);
    __currentMineSite = site;
    window.__currentMineSite = site; // debug

    // state guard
    if (!window.state) window.state = {};
    if (!state.mining) state.mining = { activeIndex: 0, progress: 0, progressTarget: 6 };

    const idx = __mineSites.findIndex(s => s.id === site.id);
    state.mining.activeIndex = clamp(idx < 0 ? 0 : idx, 0, Math.max(0, __mineSites.length - 1));
    state.mining.progress = 0;

    const ms = $("mineScene");
    if (!ms) return;

    const bg = $("mineBg");
    if (bg) {
      if (site.bg) bg.style.backgroundImage = `url('${site.bg}')`;
      else bg.style.backgroundImage = "none";
    }

    setText("mineTitle", `${site.code || site.id} · ${site.name || ""}`.trim());
    setText("mineSub", T("mine.subtitle"));

    const ship = $("mineShip");
    if (ship) ship.src = getActiveShipConfig().sprite;

    ms.classList.remove("hidden");
    ms.classList.add("show");
    ms.setAttribute("aria-hidden", "false");
    document.body.classList.add("mine-open");

    ensureMineSparkCanvasSize();
    ms.classList.add("show-ship");
  };
}

if (typeof window.exitMineScene !== "function") {
  window.exitMineScene = function exitMineScene() {
    const ms = $("mineScene");
    if (!ms) return;

    ms.classList.remove("show-ship");
    ms.classList.remove("show");
    ms.classList.add("hidden");
    ms.setAttribute("aria-hidden", "true");
    document.body.classList.remove("mine-open");
  };
}

// Fallback FX only if missing
if (typeof window.spawnMineBeam !== "function") {
  window.spawnMineBeam = function spawnMineBeam() {
    const layer = $("mineFXLayer");
    const shipEl = $("mineShip");
    if (!layer || !shipEl) return;

    const r = layer.getBoundingClientRect();
    const sr = shipEl.getBoundingClientRect();

    const sx = (sr.left + sr.width * 0.58) - r.left;
    const sy = (sr.top + sr.height * 0.34) - r.top;

    const tx = (Number(__currentMineSite?.target?.x) || 0.62) * r.width;
    const ty = (Number(__currentMineSite?.target?.y) || 0.78) * r.height;

    const dx = tx - sx, dy = ty - sy;
    const len = Math.max(120, Math.hypot(dx, dy));
    const ang = Math.atan2(dy, dx) * 180 / Math.PI;

    const beam = document.createElement("div");
    beam.className = "mine-beam";
    beam.style.left = sx + "px";
    beam.style.top = sy + "px";
    beam.style.width = len + "px";
    beam.style.setProperty("--ang", ang + "deg");
    layer.appendChild(beam);

    const hit = document.createElement("div");
    hit.className = "mine-hit";
    hit.style.left = (tx - 8) + "px";
    hit.style.top = (ty - 8) + "px";
    layer.appendChild(hit);

    setTimeout(() => beam.remove(), 450);
    setTimeout(() => hit.remove(), 450);
  };
}

if (typeof window.tickMineFX !== "function") {
  const fx = { t: 0, sparks: [] };

  window.tickMineFX = function tickMineFX(dt) {
    // mine açık değilse boşuna çizme
    const ms = $("mineScene");
    if (!ms || ms.classList.contains("hidden")) return;

    // canvas ölçülerini garanti et
    const sc = $("mineSparkCanvas");
    const sctx = sc ? sc.getContext("2d") : null;
    if (!sctx) return;

    // Not: ensureMineSparkCanvasSize() canvas transform'u ayarlıyor
    // ama responsive resize olursa burada da yakalansın:
    if (sc.width === 0 || sc.height === 0) ensureMineSparkCanvasSize();

    // CSS px ölçüsü için layer rect'i kullan
    const layer = $("mineFXLayer");
    const r = layer ? layer.getBoundingClientRect() : { width: sc.clientWidth, height: sc.clientHeight };
    const w = r.width || sc.clientWidth || 1;
    const h = r.height || sc.clientHeight || 1;

    fx.t += dt;
    if (fx.t > 0.18) {
      fx.t = 0;
      spawnMineBeam();

      const tx = (Number(__currentMineSite?.target?.x) || 0.62) * w;
      const ty = (Number(__currentMineSite?.target?.y) || 0.78) * h;

      for (let i = 0; i < 8; i++) {
        fx.sparks.push({
          x: tx + (Math.random() - 0.5) * 18,
          y: ty + (Math.random() - 0.5) * 18,
          vx: (Math.random() - 0.5) * 120,
          vy: (Math.random() - 0.5) * 120,
          life: 0.35 + Math.random() * 0.25
        });
      }
    }

    // clear (CSS px space)
    sctx.clearRect(0, 0, w, h);

    const next = [];
    for (const p of fx.sparks) {
      p.life -= dt;
      p.x += p.vx * dt;
      p.y += p.vy * dt;
      p.vx *= Math.pow(0.90, dt * 60);
      p.vy *= Math.pow(0.90, dt * 60);

      if (p.life > 0) {
        next.push(p);
        const a = clamp(p.life / 0.6, 0, 1);
        sctx.globalAlpha = a;
        sctx.fillStyle = "rgba(34,197,94,1)";
        sctx.beginPath();
        sctx.arc(p.x, p.y, 2.2, 0, Math.PI * 2);
        sctx.fill();
      }
    }
    fx.sparks = next;
    sctx.globalAlpha = 1;
  };
}

  // =========================================================
  // APPLY LANG (override-safe)
  // =========================================================
  window.applyLang = function applyLang(lang){
    const safe = (I18N && I18N[lang]) ? lang : "en";
    currentLang = safe;
    try{ localStorage.setItem(LANG_STORAGE_KEY, currentLang); }catch{}

    document.querySelectorAll(".lang-btn").forEach(b=>{
      b.classList.toggle("active", b.dataset.lang === currentLang);
    });

    // HUD
    setText("hudLabelLevel", T("hudLevel"));
    setText("hudLabelQC",    T("hudQC"));
    setText("hudLabelTGWT",  T("hudTGWT"));
    setText("hudLabelXP",    T("hudXP"));
    setText("hintText",      T("hint"));

    // Bottom nav
    setText("navStartText",    T("navStart"));
    setText("navWalletText",   T("navWallet"));
    setText("navTopText",      T("navTop"));
    setText("navStoreText",    T("navStore"));
    setText("navTGWTText",     T("navTGWT"));
    setText("navSocialText",   T("navSocial"));
    setText("navWatchText",    T("navWatch"));
    setText("navSettingsText", T("navSettings"));

    // Mine overlay (subtitle + button)
    setText("mineSub",      T("mine.subtitle"));
    setText("mineCloseBtn", T("mine.backToMoon"));

    // Panel headers (when open)
    if(currentUiMode && ["wallet","top","store","tgwt","social","settings"].includes(currentUiMode)){
      setText("uiPanelTitle", T(`panelTitles.${currentUiMode}`));
      setText("uiPanelBody",  T(`panelBodies.${currentUiMode}`));
    }

    // Store / Social / TGWT / Wallet / Watch refresh
    if (typeof renderStore === "function") renderStore();
    if (typeof renderSocialLinks === "function") renderSocialLinks();
    refreshTGWTPanelIfOpen();
    refreshWalletPanelIfOpen();
    refreshWatchUI();
  };

  // =========================================================
  // UI WIRING (bind once)
  // =========================================================
  function wireUIOnce(){
    if (wireUIOnce.__done) return;
    wireUIOnce.__done = true;

    // lang buttons
    document.querySelectorAll(".lang-btn").forEach(btn=>{
      btn.addEventListener("click", ()=> applyLang(btn.dataset.lang));
    });

    // bottom nav
    document.querySelectorAll(".nav-btn").forEach(btn=>{
      btn.addEventListener("click", ()=>{
        const key = btn.dataset.nav;
        if(key === "start"){
          closeAllPanels();
          currentUiMode = "start";
          setActiveNav("start");
          return;
        }
        if(key === "watch"){
          showWatchPanel();
          return;
        }
        openPanel(key);
      });
    });

    // ui close
    $("uiCloseBtn")?.addEventListener("click", closeAllPanels);
    $("uiPanel")?.addEventListener("click", (e)=>{
      if(e.target && e.target.id === "uiPanel") closeAllPanels();
    });

    // watch close
    $("watchCloseBtn")?.addEventListener("click", ()=> hideWatchPanel(true));

    // watch start
    $("watchStartBtn")?.addEventListener("click", ()=>{
      if(watchState.rewardGranted){
        // yeni tur
        watchState.rewardGranted = false;
        watchState.elapsed = 0;
      }
      watchState.running = !watchState.running;
      refreshWatchUI();
    });

    // mine close
    $("mineCloseBtn")?.addEventListener("click", ()=> window.exitMineScene());

    // wallet buttons
    $("btnConnectMetaMask")?.addEventListener("click", connectMetaMask);

  }

  // =========================================================
  // BOOT
  // =========================================================
  function boot(){
    // saved lang
    try{
      const saved = localStorage.getItem(LANG_STORAGE_KEY);
      if(saved && I18N[saved]) currentLang = saved;
    }catch{}

    resize();
    wireUIOnce();

    // initial UI
    applyLang(currentLang || "en");
    renderSocialLinks();
    renderStore();
    refreshHUDFromState();

    // start loop (you already have startMainLoop in your file)
    if (typeof startMainLoop === "function") startMainLoop();

    // ensure TGWT header
    setText("tgwtTitle", T("panelTitles.tgwt"));
  }

  // script body sonunda olduğumuz için direkt çağırıyoruz
  boot();
})();
// ===============================
// MAIN RAF LOOP (minimal patch)
// ===============================
let __qc_lastTS = 0;
let __qc_raf = 0;

function __qc_startLoop() {
  if (__qc_raf) return;
  __qc_lastTS = performance.now();

  const step = (ts) => {
    // dt in seconds
    let dt = (ts - __qc_lastTS) / 1000;
    __qc_lastTS = ts;

    // clamp (stability when tab was hidden)
    if (!isFinite(dt) || dt < 0) dt = 0;
    if (dt > 0.08) dt = 0.08;

    // Update (only if functions exist)
    if (typeof updateShip === "function") updateShip(dt);
    if (typeof updateMeteors === "function") updateMeteors(dt);
    if (typeof updateParticles === "function") updateParticles(dt);
    if (typeof updateAutoMine === "function") updateAutoMine(dt);

    // Draw
    if (typeof drawScene === "function") drawScene(dt);

    // Mine overlay FX (when open)
    if (document.body.classList.contains("mine-open")) {
      if (typeof tickMineFX === "function") tickMineFX(dt);
    }

    __qc_raf = requestAnimationFrame(step);
  };

  __qc_raf = requestAnimationFrame(step);
}
window.addEventListener("load", () => {
  // ✅ İlk layout + canvas DPI + stars/meteors init
  try { resize(); } catch (e) { console.warn("resize() failed:", e); }
  startMainLoop();
}, { once: true });


// Explicit scene extension 13

/* SOCIAL_WATCH_CLAIM_UI_V1
   Adds backend-connected TGWT claim flow for:
   - QC Follow / Social channels
   - Watch & Earn completion
*/
(function SOCIAL_WATCH_CLAIM_UI_V1(){
  if (window.__socialWatchClaimV1Applied) return;
  window.__socialWatchClaimV1Applied = true;

  const API_BASE =
    window.QC_API_BASE ||
    window.API_BASE ||
    "";

  function $(id){ return document.getElementById(id); }

  function esc(s){
    return String(s ?? "")
      .replaceAll("&","&amp;")
      .replaceAll("<","&lt;")
      .replaceAll(">","&gt;")
      .replaceAll('"',"&quot;")
      .replaceAll("'","&#39;");
  }

  const playerId=window.QCGetPlayerId;

  async function apiGet(path){
    const res = await fetch(API_BASE + path, { cache:"no-store" });
    return await res.json();
  }

  async function apiPost(path, payload){
    const res = await fetch(API_BASE + path, {
      method:"POST",
      headers:{ "Content-Type":"application/json" },
      body: JSON.stringify(payload || {})
    });
    return await res.json();
  }

  function afterTGWTChange(){
    try { if (typeof refreshTGWTApiPanel === "function") refreshTGWTApiPanel(); } catch(e){}
    try { if (typeof syncTGWTHud === "function") syncTGWTHud(); } catch(e){}
  }

  async function renderSocialClaimUI(){
    const list = $("socialLinksList");
    if (!list) return;

    let links = null;
    try {
      const data = await apiGet("/api/social/links");
      links = data.links || null;
    } catch(e){}

    links = links || {
      website: "https://qcnetwork.ai/",
      x: "https://x.com/QuantumCoinQC",
      telegram: "https://web.telegram.org/a/#-1002870924021",
      tiktok: "https://www.tiktok.com/@quantumcoin21",
      youtube: "https://www.youtube.com/@QuantumCoinHQ"
    };

    const labels = {
      website: "Website",
      x: "X / Twitter",
      telegram: "Telegram",
      tiktok: "TikTok",
      youtube: "YouTube"
    };

    list.innerHTML = Object.entries(links).map(([key, url]) => `
      <li style="display:grid;grid-template-columns:1fr auto;gap:8px;align-items:center;">
        <a href="${esc(url)}" target="_blank" rel="noopener">${esc(labels[key] || key)} — Open</a>
        <button class="wallet-btn socialClaimBtn" data-platform="${esc(key)}" type="button" style="white-space:nowrap;">Request verification</button>
      </li>
    `).join("");

    const note = $("socialRewardNote");
    if (note) {
      note.textContent = "TGWT is credited only after the provider verifies the campaign action. Opening a link or clicking a claim button is not proof.";
    }

    list.querySelectorAll(".socialClaimBtn").forEach(btn => {
      btn.addEventListener("click", async () => {
        const platform = btn.getAttribute("data-platform") || "social";
        const old = btn.textContent;
        btn.disabled = true;
        btn.textContent = "Claiming...";

        try {
          const resp = await apiPost("/api/social/claim", {
            playerId: playerId(),
            platform,
            nonce: "ui_social_" + platform + "_" + Date.now() + "_" + Math.random().toString(16).slice(2)
          });

          if (!resp.ok) {
            btn.textContent = "Failed: " + (resp.error || "error");
            setTimeout(() => { btn.disabled = false; btn.textContent = old; }, 1800);
            return;
          }

          btn.textContent = "Verified reward recorded";
          afterTGWTChange();
        } catch(e) {
          btn.textContent = "Backend error";
          setTimeout(() => { btn.disabled = false; btn.textContent = old; }, 1800);
        }
      });
    });
  }

  function ensureWatchClaimButton(){
    const startBtn = $("watchStartBtn");
    if (!startBtn || $("watchClaimTGWTBtn")) return;

    const btn = document.createElement("button");
    btn.className = "btn-watch-start";
    btn.id = "watchClaimTGWTBtn";
    btn.type = "button";
    btn.style.marginLeft = "8px";
    btn.disabled = true;
    btn.textContent = "Request verified ad reward";
    startBtn.insertAdjacentElement("afterend", btn);

    const statusLine = document.createElement("div");
    statusLine.id = "watchClaimTGWTStatus";
    statusLine.className = "watch-progress-text";
    statusLine.textContent = "An ad timer alone earns no tokens. A configured provider must verify completion.";
    btn.insertAdjacentElement("afterend", statusLine);

    btn.addEventListener("click", async () => {
      const s = $("watchClaimTGWTStatus");
      btn.disabled = true;
      btn.textContent = "Claiming...";

      try {
        const resp = await apiPost("/api/watch/claim", {
          playerId: playerId(),
          videoId: "watch_demo",
          nonce: "ui_watch_" + Date.now() + "_" + Math.random().toString(16).slice(2)
        });

        if (!resp.ok) {
          btn.textContent = "Claim failed";
          if (s) s.textContent = "Watch claim failed: " + (resp.error || "unknown");
          setTimeout(updateWatchClaimButton, 1200);
          return;
        }

        btn.textContent = "Verified reward recorded";
        if (s) s.textContent = "The server recorded a verified campaign reward. A BSC claim is a separate wallet transaction.";
        afterTGWTChange();
      } catch(e) {
        btn.textContent = "Backend error";
        if (s) s.textContent = "Watch claim backend error.";
        setTimeout(updateWatchClaimButton, 1200);
      }
    });
  }

  function updateWatchClaimButton(){
    const btn = $("watchClaimTGWTBtn");
    const s = $("watchClaimTGWTStatus");
    if (!btn) return;

    let ready = false;
    try {
      if (typeof watchState !== "undefined") {
        ready = !!watchState.rewardGranted || Number(watchState.elapsed || 0) >= Number(watchState.required || 30);
      }
    } catch(e){}

    const statusText = $("watchStatusText")?.textContent || "";
    if (/ready|hazır|completed|tamamlandı|lista|就绪/i.test(statusText)) ready = true;

    if (ready) {
      btn.disabled = false;
      if (!btn.textContent.includes("Pending")) btn.textContent = "Request verified ad reward";
      if (s && !s.textContent.includes("pending")) s.textContent = "DEV TEST: timer completed. Final mode requires verified Google rewarded-ad completion.";
    } else {
      btn.disabled = true;
      if (!btn.textContent.includes("Pending")) btn.textContent = "Request verified ad reward";
    }
  }

  window.renderSocialClaimUI = renderSocialClaimUI;
  window.ensureWatchClaimButton = ensureWatchClaimButton;
  window.updateWatchClaimButton = updateWatchClaimButton;

  document.addEventListener("click", (ev) => {
    const socialNav = ev.target && ev.target.closest ? ev.target.closest('[data-nav="social"]') : null;
    const watchNav = ev.target && ev.target.closest ? ev.target.closest('[data-nav="watch"]') : null;

    if (socialNav) setTimeout(renderSocialClaimUI, 250);
    if (watchNav) {
      setTimeout(ensureWatchClaimButton, 250);
      setTimeout(updateWatchClaimButton, 500);
    }
  });

  document.addEventListener("DOMContentLoaded", () => {
    setTimeout(renderSocialClaimUI, 800);
    setTimeout(ensureWatchClaimButton, 900);
    setInterval(updateWatchClaimButton, 1000);
  });

  setTimeout(renderSocialClaimUI, 1200);
  setTimeout(ensureWatchClaimButton, 1200);
  setInterval(updateWatchClaimButton, 1000);
})();

// Explicit scene extension 14

/* WATCH_REWARDED_AD_POLICY_UI_V1
   Policy note:
   Watch & Earn final mode must use verified Google Rewarded Ads completion.
   Current timer/claim is DEV TEST only and should not be treated as final ad monetization.
*/
window.WATCH_REWARDED_AD_POLICY_V1 = {
  currentMode: "DEV_TEST_ONLY",
  finalMode: "GOOGLE_REWARDED_ADS_VERIFIED_COMPLETION",
  reward: "TGWT pending after verified rewarded ad",
  note: "No ordinary display ad incentivization; final reward requires official rewarded-ad completion verification."
};

// Explicit scene extension 16

/* WATCH_CLOSE_INSIDE_CARD_V1
   Physical DOM fix: #watchCloseBtn is moved into the first Watch card.
*/
(function WATCH_CLOSE_INSIDE_CARD_V1(){
  if (window.__watchCloseInsideCardV1Applied) return;
  window.__watchCloseInsideCardV1Applied = true;

  function moveWatchCloseInside(){
    const btn = document.getElementById("watchCloseBtn");
    const firstCard = document.querySelector("#panel-watch .watch-card");
    if (!btn || !firstCard) return;

    if (btn.parentElement !== firstCard) {
      firstCard.appendChild(btn);
    }

    btn.style.position = "absolute";
    btn.style.top = "14px";
    btn.style.right = "14px";
    btn.style.zIndex = "20";
  }

  document.addEventListener("DOMContentLoaded", () => {
    setTimeout(moveWatchCloseInside, 200);
    setTimeout(moveWatchCloseInside, 800);
  });

  document.addEventListener("click", (ev) => {
    const watchNav = ev.target && ev.target.closest ? ev.target.closest('[data-nav="watch"]') : null;
    if (watchNav) {
      setTimeout(moveWatchCloseInside, 200);
      setTimeout(moveWatchCloseInside, 800);
    }
  });

  setTimeout(moveWatchCloseInside, 600);
  setInterval(moveWatchCloseInside, 2000);

  window.moveWatchCloseInside = moveWatchCloseInside;
})();

// Explicit scene extension 18

/* XP_PROGRESS_CONTROL_V1
   Small runtime monitor for XP speed.
   XP is visual progression only; QC balance still changes only from real chain/block rewards.
*/
(function XP_PROGRESS_CONTROL_V1(){
  if (window.__xpProgressControlV1Applied) return;
  window.__xpProgressControlV1Applied = true;

  let lastXP = null;
  let lastTs = null;

  function $(id){ return document.getElementById(id); }

  function ensureXPDebugLine(){
    const xpBar = $("hudXpText");
    if (!xpBar || $("xpDebugLine")) return;

    const line = document.createElement("div");
    line.id = "xpDebugLine";
    line.textContent = "XP speed monitor active";
    xpBar.closest(".hud-item")?.appendChild(line);
  }

  function getXP(){
    try {
      if (window.state && typeof window.state.xpTotal !== "undefined") {
        return Number(window.state.xpTotal || 0);
      }
    } catch(e){}
    try {
      if (typeof state !== "undefined" && typeof state.xpTotal !== "undefined") {
        return Number(state.xpTotal || 0);
      }
    } catch(e){}
    return null;
  }

  function tick(){
    ensureXPDebugLine();

    const line = $("xpDebugLine");
    const xp = getXP();
    const now = Date.now();

    if (xp === null) {
      if (line) line.textContent = "XP monitor: state not ready";
      return;
    }

    if (lastXP === null) {
      lastXP = xp;
      lastTs = now;
      if (line) line.textContent = "XP monitor: baseline set";
      return;
    }

    const dt = Math.max(1, (now - lastTs) / 1000);
    const dxp = xp - lastXP;
    const rate = dxp / dt;

    if (line) {
      line.textContent = "XP rate ≈ " + rate.toFixed(3) + "/s";
    }

    lastXP = xp;
    lastTs = now;
  }

  window.qcXPMonitorTick = tick;

  document.addEventListener("DOMContentLoaded", () => {
    setTimeout(ensureXPDebugLine, 800);
    setInterval(tick, 5000);
  });

  setTimeout(ensureXPDebugLine, 1000);
  setInterval(tick, 5000);
})();

// Explicit scene extension 20

/* WATCH_START_FORCE_BIND_V2
   Final DEV TEST watch timer fix:
   - removes old Start Watching click listeners by cloning the button
   - progress starts only after user click
   - no auto-fill on panel open
   - final real mode remains Google Rewarded Ads verified completion
*/
(function WATCH_START_FORCE_BIND_V2(){
  if (window.__watchStartForceBindV2Applied) return;
  window.__watchStartForceBindV2Applied = true;

  const state = {
    running: false,
    elapsed: 0,
    required: 30,
    rewardReady: false,
    timer: null,
    lastTs: 0,
    installed: false
  };

  function $(id){ return document.getElementById(id); }

  function mmss(s){
    s = Math.max(0, Math.floor(Number(s) || 0));
    const m = String(Math.floor(s / 60)).padStart(2, "0");
    const r = String(s % 60).padStart(2, "0");
    return `${m}:${r}`;
  }

  function syncOldWatchState(){
    try {
      if (typeof watchState !== "undefined" && watchState) {
        watchState.running = state.running;
        watchState.elapsed = state.elapsed;
        watchState.required = state.required;
        watchState.rewardGranted = state.rewardReady;
      }
    } catch(e){}
    try {
      window.watchState = window.watchState || {};
      window.watchState.running = state.running;
      window.watchState.elapsed = state.elapsed;
      window.watchState.required = state.required;
      window.watchState.rewardGranted = state.rewardReady;
    } catch(e){}
  }

  function render(){
    const fill = $("watch-progress-fill");
    const elapsedEl = $("watchElapsed");
    const requiredEl = $("watchRequiredTime");
    const status = $("watchStatusText");
    const hint = $("watchHintText");
    const startBtn = $("watchStartBtn");
    const claimBtn = $("watchClaimTGWTBtn");
    const claimStatus = $("watchClaimTGWTStatus");

    const pct = state.required > 0 ? Math.max(0, Math.min(100, (state.elapsed / state.required) * 100)) : 0;

    if (fill) fill.style.width = pct.toFixed(1) + "%";
    if (elapsedEl) elapsedEl.textContent = mmss(state.elapsed);
    if (requiredEl) requiredEl.textContent = mmss(state.required);

    if (status) {
      if (state.rewardReady) status.textContent = "DEV TEST completed — reward ready";
      else if (state.running) status.textContent = "DEV TEST watching in progress...";
      else status.textContent = "Not started yet";
    }

    if (hint) {
      if (state.rewardReady) {
        hint.textContent = "DEV TEST completed. Final mode will require verified Google rewarded-ad completion.";
      } else if (state.running) {
        hint.textContent = "DEV TEST running. Final mode will run only after the user starts a Google rewarded ad.";
      } else {
        hint.textContent = "Press Start Watching. The progress bar will not fill automatically.";
      }
    }

    if (startBtn) {
      if (state.rewardReady) startBtn.textContent = "Restart Watch Test";
      else if (state.running) startBtn.textContent = "Pause Watching";
      else startBtn.textContent = "Start Watching";
      startBtn.disabled = false;
    }

    if (claimBtn) {
      if (state.rewardReady) {
        claimBtn.disabled = false;
        if (!claimBtn.textContent.includes("Pending")) {
          claimBtn.textContent = "Request verified ad reward";
        }
      } else {
        claimBtn.disabled = true;
        if (!claimBtn.textContent.includes("Pending")) {
          claimBtn.textContent = "Request verified ad reward";
        }
      }
    }

    if (claimStatus && !state.rewardReady) {
      claimStatus.textContent = "Start Watching first. The progress bar will not fill automatically.";
    }

    syncOldWatchState();
  }

  function stopTimer(){
    if (state.timer) clearInterval(state.timer);
    state.timer = null;
    state.lastTs = 0;
  }

  function tick(){
    if (!state.running) return;

    const now = performance.now();
    if (!state.lastTs) state.lastTs = now;

    const dt = Math.max(0, (now - state.lastTs) / 1000);
    state.lastTs = now;

    state.elapsed += dt;

    if (state.elapsed >= state.required) {
      state.elapsed = state.required;
      state.running = false;
      state.rewardReady = true;
      stopTimer();
    }

    render();
  }

  function startTimer(){
    stopTimer();
    state.lastTs = performance.now();
    state.timer = setInterval(tick, 250);
  }

  function onStartClick(ev){
    ev.preventDefault();
    ev.stopPropagation();
    if (typeof ev.stopImmediatePropagation === "function") ev.stopImmediatePropagation();

    if (state.rewardReady) {
      state.elapsed = 0;
      state.rewardReady = false;
      state.running = true;
      startTimer();
    } else if (state.running) {
      state.running = false;
      stopTimer();
    } else {
      state.running = true;
      startTimer();
    }

    render();
  }

  function install(){
    const oldBtn = $("watchStartBtn");
    if (!oldBtn) return;

    if (!oldBtn.__forceBindV2) {
      const newBtn = oldBtn.cloneNode(true);
      newBtn.id = "watchStartBtn";
      newBtn.__forceBindV2 = true;
      oldBtn.parentNode.replaceChild(newBtn, oldBtn);
      newBtn.addEventListener("click", onStartClick, true);
    }

    // Always reset on install/panel open unless the V2 timer is actively running.
    if (!state.running && !state.rewardReady) {
      state.elapsed = 0;
    }

    render();
  }

  function onWatchOpen(){
    setTimeout(install, 100);
    setTimeout(install, 400);
    setTimeout(render, 700);
  }

  window.watchStartForceBindV2Install = install;
  window.watchStartForceBindV2State = state;

  document.addEventListener("DOMContentLoaded", () => {
    setTimeout(install, 500);
    setTimeout(render, 900);
  });

  document.addEventListener("click", (ev) => {
    const watchNav = ev.target && ev.target.closest ? ev.target.closest('[data-nav="watch"]') : null;
    if (watchNav) onWatchOpen();
  });

  setTimeout(install, 1000);
  setInterval(render, 1000);
})();

// Explicit scene extension 21

/* SHIP_SALES_USDT_MODEL_V1
   GitHub-safe ship sales model.
   - Free starter ship
   - 5 paid ship configs
   - All committed/public USDT prices are 0
   - Real USDT values must come from private config/backend later
   - No direct QC minting, no block reward change, no supply impact
*/
(function SHIP_SALES_USDT_MODEL_V1(){
  if (window.__shipSalesUsdtModelV1Applied) return;
  window.__shipSalesUsdtModelV1Applied = true;

  const OWNED_KEY = "qc_ship_sales_owned_v1";
  const ACTIVE_KEY = "qc_ship_sales_active_v1";

  const PUBLIC_CATALOG = [
    {
      id: "starter",
      name: "Starter Ship",
      tier: "Free",
      currency: "FREE",
      priceUSDT: 0,
      priceVisible: true,
      xpMultiplier: 1.00,
      miningComfortMultiplier: 1.00,
      cosmeticFx: "starter_thruster",
      ownedByDefault: true,
      paymentRequired: false,
      finalPaymentNote: "Free starter ship. No payment required."
    },
    {
      id: "nova_scout",
      name: "Nova Scout",
      tier: "Basic",
      currency: "USDT",
      priceUSDT: 0,
      priceVisible: false,
      xpMultiplier: 1.03,
      miningComfortMultiplier: 1.00,
      cosmeticFx: "blue_thruster",
      ownedByDefault: false,
      paymentRequired: true,
      finalPaymentNote: "USDT payment verification later. Public Git price is 0."
    },
    {
      id: "lunar_miner",
      name: "Lunar Miner",
      tier: "Standard",
      currency: "USDT",
      priceUSDT: 0,
      priceVisible: false,
      xpMultiplier: 1.05,
      miningComfortMultiplier: 1.02,
      cosmeticFx: "lunar_drill_fx",
      ownedByDefault: false,
      paymentRequired: true,
      finalPaymentNote: "USDT payment verification later. Public Git price is 0."
    },
    {
      id: "quantum_falcon",
      name: "Quantum Falcon",
      tier: "Advanced",
      currency: "USDT",
      priceUSDT: 0,
      priceVisible: false,
      xpMultiplier: 1.08,
      miningComfortMultiplier: 1.03,
      cosmeticFx: "quantum_laser_fx",
      ownedByDefault: false,
      paymentRequired: true,
      finalPaymentNote: "USDT payment verification later. Public Git price is 0."
    },
    {
      id: "titan_freighter",
      name: "Titan Freighter",
      tier: "Pro",
      currency: "USDT",
      priceUSDT: 0,
      priceVisible: false,
      xpMultiplier: 1.12,
      miningComfortMultiplier: 1.04,
      cosmeticFx: "heavy_engine_fx",
      ownedByDefault: false,
      paymentRequired: true,
      finalPaymentNote: "USDT payment verification later. Public Git price is 0."
    },
    {
      id: "eclipse_legendary",
      name: "Eclipse Legendary",
      tier: "Legendary",
      currency: "USDT",
      priceUSDT: 0,
      priceVisible: false,
      xpMultiplier: 1.15,
      miningComfortMultiplier: 1.05,
      cosmeticFx: "legendary_eclipse_fx",
      ownedByDefault: false,
      paymentRequired: true,
      finalPaymentNote: "USDT payment verification later. Public Git price is 0."
    }
  ];

  function safeClone(x){
    return JSON.parse(JSON.stringify(x));
  }

  function readOwned(){
    try {
      const raw = localStorage.getItem(OWNED_KEY);
      const arr = raw ? JSON.parse(raw) : null;
      if (Array.isArray(arr) && arr.length) {
        if (!arr.includes("starter")) arr.unshift("starter");
        return Array.from(new Set(arr));
      }
    } catch(e){}
    return ["starter"];
  }

  function writeOwned(ids){
    try {
      const safe = Array.from(new Set(["starter"].concat(ids || [])));
      localStorage.setItem(OWNED_KEY, JSON.stringify(safe));
      return safe;
    } catch(e){
      return ["starter"];
    }
  }

  function getActiveId(){
    try {
      const active = localStorage.getItem(ACTIVE_KEY);
      if (active) return active;
    } catch(e){}
    return "starter";
  }

  function setActiveId(id){
    try { localStorage.setItem(ACTIVE_KEY, id); } catch(e){}
  }

  function getLocalPrice(id){
    try {
      const local = window.QC_SHIP_USDT_PRICES_LOCAL || {};
      if (Object.prototype.hasOwnProperty.call(local, id)) {
        const v = Number(local[id] || 0);
        return Number.isFinite(v) ? v : 0;
      }
    } catch(e){}
    return 0;
  }

  function normalizeShip(item){
    const x = safeClone(item);
    const localPrice = getLocalPrice(x.id);

    if (x.currency === "USDT") {
      x.priceUSDT = localPrice > 0 ? localPrice : 0;
      x.priceVisible = x.priceUSDT > 0;
      x.githubSafePrice = x.priceUSDT === 0;
      x.realPriceSource = x.priceVisible ? "LOCAL_PRIVATE_CONFIG" : "PRIVATE_BACKEND_LATER";
    } else {
      x.githubSafePrice = true;
      x.realPriceSource = "FREE";
    }

    const owned = readOwned();
    const activeId = getActiveId();

    x.owned = owned.includes(x.id) || x.ownedByDefault === true;
    x.active = x.id === activeId;
    x.priceLabel = x.currency === "FREE"
      ? "Free"
      : (x.priceVisible ? (x.priceUSDT + " USDT") : "USDT price hidden");

    return x;
  }

  function getCatalog(){
    return PUBLIC_CATALOG.map(normalizeShip);
  }

  function getShip(id){
    return getCatalog().find(x => x.id === id) || null;
  }

  function getOwnedShips(){
    return getCatalog().filter(x => x.owned);
  }

  function getActiveShip(){
    const active = getCatalog().find(x => x.active && x.owned);
    return active || getShip("starter");
  }

  function activateShip(id){
    const ship = getShip(id);
    if (!ship) {
      return { ok:false, error:"unknown_ship", shipId:id };
    }

    if (!ship.owned) {
      return {
        ok:false,
        error:"ship_not_owned",
        shipId:id,
        paymentRequired: ship.paymentRequired,
        priceUSDT: ship.priceUSDT,
        priceVisible: ship.priceVisible,
        message:"Ship is not owned. USDT payment verification will be connected later."
      };
    }

    setActiveId(id);

    try {
      if (typeof window.renderStoreGrid === "function") {
        window.renderStoreGrid();
      }
    } catch(e){}

    return {
      ok:true,
      activeShipId:id,
      activeShip:getActiveShip()
    };
  }

  function createOrderPreview(id){
    const ship = getShip(id);
    if (!ship) {
      return { ok:false, error:"unknown_ship", shipId:id };
    }

    if (ship.id === "starter") {
      return { ok:false, error:"starter_is_free", shipId:id };
    }

    return {
      ok:true,
      orderMode:"PREVIEW_ONLY",
      paymentEnabled:false,
      shipId:id,
      shipName:ship.name,
      currency:"USDT",
      priceUSDT:0,
      priceVisible:false,
      status:"PRICE_CONFIG_REQUIRED",
      message:"Order structure ready. Real USDT value and BEP-20 payment verification must be configured privately/backend later.",
      policy:{
        realPricesNotCommitted:true,
        publicGitPriceZero:true,
        noDirectQCReward:true,
        noBlockRewardChange:true,
        noSupplyImpact:true
      }
    };
  }

  function devGrantOwned(id){
    // DEV helper only. Do not use as final payment unlock.
    const ship = getShip(id);
    if (!ship) return { ok:false, error:"unknown_ship", shipId:id };

    const owned = readOwned();
    owned.push(id);
    writeOwned(owned);

    return { ok:true, mode:"DEV_GRANT_ONLY", owned:readOwned() };
  }

  window.SHIP_SALES_USDT_MODEL_V1 = {
    version:"SHIP_SALES_USDT_MODEL_V1",
    currency:"USDT",
    publicPrices:"ZERO_ONLY",
    finalPayment:"USDT_BEP20_VERIFICATION_LATER",
    noDirectQCReward:true,
    noBlockRewardChange:true,
    noSupplyImpact:true,
    getCatalog,
    getShip,
    getOwnedShips,
    getActiveShip,
    activateShip,
    createOrderPreview,
    devGrantOwned
  };

  // Optional compatibility helpers.
  window.getShipSalesCatalogV1 = getCatalog;
  window.getActiveShipSalesShipV1 = getActiveShip;
  window.activateShipSalesShipV1 = activateShip;
  window.createShipSalesOrderPreviewV1 = createOrderPreview;

  try {
    const active = getActiveId();
    const owned = readOwned();
    if (!owned.includes(active)) setActiveId("starter");
  } catch(e){}
})();

// Explicit scene extension 22

/* SHIP_SALES_STORE_UI_BRIDGE_V1
   Connects existing Store cards to SHIP_SALES_USDT_MODEL_V1 safely.
   - Prevents old QC-spend purchase flow for paid ships
   - Shows USDT order preview instead
   - Keeps public/Git prices hidden/0
   - Starter ship remains free/owned
*/
(function SHIP_SALES_STORE_UI_BRIDGE_V1(){
  if (window.__shipSalesStoreBridgeV1Applied) return;
  window.__shipSalesStoreBridgeV1Applied = true;

  const paidSalesIds = [
    "nova_scout",
    "lunar_miner",
    "quantum_falcon",
    "titan_freighter",
    "eclipse_legendary"
  ];

  function salesIdForConfig(cfg, index){
    if (!cfg) return "starter";
    if (cfg.id === "starter") return "starter";

    try {
      if (window.SHIP_SALES_USDT_MODEL_V1 && window.SHIP_SALES_USDT_MODEL_V1.getShip) {
        const direct = window.SHIP_SALES_USDT_MODEL_V1.getShip(cfg.id);
        if (direct) return cfg.id;
      }
    } catch(e){}

    return paidSalesIds[Math.max(0, index - 1)] || cfg.id;
  }

  function getSalesModel(shipId){
    try {
      if (window.SHIP_SALES_USDT_MODEL_V1 && window.SHIP_SALES_USDT_MODEL_V1.getShip) {
        return window.SHIP_SALES_USDT_MODEL_V1.getShip(shipId);
      }
    } catch(e){}
    return null;
  }

  function isOwnedCompat(cfg, salesId){
    try {
      if (typeof window.isShipOwned === "function" && window.isShipOwned(cfg.id)) return true;
    } catch(e){}

    try {
      const model = getSalesModel(salesId);
      if (model && model.owned) return true;
    } catch(e){}

    return cfg && cfg.id === "starter";
  }

  function isActiveCompat(cfg, salesId){
    try {
      if (window.shipState && window.shipState.activeId === cfg.id) return true;
    } catch(e){}

    try {
      const model = getSalesModel(salesId);
      if (model && model.active) return true;
    } catch(e){}

    return cfg && cfg.id === "starter";
  }

  function createOrderPreview(salesId){
    try {
      if (window.SHIP_SALES_USDT_MODEL_V1 && window.SHIP_SALES_USDT_MODEL_V1.createOrderPreview) {
        return window.SHIP_SALES_USDT_MODEL_V1.createOrderPreview(salesId);
      }
    } catch(e){}
    return {
      ok:false,
      error:"ship_sales_model_missing",
      message:"Ship sales model is not ready."
    };
  }

  function activateOwnedShip(cfg, salesId){
    try {
      if (window.SHIP_SALES_USDT_MODEL_V1 && window.SHIP_SALES_USDT_MODEL_V1.activateShip) {
        const r = window.SHIP_SALES_USDT_MODEL_V1.activateShip(salesId);
        if (r && r.ok === false) return r;
      }
    } catch(e){}

    try {
      if (window.shipState) {
        window.shipState.activeId = cfg.id;
        if (typeof window.saveShipStateToStorage === "function") window.saveShipStateToStorage();
      }
    } catch(e){}

    try {
      if (typeof window.renderStoreGrid === "function") window.renderStoreGrid();
      if (typeof window.renderStore === "function") window.renderStore();
    } catch(e){}

    return { ok:true, activeShipId: salesId };
  }

  function findStoreCards(){
    return Array.from(document.querySelectorAll(".store-card"));
  }

  function findActionButton(card){
    if (!card) return null;
    const buttons = Array.from(card.querySelectorAll("button"));
    return buttons[buttons.length - 1] || null;
  }

  function patchCard(card, cfg, index){
    if (!card || !cfg) return;

    const salesId = salesIdForConfig(cfg, index);
    const model = getSalesModel(salesId);
    const owned = isOwnedCompat(cfg, salesId);
    const active = isActiveCompat(cfg, salesId);
    const btn = findActionButton(card);

    if (!btn) return;

    // Update visible badge/price area if a price badge exists.
    const badges = Array.from(card.querySelectorAll(".store-badge"));
    const mainBadge = badges[0];

    if (mainBadge && !owned && salesId !== "starter") {
      mainBadge.textContent = "USDT price hidden";
    }

    let note = card.querySelector(".ship-sales-note-v1");
    if (!note) {
      note = document.createElement("div");
      note.className = "ship-sales-note-v1";
      note.style.marginTop = "6px";
      note.style.fontSize = "12px";
      note.style.opacity = ".72";
      note.style.lineHeight = "1.35";
      card.insertBefore(note, btn);
    }

    if (salesId === "starter") {
      note.textContent = "Free starter ship.";
      return;
    }

    if (owned) {
      note.textContent = "Owned ship. Activation does not change QC block reward or total supply.";

      if (!active && !btn.__shipSalesBridgeOwned) {
        const newBtn = btn.cloneNode(true);
        newBtn.__shipSalesBridgeOwned = true;
        newBtn.textContent = "Set active ship";
        newBtn.disabled = false;
        btn.parentNode.replaceChild(newBtn, btn);

        newBtn.addEventListener("click", function(ev){
          ev.preventDefault();
          ev.stopPropagation();
          if (typeof ev.stopImmediatePropagation === "function") ev.stopImmediatePropagation();

          const r = activateOwnedShip(cfg, salesId);
          if (!r.ok) alert(r.message || r.error || "Could not activate ship.");
        }, true);
      }

      return;
    }

    note.textContent = "USDT payment/order verification later. Public Git price is 0.";

    if (btn.__shipSalesBridgePaid) return;

    const newBtn = btn.cloneNode(true);
    newBtn.__shipSalesBridgePaid = true;
    newBtn.textContent = "USDT Order Preview";
    newBtn.disabled = false;

    btn.parentNode.replaceChild(newBtn, btn);

    newBtn.addEventListener("click", function(ev){
      ev.preventDefault();
      ev.stopPropagation();
      if (typeof ev.stopImmediatePropagation === "function") ev.stopImmediatePropagation();

      const preview = createOrderPreview(salesId);

      const name = model ? model.name : salesId;
      const msg =
        "Ship: " + name + "\n" +
        "Currency: USDT\n" +
        "Public Git price: 0\n" +
        "Status: " + (preview.status || "PRICE_CONFIG_REQUIRED") + "\n\n" +
        "Real USDT payment verification will be connected later.\n" +
        "No direct QC reward, no block reward change, no supply impact.";

      alert(msg);
    }, true);
  }

  function patchStoreUI(){
    if (!Array.isArray(window.SHIP_CONFIG)) return;

    const cards = findStoreCards();
    if (!cards.length) return;

    cards.forEach((card, index) => {
      const cfg = window.SHIP_CONFIG[index];
      patchCard(card, cfg, index);
    });
  }

  function schedulePatch(){
    setTimeout(patchStoreUI, 100);
    setTimeout(patchStoreUI, 400);
    setTimeout(patchStoreUI, 900);
  }

  const oldRenderStoreGrid = window.renderStoreGrid;
  if (typeof oldRenderStoreGrid === "function" && !oldRenderStoreGrid.__shipSalesBridgeWrapped) {
    const wrapped = function(){
      const r = oldRenderStoreGrid.apply(this, arguments);
      schedulePatch();
      return r;
    };
    wrapped.__shipSalesBridgeWrapped = true;
    window.renderStoreGrid = wrapped;
  }

  const oldRenderStore = window.renderStore;
  if (typeof oldRenderStore === "function" && !oldRenderStore.__shipSalesBridgeWrapped) {
    const wrapped2 = function(){
      const r = oldRenderStore.apply(this, arguments);
      schedulePatch();
      return r;
    };
    wrapped2.__shipSalesBridgeWrapped = true;
    window.renderStore = wrapped2;
  }

  document.addEventListener("click", (ev) => {
    const storeNav = ev.target && ev.target.closest ? ev.target.closest('[data-nav="store"]') : null;
    if (storeNav) schedulePatch();
  });

  document.addEventListener("DOMContentLoaded", () => {
    schedulePatch();
    setInterval(patchStoreUI, 2500);
  });

  setTimeout(schedulePatch, 1000);

  window.patchShipSalesStoreUIV1 = patchStoreUI;
})();

// Explicit scene extension 23

/* SETTINGS_FINAL_OVERVIEW_V1
   Final product-style Settings overview.
   Removes prototype/debug messaging from the visible Settings panel.
*/
(function SETTINGS_FINAL_OVERVIEW_V1(){
  if (window.__settingsFinalOverviewV1Applied) return;
  window.__settingsFinalOverviewV1Applied = true;

  function $(id){ return document.getElementById(id); }

  function finalSettingsHTML(){
    return `
      <div id="settingsFinalOverviewV1" class="wallet-card top-card">
        <h2>QUANTUMCOIN SPACE MINER</h2>
        <p>
          QuantumCoin Space Miner is a game-styled wallet and mining interface connected to the QuantumCoin ecosystem.
          Players explore lunar mining zones, start real mining sessions, track QC balance, manage rewards, and follow
          community reward systems from a single visual interface.
        </p>

        <div class="settings-final-grid-v1">
          <div class="settings-final-card-v1">
            <h3>What is QC?</h3>
            <p>
              QC is the native coin of the QuantumCoin network. QC balance changes only through valid blockchain actions:
              confirmed transfers, real mining rewards, or chain-synced wallet updates. Clicking a zone does not create fake QC.
            </p>
          </div>

          <div class="settings-final-card-v1">
            <h3>Gameplay</h3>
            <p>
              Choose a lunar mining zone, enter the mining scene, and start the mining flow. The game gives a visual
              mining experience while the real reward logic remains connected to the QuantumCoin chain/backend.
            </p>
          </div>

          <div class="settings-final-card-v1">
            <h3>Mining Logic</h3>
            <p>
              Mining points are entry zones, not instant reward buttons. QC is added only when a real block/reward flow
              is processed and the wallet balance is refreshed from the backend or QuantumCoin node.
            </p>
          </div>

          <div class="settings-final-card-v1">
            <h3>XP and Levels</h3>
            <p>
              XP is a gameplay progression layer. It helps show activity, long-term growth, and future unlocks.
              XP does not mint QC and does not change the blockchain reward rules.
            </p>
          </div>

          <div class="settings-final-card-v1">
            <h3>TGWT Rewards</h3>
            <p>
              TGWT is the community and game reward layer. Social tasks, Watch & Earn, and future events can add
              ledger-backed TGWT rewards. Final payout flow will use verified reward and withdrawal rules.
            </p>
          </div>

          <div class="settings-final-card-v1">
            <h3>Watch & Earn</h3>
            <p>
              Watch & Earn is designed for verified rewarded-ad completion. In development mode it uses a test timer,
              but final rewards must require a verified rewarded-ad flow before TGWT is credited.
            </p>
          </div>

          <div class="settings-final-card-v1">
            <h3>Wallet and Security</h3>
            <p>
              The wallet area is used to receive, send, and inspect QC balances. Seed phrase handling must remain secure:
              never share seed phrases or private keys with anyone.
            </p>
          </div>

          <div class="settings-final-card-v1">
            <h3>Premium Ships</h3>
            <p>
              Everyone starts with a free ship. Premium ships are planned as USDT-based purchases with cosmetic effects,
              badges, and controlled progression advantages. They do not directly mint QC or change block rewards.
            </p>
          </div>

          <div class="settings-final-card-v1">
            <h3>Network Alignment</h3>
            <p>
              Space Miner does not create a separate blockchain. It is a game-style interface that stays aligned with
              the real QuantumCoin wallet, mining, and reward rules.
            </p>
          </div>
        </div>

        <div class="settings-final-card-v1" style="margin-top:16px;">
          <h3>Quick Start</h3>
          <ol class="settings-final-steps-v1">
            <li>Select a mining zone on the Moon map.</li>
            <li>Enter the mining scene and start the real mining flow.</li>
            <li>When a block/reward is confirmed, refresh the wallet balance from the chain/backend.</li>
            <li>Use Wallet to receive, send, and check QC.</li>
            <li>Use TGWT, Social, and Watch & Earn panels to follow reward activity.</li>
            <li>Use Ship Store to manage the active ship and preview future premium ship purchases.</li>
          </ol>
        </div>

        <div class="settings-final-note-v1">
          <strong>Core rule:</strong> QC is the real network coin, XP is gameplay progression, and TGWT is the community/game
          reward layer. The game interface must never fake QC rewards or change QuantumCoin supply rules.
        </div>
      </div>
    `;
  }

  function renderFinalSettings(){
    const title = $("uiPanelTitle");
    const body = $("uiPanelBody");
    const inner = document.querySelector("#uiPanel .ui-panel-inner");

    if (!inner) return;

    if (title) title.textContent = "Settings";
    if (body) body.textContent = "Game overview, wallet basics, reward systems and gameplay logic.";

    Array.from(inner.children).forEach(ch => {
      if (ch.id === "uiCloseBtn" || ch.id === "uiPanelTitle" || ch.id === "uiPanelBody") return;
      if (ch.id === "settingsFinalOverviewV1") return;
      ch.style.display = "none";
      ch.setAttribute("data-settings-final-hidden", "1");
    });

    let mount = $("settingsFinalOverviewV1");
    if (!mount) {
      const wrap = document.createElement("div");
      wrap.innerHTML = finalSettingsHTML();
      mount = wrap.firstElementChild;
      inner.appendChild(mount);
    } else {
      mount.style.display = "";
    }
  }

  function restoreNonSettings(){
    const mount = $("settingsFinalOverviewV1");
    if (mount) mount.style.display = "none";

    document.querySelectorAll("[data-settings-final-hidden='1']").forEach(el => {
      el.style.display = "";
      el.removeAttribute("data-settings-final-hidden");
    });
  }

  const oldOpenPanel = window.openPanel;
  if (typeof oldOpenPanel === "function" && !oldOpenPanel.__settingsFinalWrapped) {
    const wrapped = function(key){
      const r = oldOpenPanel.apply(this, arguments);
      if (key === "settings") setTimeout(renderFinalSettings, 80);
      else restoreNonSettings();
      return r;
    };
    wrapped.__settingsFinalWrapped = true;
    window.openPanel = wrapped;
  }

  const oldShowUiPanel = window.showUiPanel;
  if (typeof oldShowUiPanel === "function" && !oldShowUiPanel.__settingsFinalWrapped) {
    const wrapped2 = function(key){
      const r = oldShowUiPanel.apply(this, arguments);
      if (key === "settings") setTimeout(renderFinalSettings, 80);
      else restoreNonSettings();
      return r;
    };
    wrapped2.__settingsFinalWrapped = true;
    window.showUiPanel = wrapped2;
  }

  document.addEventListener("click", (ev) => {
    const settingsNav = ev.target && ev.target.closest ? ev.target.closest('[data-nav="settings"]') : null;
    if (settingsNav) {
      setTimeout(renderFinalSettings, 150);
      setTimeout(renderFinalSettings, 500);
    } else {
      const nav = ev.target && ev.target.closest ? ev.target.closest(".nav-btn") : null;
      if (nav && nav.dataset && nav.dataset.nav !== "settings") {
        restoreNonSettings();
      }
    }
  });

  window.renderFinalSettingsOverviewV1 = renderFinalSettings;
})();

// Explicit scene extension 24

/* SHIP_STORE_FINAL_USDT_UI_V2
   Final public Store UI layer:
   - hides committed prices for paid ships
   - replaces Buy ship with USDT Order Preview
   - prevents old QC spend flow for paid ships
   - starter remains free/owned/active
*/
(function SHIP_STORE_FINAL_USDT_UI_V2(){
  if (window.__shipStoreFinalUsdtUiV2Applied) return;
  window.__shipStoreFinalUsdtUiV2Applied = true;

  const saleMapByIndex = [
    "starter",
    "nova_scout",
    "lunar_miner",
    "quantum_falcon",
    "titan_freighter",
    "eclipse_legendary"
  ];

  function cfgList(){
    try {
      if (typeof SHIP_CONFIG !== "undefined" && Array.isArray(SHIP_CONFIG)) return SHIP_CONFIG;
    } catch(e){}
    if (Array.isArray(window.SHIP_CONFIG)) return window.SHIP_CONFIG;
    return [];
  }

  function getSales(shipId){
    try {
      if (window.SHIP_SALES_USDT_MODEL_V1 && window.SHIP_SALES_USDT_MODEL_V1.getShip) {
        return window.SHIP_SALES_USDT_MODEL_V1.getShip(shipId);
      }
    } catch(e){}
    return null;
  }

  function preview(shipId){
    try {
      if (window.SHIP_SALES_USDT_MODEL_V1 && window.SHIP_SALES_USDT_MODEL_V1.createOrderPreview) {
        return window.SHIP_SALES_USDT_MODEL_V1.createOrderPreview(shipId);
      }
    } catch(e){}
    return { status:"PRICE_CONFIG_REQUIRED" };
  }

  function storeCards(){
    let cards = Array.from(document.querySelectorAll(".store-card"));
    if (cards.length) return cards;

    cards = Array.from(document.querySelectorAll(".ship-card"));
    if (cards.length) return cards;

    const maybeGrids = Array.from(document.querySelectorAll("[id*='store'], [class*='store']"));
    for (const grid of maybeGrids) {
      const children = Array.from(grid.children || []).filter(x => x.querySelector && x.querySelector("button"));
      if (children.length >= 2) return children;
    }

    return [];
  }

  function replacePriceText(card, isStarter){
    if (!card) return;

    const all = Array.from(card.querySelectorAll("*"));

    for (const el of all) {
      const t = (el.textContent || "").trim();

      if (!t) continue;

      if (isStarter) {
        if (/Price\s*:/i.test(t)) {
          el.textContent = "Price: Free";
        }
        continue;
      }

      if (/Price\s*:/i.test(t) || /10\.000|20\.000|30\.000|40\.000|50\.000/i.test(t)) {
        el.textContent = "Price: USDT hidden";
      }

      if (/Not enough QC/i.test(t)) {
        el.textContent = "USDT payment verification later.";
      }
    }
  }

  function ensureNote(card, text){
    let note = card.querySelector(".ship-store-final-note-v2");
    if (!note) {
      note = document.createElement("div");
      note.className = "ship-store-final-note-v2";
      note.style.marginTop = "8px";
      note.style.fontSize = "12px";
      note.style.lineHeight = "1.35";
      note.style.opacity = ".72";

      const btn = Array.from(card.querySelectorAll("button")).pop();
      if (btn && btn.parentNode === card) card.insertBefore(note, btn);
      else card.appendChild(note);
    }
    note.textContent = text;
  }

  function replaceButton(card, salesId, isStarter){
    const buttons = Array.from(card.querySelectorAll("button"));
    const btn = buttons[buttons.length - 1];
    if (!btn) return;

    if (isStarter) {
      btn.textContent = "Active ship";
      btn.disabled = true;
      return;
    }

    if (btn.__shipStoreFinalV2) {
      btn.textContent = "USDT Order Preview";
      btn.disabled = false;
      return;
    }

    const newBtn = btn.cloneNode(true);
    newBtn.__shipStoreFinalV2 = true;
    newBtn.textContent = "USDT Order Preview";
    newBtn.disabled = false;

    btn.parentNode.replaceChild(newBtn, btn);

    newBtn.addEventListener("click", function(ev){
      ev.preventDefault();
      ev.stopPropagation();
      if (typeof ev.stopImmediatePropagation === "function") ev.stopImmediatePropagation();

      const s = getSales(salesId);
      const p = preview(salesId);
      const name = (s && s.name) ? s.name : salesId;

      alert(
        "Ship: " + name + "\n" +
        "Currency: USDT\n" +
        "Public Git price: 0\n" +
        "Visible price: hidden\n" +
        "Status: " + (p.status || "PRICE_CONFIG_REQUIRED") + "\n\n" +
        "Final payment flow: USDT / BEP-20 verification later.\n" +
        "This preview does not spend QC and does not unlock the ship yet."
      );
    }, true);
  }

  function patchCards(){
    const cfgs = cfgList();
    const cards = storeCards();
    if (!cards.length) return;

    cards.forEach((card, index) => {
      const salesId = saleMapByIndex[index] || "paid_ship_" + index;
      const isStarter = salesId === "starter";

      replacePriceText(card, isStarter);

      if (isStarter) {
        ensureNote(card, "Free starter ship. Included for every player.");
      } else {
        ensureNote(card, "USDT payment/order verification later. Public Git price is 0.");
      }

      replaceButton(card, salesId, isStarter);

      if (!isStarter) {
        const badges = Array.from(card.querySelectorAll(".store-badge"));
        if (badges[0] && /Price|10\.000|20\.000|30\.000|40\.000|50\.000/i.test(badges[0].textContent || "")) {
          badges[0].textContent = "USDT price hidden";
        }
      }
    });
  }

  function schedule(){
    setTimeout(patchCards, 100);
    setTimeout(patchCards, 400);
    setTimeout(patchCards, 900);
    setTimeout(patchCards, 1500);
  }

  const oldRenderStoreGrid = window.renderStoreGrid;
  if (typeof oldRenderStoreGrid === "function" && !oldRenderStoreGrid.__shipStoreFinalV2Wrapped) {
    const wrapped = function(){
      const r = oldRenderStoreGrid.apply(this, arguments);
      schedule();
      return r;
    };
    wrapped.__shipStoreFinalV2Wrapped = true;
    window.renderStoreGrid = wrapped;
  }

  const oldRenderStore = window.renderStore;
  if (typeof oldRenderStore === "function" && !oldRenderStore.__shipStoreFinalV2Wrapped) {
    const wrapped2 = function(){
      const r = oldRenderStore.apply(this, arguments);
      schedule();
      return r;
    };
    wrapped2.__shipStoreFinalV2Wrapped = true;
    window.renderStore = wrapped2;
  }

  document.addEventListener("click", (ev) => {
    const storeNav = ev.target && ev.target.closest ? ev.target.closest('[data-nav="store"]') : null;
    if (storeNav) schedule();
  });

  document.addEventListener("DOMContentLoaded", () => {
    schedule();
    setInterval(patchCards, 2000);
  });

  setTimeout(schedule, 1000);

  window.patchShipStoreFinalUsdtUiV2 = patchCards;
})();

Object.defineProperty(window,"QCScene",{value:Object.freeze({applyAccount(wallet,ledger){state.resources=Number(wallet.qc_balance||0);state.tgwt=Number(ledger?.balances?.verified_u||0)/1000000;state.tgwtDisplay=window.QCAccount.units(ledger?.balances?.verified_u);state.xpTotal=Number(wallet.xpTotal||0);state.level=Number(wallet.level||1);refreshHUDFromState();}})});

})();
