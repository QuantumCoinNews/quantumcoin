"""Regression tests for release trust and package boundaries."""

import argparse
import contextlib
import copy
import hashlib
import json
import io
import os
from pathlib import Path
import shutil
import sys
import tempfile
import unittest
from unittest.mock import patch

import qc_release as release


class ReleaseTrustTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="qc-release-test-")
        self.addCleanup(temporary.cleanup)
        base = Path(temporary.name)
        self.package = base / "package"
        self.package.mkdir()
        self.key = base / "fixture-key"
        self.signers = base / "allowed-signers"
        release.run("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", str(self.key))
        public_key = self.key.with_suffix(".pub").read_text(encoding="utf-8")
        self.signers.write_text(f"{release.SIGNER} {public_key}", encoding="utf-8")
        self.commit = "a" * 40
        self.tree = "b" * 40
        self.policy_path = release.REPO / "configs/windows-build-toolchain.json"
        policy = json.loads(self.policy_path.read_bytes())
        self.go_version = policy["go"]["version"]
        self.cc_version = policy["mingw"]["version"]
        self.deps = {name: {"fixture/dependency": "v1.0.0"} for name in release.TARGETS}
        for name in release.BUILD_OUTPUTS:
            (self.package / name).write_bytes(name.encode())
        (self.package / release.SBOM).write_bytes(release.json_bytes(release.sbom_for("fixture", self.go_version, self.deps)))
        subjects = release.artifacts(self.package, release.BUILD_OUTPUTS | {release.SBOM})
        (self.package / release.PROVENANCE).write_bytes(release.json_bytes({
            "_type": release.STATEMENT,
            "predicateType": release.PREDICATE,
            "subject": [{"name": name, "digest": {"sha256": entry["sha256"]}} for name, entry in subjects.items()],
            "predicate": {
                "sourceCommit": self.commit, "sourceTree": self.tree, "target": "windows/amd64",
                "goVersion": self.go_version, "buildFlags": release.BUILD_FLAGS,
                "goCommandSha256": "d" * 64, "cCompilerVersion": self.cc_version,
                "cCompilerSha256": "e" * 64,
                "toolchainLockSha256": hashlib.sha256(self.policy_path.read_bytes()).hexdigest(),
                "walletCgoLdflags": release.CGO_LDFLAGS,
                "reproduction": release.REPRODUCTION,
            },
        }))
        self.manifest = self.package / release.MANIFEST
        self.manifest.write_bytes(release.json_bytes({
            "schema": "QC_RELEASE_V1", "version": "fixture", "target": "windows/amd64",
            "sourceCommit": self.commit, "sourceTree": self.tree, "goVersion": self.go_version,
            "files": release.artifacts(self.package, release.PAYLOAD),
        }))

    def sign(self):
        (self.package / release.SIGNATURE).unlink(missing_ok=True)
        release.run("ssh-keygen", "-Y", "sign", "-f", str(self.key), "-n", release.NAMESPACE, str(self.manifest))

    def verify(self, commit=None, signers=None, toolchain_lock=None):
        args = argparse.Namespace(directory=self.package, commit=commit or self.commit, trusted_signers=signers or self.signers, toolchain_lock=toolchain_lock)
        with patch.object(release, "inspect_binary", side_effect=lambda _go, _file, name, _rev, _tool: self.deps[name]):
            with contextlib.redirect_stdout(io.StringIO()) as output:
                release.verify(args)
        return output.getvalue()

    def rewrite_provenance(self, value):
        subjects = release.artifacts(self.package, release.BUILD_OUTPUTS | {release.SBOM})
        value["subject"] = [{"name": name, "digest": {"sha256": entry["sha256"]}} for name, entry in subjects.items()]
        (self.package / release.PROVENANCE).write_bytes(release.json_bytes(value))
        manifest = json.loads(self.manifest.read_bytes())
        manifest["files"] = release.artifacts(self.package, release.PAYLOAD)
        self.manifest.write_bytes(release.json_bytes(manifest))
        self.sign()

    def test_signed_provenance_requires_valid_toolchain_identity(self):
        original = json.loads((self.package / release.PROVENANCE).read_bytes())
        for field in ("goCommandSha256", "cCompilerVersion", "cCompilerSha256", "toolchainLockSha256"):
            with self.subTest(missing=field):
                value = copy.deepcopy(original)
                del value["predicate"][field]
                self.rewrite_provenance(value)
                with self.assertRaises(ValueError): self.verify()
        for field in ("goCommandSha256", "cCompilerSha256", "toolchainLockSha256"):
            with self.subTest(malformed=field):
                value = copy.deepcopy(original)
                value["predicate"][field] = "invalid-digest"
                self.rewrite_provenance(value)
                with self.assertRaises(ValueError): self.verify()

    def test_signed_release_cannot_choose_an_older_toolchain(self):
        value = json.loads((self.package / release.PROVENANCE).read_bytes())
        value["predicate"]["goVersion"] = "go1.24.5"
        (self.package / release.SBOM).write_bytes(release.json_bytes(release.sbom_for("fixture", "go1.24.5", self.deps)))
        manifest = json.loads(self.manifest.read_bytes())
        manifest["goVersion"] = "go1.24.5"
        self.manifest.write_bytes(release.json_bytes(manifest))
        self.rewrite_provenance(value)
        with self.assertRaises(ValueError): self.verify()

    def test_package_cannot_supply_its_own_toolchain_trust(self):
        self.sign()
        with self.assertRaises(ValueError): self.verify(toolchain_lock=self.manifest)

    def test_unsigned_and_wrong_commit_are_rejected(self):
        with self.assertRaisesRegex(ValueError, "including signature"):
            self.verify()
        self.sign()
        self.assertIn("VERIFIED_RELEASE", self.verify())
        with self.assertRaisesRegex(ValueError, "authorized commit"):
            self.verify(commit="c" * 40)

    def test_tampered_binary_and_extra_candidate_are_rejected(self):
        self.sign()
        candidate = self.package / "quantumcoin_saved.exe"
        candidate.write_bytes(b"extra executable")
        with self.assertRaisesRegex(ValueError, "unexpected or missing"):
            self.verify()
        candidate.unlink()
        (self.package / "quantumcoin.exe").write_bytes(b"tampered executable")
        with self.assertRaisesRegex(ValueError, "SHA-256 mismatch"):
            self.verify()

    def test_manifest_and_external_signer_must_be_trusted(self):
        self.sign()
        inside = self.package / "allowed-signers"
        inside.write_bytes(self.signers.read_bytes())
        with self.assertRaisesRegex(ValueError, "outside"):
            self.verify(signers=inside)
        inside.unlink()
        self.manifest.write_bytes(self.manifest.read_bytes().replace(b'"fixture"', b'"modified"'))
        with self.assertRaisesRegex(ValueError, "ssh-keygen"):
            self.verify()

    def test_runtime_files_cannot_enter_the_release(self):
        self.sign()
        for name in ("wallet_priv.hex", "wallet_recovery_words.txt", "chain_data.dat", "security_events.json"):
            with self.subTest(name=name):
                extra = self.package / name
                extra.write_text("fixture only", encoding="utf-8")
                with self.assertRaisesRegex(ValueError, "unexpected or missing"):
                    self.verify()
                extra.unlink()
        checkpoint = self.package / "_pre_testnet_checkpoints"
        checkpoint.mkdir()
        with self.assertRaisesRegex(ValueError, "unexpected or missing"):
            self.verify()


class ReleaseBuildTest(unittest.TestCase):
    @unittest.skipUnless(os.environ.get("QC_RELEASE_E2E_CC"), "requires an explicit MinGW compiler")
    def test_two_clean_builds_and_signed_audit(self):
        with tempfile.TemporaryDirectory(prefix="qc-release-build-fixture-") as temporary:
            base = Path(temporary)
            repo, package = base / "source", base / "package"
            sources = {
                "go.mod": "module quantumcoin\n\ngo 1.21\n",
                "main.go": 'package main\nimport "fmt"\nfunc main() { fmt.Println("fixture node") }\n',
                "cmd/quantumcoin-gui/main.go": 'package main\nfunc main() {}\n',
                "cmd/wallet/main.go": (
                    'package main\n// int testValue(void) { return 4; }\nimport "C"\n'
                    'import "fmt"\nfunc main() { fmt.Println(C.testValue()) }\n'
                ),
            }
            for name, source in sources.items():
                path = repo / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(source, encoding="utf-8")
            (repo / "tools").mkdir()
            (repo / "configs").mkdir()
            shutil.copyfile(release.REPO / release.TOOLCHAIN_LOCK, repo / release.TOOLCHAIN_LOCK)
            for name in (".gitignore", "tools/qc_release.py", "tools/qc_release_audit.ps1", "tools/qc_repo_safety_check.ps1"):
                shutil.copyfile(release.REPO / name, repo / name)
            (repo / "release").mkdir()
            for path in release.MINER_SCRIPTS.values():
                shutil.copyfile(release.REPO / path, repo / path)
            release.run("git", "init", "-q", cwd=repo)
            release.run("git", "add", ".", cwd=repo)
            release.run("git", "-c", "user.name=QC Fixture", "-c", "user.email=fixture@example.invalid",
                        "commit", "-qm", "release fixture", cwd=repo)
            commit = release.run("git", "rev-parse", "HEAD", cwd=repo).decode().strip()
            cc = os.environ["QC_RELEASE_E2E_CC"]
            cc_version = release.run(cc, "-dumpfullversion").decode().strip()
            go_version = release.run("go", "version").decode().split()[2]
            output = release.run(
                sys.executable, str(release.REPO / "tools/qc_release.py"), "build",
                "--repo", str(repo), "--commit", commit, "--version", "fixture-verification",
                "--go-version", go_version, "--cc", cc, "--cc-version", cc_version, "--output", str(package),
            )
            self.assertIn(b"UNSIGNED_CANDIDATE", output)
            key, trust = base / "fixture-key", base / "allowed-signers"
            release.run("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", str(key))
            trust.write_text(f"{release.SIGNER} " + key.with_suffix(".pub").read_text(encoding="utf-8"), encoding="utf-8")
            release.run("ssh-keygen", "-Y", "sign", "-f", str(key), "-n", release.NAMESPACE, str(package / release.MANIFEST))
            output = release.run(
                "pwsh", "-NoProfile", "-NonInteractive", "-File", str(repo / "tools/qc_repo_safety_check.ps1"),
                "-ReleaseDirectory", str(package), "-ReleaseCommit", commit, "-TrustedSigners", str(trust),
            )
            self.assertIn(b"VERIFIED_RELEASE", output)
            self.assertIn(b"QC_01_REPO_SAFETY_CHECK_PASS", output)
            (repo / "main.go").write_text("package main\nfunc main() {}\n", encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "clean source checkout"):
                release.run(
                    "pwsh", "-NoProfile", "-NonInteractive", "-File", str(repo / "tools/qc_repo_safety_check.ps1"),
                    "-ReleaseDirectory", str(package), "-ReleaseCommit", commit, "-TrustedSigners", str(trust),
                )


if __name__ == "__main__":
    unittest.main()
