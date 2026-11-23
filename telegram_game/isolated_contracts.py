"""unittest discovery adapter for isolated script and policy contracts.

Discovery imports no ASGI runtime and opens no application database. Every
scenario gets a fresh working directory/database and test-only credentials.
"""
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent
REPO = ROOT.parent


def test_environment(directory):
    allowed = {"PATH", "SYSTEMROOT", "WINDIR", "COMSPEC", "PATHEXT", "TEMP", "TMP",
               "HOME", "USERPROFILE", "LOCALAPPDATA", "GOROOT", "GOPATH", "GOCACHE", "CGO_ENABLED"}
    env = {k:v for k,v in os.environ.items() if k.upper() in allowed}
    env.update(PYTHONPATH=str(ROOT)+os.pathsep+str(ROOT/"telegram_game"),
               QC_GAME_DB=str(Path(directory)/"test-only.sqlite3"), QC_GAME_LOAD_DOTENV="0",
               QC_API_BASE="http://127.0.0.1:1", QC_API_TOKEN="test-only-discovery-node",
               QC_TEST_PYTHON=sys.executable,
               TELEGRAM_BOT_TOKEN="test-only-discovery-bot", TELEGRAM_GAME_ADMIN_TOKEN="test-only-discovery-admin",
               QC_ENABLE_CHAIN_NFT_MINT="0", QC_ENABLE_TG_NFT_MINT_BRIDGE="0")
    return env


class Resources:
    def __init__(self, directory, contract):
        self.directory = Path(directory)
        self.contract = contract
        self.failure = None
        self.wallet_built = False

    def environment(self, directory):
        env = test_environment(directory)
        if self.contract:
            if not self.wallet_built:
                generated = subprocess.run([sys.executable, str(REPO/"scripts/build_wallet_crypto.py")],
                    cwd=REPO, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=240)
                if generated.returncode:
                    raise AssertionError("Local wallet asset build failed:\n"+generated.stdout.decode("utf-8",errors="replace")[-8000:])
                self.wallet_built = True
            path = self.directory/"test-only-nft-contract.json"
            if not path.exists() and self.failure is None:
                generated = subprocess.run(["go", "test", "./node", "-run", "^TestNFTVerifiedLifecycleContractV1$", "-count=1", "-timeout", "180s"],
                    cwd=REPO, env={**env,"QMA_NFT_LIFECYCLE_CONTRACT_PATH":str(path)},
                    stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=210)
                if generated.returncode or not path.is_file():
                    self.failure = generated.stdout.decode("utf-8",errors="replace")[-8000:]
            if self.failure is not None:
                raise AssertionError("Go test contract generation failed:\n"+self.failure)
            env.update(QMA_NFT_LIFECYCLE_CONTRACT_PATH=str(path),
                       QMA_NFT_EVIDENCE_OUTPUT_DIR=str(self.directory/"test-only-evidence"))
        return env


class ScriptCase(unittest.TestCase):
    def __init__(self, script, resources):
        super().__init__("runTest")
        self.script, self.resources = script, resources

    def id(self):
        return "isolated."+self.script.stem

    def shortDescription(self):
        return str(self.script.relative_to(REPO))

    def runTest(self):
        with tempfile.TemporaryDirectory(prefix="qc-discovery-case-") as directory:
            executable = "node" if self.script.suffix == ".js" else sys.executable
            result = subprocess.run([executable,str(self.script)],cwd=directory,
                env=self.resources.environment(directory),stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=240)
        output = result.stdout.decode("utf-8",errors="replace")
        self.assertEqual(result.returncode,0,output[-12000:])


class IsolatedSuite(unittest.TestSuite):
    def __init__(self, scripts, contract):
        super().__init__()
        self.scripts, self.contract = list(scripts), contract

    def run(self, result, debug=False):
        with tempfile.TemporaryDirectory(prefix="qc-discovery-suite-") as directory:
            resources = Resources(directory,self.contract)
            self.addTests(ScriptCase(path,resources) for path in self.scripts)
            return super().run(result,debug)

    def countTestCases(self):
        return len(self.scripts)


def policy_suite():
    return IsolatedSuite(sorted((ROOT/"telegram_game").glob("*_test.py")),False)


def runtime_suite():
    scripts=sorted((ROOT/"tests").glob("test_*.py"))
    scripts+=sorted((ROOT/"tests").glob("test_*.js"))
    scripts.append(REPO/"scripts/test_monthly_nft_catalog_v1.py")
    return IsolatedSuite(scripts,True)
