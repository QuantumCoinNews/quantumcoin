"""X OAuth/PKCE and read-only proof checks; all provider requests are mocked."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile

if os.getenv("QC_X_REWARD_TEST_CHILD")!="1":
    with tempfile.TemporaryDirectory(prefix="qc-x-provider-") as directory:
        env=os.environ.copy()
        env.update(QC_X_REWARD_TEST_CHILD="1",QC_PLAYER_RUNTIME_TEST_CHILD="1",QC_VERIFIED_REWARDS_TEST_CHILD="1",
            QC_GAME_DB=str(Path(directory)/"test-only-x.sqlite"),TELEGRAM_BOT_TOKEN="test-only-x-bot",
            TELEGRAM_GAME_ADMIN_TOKEN="test-only-x-admin",QC_API_BASE="http://127.0.0.1:1",QC_API_TOKEN="test-only-no-node")
        raise SystemExit(subprocess.call([sys.executable,str(Path(__file__).resolve())],cwd=directory,env=env))

import asyncio
import base64
from contextlib import closing
import hashlib
import json
import time
import unittest
from urllib.parse import parse_qs,parse_qsl,urlparse
from unittest.mock import patch
import httpx
from cryptography.hazmat.primitives import serialization
sys.path.insert(0,str(Path(__file__).resolve().parent))
import test_verified_rewards_pow_replay_v1 as helpers
from telegram_game import dev_api as api, reward_evidence as rewards, x_reward_verifier as x


class XRewardVerificationTests(unittest.TestCase):
    def setUp(self):
        helpers.VerifiedRewardPoWTests.setUp(self)
        self.config={"clientId":"test-only-client","clientSecret":"test-only-client-secret","callback":"https://game.test/oauth/x/callback",
                     "signingKey":self.key,"stateKey":bytes(32)}
        self.real_configuration=x.configuration
        self.config_patch=patch.object(x,"configuration",return_value=self.config.copy());self.config_patch.start();self.addCleanup(self.config_patch.stop)
        self.gate_patch=patch.object(x,"social_rewards_enabled",return_value=True);self.gate_patch.start();self.addCleanup(self.gate_patch.stop)
        with closing(api._db()) as conn,conn:conn.execute("DELETE FROM x_reward_oauth_v1")
        self.calls=[]

    def campaign(self,name="test-x-follow",action="SOCIAL_FOLLOW"):
        policy={"campaignId":name,"provider":"X","publicKey":self.key.public_key().public_bytes(serialization.Encoding.Raw,serialization.PublicFormat.Raw).hex(),
                "action":action,"target":"123456","reward_u":1_000_000,"budget_u":10_000_000,"startsAt":self.now-10,"endsAt":self.now+86400}
        self.assertTrue(rewards.create_campaign(api._db,policy)["ok"])
        return policy

    def started(self,policy):
        response=self.client.post("/api/social/x/start",headers=self.headers,json={"playerId":"tg_9001","campaignId":policy["campaignId"]})
        self.assertTrue(response.json()["ok"],response.text)
        return response.json()

    def complete(self,started):
        return self.client.post("/api/social/x/complete",headers=self.headers,
            json={"playerId":"tg_9001","state":started["state"],"code":"test-only-authorization-code"})

    def mock_provider(self,started=None,mode="follow"):
        original=httpx.AsyncClient
        def handler(request):
            self.calls.append(request)
            if request.url.path=="/2/oauth2/token":
                values=dict(parse_qsl(request.content.decode()))
                self.assertEqual(values["grant_type"],"authorization_code")
                self.assertEqual(values["redirect_uri"],self.config["callback"])
                if started:
                    expected=parse_qs(urlparse(started["authorizeUrl"]).query)["code_challenge"][0]
                    self.assertEqual(base64.urlsafe_b64encode(hashlib.sha256(values["code_verifier"].encode()).digest()).rstrip(b"=").decode(),expected)
                return httpx.Response(200,json={"access_token":"x-test","scope":"tweet.read users.read follows.read like.read"})
            self.assertEqual(request.headers["Authorization"],"Bearer x-test")
            if request.url.path=="/2/users/me":return httpx.Response(200,json={"data":{"id":"77"}})
            self.assertEqual(request.method,"GET","adapter must never like/follow automatically")
            if mode=="denied":return httpx.Response(403,json={"error":"x-test"})
            if mode=="not-following":return httpx.Response(200,json={"data":[],"meta":{}})
            if mode=="cycle":return httpx.Response(200,json={"data":[],"meta":{"next_token":"same"}})
            if mode=="like":
                self.assertEqual(request.url.path,"/2/users/77/liked_tweets")
                return httpx.Response(200,json={"data":[{"id":"123456"}]})
            self.assertEqual(request.url.path,"/2/users/77/following")
            if "pagination_token" not in request.url.params:return httpx.Response(200,json={"data":[{"id":"999"}],"meta":{"next_token":"page-2"}})
            return httpx.Response(200,json={"data":[{"id":"123456"}]})
        def client(**kwargs):
            self.assertFalse(kwargs["follow_redirects"])
            return original(**kwargs,transport=httpx.MockTransport(handler))
        return patch.object(x.httpx,"AsyncClient",side_effect=client)

    def test_follow_oauth_reads_authenticated_x_identity_then_credits_one_tgwt(self):
        started=self.started(self.campaign())
        query=parse_qs(urlparse(started["authorizeUrl"]).query)
        self.assertEqual(query["code_challenge_method"],["S256"])
        self.assertNotIn("write",query["scope"][0]);self.assertNotIn("offline.access",query["scope"][0])
        with self.mock_provider(started):result=self.complete(started)
        self.assertTrue(result.json()["ok"],result.text);self.assertEqual(result.json()["credited_u"],1_000_000)
        self.assertFalse(result.json()["onChainPaid"])
        for secret in ("x-test","test-only-client-secret","test-only-authorization-code"):
            self.assertNotIn(secret,result.text)
        with closing(api._db()) as conn:
            stored=repr(conn.execute("SELECT * FROM x_reward_oauth_v1").fetchall())
            self.assertNotIn("x-test",stored)
            self.assertEqual(conn.execute("SELECT distributed_u FROM tgwt_pool WHERE id=1").fetchone()[0],1_000_000)

    def test_like_and_follow_are_separate_one_tgwt_tasks(self):
        follow=self.started(self.campaign())
        with self.mock_provider(follow):self.assertTrue(self.complete(follow).json()["ok"])
        like=self.started(self.campaign("test-x-like","SOCIAL_LIKE"))
        with self.mock_provider(like,"like"):self.assertTrue(self.complete(like).json()["ok"])
        with closing(api._db()) as conn:
            self.assertEqual(conn.execute("SELECT distributed_u FROM tgwt_pool WHERE id=1").fetchone()[0],2_000_000)

    def test_wrong_actor_and_replayed_or_expired_state_never_call_provider(self):
        started=self.started(self.campaign())
        result=asyncio.run(x.complete(api._db,"tg_9002",started["state"],"test-code",self.now))
        self.assertEqual(result["error"],"x_oauth_state_invalid_expired_or_consumed")
        with self.mock_provider(started):self.assertTrue(self.complete(started).json()["ok"])
        with patch.object(x,"verify_provider_action",side_effect=AssertionError("replayed OAuth reached provider")):
            self.assertEqual(self.complete(started).json()["error"],"x_oauth_state_invalid_expired_or_consumed")
        self.assertTrue(x.begin(api._db,"tg_9001","test-x-follow",self.now)["alreadyCredited"])

    def test_no_action_access_denied_or_pagination_cycle_cannot_reward(self):
        policy=self.campaign()
        for mode in ("not-following","denied","cycle"):
            started=self.started(policy)
            with self.mock_provider(started,mode):result=self.complete(started)
            self.assertFalse(result.json()["ok"],result.text)
        with closing(api._db()) as conn:self.assertEqual(conn.execute("SELECT distributed_u FROM tgwt_pool WHERE id=1").fetchone()[0],0)

    def test_unconfigured_x_is_fail_closed_and_variable_reward_is_rejected(self):
        with patch.object(x,"configuration",return_value=None):
            response=self.client.post("/api/social/x/start",headers=self.headers,json={"playerId":"tg_9001","campaignId":"unknown"})
        self.assertEqual(response.json()["error"],"x_oauth_not_configured")
        policy=self.campaign();policy={**policy,"campaignId":"wrong-amount","reward_u":2_000_000}
        self.assertFalse(rewards.create_campaign(api._db,policy)["ok"])

    def test_testnet_gate_blocks_x_start_and_completion(self):
        policy=self.campaign()
        started=x.begin(api._db,"tg_9001",policy["campaignId"],self.now)
        self.assertTrue(started["ok"])
        with patch.object(x,"social_rewards_enabled",return_value=False):
            denied=asyncio.run(x.complete(api._db,"tg_9001",started["state"],"test-only-code",self.now))
            denied_start=x.begin(api._db,"tg_9002",policy["campaignId"],self.now)
        self.assertEqual(denied["error"],"social_rewards_disabled_mainnet_only")
        self.assertEqual(denied_start["error"],"social_rewards_disabled_mainnet_only")
        self.assertEqual(self.calls,[])
        with closing(api._db()) as conn:
            self.assertEqual(conn.execute("SELECT distributed_u FROM tgwt_pool WHERE id=1").fetchone()[0],0)

    def test_callback_escapes_script_and_cannot_itself_credit_money(self):
        response=self.client.get("/oauth/x/callback",params={"code":"</script><script>alert(1)</script>","state":"test-state"})
        self.assertEqual(response.status_code,200)
        self.assertNotIn("</script><script>",response.text)
        self.assertIn("postMessage",response.text)
        self.assertEqual(response.headers["referrer-policy"],"no-referrer")
        self.assertEqual(response.headers["cache-control"],"no-store")
        self.assertIn("frame-ancestors 'none'",response.headers["content-security-policy"])
        with closing(api._db()) as conn:self.assertEqual(conn.execute("SELECT distributed_u FROM tgwt_pool WHERE id=1").fetchone()[0],0)

    def test_provider_budget_failure_never_exposes_tokens_or_rewards(self):
        started=self.started(self.campaign())
        with patch("telegram_game.durable_limits.admit",return_value=False):
            result=asyncio.run(x.complete(api._db,"tg_9001",started["state"],"test-code",self.now))
        self.assertEqual(result["error"],"x_verification_unavailable_no_reward")

    def test_deployment_configuration_requires_https_and_external_protected_seed(self):
        with tempfile.TemporaryDirectory(prefix="qc-test-x-config-") as directory:
            key_file=Path(directory)/"test-only.seed"
            key_file.write_bytes(self.key.private_bytes(serialization.Encoding.Raw,serialization.PrivateFormat.Raw,serialization.NoEncryption()))
            env={"QC_X_CLIENT_ID":"test-client","QC_X_CALLBACK_URL":"https://game.test/oauth/x/callback",
                 "QC_X_REWARD_SIGNING_SEED_FILE":str(key_file.resolve()),"QC_X_STATE_ENCRYPTION_KEY":"01"*32}
            with patch.dict(os.environ,env):
                self.assertIsNotNone(self.real_configuration())
                with patch.dict(os.environ,{"QC_X_CALLBACK_URL":"http://game.test/oauth/x/callback"}):
                    self.assertIsNone(self.real_configuration())
                with patch.dict(os.environ,{"QC_X_STATE_ENCRYPTION_KEY":""}):
                    self.assertIsNone(self.real_configuration())


if __name__=="__main__":unittest.main(verbosity=2)
