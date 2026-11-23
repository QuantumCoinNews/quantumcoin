"""Isolated accounting, policy authority and action-idempotency contracts."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile
if os.getenv("QC_EXACT_TEST_CHILD") != "1":
    with tempfile.TemporaryDirectory(prefix="qc-exact-") as directory:
        env = os.environ.copy()
        env.update(QC_EXACT_TEST_CHILD="1", QC_GAME_DB=str(Path(directory)/"state.sqlite3"),
                   TELEGRAM_BOT_TOKEN="test-only-exact-bot", TELEGRAM_GAME_ADMIN_TOKEN="test-only-exact-admin",
                   QC_API_BASE="http://127.0.0.1:1", QC_API_TOKEN="test-only-node")
        raise SystemExit(subprocess.call([sys.executable, str(Path(__file__).resolve())], cwd=directory, env=env))

import asyncio
from contextlib import closing
from decimal import Decimal, localcontext
import json
import sqlite3
import time
import unittest
from unittest.mock import patch, AsyncMock
from types import SimpleNamespace, ModuleType
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives import serialization
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from fastapi.testclient import TestClient
from telegram_game import dev_api as api, runtime, exact_numbers as numbers, player_auth
from telegram_game import reward_evidence as rewards, withdraw_ledger
from telegram_game.utils import qc_api


class ExactTests(unittest.TestCase):
    def setUp(self):
        self.actor = "tg_"+str(time.time_ns())
        with closing(api._db()) as conn, conn:
            api._touch_player(conn, self.actor, None)
        credentials = player_auth.issue_session(api._db, self.actor, api._telegram_bot_token(), int(time.time()))
        self.headers = {"Authorization":"Bearer "+credentials["accessToken"]}
        self.client = TestClient(runtime.app)

    def test_canonical_qc_roundtrips_and_precision(self):
        for text, atoms in (("0",0),("0.000001",1),("25.500001",25500001),("9223372036854.775807",numbers.MAX_INTEGER)):
            self.assertEqual(numbers.parse_qc(text),atoms)
            self.assertEqual(numbers.parse_qc(numbers.format_qc(atoms)),atoms)
        for value in (1.5,True,"NaN","Infinity","-1","1e6","0.0000001","9223372036854.775808"):
            with self.assertRaises(ValueError): numbers.parse_qc(value)

    def test_decimal_conversion_cannot_round_hidden_fraction(self):
        with localcontext() as context:
            context.prec=2
            self.assertEqual(numbers.decimal_atoms(Decimal("1.000001")),1000001)
            for value in (Decimal("1.00000000000000000000000000001"),Decimal("NaN"),Decimal("Infinity"), Decimal("1E+999999"),1.0,True):
                with self.assertRaises(ValueError):numbers.decimal_atoms(value)

    def test_legacy_migration_preserves_rows_without_certifying_them(self):
        with closing(sqlite3.connect(":memory:")) as conn:
            conn.execute("CREATE TABLE players(player_id TEXT,total_qc REAL,total_xp REAL)")
            conn.execute("CREATE TABLE reward_records(amount REAL,xp_amount REAL)")
            conn.execute("INSERT INTO players VALUES('old',?,?)",(float("inf"),1.25))
            conn.execute("INSERT INTO reward_records VALUES(2.5,3.75)")
            before=conn.execute("SELECT * FROM players").fetchone()
            numbers.ensure_schema(conn);numbers.ensure_schema(conn)
            self.assertEqual(conn.execute("SELECT player_id,total_qc,total_xp FROM players").fetchone(),before)
            self.assertEqual(conn.execute("SELECT qc_atoms,xp_units,numeric_version FROM players").fetchone(),(0,0,0))
            self.assertEqual(conn.execute("SELECT amount_atoms,xp_units,numeric_version FROM reward_records").fetchone(),(0,0,0))

    def test_sqlite_rejects_fractional_and_overflow_xp(self):
        with closing(api._db()) as conn,conn:
            with self.assertRaises(sqlite3.IntegrityError):conn.execute("UPDATE players SET xp_units=0.5 WHERE player_id=?",(self.actor,))
            conn.execute("UPDATE players SET xp_units=? WHERE player_id=?",(numbers.MAX_INTEGER,self.actor))
            with self.assertRaises(ValueError):numbers.add_xp(conn,self.actor,1)
            self.assertEqual(conn.execute("SELECT xp_units,typeof(xp_units) FROM players WHERE player_id=?",(self.actor,)).fetchone(),(numbers.MAX_INTEGER,"integer"))

    def test_wallet_ignores_legacy_float_entitlement(self):
        with closing(api._db()) as conn,conn:
            conn.execute("UPDATE players SET total_qc=?,total_xp=? WHERE player_id=?",(float("inf"),1e30,self.actor))
        result=self.client.get("/api/wallet",params={"playerId":self.actor},headers=self.headers)
        self.assertEqual(result.status_code,200,result.text)
        self.assertEqual(result.json()["qc_atoms"],"0")
        self.assertEqual(result.json()["xpTotal"],0)
        self.assertTrue(result.json()["legacyNumericReviewRequired"])

    def test_new_reward_uses_integer_columns_only(self):
        with closing(api._db()) as conn,conn:
            numbers.add_xp(conn,self.actor,65)
            rid=api._reward_record_insert(conn,player_id=self.actor,source="TEST",zone_id="TEST",site_id=1,amount=0,xp_amount=65)
            self.assertEqual(conn.execute("SELECT xp_units,total_xp FROM players WHERE player_id=?",(self.actor,)).fetchone(),(65,0))
            self.assertEqual(conn.execute("SELECT amount_atoms,xp_units,block_reward_atoms,numeric_version,xp_amount FROM reward_records WHERE reward_id=?",(rid,)).fetchone(),(0,65,0,1,0))
            with self.assertRaises(ValueError):api._reward_record_insert(conn,player_id=self.actor,source="TEST",zone_id="TEST",site_id=1,amount=1,xp_amount=65)

    def test_nonfinite_mining_fields_rejected_before_state_change(self):
        for token in ("NaN","Infinity","-Infinity"):
            response=self.client.post("/api/mine/finish",headers={**self.headers,"Content-Type":"application/json"},
                content='{"playerId":'+json.dumps(self.actor)+',"durationSec":'+token+'}')
            self.assertEqual(response.status_code,422,response.text)

    def test_withdraw_requires_decimal_text_before_json_float_rounding(self):
        body='{"userId":'+json.dumps(self.actor)+',"asset":"TGWT","amount":1.0000000000000000000000001,"toAddress":"0x'+"12"*20+'","nonce":"test-amount-nonce"}'
        response=self.client.post("/api/v1/withdraw/request",headers={**self.headers,"Content-Type":"application/json"},content=body)
        self.assertEqual(response.status_code,422,response.text)

    def test_legacy_qc_settlement_cannot_be_reactivated_by_dev_flag(self):
        with patch.dict(os.environ,{"QC_ALLOW_DEV_SETTLEMENT":"1"}):
            result=api.qc_withdraw_request(api.QCWithdrawRequestIn(playerId=self.actor))
        self.assertEqual(result.status_code,410)

    def test_balance_change_cannot_confirm_transaction(self):
        with patch.object(api,"qc_fetch_tx_status",AsyncMock(return_value={"ok":False})),patch.object(api,"qc_fetch_address_balance",AsyncMock()) as balance:
            result=asyncio.run(api.qc_tx_status("test-tx"))
        self.assertFalse(result["confirmed"])
        balance.assert_not_awaited()

    def test_signed_output_rejects_float_bool_nan_and_overflow(self):
        for amount in (1.0,True,float("nan"),float("inf"),1<<63):
            result=asyncio.run(qc_api.fetch_submit_signed_tx({"Inputs":[],"Outputs":[{"Amount":amount,"PubKeyHash":"test"}]}))
            self.assertFalse(result["ok"])

    def test_admin_policy_requires_go_runtime_no_python_fallback(self):
        admin={"Authorization":"Bearer "+os.environ["TELEGRAM_GAME_ADMIN_TOKEN"]}
        for section in ("mainnet-before","burn-sink","mining-transition","network-fee","founder-vesting"):
            with patch.object(qc_api,"fetch_runtime_policy",AsyncMock(return_value={"ok":False})):
                result=self.client.get("/api/admin/economy/"+section,headers=admin)
                self.assertEqual(result.status_code,503,result.text)
        snapshot={"ok":True,"version":"NODE_RUNTIME_POLICY_V1","source":"quantumcoin_go_node","context":{"height":123,"nextProtocol":"QC_ATOM_PROTOCOL_V3"},"sections":{"network-fee":{"targetAtoms":7}}}
        with patch.object(qc_api,"fetch_runtime_policy",AsyncMock(return_value=snapshot)):
            result=self.client.get("/api/admin/economy/network-fee",headers=admin)
            self.assertEqual(result.json()["context"],snapshot["context"])
            self.assertEqual(result.json()["policy"],{"targetAtoms":7})
        self.assertIn(self.client.get("/api/admin/economy/network-fee",headers=self.headers).status_code,(401,403))

    def test_policy_bridge_authentication_and_redirect_failure(self):
        import httpx
        def receive(request):
            self.assertEqual(request.headers["X-QC-API-KEY"],"test-only-node")
            self.assertEqual(request.url.path,"/api/runtime/policy")
            return httpx.Response(302,headers={"Location":"https://elsewhere.invalid"})
        real_client=httpx.AsyncClient
        with patch.object(qc_api.httpx,"AsyncClient",side_effect=lambda **kwargs:real_client(transport=httpx.MockTransport(receive),**kwargs)):
            self.assertFalse(asyncio.run(qc_api.fetch_runtime_policy())["ok"])

    def test_bot_opens_webapp_without_random_rewards_or_jobs(self):
        from telegram_game import main_bot
        handlers=[];messages=[]
        application=SimpleNamespace(add_handler=handlers.append)
        builder=SimpleNamespace(token=lambda token:builder,build=lambda:application)
        telegram=ModuleType("telegram");extension=ModuleType("telegram.ext")
        telegram.WebAppInfo=lambda **kwargs:kwargs
        telegram.InlineKeyboardButton=lambda *args,**kwargs:kwargs
        telegram.InlineKeyboardMarkup=lambda buttons:buttons
        extension.Application=SimpleNamespace(builder=lambda:builder)
        extension.CommandHandler=lambda command,callback:(command,callback)
        async def reply(text,**kwargs):messages.append((text,kwargs))
        with patch.dict(sys.modules,{"telegram":telegram,"telegram.ext":extension}),patch.dict(os.environ,{"QC_GAME_WEBAPP_URL":"https://game.example.invalid/game"}):
            self.assertIs(main_bot.build_application(),application)
            self.assertEqual([item[0] for item in handlers],["start"])
            asyncio.run(handlers[0][1](SimpleNamespace(effective_message=SimpleNamespace(reply_text=reply)),None))
        self.assertEqual(len(messages),1)
        self.assertNotIn("balance updated",messages[0][0].lower())
        self.assertEqual(messages[0][1]["reply_markup"][0][0]["web_app"]["url"],"https://game.example.invalid/game")

    def test_action_idempotency_does_not_consume_legacy_global_nonce(self):
        now=int(time.time());key=Ed25519PrivateKey.generate()
        public=key.public_key().public_bytes(serialization.Encoding.Raw,serialization.PublicFormat.Raw).hex()
        nonce="same-client-nonce"
        with closing(api._db()) as conn,conn:
            conn.execute("INSERT INTO tgwt_nonces VALUES(?,?,?)",(self.actor,nonce,now))
        for action in ("SOCIAL_FOLLOW","WATCH_COMPLETE","STORE_PURCHASE"):
            campaign=action+"_"+self.actor
            policy={"campaignId":campaign,"provider":"test-only-provider","publicKey":public,"action":action,"target":"test-target","reward_u":1000000,"budget_u":10000000,"startsAt":now-1,"endsAt":now+300}
            self.assertTrue(rewards.create_campaign(api._db,policy)["ok"])
            statement={"version":rewards.VERSION,"audience":"quantumcoin.telegram-game.tgwt","campaignId":campaign,"campaignHash":rewards.policy_hash(policy),"playerId":self.actor,"provider":policy["provider"],"subjectId":self.actor,"action":action,"target":"test-target","eventId":campaign,"issuedAt":now,"expiresAt":now+100}
            proof={"statement":statement,"signature":key.sign(rewards.signing_bytes(statement)).hex()}
            self.assertTrue(rewards.claim(api._db,self.actor,campaign,proof,now)["ok"])
            self.assertTrue(rewards.claim(api._db,self.actor,campaign,proof,now)["alreadyCredited"])
        first=withdraw_ledger.reserve(api._db,self.actor,"0x"+"12"*20,"1",nonce,now)
        again=withdraw_ledger.reserve(api._db,self.actor,"0x"+"12"*20,"1",nonce,now)
        self.assertTrue(first["ok"],first);self.assertEqual(first["requestId"],again["requestId"])
        self.assertFalse(withdraw_ledger.reserve(api._db,self.actor,"0x"+"12"*20,"2",nonce,now)["ok"])
        with closing(api._db()) as conn:
            self.assertEqual(conn.execute("SELECT COUNT(*) FROM tgwt_nonces WHERE user_id=?",(self.actor,)).fetchone()[0],1)
        self.assertFalse(hasattr(api,"_nonce_once"))


if __name__=="__main__":unittest.main()
