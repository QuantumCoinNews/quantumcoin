"""YouTube OAuth/PKCE subscription verification; all Google calls are mocked."""
import base64
import hashlib
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from contextlib import closing
from urllib.parse import parse_qs, parse_qsl, urlparse
from unittest.mock import patch

if os.getenv("QC_TEST_YOUTUBE_REWARD_CHILD") != "1":
    with tempfile.TemporaryDirectory(prefix="qc-youtube-social-") as directory:
        env=os.environ.copy()
        env.update(QC_TEST_YOUTUBE_REWARD_CHILD="1",QC_X_REWARD_TEST_CHILD="1",
            QC_PLAYER_RUNTIME_TEST_CHILD="1",QC_VERIFIED_REWARDS_TEST_CHILD="1",
            QC_GAME_DB=str(Path(directory)/"test-only-youtube-social.sqlite"),
            TELEGRAM_BOT_TOKEN="12345:test-only-youtube-bot",TELEGRAM_GAME_ADMIN_TOKEN="test-only-youtube-admin",
            QC_API_BASE="http://127.0.0.1:1",QC_API_TOKEN="test-only-no-node")
        raise SystemExit(subprocess.call([sys.executable,str(Path(__file__).resolve())],cwd=directory,env=env))

import httpx
from cryptography.hazmat.primitives import serialization

sys.path.insert(0,str(Path(__file__).resolve().parent))
import test_verified_rewards_pow_replay_v1 as helpers
from telegram_game import dev_api as api, reward_evidence as rewards, youtube_reward_verifier as youtube


class YouTubeRewardVerificationTests(unittest.TestCase):
    def setUp(self):
        helpers.VerifiedRewardPoWTests.setUp(self)
        self.config={"clientId":"test-only-google-client.apps.googleusercontent.com","clientSecret":"test-only-google-secret",
            "callback":"https://game.test/oauth/youtube/callback","signingKey":self.key,"stateKey":bytes(32)}
        self.config_patch=patch.object(youtube,"configuration",return_value=self.config.copy());self.config_patch.start();self.addCleanup(self.config_patch.stop)
        self.gate_patch=patch.object(youtube,"social_rewards_enabled",return_value=True);self.gate_patch.start();self.addCleanup(self.gate_patch.stop)
        self.calls=[]
        self.target="UC"+"A"*22

    def campaign(self,name="test-youtube-subscribe"):
        policy={"campaignId":name,"provider":"YOUTUBE","publicKey":self.key.public_key().public_bytes(serialization.Encoding.Raw,serialization.PublicFormat.Raw).hex(),
            "action":"SOCIAL_SUBSCRIBE","target":"@QuantumCoinHQ","url":"https://www.youtube.com/@QuantumCoinHQ",
            "reward_u":1_000_000,"budget_u":10_000_000,"startsAt":self.now-10,"endsAt":self.now+86400}
        self.assertTrue(rewards.create_campaign(api._db,policy)["ok"])
        return policy

    def started(self,policy):
        result=youtube.begin(api._db,"tg_9001",policy["campaignId"],self.now)
        self.assertTrue(result["ok"],result)
        return result

    def mock_google(self,started=None,subscribed=True,fail=False):
        original=httpx.AsyncClient
        def handler(request):
            self.calls.append(request)
            if request.url.host=="oauth2.googleapis.com":
                values=dict(parse_qsl(request.content.decode()))
                self.assertEqual(values["grant_type"],"authorization_code")
                self.assertEqual(values["redirect_uri"],self.config["callback"])
                self.assertEqual(values["client_secret"],self.config["clientSecret"])
                if started:
                    challenge=parse_qs(urlparse(started["authorizeUrl"]).query)["code_challenge"][0]
                    expected=base64.urlsafe_b64encode(hashlib.sha256(values["code_verifier"].encode()).digest()).rstrip(b"=").decode()
                    self.assertEqual(expected,challenge)
                if fail:return httpx.Response(401,json={"error":"test-only-google-secret"})
                return httpx.Response(200,json={"access_token":"x-test","token_type":"Bearer"})
            self.assertEqual(request.headers["Authorization"],"Bearer x-test")
            if request.url.host=="openidconnect.googleapis.com":return httpx.Response(200,json={"sub":"google-subject-test-12345"})
            if request.url.path=="/youtube/v3/channels":
                self.assertEqual(request.url.params["forHandle"],"QuantumCoinHQ")
                return httpx.Response(200,json={"items":[{"id":self.target}]})
            if request.url.path=="/youtube/v3/subscriptions":
                self.assertEqual(request.url.params["mine"],"true")
                self.assertEqual(request.url.params["forChannelId"],self.target)
                data=[{"snippet":{"resourceId":{"channelId":self.target}}}] if subscribed else []
                return httpx.Response(200,json={"items":data})
            raise AssertionError("unexpected fixed Google endpoint "+request.url.path)
        def client(**kwargs):
            self.assertFalse(kwargs["follow_redirects"])
            return original(**kwargs,transport=httpx.MockTransport(handler))
        return patch.object(youtube.httpx,"AsyncClient",side_effect=client)

    def complete(self,started):
        return self.client.post("/api/social/youtube/complete",headers=self.headers,
            json={"playerId":"tg_9001","state":started["state"],"code":"test-only-google-code"})

    def test_pkce_read_only_subscription_credit_and_duplicate(self):
        policy=self.campaign();started=self.started(policy)
        query=parse_qs(urlparse(started["authorizeUrl"]).query)
        self.assertEqual(query["code_challenge_method"],["S256"])
        self.assertEqual(query["access_type"],["online"])
        self.assertEqual(set(query["scope"][0].split()),{"openid","https://www.googleapis.com/auth/youtube.readonly"})
        self.assertNotIn(self.config["clientSecret"],started["authorizeUrl"])
        with self.mock_google(started):result=self.complete(started)
        self.assertTrue(result.json()["ok"],result.text)
        self.assertEqual(result.json()["credited_u"],1_000_000)
        self.assertFalse(result.json()["onChainPaid"])
        self.assertNotIn("x-test",result.text)
        self.assertNotIn("test-only-google-code",result.text)
        with closing(api._db()) as conn:
            stored=repr(conn.execute("SELECT * FROM youtube_reward_oauth_v1").fetchall())
            self.assertNotIn("x-test",stored)
            self.assertNotIn("test-only-google-code",stored)
            self.assertEqual(conn.execute("SELECT distributed_u FROM tgwt_pool WHERE id=1").fetchone()[0],1_000_000)
        replay=youtube.begin(api._db,"tg_9001",policy["campaignId"],self.now)
        self.assertTrue(replay["alreadyCredited"])

    def test_not_subscribed_provider_failure_and_replayed_state_do_not_reward(self):
        policy=self.campaign()
        for subscribed,fail in ((False,False),(True,True)):
            started=self.started(policy)
            with self.mock_google(started,subscribed,fail):result=self.complete(started)
            self.assertFalse(result.json()["ok"],result.text)
            self.assertNotIn("test-only-google-secret",result.text)
        with patch.object(youtube,"_verify_subscription",side_effect=AssertionError("replay reached provider")):
            replay=self.complete(started)
        self.assertEqual(replay.json()["error"],"youtube_oauth_state_invalid_expired_or_consumed")
        with closing(api._db()) as conn:
            self.assertEqual(conn.execute("SELECT distributed_u FROM tgwt_pool WHERE id=1").fetchone()[0],0)
            self.assertEqual(conn.execute("SELECT COUNT(*) FROM tgwt_verified_rewards_v1").fetchone()[0],0)

    def test_callback_sanitizes_query_and_testnet_gate_prevents_oauth(self):
        response=self.client.get("/oauth/youtube/callback",params={"code":"</script><script>secret</script>","state":"fake-state"})
        self.assertEqual(response.status_code,200)
        self.assertNotIn("</script><script>secret",response.text)
        self.assertEqual(response.headers["referrer-policy"],"no-referrer")
        self.assertEqual(response.headers["cache-control"],"no-store")
        self.assertIn("frame-ancestors 'none'",response.headers["content-security-policy"])
        policy=self.campaign()
        with patch.object(youtube,"social_rewards_enabled",return_value=False):
            blocked=youtube.begin(api._db,"tg_9001",policy["campaignId"],self.now)
        self.assertEqual(blocked["error"],"social_rewards_disabled_mainnet_only")


if __name__=="__main__":unittest.main()
