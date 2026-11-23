"""Isolated HTTP identity, canonical runtime and single-use reward contracts."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile

if os.environ.get("QC_PLAYER_RUNTIME_TEST_CHILD") != "1":
    with tempfile.TemporaryDirectory(prefix="qc-player-runtime-") as directory:
        env = os.environ.copy()
        env.update(QC_PLAYER_RUNTIME_TEST_CHILD="1", TELEGRAM_BOT_TOKEN="test-only-runtime-bot",
                   TELEGRAM_GAME_ADMIN_TOKEN="test-only-runtime-operator", QC_GAME_DB=str(Path(directory)/"canonical.sqlite3"),
                   QC_API_BASE="http://127.0.0.1:1", QC_API_TOKEN="test-only-unreachable-node")
        raise SystemExit(subprocess.call([sys.executable, str(Path(__file__).resolve())], cwd=directory, env=env))

import concurrent.futures
from contextlib import closing
import hashlib
import hmac
import json
import sqlite3
import time
import unittest
from urllib.parse import urlencode
from unittest.mock import patch
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from fastapi.testclient import TestClient
from telegram_game import dev_api as api, runtime, api_server
from telegram_game import player_auth, mining_sessions


def signed_data(uid=9001, auth_date=None, user=None):
    data = {"auth_date": str(int(time.time()) if auth_date is None else auth_date),
            "user": json.dumps({"id": uid} if user is None else user, separators=(",", ":"))}
    check = "\n".join(key+"="+data[key] for key in sorted(data))
    secret = hmac.new(b"WebAppData", os.environ["TELEGRAM_BOT_TOKEN"].encode(), hashlib.sha256).digest()
    data["hash"] = hmac.new(secret, check.encode(), hashlib.sha256).hexdigest()
    return urlencode(data)


class RuntimeBoundaryTests(unittest.TestCase):
    def setUp(self):
        api._QC_ACTION_BURST_EVENTS_V1.clear()
        with closing(api._db()) as conn, conn:
            conn.execute("DELETE FROM request_limits_v1")
        self.client = TestClient(runtime.app)
        self.credentials = player_auth.issue_session(api._db, "tg_9001", api._telegram_bot_token(), int(time.time()))
        self.headers = {"Authorization": "Bearer " + self.credentials["accessToken"]}

    def seed_player(self, uid="tg_9001"):
        conn = api._db()
        try:
            api._touch_player(conn, uid, None)
            api._ensure_qc_address_column(conn)
            conn.execute("UPDATE players SET qc_address=? WHERE player_id=?", ("QC_test_"+uid, uid))
            conn.execute("DELETE FROM game_pow_work_v1 WHERE player_id=?", (uid,))
            conn.execute("DELETE FROM mining_sessions WHERE player_id=?", (uid,))
            conn.execute("DELETE FROM game_mining_runs_v1 WHERE player_id=?", (uid,))
            conn.execute("DELETE FROM reward_records WHERE player_id=?", (uid,))
            conn.execute("UPDATE players SET total_qc=0,total_xp=0,qc_atoms=0,xp_units=0,sessions=0 WHERE player_id=?", (uid,))
            conn.commit()
        finally:
            conn.close()

    def opened(self):
        self.seed_player()
        result = self.client.post("/api/mine/start", headers=self.headers, json={"playerId":"tg_9001","siteId":1})
        self.assertEqual(result.status_code, 200, result.text)
        self.assertTrue(result.json()["ok"], result.text)
        return result.json()["sessionId"]

    def age_run(self, session_id, seconds=65):
        with closing(sqlite3.connect(api.APP_DB)) as conn, conn:
            conn.execute("UPDATE game_mining_runs_v1 SET started_at=?,expires_at=? WHERE session_id=?",
                         (int(time.time())-seconds, int(time.time())+3600, session_id))

    def test_entrypoints_share_one_app_and_no_independent_store(self):
        self.assertIs(runtime.app, api.app)
        self.assertIs(api_server.app, api.app)
        from telegram_game.database.store import Store
        from telegram_game.handlers import tgwt_router, withdraw_router
        with self.assertRaisesRegex(RuntimeError, "retired"):
            Store()
        self.assertEqual(tgwt_router.routes + withdraw_router.routes, [])
        self.assertEqual(api._qc_idem_db_path_v1(), api.APP_DB)
        self.assertEqual(api._qc_conflict_db_path_v1(), api.APP_DB)
        self.assertEqual(api._qc_rate_limit_db_path_v1(), api.APP_DB)

    def test_every_private_route_denies_before_state_or_network(self):
        checked = 0
        with patch.object(api, "_db", side_effect=AssertionError("unauthorized DB access")):
            for route in api.app.routes:
                path = getattr(route, "path", "")
                if not (path.startswith("/api/") or path == "/admin/settlement"):
                    continue
                for method in sorted(getattr(route, "methods", [])):
                    if method in {"HEAD", "OPTIONS"} or (method, path) in player_auth.PUBLIC:
                        continue
                    response = self.client.request(method, path, json={})
                    self.assertEqual(response.status_code, 401, (method, path, response.text))
                    checked += 1
        self.assertGreater(checked, 60)

    def test_valid_login_issues_hashed_persistent_credential_and_logout(self):
        response = self.client.post("/api/telegram/identity/verify", json={"initData":signed_data()})
        self.assertEqual(response.status_code, 200, response.text)
        data = response.json()
        token = data["accessToken"]
        self.assertEqual(data["playerId"], "tg_9001")
        self.assertEqual(response.headers["cache-control"], "no-store")
        with closing(sqlite3.connect(api.APP_DB)) as conn, conn:
            rows = conn.execute("SELECT * FROM player_auth_sessions_v1").fetchall()
        self.assertNotIn(token, repr(rows))
        # A fresh connection/process uses the persisted hash, not an in-memory map.
        self.assertEqual(player_auth.authenticate(api._db, token, api._telegram_bot_token(), int(time.time()))[0], "tg_9001")
        headers = {"Authorization":"Bearer "+token}
        self.assertEqual(self.client.get("/api/settings?playerId=tg_9001", headers=headers).status_code, 200)
        self.assertTrue(self.client.post("/api/telegram/identity/logout", headers=headers).json()["ok"])
        self.assertEqual(self.client.get("/api/settings?playerId=tg_9001", headers=headers).status_code, 401)

    def test_bad_expired_future_duplicate_and_wrong_user_signatures_fail(self):
        bad = ["user=bad&hash=bad", signed_data(auth_date=int(time.time())-901),
               signed_data(auth_date=int(time.time())+120), signed_data()+"&auth_date=1",
               signed_data(user={"id": True}), signed_data(user={"id":"9001"}), signed_data(user=[])]
        for data in bad:
            result = api._validate_telegram_webapp_init_data(data, max_age_sec=900)
            self.assertFalse(result["ok"], data)
        response = self.client.post("/api/telegram/identity/verify", json={"initData":"bad"})
        self.assertEqual(response.status_code, 401)
        self.assertNotIn("accessToken", response.json())

    def test_no_bot_configuration_never_allows_dev_fallback(self):
        with patch.object(api, "_telegram_bot_token", return_value=""):
            response = self.client.post("/api/telegram/identity/verify", json={"initData":signed_data()})
            self.assertEqual(response.status_code, 503)
            status = self.client.get("/api/telegram/identity/status").json()
            self.assertTrue(status["authRequired"])
            self.assertFalse(status["devFallbackAllowed"])
            self.assertEqual(self.client.get("/api/settings?playerId=tg_9001", headers=self.headers).status_code, 401)

    def test_expiry_and_bot_rotation_invalidate_credentials(self):
        with patch.object(api, "_telegram_bot_token", return_value="rotated-bot"):
            self.assertEqual(self.client.get("/api/settings?playerId=tg_9001", headers=self.headers).status_code, 401)
        with closing(sqlite3.connect(api.APP_DB)) as conn, conn:
            conn.execute("UPDATE player_auth_sessions_v1 SET expires_at=0 WHERE token_hash=?",
                         (hashlib.sha256(self.credentials["accessToken"].encode()).hexdigest(),))
        self.assertEqual(self.client.get("/api/settings?playerId=tg_9001", headers=self.headers).status_code, 401)

    def test_player_cannot_impersonate_another_account_or_operator(self):
        for path in ("/api/mine/start", "/api/mine/finish", "/api/social/claim", "/api/store/buy",
                     "/api/settings", "/api/wallet/link-qc-address", "/api/watch/claim", "/api/withdraw/request"):
            response = self.client.post(path, headers=self.headers, json={"playerId":"tg_9002"})
            self.assertEqual(response.status_code, 403, (path,response.text))
        for path in ("/api/wallet?playerId=tg_9002", "/api/v1/tgwt/state?user_id=tg_9002",
                     "/api/v1/tgwt/ledger?user_id=tg_9002", "/api/v1/nft/monthly/status?userId=tg_9002",
                     "/api/settings?playerId=tg_9001&playerId=tg_9002"):
            self.assertEqual(self.client.get(path, headers=self.headers).status_code, 403, path)
        for path in ("/api/v1/tgwt/earn", "/api/v1/tgwt/verify", "/api/dev/reset", "/api/settlement/mock-confirm"):
            self.assertEqual(self.client.post(path, headers=self.headers, json={"userId":"tg_9001"}).status_code, 401, path)
        with patch.dict(os.environ, {"TELEGRAM_GAME_ADMIN_TOKEN":"", "QC_TG_ADMIN_TOKEN":"", "QC_ADMIN_TOKEN":"", "ADMIN_TOKEN":""}):
            self.assertEqual(self.client.get("/api/admin/panel/registry").status_code, 401)

    def test_withdraw_status_checks_record_ownership(self):
        with closing(sqlite3.connect(api.APP_DB)) as conn, conn:
            conn.execute("INSERT OR REPLACE INTO withdraw_requests(id,user_id,asset,amount_u,to_address,status,created_at,updated_at) VALUES ('private-req','tg_9002','TGWT',100,'private-address','ADMIN_REVIEW',1,1)")
        response = self.client.get("/api/v1/withdraw/status?id=private-req", headers=self.headers)
        self.assertEqual(response.json(), {"ok":False,"error":"not_found"})
        self.assertNotIn("private-address", response.text)

    def test_legacy_completion_never_credits_signed_in_player(self):
        self.seed_player()
        for amount in (1e30, -100, 100):
            response = self.client.post("/api/session/complete", headers=self.headers,
                                        json={"playerId":"tg_9001","qcEarned":amount,"xpEarned":amount,"durationSec":1e30})
            self.assertEqual(response.json()["error"], "unverified_session_complete_disabled")
        with closing(sqlite3.connect(api.APP_DB)) as conn, conn:
            self.assertEqual(conn.execute("SELECT total_qc,total_xp,sessions FROM players WHERE player_id='tg_9001'").fetchone(), (0,0,0))

    def test_finish_requires_owned_server_run_and_server_elapsed_time(self):
        session = self.opened()
        for body,error in (({},"server_session_required"), ({"sessionId":"invented"},"mining_session_invalid_or_consumed"),
                           ({"sessionId":session,"siteId":2},"mining_session_invalid_or_consumed"),
                           ({"sessionId":session,"durationSec":1e30,"xpEarned":1e30},"mining_session_too_short")):
            response = self.client.post("/api/mine/finish", headers=self.headers, json={"playerId":"tg_9001","siteId":1,**body})
            self.assertEqual(response.json()["error"], error, response.text)

    def test_one_block_one_reward_despite_replay_and_forged_amounts(self):
        session = self.opened()
        self.age_run(session)
        with closing(api._db()) as conn, conn:
            conn.execute("INSERT INTO game_pow_work_v1(session_id,player_id,address,request_key,expires_at,status,receipt_json) VALUES (?,?,?,?,?,'ACCEPTED',?)",
                (session,"tg_9001","QC_test_tg_9001","test-only",int(time.time())+30,json.dumps({"accepted":True,"height":1,"hash":"ab"*32})))
        async def mine(address):
            raise AssertionError("game finish must never ask the server to mine")
        with patch.object(api, "_get_game_mining_mode", return_value="REAL_CHAIN"), patch.object(api, "qc_fetch_mine_block", side_effect=mine) as mined, patch.object(api, "_fetch_qc_node_health_sync", return_value={}), patch.object(api.random,"random", return_value=1):
            body = {"playerId":"tg_9001","siteId":1,"sessionId":session,"durationSec":1e30,"qcEarned":1e30,"xpEarned":1e30}
            response = self.client.post("/api/mine/finish", headers=self.headers, json=body)
            self.assertTrue(response.json()["ok"], response.text)
            self.assertEqual(response.json()["qcEarned"], 0)
            self.assertTrue(response.json()["rewardPaidByChain"])
            self.assertGreaterEqual(response.json()["xpEarned"], 65)
            self.assertLessEqual(response.json()["xpEarned"], 100)
            self.assertEqual(self.client.post("/api/mine/finish", headers=self.headers, json=body).json()["error"], "mining_session_invalid_or_consumed")
            self.assertEqual(mined.call_count, 0)
        with closing(sqlite3.connect(api.APP_DB)) as conn, conn:
            self.assertEqual(conn.execute("SELECT COUNT(*) FROM reward_records WHERE player_id='tg_9001'").fetchone()[0], 1)
            self.assertEqual(conn.execute("SELECT sessions FROM players WHERE player_id='tg_9001'").fetchone()[0], 1)

    def test_sqlite_reservation_is_atomic_between_connections(self):
        session = self.opened()
        self.age_run(session)
        with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
            results = list(pool.map(lambda _: mining_sessions.reserve_run(api._db,"tg_9001",1,session,int(time.time())), range(8)))
        self.assertEqual(sum(bool(result.get("ok")) for result in results), 1)

    def test_sqlite_reservation_is_atomic_between_processes(self):
        session = self.opened()
        self.age_run(session)
        worker = """import json, sqlite3, sys, time
from telegram_game.mining_sessions import reserve_run
def connect():
    return sqlite3.connect(sys.argv[1], timeout=20)
print(json.dumps(reserve_run(connect,'tg_9001',1,sys.argv[2],int(time.time()))))
"""
        env = os.environ.copy()
        env["PYTHONPATH"] = str(Path(__file__).resolve().parents[1]) + os.pathsep + env.get("PYTHONPATH", "")
        def reserve(_):
            result = subprocess.run([sys.executable,"-c",worker,api.APP_DB,session], env=env, capture_output=True, text=True, timeout=30)
            self.assertEqual(result.returncode, 0, result.stderr)
            return json.loads(result.stdout)
        with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
            results = list(pool.map(reserve, range(4)))
        self.assertEqual(sum(bool(result.get("ok")) for result in results), 1)

    def test_request_boundaries_and_bounded_session_count(self):
        for _ in range(10):
            player_auth.issue_session(api._db, "tg_9001", api._telegram_bot_token(), int(time.time()))
        with closing(sqlite3.connect(api.APP_DB)) as conn:
            self.assertEqual(conn.execute("SELECT COUNT(*) FROM player_auth_sessions_v1 WHERE player_id='tg_9001'").fetchone()[0],8)
        fresh = player_auth.issue_session(api._db,"tg_9001",api._telegram_bot_token(),int(time.time()))
        headers = {"Authorization":"Bearer "+fresh["accessToken"],"Content-Type":"application/json"}
        response = self.client.post("/api/settings",headers=headers,content='{"playerId":"tg_9001","playerId":"tg_9002"}')
        self.assertEqual(response.status_code,422)
        response = self.client.post("/api/settings",headers=headers,content=b"x"*262145)
        self.assertEqual(response.status_code,413)
        self.assertEqual(self.client.get("/api/settings?playerId=tg_9001",headers={"Authorization":"Bearer invented"}).status_code,401)

    def test_node_failure_does_not_credit_or_allow_uncertain_retry(self):
        session = self.opened()
        self.age_run(session)
        async def fail(address):
            return {"ok":False,"error":"test-node-failure"}
        with patch.object(api,"_get_game_mining_mode",return_value="REAL_CHAIN"), patch.object(api,"qc_fetch_mine_block",side_effect=fail) as mined:
            body = {"playerId":"tg_9001","sessionId":session,"siteId":1}
            self.assertFalse(self.client.post("/api/mine/finish",headers=self.headers,json=body).json()["ok"])
            self.assertFalse(self.client.post("/api/mine/finish",headers=self.headers,json=body).json()["ok"])
            self.assertEqual(mined.call_count,0)
        with closing(sqlite3.connect(api.APP_DB)) as conn, conn:
            self.assertEqual(conn.execute("SELECT total_qc,total_xp,sessions FROM players WHERE player_id='tg_9001'").fetchone(), (0,0,0))

    def test_raw_signed_json_is_not_rewritten_and_secrets_are_redacted(self):
        self.seed_player()
        raw = '{"transaction":{"Outputs":[{"Amount":9007199254740993}],"Sender":"QC_test_tg_9001"}}'
        async def forward(transaction):
            self.assertEqual(transaction["Outputs"][0]["Amount"], 9007199254740993)
            return {"success":True,"txid":"test-id"}
        from telegram_game.utils import qc_api
        with patch.object(qc_api,"fetch_submit_signed_tx", side_effect=forward) as forwarded:
            response = self.client.post("/api/wallet/send-qc", headers=self.headers,
                json={"playerId":"tg_9001","initData":signed_data(),"transaction":raw})
        self.assertTrue(response.json()["ok"], response.text)
        self.assertEqual(forwarded.call_count, 1)
        with patch.object(api, "qc_fetch_send_tx", side_effect=AssertionError("legacy key bridge")):
            response = self.client.post("/api/wallet/send-qc", headers=self.headers,
                json={"playerId":"tg_9001","transaction":raw,"priv_hex":"[REDACTED_SECRET]"})
        self.assertEqual(response.status_code, 422)
        self.assertNotIn("sensitive-test-key", response.text)

    def test_missing_node_reward_cannot_be_replaced_by_a_default(self):
        session = self.opened()
        self.age_run(session)
        async def invalid_receipt(address):
            return {"ok":True,"height":1,"block_hash":"ab"*32}
        with patch.object(api,"_get_game_mining_mode",return_value="REAL_CHAIN"), patch.object(api,"qc_fetch_mine_block",side_effect=invalid_receipt), patch.object(api,"_fetch_qc_node_health_sync",return_value={}):
            response = self.client.post("/api/mine/finish", headers=self.headers,
                json={"playerId":"tg_9001","siteId":1,"sessionId":session})
        self.assertEqual(response.json()["error"], "accepted_client_pow_required")
        with closing(sqlite3.connect(api.APP_DB)) as conn:
            self.assertEqual(conn.execute("SELECT COUNT(*) FROM reward_records WHERE player_id='tg_9001'").fetchone()[0],0)

    def test_expired_run_and_repeated_start_keep_server_time_authority(self):
        session = self.opened()
        self.age_run(session)
        response = self.client.post("/api/mine/start", headers=self.headers, json={"playerId":"tg_9001","siteId":1})
        self.assertEqual(response.json()["sessionId"],session)
        self.assertLess(response.json()["startedAt"],int(time.time())-60)
        with closing(sqlite3.connect(api.APP_DB)) as conn, conn:
            conn.execute("UPDATE game_mining_runs_v1 SET expires_at=? WHERE session_id=?", (int(time.time())-1,session))
        response = self.client.post("/api/mine/finish", headers=self.headers, json={"playerId":"tg_9001","siteId":1,"sessionId":session})
        self.assertEqual(response.json()["error"],"mining_session_expired")


if __name__ == "__main__":
    unittest.main()
