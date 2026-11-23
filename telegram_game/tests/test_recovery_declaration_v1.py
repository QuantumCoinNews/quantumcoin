"""Recovery display validation; no application import, wallet or runtime DB."""
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from telegram_game import backup_restore_runbook_v1 as recovery


class RecoveryDeclarationTests(unittest.TestCase):
    def test_display_is_exact_generated_go_declaration_and_detached(self):
        first = recovery.default_backup_restore_runbook_v1()
        expected = json.loads(recovery._DECLARATION.read_text(encoding="utf-8"))
        self.assertEqual(first, expected)
        first["artifacts"][0]["kind"] = "mutated"
        self.assertEqual(recovery.default_backup_restore_runbook_v1(), expected)
        summary = recovery.backup_restore_summary_v1()
        self.assertTrue(summary["ok"])
        self.assertFalse(summary["liveRestoreAvailable"])

    def test_missing_each_required_role_fails_single_validation(self):
        for name in recovery.REQUIRED_ARTIFACTS:
            with self.subTest(name=name):
                runbook = recovery.default_backup_restore_runbook_v1()
                runbook["artifacts"] = [a for a in runbook["artifacts"] if a["name"] != name]
                self.assertFalse(recovery.validate_backup_restore_runbook_v1(runbook)["ok"])

    def test_kinds_wallet_custody_and_boundary_are_strict(self):
        for field, value in (("kind", "unknown"), ("kind", "config"), ("name", " quantumcoin_chain_file ")):
            runbook = recovery.default_backup_restore_runbook_v1()
            runbook["artifacts"][0][field] = value
            self.assertFalse(recovery.validate_backup_restore_runbook_v1(runbook)["ok"])
        runbook = recovery.default_backup_restore_runbook_v1()
        wallet = next(a for a in runbook["artifacts"] if a["name"] == "wallet_encrypted_backup")
        wallet["requiresEncryption"] = False
        self.assertFalse(recovery.validate_backup_restore_runbook_v1(runbook)["ok"])
        for value in (False, 1):
            runbook = recovery.default_backup_restore_runbook_v1()
            runbook["safeBoundary"]["doesNotRunLiveRestore"] = value
            self.assertFalse(recovery.validate_backup_restore_runbook_v1(runbook)["ok"])


if __name__ == "__main__":
    unittest.main()
