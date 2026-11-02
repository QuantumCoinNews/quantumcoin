"""Four focused stdlib tests; existing FastAPI function bodies run in isolation.

The Windows run consumes JSON emitted by the real Go runtime handler. Local
preparation uses an explicit test-only contract when no Go output is requested.
This is not a test of the entire FastAPI app, UI, selection or mint path.
"""

import ast
import copy
import datetime
import http.server
import json
import os
import secrets
import sqlite3
import sys
import tempfile
import threading
import types
import unittest
from contextlib import closing
from pathlib import Path
from typing import Optional
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
from telegram_game.utils import nft_program_bridge_v1 as bridge


class NFTGameProgramBridgeTests(unittest.TestCase):
    def setUp(self):
        requested = os.getenv("QMA0066_NFT_PROGRAM_CONTRACT_PATH")
        if requested:
            # A missing/failed Go contract cannot silently fall back to a mock.
            self.program = json.loads(Path(requested).read_text(encoding="utf-8-sig"))
        else:
            at = int(datetime.datetime(2031, 12, 15, tzinfo=datetime.timezone.utc).timestamp())
            self.program = {
                "success": True, "version": bridge.VERSION, "verified": True,
                "gameActivatedAt": at, "networkID": "test-only-bridge",
                "legacyGenesisHex": "ab" * 32, "campaignID": "telegram-monthly",
                "firstAwardPeriod": "2032-01",
                "awardPeriods": [f"2032-{month:02d}" for month in range(1, 13)],
                "awardCount": 12, "maxAwardsPerMonth": 1, "mintProtocolActive": True,
            }
        self.assertIsNotNone(bridge._validate_program_v1(self.program))
        self.body = json.dumps(self.program).encode()
        self.status, self.content_type, self.redirect = 200, "application/json", None
        self.requests = []
        owner = self

        class Handler(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                owner.requests.append((self.path, dict(self.headers)))
                self.send_response(owner.status)
                self.send_header("Content-Type", owner.content_type)
                if owner.redirect:
                    self.send_header("Location", owner.redirect)
                self.end_headers()
                self.wfile.write(owner.body)

            def log_message(self, *_):
                pass

        self.server = http.server.HTTPServer(("127.0.0.1", 0), Handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.base = f"http://127.0.0.1:{self.server.server_port}"
        self.environment = patch.dict(os.environ, {
            "QC_API_BASE": self.base, "QC_API_TOKEN": "test-only-program-token",
            "QC_TESTNET": "", "QC_PUBLIC_TESTNET": "", "QC_NET_MAGIC": "",
            "TELEGRAM_GAME_CHAIN_MODE": "local",
            "QC_ENABLE_CHAIN_NFT_MINT": "1", "QC_NFT_GAME_ACTIVATED_AT": "1",
        })
        self.environment.start()
        self.directory = tempfile.TemporaryDirectory(prefix="qma0066-game-bridge-")
        self.db = str(Path(self.directory.name)/"candidate.sqlite")
        with closing(sqlite3.connect(self.db)) as conn, conn:
            conn.execute("CREATE TABLE players(player_id TEXT PRIMARY KEY, display_name TEXT, last_seen INTEGER)")
            conn.execute("CREATE TABLE monthly_nft_verified_game_sessions(session_id TEXT PRIMARY KEY, user_id TEXT, period TEXT, completed_at INTEGER, activity_verified INTEGER)")
        names = {
            "_nft_verified_mainnet_activation_at_v1", "_monthly_nft_award_periods_from_activation_v1",
            "_ensure_monthly_nft_tables", "_monthly_nft_candidate_row_to_dict",
            "_nft_safe_ensure_player", "_monthly_nft_verified_session_count_v1",
            "_monthly_nft_candidate_select_sql", "nft_monthly_candidate_create", "_nft_month_key",
        }
        path = ROOT/"telegram_game/dev_api.py"
        nodes = [n for n in ast.parse(path.read_text(encoding="utf-8-sig")).body if isinstance(n, ast.FunctionDef) and n.name in names]
        self.assertEqual({n.name for n in nodes}, names)
        for n in nodes:
            n.decorator_list = []
        self.api = {
            "Optional": Optional, "MonthlyNFTCandidateIn": types.SimpleNamespace,
            "sqlite3": sqlite3, "json": json, "secrets": secrets,
            "NFT_MONTHLY_POLICY_VERSION": "TG_MONTHLY_NFT_CANDIDATE_POOL_V1", "NFT_DEFAULT_TYPE": "rare",
            "_lock": threading.Lock(), "_db": lambda: sqlite3.connect(self.db), "_now": lambda: self.month_time(self.program["firstAwardPeriod"]),
        }
        exec(compile(ast.Module(body=nodes, type_ignores=[]), str(path), "exec"), self.api)

    def tearDown(self):
        self.environment.stop()
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(timeout=2)
        self.directory.cleanup()

    def month_time(self, period):
        year, month = map(int, period.split("-"))
        return int(datetime.datetime(year, month, 10, tzinfo=datetime.timezone.utc).timestamp())

    def add_session(self, user, period, verified=1):
        with closing(sqlite3.connect(self.db)) as conn, conn:
            conn.execute("INSERT INTO monthly_nft_verified_game_sessions VALUES(?,?,?,?,?)", (secrets.token_hex(12), user, period, self.month_time(period), verified))

    def candidate(self, user, period):
        return self.api["nft_monthly_candidate_create"](types.SimpleNamespace(userId=user, period=period, source="client", activity={"completedGameSessions": 999}))

    def candidate_count(self):
        with closing(sqlite3.connect(self.db)) as conn, conn:
            exists = conn.execute("SELECT 1 FROM sqlite_master WHERE type='table' AND name='monthly_nft_candidates'").fetchone()
            return conn.execute("SELECT COUNT(*) FROM monthly_nft_candidates").fetchone()[0] if exists else 0

    def test_verified_node_response_reaches_actual_candidate_gate(self):
        period = self.program["firstAwardPeriod"]
        self.add_session("active-user", period)
        self.assertEqual(self.api["_nft_verified_mainnet_activation_at_v1"](), self.program["gameActivatedAt"])
        result = self.candidate("active-user", period)
        self.assertTrue(result["ok"], result)
        self.assertEqual(result["candidate"]["activity"], {"verifiedCompletedGameSessions": 1})
        self.assertEqual(result["candidate"]["mintedStatus"], "NOT_MINTED")
        self.assertTrue(self.candidate("active-user", period)["alreadyExists"])
        self.assertEqual(self.candidate_count(), 1)

    def test_invalid_or_unavailable_node_never_creates_candidates(self):
        period = self.program["firstAwardPeriod"]
        self.add_session("active-user", period)
        changes = {
            "unverified": {"verified": False}, "false_success": {"success": False},
            "missing_date": {"gameActivatedAt": 0}, "boolean_date": {"gameActivatedAt": True},
            "different_version": {"version": "untrusted"}, "monthly_43": {"awardCount": 43},
            "two_each_month": {"maxAwardsPerMonth": 2}, "missing_genesis": {"legacyGenesisHex": ""},
            "altered_period": {"awardPeriods": ["2099-01"] * 12},
        }
        for name, change in changes.items():
            with self.subTest(name=name):
                data = copy.deepcopy(self.program)
                data.update(change)
                self.body = json.dumps(data).encode()
                self.assertFalse(self.candidate("active-user", period)["ok"])
        self.body = b"not json"
        self.assertFalse(self.candidate("active-user", period)["ok"])
        self.body = b" " * (bridge.MAX_RESPONSE_BYTES + 1)
        self.assertFalse(self.candidate("active-user", period)["ok"])
        self.status, self.body = 503, b"{}"
        self.assertFalse(self.candidate("active-user", period)["ok"])
        with patch.object(bridge.urllib.request.OpenerDirector, "open", side_effect=TimeoutError):
            self.assertFalse(self.candidate("active-user", period)["ok"])
        self.assertEqual(self.candidate_count(), 0)

    def test_token_only_header_and_no_redirect(self):
        self.assertIsNotNone(bridge.fetch_verified_nft_game_program_v1())
        path, headers = self.requests[-1]
        self.assertEqual(path, bridge.PROGRAM_PATH)
        self.assertEqual(headers.get("X-Qc-Api-Key"), "test-only-program-token")
        self.assertNotIn("token", path)
        before = len(self.requests)
        with patch.dict(os.environ, {"QC_API_TOKEN": ""}):
            self.assertIsNone(bridge.fetch_verified_nft_game_program_v1())
        self.assertEqual(len(self.requests), before)
        self.status, self.redirect = 302, self.base+"/should-never-be-followed"
        self.assertIsNone(bridge.fetch_verified_nft_game_program_v1())
        self.assertEqual(len(self.requests), before+1)
        self.status, self.redirect, self.content_type = 200, None, "text/html"
        self.assertIsNone(bridge.fetch_verified_nft_game_program_v1())

    def test_periods_and_verified_session_gate_keep_limits(self):
        periods = self.api["_monthly_nft_award_periods_from_activation_v1"](self.program["gameActivatedAt"])
        self.assertEqual(periods, tuple(self.program["awardPeriods"]))
        self.assertEqual(len(periods), 12)
        self.assertFalse(self.candidate("no-session", periods[0])["ok"])
        self.add_session("unverified-user", periods[0], verified=0)
        self.assertFalse(self.candidate("unverified-user", periods[0])["ok"])
        self.add_session("last-month-user", periods[-1])
        self.assertFalse(self.candidate("last-month-user", periods[-1])["ok"])
        # Advance only this isolated test clock before admitting December.
        self.api["_now"] = lambda: self.month_time(periods[-1])
        self.assertTrue(self.candidate("last-month-user", periods[-1])["ok"])
        activated_period = datetime.datetime.fromtimestamp(self.program["gameActivatedAt"], tz=datetime.timezone.utc).strftime("%Y-%m")
        self.add_session("too-early", activated_period)
        self.assertFalse(self.candidate("too-early", activated_period)["ok"])
        y, m = map(int, periods[-1].split("-"))
        next_month = f"{y+1:04d}-01" if m == 12 else f"{y:04d}-{m+1:02d}"
        self.add_session("too-late", next_month)
        self.assertFalse(self.candidate("too-late", next_month)["ok"])
        self.assertEqual(self.candidate_count(), 1)


if __name__ == "__main__":
    unittest.main(verbosity=2)
