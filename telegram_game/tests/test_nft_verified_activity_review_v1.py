"""Real signed game session -> candidate -> local review -> admin -> manual selection.

All DBs, tokens and programme dates are test-only. No real node or mint call.
"""
import datetime
import hashlib
import hmac
import json
import os
import sqlite3
import subprocess
import sys
import tempfile
import unittest
import urllib.parse
from contextlib import closing
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
TOKEN = "test-only-activity-review-telegram"
ADMIN = "test-only-activity-review-admin"
if __name__ == "__main__" and os.getenv("QMA_ACTIVITY_REVIEW_TEST_CHILD") != "1":
    with tempfile.TemporaryDirectory(prefix="qma-activity-process-") as directory:
        env = os.environ.copy()
        env.update(QMA_ACTIVITY_REVIEW_TEST_CHILD="1", QC_API_BASE="http://127.0.0.1:1",
            QC_API_TOKEN="test-only-unreachable-node", TELEGRAM_GAME_ADMIN_TOKEN=ADMIN,
            TELEGRAM_BOT_TOKEN=TOKEN, QC_ENABLE_CHAIN_NFT_MINT="0", QC_ENABLE_TG_NFT_MINT_BRIDGE="0")
        env["PYTHONPATH"] = str(ROOT) + os.pathsep + env.get("PYTHONPATH", "")
        result = subprocess.run([sys.executable, str(Path(__file__).resolve()), *sys.argv[1:]],
            cwd=directory, env=env, timeout=120)
    raise SystemExit(result.returncode)

from fastapi.testclient import TestClient

with tempfile.TemporaryDirectory(prefix="qma-activity-startup-") as startup:
    original_connect = sqlite3.connect
    def isolated_connect(database, *args, **kwargs):
        if str(database) == "dev_api_state.db":
            database = str(Path(startup) / "startup.sqlite")
        return original_connect(database, *args, **kwargs)
    with patch.object(sqlite3, "connect", side_effect=isolated_connect):
        from telegram_game import dev_api as api

UTC = datetime.timezone.utc
NOW = int(datetime.datetime(2032, 1, 20, tzinfo=UTC).timestamp())
ACTIVATION = int(datetime.datetime(2031, 12, 15, tzinfo=UTC).timestamp())
USER = "tg_701"
PERIOD = "2032-01"


def wallet():
    raw = b"\0" + b"\x07" * 20
    raw += hashlib.sha256(hashlib.sha256(raw).digest()).digest()[:4]
    alphabet, result, value = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "", int.from_bytes(raw, "big")
    while value:
        value, digit = divmod(value, 58)
        result = alphabet[digit] + result
    return "1" * (len(raw) - len(raw.lstrip(b"\0"))) + result


class VerifiedActivityReviewTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="qma-activity-review-")
        self.clock = NOW
        self.patches = [patch.object(api, "APP_DB", str(Path(self.temp.name)/"game.sqlite")),
            patch.object(api, "_now", side_effect=lambda: self.clock),
            patch.object(api, "_nft_verified_mainnet_activation_at_v1", return_value=ACTIVATION),
            patch.dict(os.environ, {"TELEGRAM_BOT_TOKEN":TOKEN, "TELEGRAM_GAME_ADMIN_TOKEN":ADMIN})]
        for item in self.patches:
            item.start()
        api._init()
        with closing(api._db()) as conn, conn:
            api._ensure_qc_address_column(conn)
            api._ensure_monthly_nft_tables(conn)
            api._ensure_monthly_nft_verified_game_sessions_v1(conn)
            api._touch_player(conn, USER, None)
            conn.execute("UPDATE players SET qc_address=? WHERE player_id=?", (wallet(), USER))
        api._QC_ACTION_BURST_EVENTS_V1.clear()
        api._QC_ACTION_BURST_BLOCKS_V1.clear()
        self.client = TestClient(api.app)

    def tearDown(self):
        self.client.close()
        for item in reversed(self.patches):
            item.stop()
        self.temp.cleanup()

    def post(self, path, body, admin=False):
        result = self.client.post(path, json=body, headers={"X-Admin-Token":ADMIN} if admin else {})
        self.assertEqual(result.status_code, 200, result.text)
        return result.json()

    def signed(self):
        values = {"auth_date":str(self.clock), "user":json.dumps({"id":701}, separators=(",",":"))}
        check = "\n".join(f"{key}={values[key]}" for key in sorted(values))
        secret = hmac.new(b"WebAppData", TOKEN.encode(), hashlib.sha256).digest()
        values["hash"] = hmac.new(secret, check.encode(), hashlib.sha256).hexdigest()
        return urllib.parse.urlencode(values)

    def completed_game(self):
        data = self.signed()
        started = self.post("/api/v1/nft/monthly/session/start", {"playerId":USER,"initData":data})
        self.assertTrue(started["ok"], started)
        self.clock += 31
        action = self.post("/api/mine/start", {"playerId":USER,"siteId":1,"initData":data,"nftSessionId":started["sessionId"]})
        self.assertTrue(action["ok"], action)
        self.clock += 30
        done = self.post("/api/v1/nft/monthly/session/complete", {"playerId":USER,"initData":data,"sessionId":started["sessionId"]})
        self.assertTrue(done["ok"], done)

    def create(self):
        return self.post("/api/v1/nft/monthly/candidate", {"userId":USER,"period":PERIOD,
            "source":"client-untrusted", "activity":{"miningSessions":999,"realBlockFound":True,"verifiedCompletedGameSessions":999}})

    def candidate(self):
        self.completed_game()
        result = self.create()
        self.assertTrue(result["ok"], result)
        return result["candidate"]["candidateId"]

    def review(self, cid):
        return self.post("/api/admin/nft/monthly/ai-review", {"candidateId":cid}, True)

    def approve(self, cid):
        return self.post("/api/admin/nft/monthly/review", {"candidateId":cid,"action":"APPROVE"}, True)

    def select(self, cid):
        return self.post("/api/admin/nft/monthly/select-winner", {"candidateId":cid}, True)

    def row(self, cid):
        with closing(api._db()) as conn:
            return conn.execute(api._monthly_nft_candidate_select_sql()+" WHERE candidate_id=?", (cid,)).fetchone()

    def balances(self):
        with closing(api._db()) as conn:
            return conn.execute("SELECT player_id,total_qc,total_xp,sessions FROM players ORDER BY player_id").fetchall()

    def test_signed_completion_reaches_review_admin_but_not_manual_selection(self):
        cid = self.candidate()
        self.assertEqual(self.row(cid)[7:10], ("PENDING_AI_REVIEW","UNKNOWN","PENDING_ADMIN_REVIEW"))
        before = self.balances()
        result = self.review(cid)
        self.assertTrue(result["ok"], result)
        self.assertEqual(result["candidate"]["aiReviewStatus"], "AI_REVIEW_PASSED")
        self.assertEqual(result["aiReview"]["verifiedSessionCount"], 1)
        self.assertEqual(result["candidate"]["activity"], {"verifiedCompletedGameSessions":1})
        self.assertEqual(result["candidate"]["adminStatus"], "PENDING_ADMIN_REVIEW")
        self.assertEqual(result["candidate"]["winnerStatus"], "NOT_SELECTED")
        for key in ("minted","winnerSelected","balanceChanged","chainMetadataWritten","qcSupplyChanged","tgwtBalanceChanged"):
            self.assertIs(result["safeBoundary"][key], False)
        self.assertEqual(self.select(cid)["error"], "admin_not_approved")
        self.assertTrue(self.approve(cid)["ok"])
        chosen = self.select(cid)
        self.assertEqual(chosen, {"ok":False,"error":"monthly_manual_selection_disabled"})
        self.assertEqual(self.row(cid)[11], "APPROVED_CANDIDATE_NOT_WINNER")
        self.assertEqual(self.balances(), before)

    def test_review_ignores_stale_or_forged_candidate_activity(self):
        cid = self.candidate()
        with closing(api._db()) as conn, conn:
            conn.execute("UPDATE monthly_nft_candidates SET source='client',activity_json=? WHERE candidate_id=?",
                ('{"miningSessions":999,"realBlockFound":true,"verifiedCompletedGameSessions":999}',cid))
            conn.execute("DELETE FROM monthly_nft_verified_game_sessions")
        result = self.review(cid)
        self.assertTrue(result["ok"], result)
        self.assertEqual(result["candidate"]["aiReviewStatus"], "AI_REVIEW_FLAGGED")
        self.assertEqual(result["candidate"]["score"], 0)
        self.assertEqual(result["aiReview"]["verifiedSessionCount"],0)
        self.assertTrue(self.approve(cid)["ok"])
        self.assertEqual(self.select(cid)["error"], "ai_review_not_passed")

    def test_candidate_gate_rejects_future_or_wrong_period_evidence(self):
        for period, at in ((PERIOD,NOW+1),("2031-12",NOW-1),(PERIOD,NOW-40*86400)):
            with self.subTest(period=period, at=at):
                with closing(api._db()) as conn, conn:
                    conn.execute("DELETE FROM monthly_nft_verified_game_sessions")
                    conn.execute("INSERT INTO monthly_nft_verified_game_sessions VALUES('forged',?,?,?,1)", (USER,period,at))
                self.assertEqual(self.create()["error"], "verified_game_session_required")

    def test_review_rechecks_verified_flag_period_time_and_user(self):
        cid = self.candidate()
        for user, period, verified, at in ((USER,PERIOD,0,NOW),("tg_other",PERIOD,1,NOW),
                (USER,"2031-12",1,NOW),(USER,PERIOD,1,self.clock+1),(USER,PERIOD,1,NOW-40*86400)):
            with self.subTest(user=user,period=period,verified=verified,at=at):
                with closing(api._db()) as conn, conn:
                    conn.execute("UPDATE monthly_nft_verified_game_sessions SET user_id=?,period=?,activity_verified=?,completed_at=?",(user,period,verified,at))
                result = self.review(cid)
                self.assertEqual(result["candidate"]["aiReviewStatus"],"AI_REVIEW_FLAGGED",result)

    def test_review_rereads_new_server_activity_without_new_candidate(self):
        cid = self.candidate()
        self.completed_game()
        result = self.review(cid)
        self.assertEqual(result["aiReview"]["verifiedSessionCount"],2)
        self.assertEqual(result["candidate"]["score"],0.65)
        self.assertEqual(result["candidate"]["activity"],{"verifiedCompletedGameSessions":2})

    def test_reevaluation_requires_fresh_admin_approval(self):
        cid = self.candidate()
        self.assertTrue(self.review(cid)["ok"])
        self.assertTrue(self.approve(cid)["ok"])
        again = self.review(cid)
        self.assertEqual(again["candidate"]["adminStatus"],"PENDING_ADMIN_REVIEW")
        self.assertEqual(self.select(cid)["error"],"admin_not_approved")

    def test_selected_or_mint_locked_candidate_cannot_be_rereviewed(self):
        cid = self.candidate()
        for winner, minted in (("SELECTED_MONTHLY_WINNER","NOT_MINTED"),("NOT_SELECTED","PENDING_LEGACY"),("NOT_SELECTED","MINTED")):
            with self.subTest(winner=winner,minted=minted):
                with closing(api._db()) as conn, conn:
                    conn.execute("UPDATE monthly_nft_candidates SET winner_status=?,minted_status=? WHERE candidate_id=?",(winner,minted,cid))
                before = self.row(cid)
                self.assertEqual(self.review(cid)["error"],"candidate_selection_locked")
                self.assertEqual(self.row(cid),before)

    def test_program_unavailable_or_outside_cannot_change_review(self):
        cid = self.candidate()
        before = self.row(cid)
        with patch.object(api,"_nft_verified_mainnet_activation_at_v1",return_value=None):
            self.assertEqual(self.review(cid)["error"],"monthly_nft_program_not_active_or_period_outside")
        self.assertEqual(self.row(cid),before)
        with closing(api._db()) as conn, conn:
            conn.execute("UPDATE monthly_nft_candidates SET period='2033-01' WHERE candidate_id=?",(cid,))
        before = self.row(cid)
        self.assertEqual(self.review(cid)["error"],"monthly_nft_program_not_active_or_period_outside")
        self.assertEqual(self.row(cid),before)

    def test_auth_required_for_advisory_review(self):
        cid = self.candidate()
        before = self.row(cid)
        for suffix in ("", "?token="+ADMIN):
            response=self.client.post("/api/admin/nft/monthly/ai-review"+suffix,json={"candidateId":cid})
            self.assertEqual(response.status_code,401)
        self.assertEqual(self.row(cid),before)

    def test_review_database_error_rolls_back_all_changes(self):
        cid = self.candidate()
        with closing(api._db()) as conn, conn:
            conn.execute("CREATE TRIGGER fail_review BEFORE UPDATE ON monthly_nft_candidates BEGIN SELECT RAISE(ABORT,'test only'); END")
        before=self.row(cid)
        self.assertEqual(self.review(cid)["error"],"monthly_ai_review_store_unavailable")
        self.assertEqual(self.row(cid),before)
        with closing(api._db()) as conn, conn:
            conn.execute("DROP TRIGGER fail_review")
        self.assertTrue(self.review(cid)["ok"])

    def test_lookup_by_user_and_period_uses_same_evidence(self):
        cid=self.candidate()
        result=self.post("/api/admin/nft/monthly/ai-review",{"userId":USER,"period":PERIOD},True)
        self.assertTrue(result["ok"],result)
        self.assertEqual(result["candidate"]["candidateId"],cid)
        self.assertEqual(result["aiReview"]["verifiedSessionCount"],1)

    def test_other_connection_cannot_select_during_review_transaction(self):
        from telegram_game.utils.nft_selection_identity_v1 import select_monthly_winner_identity_v1
        cid=self.candidate()
        self.assertTrue(self.review(cid)["ok"])
        self.assertTrue(self.approve(cid)["ok"])
        original=api._monthly_nft_local_ai_score
        competing=[]
        def review_while_other_worker_selects(candidate,count):
            competing.append(select_monthly_winner_identity_v1(
                lambda:sqlite3.connect(api.APP_DB,timeout=0.05),cid,(PERIOD,),self.clock,
                "test-only-other-worker","",api._ensure_monthly_nft_tables,api._monthly_nft_candidate_row_to_dict))
            return original(candidate,count)
        with patch.object(api,"_monthly_nft_local_ai_score",side_effect=review_while_other_worker_selects):
            self.assertTrue(self.review(cid)["ok"])
        self.assertEqual(competing,[{"ok":False,"error":"monthly_selection_store_unavailable"}])
        self.assertEqual(self.row(cid)[11],"NOT_SELECTED")
        self.assertTrue(self.approve(cid)["ok"])
        self.assertEqual(self.select(cid)["error"], "monthly_manual_selection_disabled")

    def test_missing_evidence_table_cannot_pass_review(self):
        cid=self.candidate()
        with closing(api._db()) as conn, conn:
            conn.execute("DROP TABLE monthly_nft_verified_game_sessions")
        result=self.review(cid)
        self.assertEqual(result["candidate"]["aiReviewStatus"],"AI_REVIEW_FLAGGED")
        self.assertEqual(result["aiReview"]["verifiedSessionCount"],0)


if __name__ == "__main__":
    unittest.main(verbosity=2)
