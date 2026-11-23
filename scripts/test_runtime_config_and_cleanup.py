from pathlib import Path
import unittest
from unittest.mock import patch

from runtime_config_preflight import preflight
from source_hygiene import cleanup_plan, cleanup_class, source_exclusion


class DeploymentPreflightTest(unittest.TestCase):
    def test_public_auth_implementation_is_packaged_without_allowing_secret_exports(self):
        self.assertFalse(source_exclusion("node_app_api_credentials.go"))
        self.assertIsNone(cleanup_class("node_app_api_credentials.go"))
        for private in ("credentials.json", "archive/node_app_api_credentials.go",
                        "node_app_api_credentials.go.bak", "secrets/node_app_api_credentials.go"):
            self.assertTrue(source_exclusion(private), private)

    def test_public_recovery_controller_is_packaged_without_allowing_key_exports(self):
        controller = "telegram_game/telegram_game/dev_ui/wallet_recovery.js"
        self.assertFalse(source_exclusion(controller))
        self.assertIsNone(cleanup_class(controller))
        self.assertFalse(source_exclusion("telegram_game/tests/test_wallet_recovery_ui_v1.js"))
        for private in ("wallet_recovery.json", "wallet_recovery.hex", "wallet_recovery.js",
                        "archive/wallet_recovery.js", "secrets/wallet_recovery.js",
                        controller + ".bak", "wallet_priv.hex", "private_key.json"):
            with self.subTest(path=private):
                self.assertTrue(source_exclusion(private))

    def test_security_boundaries(self):
        issues = preflight({
            "QC_API_LISTEN": "[::]:8082",
            "QC_API_BASE": "http://example.invalid",
            "QC_PUBLISHER_MONITOR_ORIGIN": "http://example.invalid",
            "QC_GAME_DB": "data.sqlite3",
        })
        self.assertEqual(set(issues), {"QC_API_TOKEN", "QC_API_BASE", "QC_PUBLISHER_MONITOR_ORIGIN",
                                       "QC_PUBLISHER_MONITOR_TOKEN", "QC_GAME_DB"})
        self.assertEqual(preflight({"QC_API_LISTEN": "::1:8082"}), {"QC_API_LISTEN": "invalid-host-or-port"})
        self.assertEqual(preflight({"QC_API_LISTEN": "[::1]:8082",
                                    "QC_API_BASE": "http://127.0.0.1:8082",
                                    "QC_PUBLISHER_MONITOR_ORIGIN": "http://[::1]:8082",
                                    "QC_PUBLISHER_MONITOR_TOKEN": "test-token"}), {})

    def test_cleanup_reads_only_inventory(self):
        index = ["release/config.testnet.json", "release/chain_data.dat", "bin/node.exe", "go.mod"]
        head = ["archive/wallet_priv.hex", "ui/startup.go.disabled", "go.mod"]
        with patch("source_hygiene.git_paths", side_effect=[index, head]):
            plan = cleanup_plan(Path("."))
        records = {entry["path"]: entry for entry in plan["records"]}
        self.assertEqual({path: entry["class"] for path, entry in records.items()}, {
            "archive/wallet_priv.hex": "secret", "bin/node.exe": "generated",
            "release/chain_data.dat": "runtime", "release/config.testnet.json": "release-source-review",
            "ui/startup.go.disabled": "archive",
        })
        self.assertFalse(records["archive/wallet_priv.hex"]["inIndex"])
        self.assertEqual(plan["mode"], "read-only")
