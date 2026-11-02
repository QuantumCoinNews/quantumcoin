"""Isolated QMA-0070A tests: temporary SQLite only; no API/node or mint."""
import datetime
import hashlib
import json
import os
import sqlite3
import sys
import tempfile
import unittest
from contextlib import closing
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from telegram_game.utils import nft_draw_freeze_v1 as freeze
from telegram_game.utils.nft_selection_identity_v1 import select_monthly_winner_identity_v1

PERIOD = "2032-01"
END = int(datetime.datetime(2032, 2, 1, tzinfo=datetime.timezone.utc).timestamp())
NOW = END + 61
FIELDS = ("candidateId","userId","period","nftType","source","activity",
          "score","aiReviewStatus","aiRiskLevel","adminStatus","adminNote",
          "winnerStatus","mintedStatus","policyVersion","createdAt","updatedAt")


def wallet(num):
    raw = b"\x00" + bytes([num])*20
    raw += hashlib.sha256(hashlib.sha256(raw).digest()).digest()[:4]
    chars, value, answer = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", int.from_bytes(raw,"big"), ""
    while value:
        value, digit = divmod(value, 58)
        answer = chars[digit] + answer
    return "1" + answer


def serialize(row):
    return dict(zip(FIELDS, row))


def ensure(conn):
    conn.execute("""CREATE TABLE IF NOT EXISTS monthly_nft_candidates (
        candidate_id TEXT PRIMARY KEY, user_id TEXT, period TEXT,
        nft_type TEXT, source TEXT, activity_json TEXT, score REAL,
        ai_review_status TEXT, ai_risk_level TEXT, admin_status TEXT,
        admin_note TEXT, winner_status TEXT, minted_status TEXT,
        policy_version TEXT, created_at INTEGER, updated_at INTEGER,
        UNIQUE(user_id,period))""")
    conn.execute("CREATE TABLE IF NOT EXISTS players(player_id TEXT PRIMARY KEY,qc_address TEXT)")
    conn.execute("""CREATE TABLE IF NOT EXISTS monthly_nft_verified_game_sessions (
        session_id TEXT PRIMARY KEY,user_id TEXT,period TEXT,completed_at INTEGER,activity_verified INTEGER)""")


class CandidateFreezeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="qma0070-freeze-")
        self.db = str(Path(self.temp.name)/"game.sqlite")
        with closing(self.connect()) as conn, conn:
            ensure(conn)
        self.add("a", "tg_101", wallet(1))
        self.add("b", "tg_102", wallet(2))

    def tearDown(self):
        self.temp.cleanup()

    def connect(self):
        return sqlite3.connect(self.db, timeout=20)

    def add(self, cid, user, address, period=PERIOD, status="APPROVED_BY_ADMIN"):
        with closing(self.connect()) as conn, conn:
            conn.execute("INSERT OR REPLACE INTO players VALUES(?,?)", (user,address))
            conn.execute("INSERT INTO monthly_nft_candidates VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
                         (cid,user,period,"rare","server_verified_game_session","{}",.65,
                          "AI_REVIEW_PASSED","LOW",status,"","APPROVED_CANDIDATE_NOT_WINNER",
                          "NOT_MINTED","test",NOW-100,NOW-100))
            conn.execute("INSERT INTO monthly_nft_verified_game_sessions VALUES(?,?,?,?,1)",
                         (cid+"-sess",user,period,END-50))

    def freeze(self, now=NOW, allowed=(PERIOD,)):
        return freeze.freeze_monthly_pool_v1(self.connect, PERIOD, allowed, now, ensure, serialize)

    def test_freeze_builds_sorted_manifest_and_no_winner(self):
        result = self.freeze()
        self.assertTrue(result["ok"], result)
        self.assertEqual(result["candidateCount"],2)
        self.assertEqual([c["userId"] for c in result["manifest"]["candidates"]], ["tg_101","tg_102"])
        self.assertTrue(freeze.verify_frozen_pool_v1(result["manifest"],result["poolHash"]))
        self.assertFalse(result["safeBoundary"]["randomnessVerified"])
        with closing(self.connect()) as conn, conn:
            self.assertEqual(conn.execute("SELECT count(*) FROM monthly_nft_candidates WHERE winner_status='SELECTED_MONTHLY_WINNER'").fetchone()[0],0)

    def test_idempotent(self):
        a,b=self.freeze(),self.freeze()
        self.assertTrue(b["alreadyFrozen"])
        self.assertEqual(a["poolHash"],b["poolHash"])
        self.assertEqual(a["frozenAt"],b["frozenAt"])

    def test_cannot_freeze_before_end_of_month(self):
        self.assertEqual(self.freeze(now=END-1)["error"], "monthly_period_not_closed")
        self.assertEqual(freeze.read_monthly_pool_v1(self.connect, PERIOD)["error"],"monthly_pool_not_frozen")

    def test_activation_must_allow_period(self):
        self.assertEqual(self.freeze(allowed=())["error"], "monthly_nft_program_not_active_or_period_outside")

    def test_candidate_without_valid_activity_rejected(self):
        with closing(self.connect()) as conn, conn:
            conn.execute("DELETE FROM monthly_nft_verified_game_sessions WHERE user_id='tg_102'")
        self.assertEqual(self.freeze()["error"], "verified_game_session_required")

    def test_duplicate_wallet_owner_rejected(self):
        with closing(self.connect()) as conn, conn:
            conn.execute("UPDATE players SET qc_address=? WHERE player_id='tg_102'",(wallet(1),))
        self.assertEqual(self.freeze()["error"],"monthly_candidate_wallet_not_unique")

    def test_unapproved_candidate_excluded(self):
        with closing(self.connect()) as conn, conn:
            conn.execute("UPDATE monthly_nft_candidates SET admin_status='ON_HOLD' WHERE candidate_id='b'")
        result=self.freeze()
        self.assertEqual(result["candidateCount"],1)
        self.assertEqual(result["manifest"]["candidates"][0]["candidateId"],"a")

    def test_minted_or_legacy_winner_blocks_freeze(self):
        for state in ("MINTED", "PENDING_LEGACY"):
            with self.subTest(state=state):
                with closing(self.connect()) as conn, conn:
                    conn.execute("UPDATE monthly_nft_candidates SET minted_status=? WHERE candidate_id='b'", (state,))
                self.assertEqual(self.freeze()["error"],"monthly_winner_already_selected")

    def test_zero_eligible_fail_closed(self):
        with closing(self.connect()) as conn, conn:
            conn.execute("UPDATE monthly_nft_candidates SET admin_status='ON_HOLD'")
        self.assertEqual(self.freeze()["error"],"monthly_no_eligible_candidates")

    def test_tamper_proof_detected_and_trigger_protects_record(self):
        record=self.freeze()
        altered=json.loads(json.dumps(record["manifest"]))
        altered["candidates"][0]["userId"]="different"
        with self.assertRaises(freeze.SelectionRejected):
            freeze.verify_frozen_pool_v1(altered,record["poolHash"])
        with closing(self.connect()) as conn, conn:
            with self.assertRaises(sqlite3.IntegrityError):
                conn.execute("UPDATE monthly_nft_pool_freezes_v1 SET pool_sha256=?", ("0"*64,))
        self.assertEqual(freeze.read_monthly_pool_v1(self.connect, PERIOD)["poolHash"],record["poolHash"])

    def test_manual_selection_blocked_after_freeze(self):
        self.assertTrue(self.freeze()["ok"])
        result=select_monthly_winner_identity_v1(self.connect,"a",(PERIOD,),NOW,"test","",ensure,serialize)
        self.assertEqual(result["error"],"monthly_pool_frozen_manual_selection_disabled")

    def test_bad_period_and_no_frozen_pool(self):
        self.assertEqual(freeze.read_monthly_pool_v1(self.connect,"2032-13")["error"],"invalid_freeze_period")
        self.assertEqual(freeze.read_monthly_pool_v1(self.connect,PERIOD)["error"],"monthly_pool_not_frozen")


if __name__ == "__main__":
    unittest.main(verbosity=2)
