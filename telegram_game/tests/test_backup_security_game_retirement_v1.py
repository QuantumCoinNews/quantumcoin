"""Read-only declaration and retired bridge contracts; no runtime DB or keys."""
import asyncio
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from telegram_game import backup_restore_runbook_v1 as recovery
from telegram_game.utils import node_score_bridge_v1 as bridge


class BackupAndGameAuthorityTests(unittest.TestCase):
    def test_every_artifact_requires_encryption_and_chain_is_full_file(self):
        runbook = recovery.default_backup_restore_runbook_v1()
        for artifact in runbook["artifacts"]:
            with self.subTest(name=artifact["name"]):
                self.assertIs(artifact["requiresEncryption"], True)
                changed = recovery.default_backup_restore_runbook_v1()
                next(a for a in changed["artifacts"] if a["name"] == artifact["name"])["requiresEncryption"] = False
                self.assertFalse(recovery.validate_backup_restore_runbook_v1(changed)["ok"])
        chain = next(a for a in runbook["artifacts"] if a["name"] == "quantumcoin_chain_file")
        self.assertIs(chain["appendOnlyPreferred"], False)

    def test_retired_bridge_never_sends_keys_tokens_or_scores(self):
        with patch("socket.create_connection", side_effect=AssertionError("network access forbidden")):
            with self.assertRaisesRegex(ValueError, "root_game_retired"):
                asyncio.run(bridge.publish_authoritative_score("https://example.invalid", {"score": 999}, "test-only-token"))


if __name__ == "__main__":
    unittest.main()
