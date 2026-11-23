"""Installation/discovery contracts; no application import or live database."""
from pathlib import Path
import importlib.util
import re
import os
import sys
import tempfile
import unittest
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1]


class InstallationContracts(unittest.TestCase):
    def test_one_hash_locked_installation_including_bot_and_http_client(self):
        lock=(ROOT/"requirements.lock").read_text()
        packages=re.findall(r"^([a-zA-Z0-9_.-]+)==[^\n]+(?:\\\n[ \t]+--hash=sha256:[a-f0-9]{64})+",lock,re.M)
        self.assertGreater(len(packages),20)
        for name in ("fastapi","uvicorn","pydantic","httpx","python-telegram-bot","cryptography","py-ecc","python-dotenv"):
            self.assertIn(name,packages)
        self.assertIn("--require-hashes",(ROOT/"requirements.txt").read_text())
        for relative in ("telegram_game/requirements.txt","requirements-nft-evidence.txt","requirements-nft-witness.txt"):
            text=(ROOT/relative).read_text()
            self.assertIn("requirements.txt",text)
            self.assertFalse(re.search(r"^[a-zA-Z][\w-]*(?:==|>=)",text,re.M))

    def test_discovery_constructs_isolated_cases_without_runtime_import(self):
        spec=importlib.util.spec_from_file_location("isolated_test_support",ROOT/"isolated_contracts.py")
        module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        self.assertEqual(module.policy_suite().countTestCases(),31)
        suite=module.runtime_suite()
        self.assertGreaterEqual(suite.countTestCases(),37)
        self.assertEqual(len(suite._tests),0,"Discovery itself must not execute scripts")
        self.assertTrue(all(path.is_file() for path in suite.scripts))
        env=module.test_environment(ROOT/"tests/fixtures/test-only")
        self.assertEqual(env["QC_GAME_LOAD_DOTENV"],"0")
        self.assertEqual(env["QC_API_BASE"],"http://127.0.0.1:1")

    def test_one_page_and_external_controllers_with_legacy_redirects(self):
        ui=ROOT/"telegram_game/dev_ui"
        page=(ui/"game.html").read_text(encoding="utf-8")
        for attributes,body in re.findall(r"<script\b([^>]*)>(.*?)</script>",page,re.S):
            self.assertEqual(body.strip(),"")
            source=re.search(r'src="([^"]+)"',attributes)[1]
            if source.startswith("/dev_ui/"):
                self.assertIn("defer",attributes)
                self.assertTrue((ui/source.rsplit("/",1)[1]).is_file())
        for alias in (ui/"index.html",ui/"space_miner_prototype.html",ROOT/"telegram_game/index.html"):
            self.assertIn("url=/game",alias.read_text())
        scene=(ui/"game_scene.js").read_text(encoding="utf-8")
        self.assertNotIn("tgwtLedger",scene)
        self.assertNotRegex(scene,r"localStorage\.(getItem|setItem)\((CORE_STORAGE_KEY|TGWT_LEDGER_KEY|WALLET_SEED_KEY)")
        self.assertEqual(scene.count("function updateXP("),1)
        mining=(ui/"mining_controller.js").read_text()
        self.assertIn('"qc:mine-enter"',mining)
        self.assertNotIn("const oldEnter",mining)

    def test_canonical_http_entry_and_all_local_controller_assets(self):
        sys.path.insert(0,str(ROOT))
        with tempfile.TemporaryDirectory(prefix="qc-page-entry-") as directory, patch.dict(os.environ,{
            "QC_GAME_DB":str(Path(directory)/"test-only.sqlite3"),"QC_GAME_LOAD_DOTENV":"0",
            "QC_API_BASE":"http://127.0.0.1:1","TELEGRAM_BOT_TOKEN":"test-only-page-entry"}):
            from fastapi.testclient import TestClient
            from telegram_game import dev_api
            with TestClient(dev_api.app) as client:
                game=client.get("/game")
                self.assertEqual(game.status_code,200)
                self.assertEqual(client.get("/").text,game.text)
                self.assertIn('/dev_ui/account_wallet.js',game.text)
                for source in re.findall(r'(?:src|href)="(/dev_ui/[^"]+)"',game.text):
                    self.assertEqual(client.get(source).status_code,200,source)
                self.assertEqual(client.get("/dev_ui/qc_wallet_worker.js").status_code,200)
                assets=ROOT/"telegram_game/dev_ui/assets/qc_wallet_crypto"
                if (assets/"wallet_crypto.wasm").is_file():
                    wasm=client.get("/dev_ui/assets/qc_wallet_crypto/wallet_crypto.wasm")
                    self.assertEqual(wasm.status_code,200)
                    self.assertEqual(wasm.content[:4],b"\x00asm")
                    self.assertIn("application/wasm",wasm.headers["content-type"])
                    self.assertEqual(client.get("/dev_ui/assets/qc_wallet_crypto/wasm_exec.js").status_code,200)
                self.assertIn("url=/game",client.get("/dev_ui/space_miner_prototype.html").text)


if __name__=="__main__":unittest.main()
