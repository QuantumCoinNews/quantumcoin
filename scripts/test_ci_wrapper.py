"""Exercise the real Windows CI wrapper's early rejection paths without downloads."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
import zipfile

ROOT = Path(__file__).resolve().parents[1]


@unittest.skipUnless(os.name == "nt", "Windows CI wrapper")
class CIWrapperTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="qc-ci-wrapper-")
        self.addCleanup(temporary.cleanup)
        self.base = Path(temporary.name)
        self.repo = self.base / "repo"
        (self.repo / "scripts").mkdir(parents=True)
        (self.repo / "configs").mkdir()
        self.script = self.repo / "scripts/build_windows_release_ci.ps1"
        shutil.copyfile(ROOT / "scripts/build_windows_release_ci.ps1", self.script)
        shutil.copyfile(ROOT / "configs/windows-build-toolchain.json", self.repo / "configs/windows-build-toolchain.json")
        self.env = {key: value for key, value in os.environ.items()
                    if not key.upper().startswith(("GIT_", "QC_"))}
        self.env.update(GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull,
                        GIT_AUTHOR_NAME="CI fixture", GIT_COMMITTER_NAME="CI fixture",
                        GIT_AUTHOR_EMAIL="fixture@example.invalid", GIT_COMMITTER_EMAIL="fixture@example.invalid")
        self.git("init", "--quiet")
        self.git("add", ".")
        self.git("-c", "core.hooksPath=" + str(self.base / "empty-hooks"), "-c", "commit.gpgsign=false",
                 "commit", "--quiet", "-m", "disposable CI fixture")
        self.commit = self.git("rev-parse", "HEAD").strip()

    def git(self, *args):
        return subprocess.check_output(["git", *args], cwd=self.repo, env=self.env, text=True, timeout=30)

    def invoke(self, commit, destination):
        return subprocess.run(["pwsh", "-NoProfile", "-NonInteractive", "-File", str(self.script),
                               "-Commit", commit, "-EvidenceDirectory", str(destination)],
                              cwd=self.repo, env=self.env, capture_output=True, text=True, timeout=60)

    def test_wrong_commit_cannot_accept_candidate(self):
        evidence = self.base / "evidence"
        result = self.invoke("0" * 40, evidence)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("selected commit", result.stderr)
        receipt = json.loads((evidence / "ci-gate.json").read_text(encoding="utf-8-sig"))
        self.assertIs(receipt["ok"], False)
        self.assertEqual(receipt["phase"], "started")
        self.assertEqual({p.name for p in evidence.iterdir()}, {"ci-gate.json"})

    def test_dirty_checkout_cannot_accept_candidate(self):
        self.script.write_text(self.script.read_text(encoding="utf-8") + "\n# dirty fixture\n", encoding="utf-8")
        evidence = self.base / "evidence"
        result = self.invoke(self.commit, evidence)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("clean checkout", result.stderr)
        self.assertIs(json.loads((evidence / "ci-gate.json").read_text(encoding="utf-8-sig"))["ok"], False)

    def test_receipt_inside_checkout_is_rejected_without_writing(self):
        evidence = self.repo / "evidence"
        result = self.invoke(self.commit, evidence)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(evidence.exists())

    def install_archive(self, checksum, url="https://fixture.invalid/toolchain.zip"):
        archive = self.base / "fixture.zip"
        with zipfile.ZipFile(archive, "w") as z:
            z.writestr("bin/probe.txt", "synthetic archive marker")
        if checksum == "actual":
            checksum = hashlib.sha256(archive.read_bytes()).hexdigest()
        script = self.base / "archive-fixture.ps1"
        script.write_text('''param($Wrapper,$Archive,$Destination,$Checksum,$Url)
$ErrorActionPreference='Stop'
$tokens=$null;$errors=$null
$ast=[Management.Automation.Language.Parser]::ParseFile($Wrapper,[ref]$tokens,[ref]$errors)
if($errors.Count){throw 'Invalid CI wrapper syntax'}
$function=$ast.Find({param($node) $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq 'Install-Archive'},$true)
if(!$function){throw 'Archive installer missing'}
. ([scriptblock]::Create($function.Extent.Text))
function Invoke-WebRequest {param($Uri,$OutFile) Copy-Item -LiteralPath $Archive -Destination $OutFile}
Install-Archive ([pscustomobject]@{url=$Url;sha256=$Checksum}) $Destination
''', encoding="utf-8")
        destination = self.base / "selected toolchain"
        result = subprocess.run(["pwsh", "-NoProfile", "-NonInteractive", "-File", str(script),
                                 str(self.script), str(archive), str(destination), checksum, url],
                                env=self.env, capture_output=True, text=True, timeout=60)
        return result, destination

    def test_matching_archive_is_verified_and_extracted(self):
        result, destination = self.install_archive("actual")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((destination / "unpacked/bin/probe.txt").read_text(), "synthetic archive marker")

    def test_changed_archive_cannot_be_extracted(self):
        result, destination = self.install_archive("0" * 64)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("SHA-256 mismatch", result.stderr)
        self.assertFalse((destination / "unpacked").exists())

    def test_invalid_hash_cannot_start_an_install(self):
        result, destination = self.install_archive("invalid")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(destination.exists())

    def test_plain_http_toolchain_is_rejected_before_download(self):
        result, destination = self.install_archive("actual", "http://fixture.invalid/toolchain.zip")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(destination.exists())


if __name__ == "__main__":
    unittest.main()
