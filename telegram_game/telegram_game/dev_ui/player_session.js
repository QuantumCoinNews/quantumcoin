/* Canonical player identity. Credentials stay in memory and go only to this origin. */
(function () {
  "use strict";
  if (window.QCPlayer) return;
  const originalFetch = window.fetch.bind(window);
  let credential = "";
  let identity = Object.freeze({ ok: false, checked: false, playerId: "", source: "telegram_required" });
  function playerId() { return identity.ok ? identity.playerId : ""; }
  Object.defineProperties(window, {
    QC_TG_IDENTITY: { get: () => identity }, QC_PLAYER_ID: { get: playerId },
    QCGetPlayerId: { value: playerId }, getPlayerId: { value: playerId },
    QCGetEffectivePlayerId: { value: playerId }
  });

  function locked(error) {
    credential = "";
    identity = Object.freeze({ ok: false, checked: true, playerId: "", error, source: "telegram_required" });
    const show = function () {
      if (document.getElementById("telegram-identity-required")) return;
      const notice = document.createElement("div");
      notice.id = "telegram-identity-required";
      notice.setAttribute("role", "alert");
      notice.style.cssText = "position:fixed;inset:0;z-index:2147483647;background:#020617ee;color:#fff;display:grid;place-items:center;padding:24px;text-align:center;font:18px sans-serif";
      notice.textContent = "Telegram oturumu doğrulanamadı. Oyunu Telegram içinden yeniden açın. / Open the game again from Telegram to sign in.";
      document.body.appendChild(notice);
    };
    if (document.body) show(); else document.addEventListener("DOMContentLoaded", show, { once: true });
    return window.QC_TG_IDENTITY;
  }

  async function verify() {
    const tg = window.Telegram && window.Telegram.WebApp;
    if (!tg || !tg.initData) return locked("telegram_init_data_required");
    try {
      const response = await originalFetch("/api/telegram/identity/verify", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ initData: String(tg.initData) }), cache: "no-store",
        credentials: "omit", redirect: "error"
      });
      const data = await response.json();
      if (!response.ok || !data.ok || !data.accessToken || !data.playerId) {
        return locked("telegram_identity_invalid");
      }
      credential = data.accessToken;
      identity = Object.freeze({ ok: true, checked: true, playerId: data.playerId,
        telegramUser: data.telegramUser, expiresAt: data.expiresAt, source: "server_verified_telegram_session" });
      return window.QC_TG_IDENTITY;
    } catch (_) {
      return locked("telegram_identity_unavailable");
    }
  }

  const ready = verify();
  Object.defineProperty(window, "QCPlayer", { value: Object.freeze({ id: playerId, ready }) });
  window.QC_TELEGRAM_IDENTITY_READY = ready;
  // Reopening the WebApp obtains fresh Telegram data; do not silently keep playing
  // under a stored player ID after authentication fails or a token expires.
  window.QCVerifyTelegramIdentity = function () { return ready; };
  window.fetch = async function (input, options) {
    const url = new URL(typeof input === "string" || input instanceof URL ? input : input.url, window.location.href);
    if (!url.pathname.startsWith("/api/")) return originalFetch(input, options);
    if (url.origin !== window.location.origin) throw new Error("game_api_origin_mismatch");
    const identity = await ready;
    if (!identity.ok || !credential) throw new Error("telegram_session_required");
    const headers = new Headers(input instanceof Request ? input.headers : undefined);
    new Headers((options || {}).headers).forEach(function (value, key) { headers.set(key, value); });
    headers.set("Authorization", "Bearer " + credential);
    // Preserve the original body verbatim, including signed JSON strings/int64.
    const response = await originalFetch(input, { ...options, headers, credentials: "omit", redirect: "error" });
    if (response.status === 401) locked("telegram_session_expired");
    return response;
  };
})();
