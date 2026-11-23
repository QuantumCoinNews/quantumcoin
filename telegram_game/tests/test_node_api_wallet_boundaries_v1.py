"""Isolated HTTP bridge privacy, health and authoritative score contracts."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile

if os.environ.get("QC_WALLET_BOUNDARY_TEST_CHILD") != "1":
    with tempfile.TemporaryDirectory(prefix="qc-wallet-boundaries-") as directory:
        env = os.environ.copy()
        env.update(QC_WALLET_BOUNDARY_TEST_CHILD="1", QC_API_BASE="http://127.0.0.1:1", QC_API_TOKEN="test-boundary-token", TELEGRAM_BOT_TOKEN="test-only-bot")
        raise SystemExit(subprocess.call([sys.executable, str(Path(__file__).resolve())], cwd=directory, env=env))

import asyncio
import json
import unittest
from unittest.mock import patch
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import httpx
from fastapi.testclient import TestClient
from telegram_game import dev_api as api
from telegram_game.utils import qc_api, node_score_bridge_v1 as score
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey


class WalletBoundaryTests(unittest.TestCase):
    def test_legacy_private_key_fields_are_rejected_without_echo_or_network(self):
        secret = "sensitive-test-key-that-must-not-appear"
        with TestClient(api.app) as client, patch.object(qc_api.httpx, "AsyncClient", side_effect=AssertionError("network must not be reached")):
            result = client.post("/api/wallet/send-qc", json={"playerId":"tg_1","toAddress":"recipient","amount":"1","privHex":secret})
        self.assertEqual(result.status_code, 401)
        self.assertNotIn(secret, result.text)
        with patch.object(qc_api.httpx,"AsyncClient",side_effect=AssertionError("retired key bridge used network")):
            result=asyncio.run(qc_api.fetch_send_tx("from","to","1",secret))
        self.assertEqual(result["error"],"wallet_local_signing_required")

    def test_signed_submission_requires_verified_telegram_identity(self):
        with TestClient(api.app) as client, patch.object(api,"_db",side_effect=AssertionError("unauthenticated DB access")):
            response=client.post("/api/wallet/send-qc",json={"playerId":"tg_1","transaction":'{"transaction":{"Sender":"owner"}}'})
        self.assertEqual(response.status_code,401)
        self.assertEqual(response.json()["error"],"telegram_session_required")

    def test_bridge_sends_public_signatures_and_exact_integer_with_token(self):
        requests=[]
        class Client:
            def __init__(self,**kwargs): self.kwargs=kwargs
            async def __aenter__(self): return self
            async def __aexit__(self,*args): pass
            async def post(self,url,**kwargs):
                requests.append((url,kwargs,self.kwargs))
                return httpx.Response(200,json={"success":True,"txid":"ab"},request=httpx.Request("POST",url))
        tx={"ID":"public","Sender":"owner","Inputs":[{"TxID":"outpoint","OutIndex":0,"Signature":"public-signature","PubKey":"public-key"}],"Outputs":[{"Amount":9007199254740993,"PubKeyHash":"recipient"}],"Timestamp":"2026-11-03T00:00:00Z","Amount":1}
        with patch.object(qc_api.httpx,"AsyncClient",Client):
            result=asyncio.run(qc_api.fetch_submit_signed_tx(tx))
        self.assertTrue(result["ok"])
        self.assertEqual(requests[0][1]["json"]["transaction"]["Outputs"][0]["Amount"],9007199254740993)
        self.assertEqual(requests[0][1]["headers"]["X-QC-API-KEY"],"test-boundary-token")
        self.assertFalse(requests[0][2]["follow_redirects"])
        tx["Inputs"][0]["priv_hex"]="must-never-forward"
        with patch.object(qc_api.httpx,"AsyncClient",side_effect=AssertionError("nested secret forwarded")):
            self.assertFalse(asyncio.run(qc_api.fetch_submit_signed_tx(tx))["ok"])

    def test_health_distinguishes_reachable_authentication_required(self):
        captured=[]
        class Client:
            def __init__(self,**kwargs): pass
            async def __aenter__(self): return self
            async def __aexit__(self,*args): pass
            async def get(self,url,**kwargs):
                captured.append(kwargs)
                return httpx.Response(401,json={"success":False},headers={"www-authenticate":'Bearer realm="QuantumCoin API"'},request=httpx.Request("GET",url))
        with patch.object(qc_api.httpx,"AsyncClient",Client): result=asyncio.run(qc_api.fetch_health())
        self.assertTrue(result["reachable"])
        self.assertFalse(result["authenticated"])
        self.assertEqual(captured[0]["headers"]["X-QC-API-KEY"],"test-boundary-token")

    def test_score_authority_domain_and_bounds(self):
        event={"version":score.VERSION,"network":"mainnet","genesis":"ab"*32,"player":"tg_123","score":200,"sequence":1,"expires":1700000120}
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/"test-only-ed25519.seed";path.write_bytes(bytes(32))
            signed=score.sign_authoritative_score(event,path)
            Ed25519PrivateKey.from_private_bytes(bytes(32)).public_key().verify(bytes.fromhex(signed["signature"]),score.signing_bytes(event))
        vector=json.loads((Path(__file__).resolve().parents[2]/"game/testdata/score_event.json").read_text())
        self.assertEqual(signed,vector["event"])
        self.assertEqual(score.signing_bytes(event).hex(),vector["signingBytesHex"])
        for field,value in (("score",-1),("score",True),("sequence",0),("player","bad/player")):
            with self.assertRaises(ValueError): score.signing_bytes({**event,field:value})


if __name__=="__main__": unittest.main()
