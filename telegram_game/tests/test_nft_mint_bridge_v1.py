"""Real Go-produced public contract + loopback HTTP + temporary SQLite.

Set QMA0066_TG_MINT_CLIENT_CONTRACT_PATH to refresh the contract with the Go
runtime test. The bundled fixture is also emitted by that exact test; there is
no hand-written successful mint receipt fallback.
"""

import concurrent.futures
import copy
import datetime
import hashlib
import http.server
import json
import os
import socket
import sqlite3
import subprocess
import sys
import tempfile
import threading
import unittest
from contextlib import closing, contextmanager
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
from telegram_game.utils import nft_mint_bridge_v1 as bridge


def address_for_owner(owner):
    raw = b"\0" + bytes.fromhex(owner)
    raw += hashlib.sha256(hashlib.sha256(raw).digest()).digest()[:4]
    alphabet, encoded, number = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "", int.from_bytes(raw, "big")
    while number:
        number, remainder = divmod(number, 58)
        encoded = alphabet[remainder] + encoded
    return "1" * (len(raw) - len(raw.lstrip(b"\0"))) + encoded


class NFTMintBridgeTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        path = Path(os.getenv("QMA0066_TG_MINT_CLIENT_CONTRACT_PATH") or Path(__file__).parent / "fixtures/nft_game_mint_client_contract_v1.json")
        cls.contract = json.loads(path.read_text(encoding="utf-8-sig"))

    def setUp(self):
        # These tests isolate transport/persistence. The full evidence gate is
        # exercised without this patch by test_nft_verified_lifecycle_v1.py.
        self.evidence_patch = patch.object(bridge, "_lifecycle_evidence", return_value=["test-only-draw", "test-only-final-approval"])
        self.evidence_patch.start()
        self.addCleanup(self.evidence_patch.stop)
        self.temp = tempfile.TemporaryDirectory(prefix="qma-tg-mint-bridge-")
        self.root = Path(self.temp.name)
        self.db = self.root / "game.sqlite"
        self.payload = copy.deepcopy(self.contract["request"])
        self.program = copy.deepcopy(self.contract["program"])
        self.receipt = copy.deepcopy(self.contract["receipt"])
        self.period = self.payload["record"]["awardPeriod"]
        self.address = address_for_owner(self.payload["record"]["recipientPubKeyHash"])
        self.catalog_id = "test-stock-1"
        self.catalog_path = self.root / "nft/catalog/monthly_rare_nft_stock_2026.json"
        self.catalog_path.parent.mkdir(parents=True)
        items = []
        for i, period in enumerate(self.program["awardPeriods"]):
            image = self.root / f"images/{i}.png"
            metadata = self.root / f"metadata/{i}.json"
            image.parent.mkdir(exist_ok=True)
            metadata.parent.mkdir(exist_ok=True)
            image.write_bytes(b"test-only-artwork-" + str(i).encode())
            metadata.write_bytes(self.contract["metadata"].encode() if period == self.period else json.dumps({"test-only": i}).encode())
            items.append({"id": f"test-stock-{i+1}", "period": period, "name": f"test-name-{i}", "concept": f"test-concept-{i}", "rarity": "rare", "finalReady": True,
                          "imagePath": str(image.relative_to(self.root)), "metadataPath": str(metadata.relative_to(self.root)),
                          "finalImageSha256": hashlib.sha256(image.read_bytes()).hexdigest(), "finalMetadataSha256": hashlib.sha256(metadata.read_bytes()).hexdigest(),
                          "artworkStatus": "FINAL_READY", "metadataStatus": "FINAL_READY", "uniquenessCheck": "PASSED", "aiReviewStatus": "APPROVED", "adminApprovalStatus": "APPROVED"})
        self.catalog = {"totalPlanned": 12, "awardsPerMonth": 1, "items": items}
        self.catalog_path.write_text(json.dumps(self.catalog))
        at = int(datetime.datetime.strptime(self.period + "-10", "%Y-%m-%d").replace(tzinfo=datetime.timezone.utc).timestamp())
        with self.transaction() as conn:
            conn.executescript("""CREATE TABLE players(player_id TEXT PRIMARY KEY,qc_address TEXT);
                CREATE TABLE monthly_nft_candidates(candidate_id TEXT PRIMARY KEY,user_id TEXT,period TEXT,ai_review_status TEXT,ai_risk_level TEXT,admin_status TEXT,winner_status TEXT,minted_status TEXT,updated_at INTEGER);
                CREATE TABLE monthly_nft_verified_game_sessions(user_id TEXT,activity_verified INTEGER,completed_at INTEGER);""")
            conn.execute("INSERT INTO players VALUES(?,?)", ("active-user", self.address))
            conn.execute("INSERT INTO monthly_nft_candidates VALUES(?,?,?,?,?,?,?,?,?)", ("candidate-1", "active-user", self.period, "AI_REVIEW_PASSED", "LOW", "APPROVED_BY_ADMIN", "SELECTED_MONTHLY_WINNER", "NOT_MINTED", 0))
            conn.execute("INSERT INTO monthly_nft_verified_game_sessions VALUES(?,?,?)", ("active-user", 1, at))
        self.calls, self.mint_posts, self.logical_mints = [], 0, 0
        self.minted, self.mode = False, "normal"
        self.server_lock = threading.Lock()
        owner = self

        class Handler(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                owner.calls.append((self.path, dict(self.headers)))
                self.send_json(owner.program)

            def do_POST(self):
                body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
                payload = json.loads(body)
                with owner.server_lock:
                    owner.calls.append((self.path, dict(self.headers)))
                    if payload != owner.payload:
                        return self.send_json({"success": False}, 422)
                    if owner.mode == "redirect":
                        self.send_response(302)
                        self.send_header("Location", "/must-not-follow")
                        self.end_headers()
                        return
                    if self.path.endswith("/mint"):
                        owner.mint_posts += 1
                        if not owner.minted:
                            owner.logical_mints += 1
                            owner.minted = True
                        if owner.mode in {"lost", "pending"}:
                            self.close_connection = True
                            self.connection.shutdown(socket.SHUT_RDWR)
                            self.connection.close()
                            return
                    if owner.mode == "unavailable" or (owner.mode == "pending" and owner.minted):
                        return self.send_json({}, 503)
                    if not owner.minted:
                        return self.send_json({}, 404)
                    receipt = copy.deepcopy(owner.receipt)
                    receipt["alreadyMinted"] = True
                    if owner.mode == "wrong_recipient":
                        receipt["recipientPubKeyHash"] = "22" * 20
                    if owner.mode == "unpersisted":
                        receipt["persisted"] = False
                    self.send_json(receipt)

            def send_json(self, value, status=200):
                self.send_response(status)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(json.dumps(value).encode())

            def log_message(self, *_):
                pass

        self.server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.env = patch.dict(os.environ, {"QC_API_BASE": f"http://127.0.0.1:{self.server.server_port}", "QC_API_TOKEN": "test-only-node-token",
            "QC_ENABLE_CHAIN_NFT_MINT": "1", "QC_ENABLE_TG_NFT_MINT_BRIDGE": "1", "TELEGRAM_GAME_CHAIN_MODE": "mainnet", "QC_TESTNET": "", "QC_PUBLIC_TESTNET": "", "QC_NET_MAGIC": ""})
        self.env.start()

    def tearDown(self):
        self.env.stop()
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(timeout=2)
        self.temp.cleanup()

    def connect(self):
        return sqlite3.connect(self.db, timeout=10)

    @contextmanager
    def transaction(self):
        # sqlite3's context manager commits/rolls back but does not close.
        # Release the handle before TemporaryDirectory cleanup on Windows.
        with closing(self.connect()) as conn:
            with conn:
                yield conn

    def invoke(self, connect=None):
        return bridge.controlled_monthly_nft_mint_v1(connect or self.connect, self.root, "candidate-1", self.catalog_id, self.payload)

    def status(self):
        with self.transaction() as conn:
            return conn.execute("SELECT minted_status FROM monthly_nft_candidates").fetchone()[0]

    def test_canonical_hashes_match_real_go_record_and_receipt(self):
        _, token, digest, _ = bridge.canonical_award_v1(self.payload)
        self.assertEqual(token, self.receipt["tokenID"])
        self.assertEqual(digest, self.receipt["recordDigest"])
        self.assertEqual(bridge.validate_receipt_v1(self.payload, self.receipt), self.receipt)

    def test_confirmed_award_updates_candidate_and_ledger_atomically(self):
        result = self.invoke()
        self.assertTrue(result["ok"], result)
        self.assertEqual(self.status(), "MINTED_CONFIRMED")
        with self.transaction() as conn:
            state, receipt = conn.execute("SELECT status,receipt_json FROM monthly_nft_mint_intents_v1").fetchone()
        self.assertEqual(state, "CONFIRMED")
        self.assertEqual(json.loads(receipt)["recordDigest"], self.receipt["recordDigest"])
        self.assertTrue(self.invoke()["ok"])
        self.assertEqual(self.mint_posts, 1)
        self.assertEqual(self.logical_mints, 1)

    def test_lost_mint_response_is_reconciled_without_second_mint(self):
        self.mode = "lost"
        result = self.invoke()
        self.assertTrue(result["ok"], result)
        self.assertEqual(self.mint_posts, 1)
        self.assertEqual(self.logical_mints, 1)

    def test_pending_response_survives_retry_without_false_award(self):
        self.mode = "pending"
        result = self.invoke()
        self.assertFalse(result["ok"])
        self.assertEqual(self.status(), "NOT_MINTED")
        with self.transaction() as conn:
            self.assertEqual(conn.execute("SELECT status,receipt_json FROM monthly_nft_mint_intents_v1").fetchone(), ("PENDING", None))
        self.mode = "normal"
        self.assertTrue(self.invoke()["ok"])
        self.assertEqual(self.mint_posts, 1)

    def test_wrong_recipient_or_ram_only_receipt_cannot_credit_user(self):
        for mode in ("wrong_recipient", "unpersisted"):
            with self.subTest(mode=mode):
                self.mode = mode
                self.minted = True
                self.assertFalse(self.invoke()["ok"])
                self.assertEqual(self.status(), "NOT_MINTED")
        self.assertEqual(self.mint_posts, 0)

    def test_disabled_or_invalid_evidence_never_reaches_node(self):
        for flag in ("QC_ENABLE_CHAIN_NFT_MINT", "QC_ENABLE_TG_NFT_MINT_BRIDGE"):
            with patch.dict(os.environ, {flag: "0"}):
                self.assertFalse(self.invoke()["ok"])
        self.payload["record"]["recipientPubKeyHash"] = "22" * 20
        self.assertFalse(self.invoke()["ok"])
        self.assertEqual(self.calls, [])
        self.assertEqual(self.status(), "NOT_MINTED")

    def test_testnet_and_local_modes_never_send_mint_requests(self):
        for mode in ("testnet", "local"):
            with patch.dict(os.environ, {"TELEGRAM_GAME_CHAIN_MODE": mode}):
                result = self.invoke()
                self.assertEqual(result["error"], "monthly_nft_mainnet_mode_required")
        self.assertEqual(self.calls, [])

    def test_receipt_commitment_scope_and_height_mutations_are_rejected(self):
        changes = {"tokenID": "ab" * 32, "recordDigest": "ab" * 32,
                   "legacyGenesisHex": "ab" * 32, "metadataHash": "ab" * 32,
                   "networkID": "other-network", "campaignID": "other-campaign",
                   "awardPeriod": "1999-01", "serial": True,
                   "mintHeight": True, "chainHeight": 0,
                   "persistedTipHash": "ab" * 32, "alreadyMinted": 1}
        for field, value in changes.items():
            with self.subTest(field=field):
                receipt = copy.deepcopy(self.receipt)
                receipt[field] = value
                with self.assertRaises(bridge.NFTMintBridgeError):
                    bridge.validate_receipt_v1(self.payload, receipt)

    def test_catalog_draft_tampering_and_wrong_program_fail_closed(self):
        original = copy.deepcopy(self.catalog)
        for kind in ("draft", "tampered", "wrong_periods"):
            self.catalog = copy.deepcopy(original)
            if kind == "draft":
                self.catalog["items"][0]["finalReady"] = False
            elif kind == "tampered":
                self.catalog["items"][0]["finalMetadataSha256"] = "ab" * 32
            else:
                self.catalog["items"][0]["period"] = "1999-01"
            self.catalog_path.write_text(json.dumps(self.catalog))
            self.assertFalse(self.invoke()["ok"], kind)
        self.assertEqual(self.mint_posts, 0)
        self.assertEqual(self.status(), "NOT_MINTED")

    def test_unapproved_inactive_or_changed_linked_wallet_cannot_mint(self):
        for field, bad, good in (("ai_review_status", "FLAGGED", "AI_REVIEW_PASSED"), ("admin_status", "REJECTED", "APPROVED_BY_ADMIN"), ("winner_status", "NOT_SELECTED", "SELECTED_MONTHLY_WINNER")):
            with self.transaction() as conn:
                conn.execute(f"UPDATE monthly_nft_candidates SET {field}=?", (bad,))
            self.assertFalse(self.invoke()["ok"], field)
            with self.transaction() as conn:
                conn.execute(f"UPDATE monthly_nft_candidates SET {field}=?", (good,))
        with self.transaction() as conn:
            conn.execute("DELETE FROM monthly_nft_verified_game_sessions")
        self.assertFalse(self.invoke()["ok"])
        with self.transaction() as conn:
            conn.execute("UPDATE players SET qc_address=?", (address_for_owner("33" * 20),))
        self.assertFalse(self.invoke()["ok"])
        self.assertEqual(self.mint_posts, 0)

    def test_redirect_never_forwards_token_or_mints(self):
        self.mode = "redirect"
        self.assertFalse(self.invoke()["ok"])
        self.assertFalse(any(path == "/must-not-follow" for path, _ in self.calls))
        self.assertEqual(self.mint_posts, 0)
        self.assertTrue(all(headers.get("X-Qc-Api-Key") == "test-only-node-token" for _, headers in self.calls))
        self.assertTrue(all("token=" not in path for path, _ in self.calls))

    def test_database_failure_after_chain_mint_recovers_by_receipt(self):
        owner = self
        calls = 0

        class FailCommit:
            def __init__(self):
                self.conn = owner.connect()
            def execute(self, sql, args=()):
                if "SET minted_status=" in sql:
                    raise sqlite3.OperationalError("test-only-db-failure")
                return self.conn.execute(sql, args)
            def commit(self):
                self.conn.commit()
            def close(self):
                self.conn.close()

        def connect():
            nonlocal calls
            calls += 1
            return FailCommit() if calls == 2 else self.connect()

        result = self.invoke(connect)
        self.assertFalse(result["ok"])
        self.assertEqual(self.status(), "NOT_MINTED")
        with self.transaction() as conn:
            self.assertEqual(conn.execute("SELECT status FROM monthly_nft_mint_intents_v1").fetchone()[0], "PENDING")
        self.assertTrue(self.invoke()["ok"])
        self.assertEqual(self.mint_posts, 1)

    def test_concurrent_workers_confirm_one_canonical_award(self):
        with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:
            results = list(pool.map(lambda _: self.invoke(), range(3)))
        self.assertTrue(all(x["ok"] for x in results), results)
        self.assertEqual(self.logical_mints, 1)
        self.assertEqual(self.status(), "MINTED_CONFIRMED")
        with self.transaction() as conn:
            self.assertEqual(conn.execute("SELECT COUNT(*) FROM monthly_nft_mint_intents_v1").fetchone()[0], 1)

    def test_reorg_missing_receipt_requires_reconciliation_without_remint(self):
        self.assertTrue(self.invoke()["ok"])
        self.assertEqual(self.mint_posts, 1)
        self.minted = False  # Canonical branch no longer contains the mint.
        for _ in range(2):
            self.assertEqual(self.invoke()["error"], "canonical_mint_receipt_missing")
            self.assertEqual(self.status(), "MINT_RECONCILIATION_REQUIRED")
        self.assertEqual(self.mint_posts, 1)
        self.minted = True  # Same exact record returned to the canonical chain.
        self.assertTrue(self.invoke()["ok"])
        self.assertEqual(self.status(), "MINTED_CONFIRMED")
        self.assertEqual(self.mint_posts, 1)

    def test_admin_route_authentication_and_default_disabled_mode(self):
        code = r'''
import os
from fastapi.testclient import TestClient
from telegram_game.dev_api import app
os.environ['TELEGRAM_GAME_ADMIN_TOKEN'] = 'test-only-admin-token'
os.environ['QC_ENABLE_TG_NFT_MINT_BRIDGE'] = '0'
client = TestClient(app)
path = '/api/admin/nft/monthly/controlled-mint'
body = {'candidateId': 'candidate-1', 'catalogItemId': 'test-stock-1'}
assert client.post(path,json=body).status_code == 401
assert client.post(path+'?token=test-only-admin-token',json=body).status_code == 401
response = client.post(path,json=body,headers={'X-Admin-Token':'test-only-admin-token'})
assert response.status_code == 200, response.text
assert response.json()['error'] == 'telegram_nft_mint_bridge_disabled', response.text
'''
        env = os.environ.copy()
        env["PYTHONPATH"] = str(ROOT) + os.pathsep + env.get("PYTHONPATH", "")
        result = subprocess.run([sys.executable, "-c", code], cwd=self.root, env=env, text=True, capture_output=True, timeout=30)
        self.assertEqual(result.returncode, 0, result.stderr[-2000:])
        self.assertEqual(self.mint_posts, 0)


if __name__ == "__main__":
    unittest.main(verbosity=2)
