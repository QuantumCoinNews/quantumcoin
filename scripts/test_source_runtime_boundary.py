"""Real Git and snapshot boundaries, with synthetic runtime/custody data only."""
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import go_source_audit
from source_hygiene import archive, check_source, cleanup_class, layout_inventory, source_exclusion, working_source_inventory


class SourceRuntimeBoundaryTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="qc-source-boundary-")
        self.addCleanup(temporary.cleanup)
        self.base = Path(temporary.name).resolve()
        self.repo = self.base / "source"
        self.repo.mkdir()
        self.source = {"main.go", "node_app_api_credentials.go", "go.mod", "go.sum", "package-lock.json",
                       "telegram_game/requirements.lock", "config.json.example", "configs/testnet.json",
                       ".env.example", "autopublisher/.env.example", "autopublisher/src/main.ts",
                       ".vscode/settings.json", ".vscode/launch.json",
                       "autopublisher/src/content/pool/topics_pool.json", "webui/dist/index.html",
                       "webui/dist/canonical-pow.mjs", "telegram_game/telegram_game/dev_ui/wallet_recovery.js"}
        self.excluded = {"config.json", ".wallet", ".env", "_goenv.txt", ".vscode/tasks.json",
                         "telegram_game/telegram_game/.vscode", "id_ed25519_quantumcoin_release",
                         "telegram_game/telegram_game/dev_ui/.check_scripts/script_01.js",
                         "ui/main_ui.go.bak", "gui/startup.go.disabled", "main.go.before_old",
                         "chain_data.dat", "state.sqlite3-wal",
                         "qc_blockchain.db.lock", "security_events.json", ".security-record.lock", "ai_state.json",
                         "node_modules/pkg/index.js", ".go-tmp/build/main.go", ".pytest_cache/state.json",
                         "autopublisher/dist/main.js", "autopublisher/.build/manifest.json",
                         "autopublisher/state/publish/ledger.json", "autopublisher/last.json",
                         "autopublisher/src/content/pool/pool_state.json", "autopublisher/local.dpapi.json",
                         "release/_testnet_backups/manifest.json", "webwallet/dist/index.html", "desktop.ini"}
        for name in self.source | self.excluded:
            path = self.repo / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(b"synthetic fixture only\n")
        self.git("init", "--quiet")
        self.git("config", "core.autocrlf", "false")
        self.git("add", "--force", ".")
        self.git("-c", "user.name=Boundary fixture", "-c", "user.email=fixture@example.invalid",
                 "-c", "commit.gpgsign=false", "-c", "core.hooksPath=" + str(self.base/"no-hooks"),
                 "commit", "--quiet", "-m", "synthetic source and runtime fixture")

    def git(self, *args):
        return subprocess.check_output(["git", *args], cwd=self.repo)

    def test_canonical_source_keeps_auth_assets_templates_and_locks(self):
        result = working_source_inventory(self.repo)
        self.assertEqual(set(result["sourcePaths"]), self.source)
        for name in self.source:
            self.assertFalse(source_exclusion(name), name)

    def test_backups_and_durable_ledgers_are_not_disposable_build_output(self):
        for name in ("offline-recovery/custody.json", "_backups/wallets.json",
                     "autopublisher/state/publish/ledger.json", "autopublisher/media/audio/original.wav"):
            with self.subTest(path=name):
                self.assertTrue(source_exclusion(name))
                self.assertIn(cleanup_class(name), {"archive", "runtime", "secret"})

    def test_snapshot_and_inventory_have_identical_membership_and_never_read_runtime(self):
        original = Path.read_bytes
        def guarded(path):
            if path.is_relative_to(self.repo) and path.relative_to(self.repo).as_posix() in self.excluded:
                raise AssertionError("Runtime/custody content was opened")
            return original(path)
        destination = self.base / "snapshot"
        destination.mkdir()
        with patch.object(go_source_audit, "ROOT", self.repo), patch.object(Path, "read_bytes", guarded):
            result = go_source_audit.snapshot(destination)
        self.assertEqual(result["files"], len(self.source))
        self.assertEqual({path.relative_to(destination).as_posix() for path in destination.rglob("*") if path.is_file()}, self.source)

    def test_metadata_layout_does_not_open_files_or_misreport_gate(self):
        with patch.object(Path, "open", side_effect=AssertionError("Metadata inventory opened a file")):
            result = layout_inventory(self.repo)
        self.assertIs(result["runtimeContentsRead"], False)
        self.assertIs(result["sourceGate"]["ok"], False)
        paths = {row["path"] for row in result["excludedWorkingPaths"]}
        self.assertIn("node_modules", paths)
        self.assertNotIn("node_modules/pkg/index.js", paths)
        self.assertIn(".wallet", paths)
        self.assertIn(".vscode/tasks.json", paths)
        self.assertNotIn(".vscode/settings.json", paths)
        self.assertNotIn(".vscode/launch.json", paths)

    def test_pruned_legacy_directories_require_custody_review(self):
        for name in ("bin", "build", "release/dist", "release_clean_test", "release_clean_test_old_20260517"):
            with self.subTest(path=name):
                self.assertEqual(cleanup_class(name + "/__layout_probe__"), "mixed-custody-review")
        self.assertEqual(cleanup_class("bin/wallet_priv.hex"), "secret")
        self.assertEqual(cleanup_class("bin/chain_data.dat"), "runtime")

    def test_encrypted_wallet_formats_are_private_in_git_source_and_export(self):
        real_root = Path(__file__).resolve().parents[1]
        for name in (".gitignore", ".gitattributes"):
            (self.repo / name).write_bytes((real_root / name).read_bytes())
        names = ["operator-backup." + extension for extension in ("qcwallet", "qcvault", "qcbak")]
        names += ["nested/.qc-wallet-store-fixture", "nested/.qc-wallet-fixture", "nested/.qc-encrypted-backup-fixture"]
        for name in names:
            with self.subTest(path=name):
                self.assertTrue(source_exclusion(name))
                self.assertEqual(cleanup_class(name), "secret")
                self.git("check-ignore", "--no-index", "--", name)
                self.assertIn(b"export-ignore: set", self.git("check-attr", "export-ignore", "--", name))

    def test_contaminated_head_is_rejected_even_after_index_removal(self):
        self.assertFalse(check_source(self.repo)["ok"])
        for name in sorted(self.excluded):
            self.git("update-index", "--force-remove", "--", name)
        result = check_source(self.repo)
        self.assertEqual(result["trackedExcludedPaths"], [])
        self.assertTrue(result["committedExcludedPaths"])
        target = self.base / "source.zip"
        with self.assertRaises(ValueError):
            archive(self.repo, target)
        self.assertFalse(target.exists())


if __name__ == "__main__":
    unittest.main()
