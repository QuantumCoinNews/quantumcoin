"""Telegram Bot API membership checks are mocked; no real reward is sent."""
import asyncio
import os
from pathlib import Path
import subprocess
import sys
import tempfile

if os.getenv("QC_TEST_TELEGRAM_REWARD_CHILD") != "1":
    with tempfile.TemporaryDirectory(prefix="qc-telegram-social-") as directory:
        env=os.environ.copy()
        env.update(QC_TEST_TELEGRAM_REWARD_CHILD="1",QC_X_REWARD_TEST_CHILD="1",
            QC_PLAYER_RUNTIME_TEST_CHILD="1",QC_VERIFIED_REWARDS_TEST_CHILD="1",
            QC_GAME_DB=str(Path(directory)/"test-only-telegram-social.sqlite"),
            TELEGRAM_BOT_TOKEN="12345:test-only-telegram-bot-token-value",
            TELEGRAM_GAME_ADMIN_TOKEN="test-only-telegram-admin",QC_API_BASE="http://127.0.0.1:1",
            QC_API_TOKEN="test-only-no-node")
        raise SystemExit(subprocess.call([sys.executable,str(Path(__file__).resolve())],cwd=directory,env=env))

import hashlib
import json
import unittest
from contextlib import closing
from unittest.mock import patch

import httpx
from cryptography.hazmat.primitives import serialization

sys.path.insert(0,str(Path(__file__).resolve().parent))
import test_verified_rewards_pow_replay_v1 as helpers
from telegram_game import dev_api as api, reward_evidence as rewards, telegram_reward_verifier as telegram


class TelegramRewardVerificationTests(unittest.TestCase):
    def setUp(self):
        helpers.VerifiedRewardPoWTests.setUp(self)
        self.config={"botToken":"12345:test-only-telegram-bot-token-value","signingKey":self.key}
        self.config_patch=patch.object(telegram,"configuration",return_value=self.config.copy());self.config_patch.start();self.addCleanup(self.config_patch.stop)
        self.gate_patch=patch.object(telegram,"social_rewards_enabled",return_value=True);self.gate_patch.start();self.addCleanup(self.gate_patch.stop)
        self.calls=[]

    def campaign(self,name="test-telegram-join"):
        policy={"campaignId":name,"provider":"TELEGRAM","publicKey":self.key.public_key().public_bytes(serialization.Encoding.Raw,serialization.PublicFormat.Raw).hex(),
            "action":"SOCIAL_JOIN","target":"-1002870924021","url":"https://web.telegram.org/a/#-1002870924021",
            "reward_u":1_000_000,"budget_u":10_000_000,"startsAt":self.now-10,"endsAt":self.now+86400}
        self.assertTrue(rewards.create_campaign(api._db,policy)["ok"])
        return policy

    def mock_provider(self,status="member"):
        original=httpx.AsyncClient
        def handler(request):
            self.calls.append(request)
            self.assertEqual(request.url.host,"api.telegram.org")
            self.assertEqual(request.url.path,"/bot12345:test-only-telegram-bot-token-value/getChatMember")
            self.assertEqual(request.url.params["chat_id"],"-1002870924021")
            self.assertEqual(request.url.params["user_id"],"9001")
            if status=="error":return httpx.Response(403,json={"ok":False,"description":"test-only-bot-secret"})
            result={"user":{"id":9001},"status":status}
            if status=="restricted":result["is_member"]=True
            return httpx.Response(200,json={"ok":True,"result":result})
        def client(**kwargs):
            self.assertFalse(kwargs["follow_redirects"])
            return original(**kwargs,transport=httpx.MockTransport(handler))
        return patch.object(telegram.httpx,"AsyncClient",side_effect=client)

    def test_member_receives_exactly_one_off_chain_tgwt_and_repeat_is_free(self):
        policy=self.campaign()
        with self.mock_provider():first=asyncio.run(telegram.verify(api._db,"tg_9001",policy["campaignId"],self.now))
        self.assertTrue(first["ok"],first)
        self.assertEqual(first["credited_u"],1_000_000)
        self.assertFalse(first["onChainPaid"])
        self.assertEqual(first["status"],"VERIFIED_OFFCHAIN_CLAIMABLE")
        with self.mock_provider():again=asyncio.run(telegram.verify(api._db,"tg_9001",policy["campaignId"],self.now))
        self.assertTrue(again["alreadyCredited"])
        self.assertEqual(again["credited_u"],0)
        self.assertEqual(len(self.calls),1)
        with closing(api._db()) as conn:
            self.assertEqual(conn.execute("SELECT distributed_u FROM tgwt_pool WHERE id=1").fetchone()[0],1_000_000)
            proof=conn.execute("SELECT proof_hash FROM tgwt_verified_rewards_v1").fetchone()
            self.assertTrue(proof[0])

    def test_nonmember_wrong_actor_and_provider_failure_never_credit(self):
        policy=self.campaign()
        for status in ("left","kicked","error"):
            with self.mock_provider(status):result=asyncio.run(telegram.verify(api._db,"tg_9001",policy["campaignId"],self.now))
            self.assertFalse(result["ok"])
            self.assertNotIn("test-only-bot-secret",json.dumps(result))
        with self.mock_provider():wrong=asyncio.run(telegram.verify(api._db,"tg_9002",policy["campaignId"],self.now))
        self.assertFalse(wrong["ok"])
        with closing(api._db()) as conn:
            self.assertEqual(conn.execute("SELECT distributed_u FROM tgwt_pool WHERE id=1").fetchone()[0],0)
            self.assertEqual(conn.execute("SELECT COUNT(*) FROM tgwt_verified_rewards_v1").fetchone()[0],0)

    def test_restricted_current_member_counts_but_testnet_gate_blocks_api(self):
        policy=self.campaign()
        with self.mock_provider("restricted"):
            result=asyncio.run(telegram.verify(api._db,"tg_9001",policy["campaignId"],self.now))
        self.assertTrue(result["ok"])
        with closing(api._db()) as conn, conn:
            conn.execute("DELETE FROM tgwt_verified_rewards_v1")
            conn.execute("DELETE FROM tgwt_events")
            conn.execute("DELETE FROM tgwt_state")
            conn.execute("UPDATE tgwt_pool SET distributed_u=0 WHERE id=1")
        with patch.object(telegram,"social_rewards_enabled",return_value=False):
            blocked=asyncio.run(telegram.verify(api._db,"tg_9001",policy["campaignId"],self.now))
            api_blocked=self.client.post("/api/social/telegram/verify",headers=self.headers,
                json={"playerId":"tg_9001","campaignId":policy["campaignId"]})
        self.assertEqual(blocked["error"],"social_rewards_disabled_mainnet_only")
        self.assertEqual(api_blocked.json()["error"],"social_rewards_disabled_mainnet_only")
        self.assertEqual(len(self.calls),1)
        with closing(api._db()) as conn:self.assertEqual(conn.execute("SELECT distributed_u FROM tgwt_pool WHERE id=1").fetchone()[0],0)


if __name__=="__main__":unittest.main()
