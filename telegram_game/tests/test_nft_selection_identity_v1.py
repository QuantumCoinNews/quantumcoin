"""Real admin routes + temporary SQLite; activation is test-only and fixed."""
import datetime
import hashlib
import json
import os
import sqlite3
import subprocess
import sys
import tempfile
import unittest
from contextlib import closing
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

if __name__ == "__main__" and os.getenv("QMA0070_SELECTION_TEST_CHILD") != "1":
    with tempfile.TemporaryDirectory(prefix="qma0070-api-process-") as directory:
        env = os.environ.copy()
        env.update(QMA0070_SELECTION_TEST_CHILD="1", QC_API_BASE="http://127.0.0.1:1",
            QC_API_TOKEN="test-only-unreachable-node", TELEGRAM_GAME_ADMIN_TOKEN="test-only-selection-admin",
            TELEGRAM_BOT_TOKEN="test-only-telegram", QC_ENABLE_CHAIN_NFT_MINT="0", QC_ENABLE_TG_NFT_MINT_BRIDGE="0")
        env["PYTHONPATH"] = str(ROOT) + os.pathsep + env.get("PYTHONPATH", "")
        result = subprocess.run([sys.executable, str(Path(__file__).resolve()), *sys.argv[1:]], cwd=directory, env=env, timeout=120)
    raise SystemExit(result.returncode)

from fastapi.testclient import TestClient

# Also keep module collection by another test runner away from the real DB:
# dev_api initializes a relative dev_api_state.db during import.
with tempfile.TemporaryDirectory(prefix="qma0070-startup-") as startup:
    original_connect = sqlite3.connect
    def isolated_connect(database, *args, **kwargs):
        if str(database) == "dev_api_state.db":
            database = str(Path(startup) / "startup.sqlite")
        return original_connect(database, *args, **kwargs)
    with patch.object(sqlite3, "connect", side_effect=isolated_connect):
        from telegram_game import dev_api as api
from telegram_game.utils import nft_selection_identity_v1 as selection

ACTIVATION = int(datetime.datetime(2031, 12, 15, tzinfo=datetime.timezone.utc).timestamp())
NOW = int(datetime.datetime(2032, 1, 20, tzinfo=datetime.timezone.utc).timestamp())


def wallet(number):
    raw = b"\0" + bytes([number]) * 20
    raw += hashlib.sha256(hashlib.sha256(raw).digest()).digest()[:4]
    alphabet, result, value = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "", int.from_bytes(raw, "big")
    while value:
        value, digit = divmod(value, 58)
        result = alphabet[digit] + result
    return "1" * (len(raw) - len(raw.lstrip(b"\0"))) + result


class MonthlySelectionIdentityTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="qma0070-selection-")
        self.db = str(Path(self.temp.name) / "game.sqlite")
        self.patches = [patch.object(api, "APP_DB", self.db), patch.object(api, "_now", return_value=NOW),
            patch.object(api, "_nft_verified_mainnet_activation_at_v1", return_value=ACTIVATION)]
        for item in self.patches:
            item.start()
        api._init()
        with closing(api._db()) as conn, conn:
            api._ensure_qc_address_column(conn)
            api._ensure_monthly_nft_tables(conn)
            api._ensure_monthly_nft_verified_game_sessions_v1(conn)
        api._QC_ACTION_BURST_EVENTS_V1.clear()
        api._QC_ACTION_BURST_BLOCKS_V1.clear()
        self.client = TestClient(api.app)
        self.add_candidate("candidate-a", "tg_101", wallet(1))
        self.add_candidate("candidate-b", "tg_102", wallet(2))

    def tearDown(self):
        self.client.close()
        for item in reversed(self.patches):
            item.stop()
        self.temp.cleanup()

    def add_candidate(self, candidate_id, user, address, period="2032-01"):
        year, month = map(int, period.split("-"))
        completed = int(datetime.datetime(year, month, 10, tzinfo=datetime.timezone.utc).timestamp())
        with closing(api._db()) as conn, conn:
            api._touch_player(conn, user, None)
            conn.execute("UPDATE players SET qc_address=? WHERE player_id=?", (address, user))
            conn.execute("INSERT INTO monthly_nft_candidates VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", (
                candidate_id, user, period, "rare", "server_verified_game_session", '{"verifiedCompletedGameSessions":1}',
                .9, "AI_REVIEW_PASSED", "LOW", "APPROVED_BY_ADMIN", "", "APPROVED_CANDIDATE_NOT_WINNER", "NOT_MINTED", "test-only-selection-policy", NOW, NOW))
            conn.execute("INSERT INTO monthly_nft_verified_game_sessions VALUES(?,?,?,?,1)", (candidate_id+"-session", user, period, completed))

    def select(self, candidate="candidate-a", headers=None):
        response = self.client.post("/api/admin/nft/monthly/select-winner", json={"candidateId": candidate},
            headers=headers if headers is not None else {"X-Admin-Token": "test-only-selection-admin"})
        self.assertEqual(response.status_code, 200, response.text)
        return response.json()

    def winners(self):
        with closing(api._db()) as conn:
            return conn.execute("SELECT candidate_id FROM monthly_nft_candidates WHERE winner_status='SELECTED_MONTHLY_WINNER' ORDER BY candidate_id").fetchall()

    def test_admin_authentication_still_required(self):
        url = "/api/admin/nft/monthly/select-winner"
        for suffix in ("", "?token=test-only-selection-admin"):
            response = self.client.post(url+suffix, json={"candidateId": "candidate-a"})
            self.assertEqual(response.status_code, 401)
        self.assertEqual(self.winners(), [])

    def seed_verified_winner_fixture(self, connect=None):
        # A historical winner is a fixture, never created by the retired API.
        with closing((connect or api._db)()) as conn, conn:
            conn.execute("CREATE UNIQUE INDEX IF NOT EXISTS idx_monthly_nft_one_winner_identity_v1 ON monthly_nft_candidates(period) WHERE winner_status='SELECTED_MONTHLY_WINNER' OR minted_status IS NOT 'NOT_MINTED'")
            conn.execute("UPDATE monthly_nft_candidates SET winner_status='SELECTED_MONTHLY_WINNER' WHERE candidate_id='candidate-a'")

    def test_valid_manual_selection_cannot_change_candidate_or_player_state(self):
        with closing(api._db()) as conn:
            before = conn.execute("SELECT player_id,total_qc,total_xp,sessions FROM players ORDER BY player_id").fetchall()
            candidates = conn.execute("SELECT * FROM monthly_nft_candidates ORDER BY candidate_id").fetchall()
        result = self.select()
        self.assertEqual(result, {"ok": False, "error": "monthly_manual_selection_disabled"})
        with closing(api._db()) as conn:
            self.assertEqual(conn.execute("SELECT player_id,total_qc,total_xp,sessions FROM players ORDER BY player_id").fetchall(), before)
            self.assertEqual(conn.execute("SELECT * FROM monthly_nft_candidates ORDER BY candidate_id").fetchall(), candidates)

    def test_duplicate_wallet_cannot_select(self):
        with closing(api._db()) as conn, conn:
            conn.execute("UPDATE players SET qc_address=? WHERE player_id='tg_102'", (wallet(1),))
        result = self.select()
        self.assertFalse(result["ok"], result)
        self.assertEqual(result["error"], "monthly_candidate_wallet_not_unique")
        self.assertEqual(self.winners(), [])

    def test_wallet_whitespace_alias_cannot_gain_second_identity(self):
        with closing(api._db()) as conn, conn:
            conn.execute("UPDATE players SET qc_address=? WHERE player_id='tg_102'", ("  "+wallet(1)+"  ",))
        self.assertEqual(self.select()["error"], "monthly_candidate_wallet_not_unique")
        self.assertEqual(self.winners(), [])

    def test_invalid_wallet_format_is_rejected(self):
        for address in ("QC_FAKE", wallet(1)[:-1]+"0", "", "1" * 200):
            with self.subTest(address=address):
                with closing(api._db()) as conn, conn:
                    conn.execute("UPDATE players SET qc_address=? WHERE player_id='tg_101'", (address,))
                self.assertFalse(self.select()["ok"])
        self.assertEqual(self.winners(), [])

    def test_current_period_evidence_cannot_be_replaced_by_client_score(self):
        for period, verified, completed in (("2032-01",0,NOW-10),("2031-12",1,NOW-10),
                ("2032-01",1,NOW+86400),("2032-01",1,NOW-40*86400)):
            with self.subTest(period=period, verified=verified, completed=completed):
                with closing(api._db()) as conn, conn:
                    conn.execute("UPDATE monthly_nft_verified_game_sessions SET period=?,activity_verified=?,completed_at=? WHERE user_id='tg_101'", (period,verified,completed))
                    conn.execute("UPDATE monthly_nft_candidates SET score=999,activity_json=? WHERE candidate_id='candidate-a'", ('{"miningSessions":99999}',))
                self.assertEqual(self.select()["error"], "verified_game_session_required")
        self.assertEqual(self.winners(), [])

    def test_period_outside_verified_program_is_rejected(self):
        with closing(api._db()) as conn, conn:
            conn.execute("UPDATE monthly_nft_candidates SET period='2033-01' WHERE candidate_id='candidate-a'")
        self.assertEqual(self.select()["error"], "monthly_nft_program_not_active_or_period_outside")

    def test_legacy_or_unknown_mint_state_keeps_month_occupied(self):
        for state in ("MINTED_LEGACY", "PENDING_LEGACY"):
            with self.subTest(state=state):
                with closing(api._db()) as conn, conn:
                    conn.execute("UPDATE monthly_nft_candidates SET minted_status=? WHERE candidate_id='candidate-b'",(state,))
                self.assertEqual(self.select()["error"],"monthly_winner_already_selected")
        self.assertEqual(self.winners(),[])

    def test_legacy_duplicate_user_candidates_are_rejected(self):
        with closing(api._db()) as conn, conn:
            conn.execute("ALTER TABLE monthly_nft_candidates RENAME TO previous_candidates")
            conn.execute("CREATE TABLE monthly_nft_candidates AS SELECT * FROM previous_candidates")
            conn.execute("DROP TABLE previous_candidates")
            conn.execute("UPDATE monthly_nft_candidates SET user_id='tg_101' WHERE candidate_id='candidate-b'")
        self.assertEqual(self.select()["error"], "monthly_candidate_user_not_unique")
        self.assertEqual(self.winners(), [])

    def test_review_cannot_reset_selected_winner(self):
        self.seed_verified_winner_fixture()
        for action in ("APPROVE", "REJECT", "HOLD"):
            response = self.client.post("/api/admin/nft/monthly/review", json={"candidateId":"candidate-a","action":action}, headers={"X-Admin-Token":"test-only-selection-admin"})
            self.assertEqual(response.json()["error"], "candidate_selection_locked", response.text)
        self.assertEqual(self.select("candidate-b")["error"], "monthly_winner_already_selected")
        self.assertEqual(self.winners(), [("candidate-a",)])

    def test_monthly_limit_applies_across_nft_types(self):
        with closing(api._db()) as conn, conn:
            conn.execute("UPDATE monthly_nft_candidates SET nft_type='legacy-other-type',winner_status='SELECTED_MONTHLY_WINNER' WHERE candidate_id='candidate-b'")
        self.assertEqual(self.select()["error"], "monthly_winner_already_selected")

    def test_manual_selection_is_disabled_in_other_month_too(self):
        self.seed_verified_winner_fixture()
        self.add_candidate("candidate-next-month", "tg_101", wallet(1), "2032-02")
        later = int(datetime.datetime(2032,2,20,tzinfo=datetime.timezone.utc).timestamp())
        with patch.object(api,"_now",return_value=later):
            self.assertEqual(self.select("candidate-next-month")["error"], "monthly_manual_selection_disabled")
        self.assertEqual(len(self.winners()),1)

    def test_stale_admin_review_cannot_overwrite_concurrent_selection(self):
        original = api._db
        owner = self
        class StaleCursor:
            def __init__(self,cursor): self.cursor=cursor
            def fetchone(self):
                row=self.cursor.fetchone()
                self.cursor.close()
                owner.seed_verified_winner_fixture(original)
                return row
        class StaleReview:
            def __init__(self): self.conn=original();self.injected=False
            def execute(self,sql,args=()):
                cursor=self.conn.execute(sql,args)
                if not self.injected and "FROM monthly_nft_candidates" in sql and "WHERE candidate_id=?" in sql:
                    self.injected=True
                    return StaleCursor(cursor)
                return cursor
            def __getattr__(self,name): return getattr(self.conn,name)
        with patch.object(api,"_db",side_effect=StaleReview):
            response=self.client.post("/api/admin/nft/monthly/review",json={"candidateId":"candidate-a","action":"REJECT"},headers={"X-Admin-Token":"test-only-selection-admin"})
        self.assertEqual(response.json()["error"],"candidate_selection_locked",response.text)
        self.assertEqual(self.winners(),[("candidate-a",)])

    def test_database_unique_index_rejects_second_writer(self):
        self.seed_verified_winner_fixture()
        with closing(api._db()) as conn:
            with self.assertRaises(sqlite3.IntegrityError), conn:
                conn.execute("UPDATE monthly_nft_candidates SET winner_status='SELECTED_MONTHLY_WINNER' WHERE candidate_id='candidate-b'")
        self.assertEqual(self.winners(), [("candidate-a",)])

    def test_eight_processes_cannot_select_any_manual_winner(self):
        code = r'''
import json,sqlite3,sys
from telegram_game.utils.nft_selection_identity_v1 import select_monthly_winner_identity_v1
keys=('candidateId','userId','period','nftType','source','activity','score','aiReviewStatus','aiRiskLevel','adminStatus','adminNote','winnerStatus','mintedStatus','policyVersion','createdAt','updatedAt')
result=select_monthly_winner_identity_v1(lambda:sqlite3.connect(sys.argv[1],timeout=20),sys.argv[2],('2032-01',),int(sys.argv[3]),'test-only','',lambda conn:None,lambda row:dict(zip(keys,row)))
print(json.dumps(result))
'''
        workers = [subprocess.Popen([sys.executable,"-c",code,self.db,"candidate-a" if i%2 else "candidate-b",str(NOW)],stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True) for i in range(8)]
        results=[]
        try:
            for worker in workers:
                stdout,stderr=worker.communicate(timeout=35)
                self.assertEqual(worker.returncode,0,stderr)
                results.append(json.loads(stdout))
        finally:
            for worker in workers:
                if worker.poll() is None:
                    worker.kill()
                worker.communicate(timeout=10)
        self.assertTrue(all(r == {"ok":False,"error":"monthly_manual_selection_disabled"} for r in results), results)
        self.assertEqual(self.winners(),[])

    def test_disabled_path_never_attempts_winner_update(self):
        original=api._db
        class FailedUpdate:
            def __init__(self): self.conn=original()
            def execute(self,sql,args=()):
                if "SET winner_status='SELECTED_MONTHLY_WINNER'" in sql:
                    raise sqlite3.OperationalError("test-only-update-failure")
                return self.conn.execute(sql,args)
            def __getattr__(self,name): return getattr(self.conn,name)
        with patch.object(api,"_db",side_effect=FailedUpdate):
            self.assertEqual(self.select()["error"],"monthly_manual_selection_disabled")
        self.assertEqual(self.winners(),[])
        self.assertEqual(self.select()["error"],"monthly_manual_selection_disabled")


if __name__ == "__main__":
    unittest.main(verbosity=2)
