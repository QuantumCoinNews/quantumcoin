"""Isolated adversarial wallet/ledger/transport contracts. No live data or RPC."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile
if os.getenv("QC_WALLET_LEDGER_TEST_CHILD")!="1":
    with tempfile.TemporaryDirectory(prefix="qc-wallet-ledger-") as directory:
        env=os.environ.copy()
        env.update(QC_WALLET_LEDGER_TEST_CHILD="1",QC_GAME_DB=str(Path(directory)/"isolated.sqlite"),
                   TELEGRAM_BOT_TOKEN="test-wallet-ledger-bot",TELEGRAM_GAME_ADMIN_TOKEN="test-wallet-ledger-admin",
                   QC_WALLET_LINK_NETWORK_ID="test-only-network",QC_WALLET_LINK_GENESIS_HEX="11"*32,QC_API_TOKEN="test-node-token")
        raise SystemExit(subprocess.call([sys.executable,str(Path(__file__).resolve())],cwd=directory,env=env))

import asyncio
import concurrent.futures
from contextlib import closing
import hashlib
from http.server import BaseHTTPRequestHandler,ThreadingHTTPServer
import json
import sqlite3
import threading
import time
import unittest
from unittest.mock import patch
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
sys.path.insert(0,str(Path(__file__).resolve().parents[2]/"scripts"))
from fastapi.testclient import TestClient
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import ec
from telegram_game import dev_api as api, wallet_link, withdraw_ledger as ledger, player_auth, game_pow
from telegram_game.utils import qc_api
from sign_wallet_link import sign


def address_for(key):
    public=key.public_key().public_bytes(serialization.Encoding.X962,serialization.PublicFormat.UncompressedPoint)
    raw=b"\x00"+hashlib.new("ripemd160",hashlib.sha256(public[1:]).digest()).digest()
    raw+=hashlib.sha256(hashlib.sha256(raw).digest()).digest()[:4]
    number=int.from_bytes(raw,"big"); chars=""
    alphabet="123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
    while number:number,mod=divmod(number,58);chars=alphabet[mod]+chars
    return "1"*(len(raw)-len(raw.lstrip(b"\x00")))+chars


class SecurityTests(unittest.TestCase):
    def setUp(self):
        self.actor="tg_90101";self.other="tg_90102";self.now=int(time.time())
        self.key=ec.generate_private_key(ec.SECP256K1());self.address=address_for(self.key)
        self.secret=f"{self.key.private_numbers().private_value:064x}"
        with closing(api._db()) as conn,conn:
            wallet_link.schema(conn);ledger.schema(conn)
            for table in ("wallet_link_challenges_v1","wallet_link_proofs_v1","wallet_link_legacy_history_v1","tgwt_withdraw_confirmations_v1","tgwt_withdraw_reservations_v1","withdraw_requests","tgwt_events","tgwt_state","request_limits_v1"):
                conn.execute("DELETE FROM "+table)
            for actor in (self.actor,self.other):
                api._touch_player(conn,actor,None)
                api._ensure_qc_address_column(conn)
                conn.execute("UPDATE players SET qc_address='' WHERE player_id=?",(actor,))
                conn.execute("INSERT INTO tgwt_state(user_id,earned_u,verified_u) VALUES(?,?,?)",(actor,10_000_000,10_000_000))
        self.client=TestClient(api.app)
        credentials=player_auth.issue_session(api._db,self.actor,api._telegram_bot_token(),self.now)
        self.headers={"Authorization":"Bearer "+credentials["accessToken"]}

    def proof(self,actor=None):
        actor=actor or self.actor
        issued=wallet_link.issue(api._db,actor,self.address,self.now)
        self.assertTrue(issued["ok"],issued)
        return sign(issued["challenge"],self.secret,self.now)

    def prove(self,p,actor=None,now=None):
        return wallet_link.consume(api._db,actor or self.actor,p["address"],p["nonce"],p["publicKey"],p["signature"],now or self.now)

    def reserve(self,nonce="request-00000001",amount="2"):
        return ledger.reserve(api._db,self.actor,"0x"+"12"*20,amount,nonce,self.now)

    def balances(self):
        with closing(api._db()) as conn:return ledger.amounts(conn,self.actor)

    def test_wallet_proof_http_and_replay(self):
        denied=self.client.post("/api/wallet/link-qc-address",headers=self.headers,json={"playerId":self.actor,"address":self.address}).json()
        self.assertEqual(denied["error"],"wallet_ownership_proof_required")
        issued=self.client.post("/api/wallet/link-challenge",headers=self.headers,json={"playerId":self.actor,"address":self.address}).json()
        p=sign(issued["challenge"],self.secret,int(time.time()))
        result=self.client.post("/api/wallet/link-qc-address",headers=self.headers,json=p).json()
        self.assertTrue(result["ownershipVerified"],result)
        self.assertFalse(self.prove(p)["ok"])

    def test_wallet_wrong_actor_key_expiry_and_domain(self):
        p=self.proof()
        self.assertFalse(self.prove(p,self.other)["ok"])
        self.assertFalse(self.prove(p,now=self.now+300)["ok"])
        wrong=self.proof();wrong["signature"]="00"*64
        self.assertFalse(self.prove(wrong)["ok"])
        with patch.dict(os.environ,{"QC_WALLET_LINK_GENESIS_HEX":"22"*32}):self.assertFalse(self.prove(p)["ok"])
        self.assertTrue(self.prove(p)["ok"])

    def test_legacy_squatter_reclaimed_but_proven_owner_protected(self):
        with closing(api._db()) as conn,conn:conn.execute("UPDATE players SET qc_address=? WHERE player_id=?",(self.address,self.other))
        self.assertTrue(self.prove(self.proof())["ok"])
        with closing(api._db()) as conn:
            self.assertEqual(conn.execute("SELECT qc_address FROM players WHERE player_id=?",(self.other,)).fetchone()[0],"")
            self.assertEqual(conn.execute("SELECT COUNT(*) FROM wallet_link_legacy_history_v1").fetchone()[0],1)
        other=self.proof(self.other)
        self.assertFalse(self.prove(other,self.other)["ok"])

    def test_wallet_proof_concurrent_consumption_only_once(self):
        p=self.proof()
        with concurrent.futures.ThreadPoolExecutor(2) as pool:results=list(pool.map(lambda _:self.prove(p),range(2)))
        self.assertEqual(sum(r["ok"] for r in results),1)

    def test_wallet_missing_network_and_fake_address_fail_closed(self):
        with patch.dict(os.environ,{"QC_WALLET_LINK_GENESIS_HEX":""}):self.assertFalse(wallet_link.issue(api._db,self.actor,self.address,self.now)["ok"])
        self.assertFalse(wallet_link.issue(api._db,self.actor,"QC_fake",self.now)["ok"])

    def test_withdraw_reserves_does_not_claim_paid(self):
        r=self.reserve();self.assertTrue(r["ok"],r)
        b=self.balances();self.assertEqual((b["verified_u"],b["reserved_u"],b["withdrawn_u"]),(8_000_000,2_000_000,0))
        self.assertTrue(b["accountingConsistent"])
        self.assertFalse(r["onChainPaid"])
        replay=self.reserve();self.assertTrue(replay["alreadyRecorded"])
        self.assertFalse(self.reserve(amount="3")["ok"])
        self.assertEqual(self.balances()["reserved_u"],2_000_000)

    def test_withdraw_exact_units_invalid_input_no_writes(self):
        for amount in ("0.0000001","NaN","Infinity","-1","0","3000001"):
            self.assertFalse(self.reserve(amount=amount)["ok"],amount)
        self.assertFalse(ledger.reserve(api._db,self.actor,"0x"+"zz"*20,"1","request-invalid",self.now)["ok"])
        self.assertTrue(self.reserve(amount="0.000001")["ok"])
        self.assertEqual(self.balances()["reserved_u"],1)

    def test_withdraw_cancel_reject_idempotent_and_actor_scoped(self):
        r=self.reserve();rid=r["requestId"]
        self.assertFalse(ledger.release(api._db,rid,self.other,self.now)["ok"])
        self.assertTrue(ledger.release(api._db,rid,self.actor,self.now)["ok"])
        self.assertTrue(ledger.release(api._db,rid,self.actor,self.now)["alreadyRecorded"])
        self.assertEqual(self.balances()["verified_u"],10_000_000)
        rid=self.reserve("request-00000002")["requestId"]
        self.assertTrue(ledger.release(api._db,rid,None,self.now,operator=True,reject=True)["ok"])
        self.assertEqual(self.balances()["reserved_u"],0)

    def test_concurrent_reservations_no_overspend(self):
        with concurrent.futures.ThreadPoolExecutor(2) as pool:
            results=list(pool.map(lambda i:self.reserve("concurrent-"+str(i),"8"),range(2)))
        self.assertEqual(sum(r["ok"] for r in results),1)
        self.assertEqual(self.balances()["reserved_u"],8_000_000)

    def test_withdraw_rollback_all_tables(self):
        with closing(api._db()) as conn,conn:conn.execute("CREATE TRIGGER test_abort_withdraw BEFORE INSERT ON tgwt_events BEGIN SELECT RAISE(ABORT,'test rollback'); END")
        try:
            with self.assertRaises(sqlite3.IntegrityError):self.reserve()
            self.assertEqual(self.balances()["verified_u"],10_000_000)
            with closing(api._db()) as conn:self.assertEqual(conn.execute("SELECT COUNT(*) FROM withdraw_requests").fetchone()[0],0)
        finally:
            with closing(api._db()) as conn,conn:conn.execute("DROP TRIGGER test_abort_withdraw")

    def test_confirmation_requires_verified_evidence_and_once(self):
        rid=self.reserve()["requestId"];tx="0x"+"ab"*32
        self.assertFalse(ledger.confirm(api._db,rid,tx,lambda *args:{"finalized":True},self.now)["ok"])
        def fixture(request,to,amount,tx):return {"requestId":request,"recipient":to,"amount_u":amount,"txHash":tx,"chainId":56,"finalized":True,"logIndex":1}
        self.assertTrue(ledger.confirm(api._db,rid,tx,fixture,self.now)["ok"])
        self.assertTrue(ledger.confirm(api._db,rid,tx,fixture,self.now)["alreadyRecorded"])
        self.assertEqual((self.balances()["reserved_u"],self.balances()["withdrawn_u"]),(0,2_000_000))
        self.assertFalse(ledger.release(api._db,rid,self.actor,self.now)["ok"])

    def test_legacy_ledger_not_silently_repaired(self):
        with closing(api._db()) as conn,conn:
            conn.execute("UPDATE tgwt_state SET withdrawn_u=1 WHERE user_id=?",(self.actor,))
        self.assertEqual(self.reserve()["error"],"legacy_ledger_reconciliation_required")
        self.assertEqual(self.balances()["withdrawn_u"],1)

    def test_old_admin_review_amount_is_not_certified_by_balanced_totals(self):
        with closing(api._db()) as conn,conn:
            conn.execute("UPDATE tgwt_state SET verified_u=8000000,withdrawn_u=2000000 WHERE user_id=?",(self.actor,))
            conn.execute("INSERT INTO withdraw_requests VALUES(?,?,?,?,?,?,?,?,?)",("old-unproven",self.actor,"TGWT",2_000_000,"0x"+"12"*20,"ADMIN_REVIEW",None,self.now,self.now))
        balance=self.balances()
        self.assertTrue(balance["accountingConsistent"])
        self.assertEqual(balance["legacyUnreconciledRequests"],1)
        self.assertEqual(self.reserve()["error"],"legacy_ledger_reconciliation_required")

    def test_http_reserve_status_cancel_and_operator_boundary(self):
        response=self.client.post("/api/v1/withdraw/request",headers=self.headers,json={"userId":self.actor,"asset":"TGWT","amount":"2.000001","toAddress":"0x"+"12"*20,"nonce":"http-request-0001"})
        result=response.json();self.assertTrue(result["ok"],result)
        self.assertEqual(result["balances"]["withdrawn_u"],0)
        request=result["requestId"]
        status=self.client.get("/api/v1/withdraw/status",headers=self.headers,params={"id":request}).json()
        self.assertEqual(status["request"]["amount_u"],2_000_001)
        self.assertEqual(self.client.post("/api/admin/tgwt/withdraw/reject",headers=self.headers,json={"requestId":request}).status_code,401)
        result=self.client.post("/api/v1/withdraw/cancel",headers=self.headers,json={"requestId":request}).json()
        self.assertTrue(result["ok"],result)
        self.assertEqual(result["balances"]["verified_u"],10_000_000)

    def test_reset_key_transport_and_unverified_payout_are_blocked(self):
        self.assertEqual(self.client.post("/api/dev/reset").status_code,401)
        admin={"Authorization":"Bearer test-wallet-ledger-admin"}
        self.assertEqual(self.client.post("/api/dev/reset",headers=admin).status_code,410)
        bad=self.client.post("/api/wallet/send-qc",headers=self.headers,json={"playerId":self.actor,"transaction":"{}","privHex":"test-never-send"})
        self.assertIn(bad.status_code,(404,422))
        self.assertEqual(self.client.post("/api/admin/tgwt/withdraw/confirm",headers=self.headers,json={"requestId":"anything"}).status_code,401)
        self.assertEqual(self.client.post("/api/admin/tgwt/withdraw/confirm",headers=admin,json={"requestId":"anything"}).status_code,503)

    def test_node_bridge_all_operations_authenticated_and_redirects_blocked(self):
        hits=[]
        class Node(BaseHTTPRequestHandler):
            redirect_mode=False
            def log_message(self,*args):pass
            def do_GET(self):
                hits.append((self.path,self.headers.get("X-QC-API-KEY")))
                if self.headers.get("X-QC-API-KEY")!="test-node-token":self.send_response(401);self.end_headers();return
                if self.redirect_mode and self.path!="/capture":
                    self.send_response(302);self.send_header("Location","/capture");self.end_headers();return
                self.send_response(200);self.send_header("Content-Type","application/json");self.end_headers();self.wfile.write(b'{"ok":true,"success":true,"id":"__precheck__","inBlock":false,"inMempool":false,"confirmedHeight":0}')
            def do_POST(self):
                self.rfile.read(int(self.headers.get("Content-Length",0)));self.do_GET()
        server=ThreadingHTTPServer(("127.0.0.1",0),Node);thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start()
        try:
            with patch.dict(os.environ,{"QC_API_BASE":f"http://127.0.0.1:{server.server_port}"}):
                self.assertTrue(asyncio.run(qc_api.fetch_health())["ok"])
                self.assertTrue(api._fetch_qc_node_health_sync()["ok"])
                self.assertTrue(asyncio.run(qc_api.fetch_address_balance(self.address))["ok"])
                self.assertTrue(asyncio.run(qc_api.fetch_tx_status("abc"))["ok"])
                self.assertTrue(asyncio.run(qc_api.fetch_submit_signed_tx({"Inputs":[],"Outputs":[]}))["ok"])
                self.assertTrue(asyncio.run(game_pow.node_request("/api/mine/job"))["ok"])
                self.assertTrue(asyncio.run(game_pow.node_request("/api/mine/submit",{}))["ok"])
                self.assertTrue(asyncio.run(api.qc_check_tx_status_endpoint_ready())["ready"])
                self.assertEqual(len(hits),8)
                self.assertTrue(all(token=="test-node-token" for _,token in hits))
                with patch.dict(os.environ,{"QC_API_TOKEN":"wrong-test-token"}):
                    self.assertFalse(asyncio.run(qc_api.fetch_health())["ok"])
                Node.redirect_mode=True
                self.assertFalse(asyncio.run(qc_api.fetch_health())["ok"])
                self.assertFalse(api._fetch_qc_node_health_sync()["ok"])
                self.assertFalse(asyncio.run(qc_api.fetch_address_balance(self.address))["ok"])
                self.assertFalse(asyncio.run(qc_api.fetch_tx_status("abc"))["ok"])
                self.assertFalse(asyncio.run(qc_api.fetch_submit_signed_tx({"Inputs":[],"Outputs":[]}))["ok"])
                self.assertFalse(asyncio.run(game_pow.node_request("/api/mine/job"))["ok"])
                self.assertFalse(asyncio.run(game_pow.node_request("/api/mine/submit",{}))["ok"])
                self.assertFalse(asyncio.run(api.qc_check_tx_status_endpoint_ready())["ready"])
                self.assertFalse(any(path=="/capture" for path,_ in hits))
        finally:server.shutdown();server.server_close();thread.join()


if __name__=="__main__":unittest.main(verbosity=2)
