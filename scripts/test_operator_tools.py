"""Exercise the PowerShell tools over loopback without running a node or miner."""

from contextlib import contextmanager
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import threading
import unittest


ROOT = Path(__file__).resolve().parents[1]
LOAD = ROOT / "tools/qc_load_test_safe_v1.ps1"
TOKEN = "loopback-fixture-only"
ADDRESS = "1" * 32
GENESIS = "a" * 64


@contextmanager
def server(response):
    records = []

    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *_):
            pass

        def respond(self):
            size = int(self.headers.get("Content-Length", "0"))
            body = self.rfile.read(size) if size else b""
            key = self.headers.get("X-QC-API-KEY")
            records.append((self.command, self.path, key, body, self.headers.get("X-QC-Expected-Genesis")))
            status, result = response(self.command, self.path, key)
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            if status == 302:
                self.send_header("Location", "/redirect-target")
            self.end_headers()
            self.wfile.write(json.dumps(result).encode())

        do_GET = do_POST = respond

    with ThreadingHTTPServer(("127.0.0.1", 0), Handler) as httpd:
        thread = threading.Thread(target=httpd.serve_forever)
        thread.start()
        try:
            yield f"http://127.0.0.1:{httpd.server_port}", records
        finally:
            httpd.shutdown()
            thread.join()


def execute(argv, token=TOKEN, **settings):
    env = {**os.environ, **settings}
    env.pop("QC_API_TOKEN", None)
    if token is not None:
        env["QC_API_TOKEN"] = token
    return subprocess.run(argv, env=env, capture_output=True, text=True, timeout=30)


class LoadTest(unittest.TestCase):
    def load(self, base, *options, token=TOKEN):
        return execute(["pwsh", "-NoProfile", "-File", str(LOAD), "-BaseUrl", base, *options], token)

    def test_plan_and_real_parallel_health(self):
        with server(lambda _method, _path, key: (200, {"ok": True}) if key == TOKEN else (401, {})) as (base, records):
            plan = self.load(base + "///", "-PlanOnly", token=None)
            self.assertEqual(plan.returncode, 0, plan.stderr)
            self.assertEqual(json.loads(plan.stdout)["endpoints"], ["/api/health"])
            self.assertEqual(records, [])
            result = self.load(base + "///", "-Requests", "4", "-Concurrency", "2")
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(json.loads(result.stdout)["okCount"], 4)
            self.assertEqual(records, [("GET", "/api/health", TOKEN, b"", None)] * 4)
            self.assertNotIn(TOKEN, result.stdout + result.stderr)

    def test_unauthorized_is_a_failed_run_and_unsafe_gets_are_rejected(self):
        with server(lambda *_: (401, {})) as (base, records):
            result = self.load(base, "-Requests", "2")
            self.assertNotEqual(result.returncode, 0)
            report = json.loads(result.stdout)
            self.assertFalse(report["ok"])
            self.assertEqual(report["failCount"], 2)
            self.assertEqual({row["status"] for row in report["results"]}, {401})
            for path in ("/api/dev/fastmine", "/api/mine/job", "/api/%6dine/job"):
                denied = self.load(base, "-Endpoint", path, "-PlanOnly")
                self.assertNotEqual(denied.returncode, 0)
            self.assertEqual(len(records), 2)


class MinerTest(unittest.TestCase):
    def run_case(self, health_status, mine_status=200, token=TOKEN, *, network="local-release", read_only=False, legacy=False):
        with tempfile.TemporaryDirectory(prefix="qc-miner-fixture-") as path:
            entry = Path(path) / "run_miner.cmd"
            shutil.copyfile(ROOT / entry.name, entry)
            runtime = entry.parent / "release"
            runtime.mkdir()
            for name in ("run_miner.cmd", "_qc_api_mempool_miner.ps1"):
                shutil.copyfile(ROOT / "release" / name, runtime / name)

            def response(_method, endpoint, _key):
                if endpoint == "/api/health":
                    health = {"ok": health_status == 200}
                    if not legacy:
                        health.update(apiReadOnly=read_only, networkIdentity={
                            "version": "QC_CHAIN_NETWORK_IDENTITY_V1", "network": network,
                            "genesisDomain": "QC_LOCAL_RELEASE_GENESIS_V2", "genesisHash": GENESIS,
                        })
                    return health_status, health
                if endpoint == "/api/mine/local-release" and mine_status == 200:
                    (runtime / "miner_stop.flag").touch()
                    return 200, {"success": True, "height": 42}
                return mine_status, {"success": False}

            with server(response) as (base, records):
                output = execute(["cmd.exe", "/d", "/c", str(entry)],
                                 token, ADDR="", QC_ADDR=ADDRESS, QC_API_BASE=base + "/")
            log = runtime / "miner_out.log"
            logged = log.read_text(encoding="utf-8") if log.exists() else ""
            self.assertNotIn(TOKEN, output.stdout + output.stderr + logged)
            return output, records, logged

    def test_health_and_mine_use_the_header_and_stop_after_one_block(self):
        result, records, log = self.run_case(200)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(records[0], ("GET", "/api/health", TOKEN, b"", None))
        self.assertEqual(records[1][:3], ("POST", "/api/mine/local-release", TOKEN))
        self.assertEqual(json.loads(records[1][3]), {"address": ADDRESS})
        self.assertEqual(records[1][4], GENESIS)
        self.assertEqual(len(records), 2)
        self.assertIn("Block mined at height 42", log)

    def test_auth_redirect_and_read_only_errors_stop_the_client(self):
        for health, mine, token, count in [(401, 200, TOKEN, 1), (302, 200, TOKEN, 1),
                                          (200, 403, TOKEN, 2), (200, 200, None, 0)]:
            with self.subTest(health=health, mine=mine, has_token=token is not None):
                result, records, log = self.run_case(health, mine, token)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(len(records), count)
                self.assertNotIn("Block mined", log)
                if health == 401:
                    self.assertIn("HTTP 401", log)
                if mine == 403:
                    self.assertIn("HTTP 403", log)

    def test_legacy_mainnet_and_observer_never_receive_a_mining_request(self):
        for settings in ({"legacy": True}, {"network": "mainnet"}, {"read_only": True}):
            with self.subTest(settings=settings):
                result, records, log = self.run_case(200, **settings)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual([request[1] for request in records], ["/api/health"])
                self.assertNotIn("Block mined", log)


if __name__ == "__main__":
    unittest.main()
