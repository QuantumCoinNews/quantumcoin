import json
import os
from pathlib import Path
import tempfile
import unittest
import subprocess
import shutil
from concurrent.futures import ThreadPoolExecutor
from unittest.mock import patch

from network_workspaces import MARKER, ROOT, historical_inventory, marker_for, plan, prepare
from source_hygiene import source_exclusion


class FreshNetworkWorkspaceTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="qc-network-workspace-test-")
        self.addCleanup(self.temporary.cleanup)
        self.base = Path(self.temporary.name).resolve()

    def test_plan_is_read_only_and_defaults_are_separate(self):
        with patch.dict(os.environ, {"LOCALAPPDATA": str(self.base)}):
            testnet, mainnet = plan("testnet"), plan("mainnet")
        self.assertNotEqual(testnet["dataDir"], mainnet["dataDir"])
        self.assertFalse(Path(testnet["dataDir"]).exists())
        self.assertFalse(Path(mainnet["dataDir"]).exists())
        self.assertFalse(testnet["balance_migration"])
        self.assertFalse(testnet["launchPerformed"])
        self.assertEqual(mainnet["genesis_profile"], "network-v4")

    def test_prepare_creates_only_marker_and_restarts_preserve_data(self):
        directory = self.base / "testnet-v4"
        result = prepare("testnet", directory)
        self.assertTrue(result["prepared"])
        self.assertEqual([p.name for p in directory.iterdir()], [MARKER])
        chain = directory / result["chainFile"]
        chain.write_bytes(b"synthetic test-only data; node still must validate actual genesis")
        before = chain.read_bytes()
        prepare("testnet", directory)
        self.assertEqual(chain.read_bytes(), before)
        with self.assertRaises(ValueError):
            prepare("mainnet", directory)
        self.assertEqual(chain.read_bytes(), before)

    def test_nonempty_legacy_directory_never_migrates_or_overwrites(self):
        for name in ("chain_data.dat", "chain_data_testnet.dat", "wallet_priv.hex"):
            with self.subTest(name=name):
                directory = self.base / name.replace(".", "-")
                directory.mkdir()
                original = directory / name
                original.write_bytes(b"test-only historical bytes")
                with self.assertRaises(ValueError):
                    prepare("testnet", directory)
                self.assertEqual(original.read_bytes(), b"test-only historical bytes")
                self.assertFalse((directory / MARKER).exists())

    def test_source_and_release_package_are_rejected(self):
        with self.assertRaises(ValueError):
            prepare("testnet", ROOT / "test-only-runtime-must-not-be-created")
        package = self.base / "signed-package"
        with self.assertRaises(ValueError):
            prepare("testnet", package / "data", package)
        self.assertFalse(package.exists())
        with self.assertRaises(ValueError):
            plan("testnet", Path("relative-data"))

    def test_invalid_or_foreign_marker_is_preserved(self):
        directory = self.base / "testnet-v4"
        directory.mkdir()
        for raw in ("{", "null", "[]", json.dumps(marker_for("mainnet")),
                    json.dumps({**marker_for("testnet"), "balance_migration": True}),
                    json.dumps({**marker_for("testnet"), "balance_migration": 0}),
                    '{"network":"mainnet","network":"testnet"}'):
            with self.subTest(raw_type=raw[:1]):
                (directory / MARKER).write_text(raw)
                with self.assertRaises(ValueError):
                    prepare("testnet", directory)
                self.assertEqual((directory / MARKER).read_text(), raw)

    def test_indirect_directory_is_rejected(self):
        target = self.base / "target"
        target.mkdir()
        alias = self.base / "alias"
        try:
            alias.symlink_to(target, target_is_directory=True)
        except OSError:
            if os.name != "nt":
                self.skipTest("Symlink creation unavailable")
            result = subprocess.run(["cmd", "/c", "mklink", "/J", str(alias), str(target)], capture_output=True)
            if result.returncode:
                self.skipTest("Windows reparse-point creation unavailable")
        try:
            with self.assertRaises(ValueError):
                prepare("testnet", alias)
            self.assertEqual(list(target.iterdir()), [])
        finally:
            if alias.is_symlink():
                alias.unlink()
            else:
                # Remove only this test-owned junction, never its target.
                os.rmdir(alias)

    def test_foreign_chain_filename_is_rejected_after_preparation(self):
        directory = self.base / "testnet-v4"
        prepare("testnet", directory)
        foreign = directory / "chain_data.dat"
        foreign.write_bytes(b"test-only legacy chain")
        with self.assertRaises(ValueError):
            prepare("testnet", directory)
        self.assertEqual(foreign.read_bytes(), b"test-only legacy chain")

    def test_failed_marker_publication_does_not_claim_prepared(self):
        directory = self.base / "testnet-v4"
        with patch("network_workspaces.os.link", side_effect=OSError("test-only disk failure")):
            with self.assertRaises(OSError):
                prepare("testnet", directory)
        self.assertFalse(plan("testnet", directory)["prepared"])
        self.assertEqual(list(directory.iterdir()), [])

    def test_concurrent_preparation_never_replaces_network_identity(self):
        directory = self.base / "testnet-v4"
        def attempt(_):
            try:
                return prepare("testnet", directory)["prepared"]
            except (ValueError, OSError):
                return False
        with ThreadPoolExecutor(max_workers=2) as pool:
            results = list(pool.map(attempt, range(2)))
        self.assertTrue(any(results))
        self.assertEqual(json.loads((directory / MARKER).read_text()), marker_for("testnet"))
        self.assertEqual([p.name for p in directory.iterdir()], [MARKER])

    def test_inventory_hashes_only_chain_files_and_does_not_read_keys(self):
        (self.base / "chain_data.dat").write_bytes(b"historical chain A")
        (self.base / "wallet_priv.hex").write_text("test-only value never read")
        (self.base / "chain_private_key.dat").write_text("test-only value never read")
        inventory = historical_inventory(self.base)
        self.assertEqual([r["path"] for r in inventory["records"]], ["chain_data.dat"])
        self.assertEqual(inventory["records"][0]["classification"], "historical-only")
        self.assertFalse(inventory["records"][0]["importToFreshNetworks"])
        self.assertEqual((self.base / "chain_data.dat").read_bytes(), b"historical chain A")

    def test_runtime_marker_is_not_source_and_mainnet_template_stays_draft(self):
        self.assertTrue(source_exclusion(MARKER))
        config = json.loads((ROOT / "configs/mainnet.fresh.example.json").read_text())
        self.assertEqual(config["genesis_profile"], "network-v4")
        self.assertEqual(config["chain_file"], "chain_data.dat")
        self.assertNotIn("premine_address", config)

    def run_launcher(self, script, directory, plan_only=True):
        shell = shutil.which("pwsh") or shutil.which("powershell")
        if not shell:
            self.skipTest("PowerShell is required for Windows launcher integration")
        command = [shell, "-NoProfile", "-File", str(ROOT / "release" / script), "-DataDir", str(directory)]
        if plan_only:
            command.append("-PlanOnly")
        return subprocess.run(command, capture_output=True, text=True, timeout=30)

    def test_both_launch_wrappers_plan_fresh_directories_without_writing(self):
        for script, mode in (("RUN_TESTNET_API_18082.ps1", "api"), ("RUN_TESTNET_NODE_18082_13001.ps1", "run")):
            with self.subTest(script=script):
                directory = self.base / mode
                result = self.run_launcher(script, directory)
                self.assertEqual(result.returncode, 0, result.stderr)
                value = json.loads(result.stdout)
                self.assertEqual(value["mode"], mode)
                self.assertFalse(value["balanceMigration"])
                self.assertEqual(value["genesisProfile"], "network-v4")
                self.assertEqual(value["p2pListen"], "127.0.0.1:13001")
                self.assertFalse(directory.exists())

    def test_launch_wrapper_rejects_legacy_data_and_requires_release_before_preparation(self):
        directory = self.base / "old"
        directory.mkdir()
        original = directory / "chain_data_testnet.dat"
        original.write_bytes(b"historical test-only data")
        result = self.run_launcher("RUN_TESTNET_API_18082.ps1", directory)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(original.read_bytes(), b"historical test-only data")
        new_directory = self.base / "fresh"
        result = self.run_launcher("RUN_TESTNET_API_18082.ps1", new_directory, plan_only=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("verified release", result.stderr)
        self.assertFalse(new_directory.exists())


if __name__ == "__main__":
    unittest.main()
