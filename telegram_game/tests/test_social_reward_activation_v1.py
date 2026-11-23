import os
from pathlib import Path
import sqlite3
import sys
import tempfile
import unittest
from unittest.mock import patch

from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives import serialization

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
from telegram_game import reward_evidence
from telegram_game.social_reward_activation import social_rewards_enabled
from config.runtime_environment import validate_environment


MAINNET = {
    "QC_TGWT_SOCIAL_REWARDS_ENABLED": "1",
    "TGWT_BSC_NETWORK": "bsc_mainnet",
    "TGWT_BSC_CHAIN_ID": "56",
    "TGWT_BSC_CLAIM_ENABLED": "1",
    "TGWT_BSC_DISTRIBUTION_WALLET_ADDRESS": "0x1111111111111111111111111111111111111111",
    "TGWT_BSC_TOKEN_CONTRACT_ADDRESS": "0x2222222222222222222222222222222222222222",
    "TGWT_BSC_CLAIM_CONTRACT_ADDRESS": "0x3333333333333333333333333333333333333333",
    "TGWT_BSC_MAINNET_CLAIM_APPROVAL": "I_APPROVE_TGWT_BSC_MAINNET_CLAIM",
}


class SocialRewardActivationTests(unittest.TestCase):
    def test_default_and_testnet_are_disabled(self):
        with patch.dict(os.environ, {}, clear=True):
            self.assertFalse(social_rewards_enabled())
        with patch.dict(os.environ, {**MAINNET, "QC_TESTNET": "1", "QC_NETWORK": "mainnet"}, clear=True):
            self.assertFalse(social_rewards_enabled())
        with patch.dict(os.environ, {**MAINNET, "TELEGRAM_GAME_NETWORK": "testnet"}, clear=True):
            self.assertFalse(social_rewards_enabled())

    def test_provider_environment_is_in_canonical_runtime_schema(self):
        env = {
            "QC_TGWT_SOCIAL_REWARDS_ENABLED": "0",
            "QC_TGWT_REWARD_SIGNING_SEED_FILE": r"C:\keys\tgwt-reward.seed",
            "QC_YOUTUBE_CLIENT_ID": "public-client-id",
            "QC_YOUTUBE_CLIENT_SECRET": "secret-only-in-env",
            "QC_YOUTUBE_CALLBACK_URL": "https://game.example/oauth/youtube/callback",
            "QC_YOUTUBE_STATE_ENCRYPTION_KEY": "01" * 32,
        }
        self.assertEqual(validate_environment(env, "game"), {})

    def test_mainnet_needs_chain_claim_and_manual_approval(self):
        with patch.dict(os.environ, MAINNET, clear=True):
            self.assertTrue(social_rewards_enabled())
        for change in ({"TGWT_BSC_CHAIN_ID": "97"}, {"TGWT_BSC_CLAIM_ENABLED": "0"},
                       {"TGWT_BSC_MAINNET_CLAIM_APPROVAL": ""},
                       {"TGWT_BSC_PRIVATE_KEY": "test-only-forbidden"}):
            with patch.dict(os.environ, {**MAINNET, **change}, clear=True):
                self.assertFalse(social_rewards_enabled(), change)

    def test_campaign_links_and_provider_actions_are_bound(self):
        key = Ed25519PrivateKey.generate()
        pub = key.public_key().public_bytes(serialization.Encoding.Raw, serialization.PublicFormat.Raw).hex()
        now = 2_000_000_000
        base = {"publicKey": pub, "reward_u": 1_000_000, "budget_u": 3_000_000,
                "startsAt": now - 1, "endsAt": now + 1000}
        campaigns = [
            {**base, "campaignId": "tg-join", "provider": "TELEGRAM", "action": "SOCIAL_JOIN",
             "target": "-1002870924021", "url": "https://web.telegram.org/a/#-1002870924021"},
            {**base, "campaignId": "yt-sub", "provider": "YOUTUBE", "action": "SOCIAL_SUBSCRIBE",
             "target": "@QuantumCoinHQ", "url": "https://www.youtube.com/@QuantumCoinHQ"},
            {**base, "campaignId": "x-follow", "provider": "X", "action": "SOCIAL_FOLLOW",
             "target": "123456", "url": "https://x.com/QuantumCoinQC"},
        ]
        with tempfile.TemporaryDirectory(prefix="qc-social-policy-") as temp:
            dbpath = Path(temp) / "policy.sqlite"
            def db():
                conn = sqlite3.connect(dbpath)
                conn.execute("CREATE TABLE IF NOT EXISTS tgwt_campaigns_v1(campaign_id TEXT PRIMARY KEY,policy_json TEXT NOT NULL,policy_hash TEXT NOT NULL,spent_u INTEGER NOT NULL DEFAULT 0,enabled INTEGER NOT NULL DEFAULT 1)")
                return conn
            for campaign in campaigns:
                self.assertTrue(reward_evidence.create_campaign(db, campaign)["ok"], campaign["campaignId"])
            invalid = [
                {**campaigns[0], "campaignId": "tg-wrong-action", "action": "SOCIAL_FOLLOW"},
                {**campaigns[0], "campaignId": "tg-wrong-link", "url": "https://evil.example/join"},
                {**campaigns[1], "campaignId": "yt-bad-target", "target": "x"},
            ]
            for campaign in invalid:
                self.assertFalse(reward_evidence.create_campaign(db, campaign)["ok"], campaign["campaignId"])


if __name__ == "__main__":
    unittest.main()
