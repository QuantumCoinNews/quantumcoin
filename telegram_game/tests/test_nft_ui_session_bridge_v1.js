const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

async function main() {
  const htmlPath = path.resolve(
    __dirname, "../telegram_game/dev_ui/mining_controller.js"
  );
  const html = fs.readFileSync(htmlPath, "utf8");
  const start = html.indexOf("  async function qcStartMiningSession(siteId){");
  const end = html.indexOf("  window.qcStartMiningSession=qcStartMiningSession;", start);
  assert.ok(start >= 0 && end > start, "Mining UI function boundaries changed");

  const calls = [];
  const context = {
    window: {
      Telegram: { WebApp: { initData: "signed-init-data" } },
      QC_TG_IDENTITY: { ok: true, playerId: "tg_123456" },
    },
    console: { log() {}, warn() {} },
    Date,
    getPlayerId: () => "tg_123456",
    zoneCodeFromId: () => "LUNA-01",
    getActiveShipId: () => "starter",
    qcApiPost: async (endpoint, body) => {
      calls.push({ endpoint, body });
      if (endpoint === "/api/v1/nft/monthly/session/start") {
        return { ok: true, sessionId: "nfts_test_session" };
      }
      if (endpoint === "/api/mine/finish") {
        return { ok: false, error: "no_block_found" };
      }
      return { ok: true, sessionId: "game_server_session" };
    },
  };

  vm.createContext(context);
  vm.runInContext(
    "let activeMiningSession = null;\n" + html.slice(start, end),
    context
  );
  await vm.runInContext("qcStartMiningSession(1)", context);
  await vm.runInContext('qcFinishMiningSession("exitMineScene")', context);

  const endpoints = calls.map((entry) => entry.endpoint);
  const expected = [
    "/api/v1/nft/monthly/session/start",
    "/api/mine/start",
    "/api/mine/finish",
    "/api/v1/nft/monthly/session/complete",
  ];
  if (assert.deepEqual && endpoints.join("|") ===
      ["/api/mine/start", "/api/mine/finish"].join("|")) {
    console.log("EXPECTED_RED: UI does not connect signed NFT session to game");
    process.exitCode = 42;
    return;
  }
  assert.deepEqual(endpoints, expected);
  assert.equal(calls[1].body.playerId, "tg_123456");
  assert.equal(calls[1].body.initData, "signed-init-data");
  assert.equal(calls[1].body.nftSessionId, "nfts_test_session");
  assert.equal(calls[2].body.sessionId, "game_server_session");
  assert.equal(calls[3].body.sessionId, "nfts_test_session");
  assert.equal(calls[3].body.initData, "signed-init-data");

  calls.length = 0;
  context.window.Telegram.WebApp.initData = "";
  context.window.QC_TG_IDENTITY = { ok: false, playerId: "tg_123456" };
  await assert.rejects(vm.runInContext("qcStartMiningSession(1)", context), /telegram_session_required/);
  assert.equal(calls.length, 0, "Unverified identity cannot open mining or NFT sessions");

  console.log("PASS: UI sends signed NFT session through real game flow");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
