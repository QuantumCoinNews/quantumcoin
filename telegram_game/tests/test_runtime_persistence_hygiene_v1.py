"""Isolated entrypoint, database policy, source hygiene and slow-node contracts."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile
if os.getenv("QC_PERSISTENCE_TEST_CHILD")!="1":
    with tempfile.TemporaryDirectory(prefix="qc-persistence-") as directory:
        env=os.environ.copy()
        env.update(QC_PERSISTENCE_TEST_CHILD="1",QC_GAME_DB=str(Path(directory)/"state.sqlite3"),
            TELEGRAM_BOT_TOKEN="test-only-persistence-bot",TELEGRAM_GAME_ADMIN_TOKEN="test-only-persistence-admin",QC_API_BASE="http://127.0.0.1:1",QC_API_TOKEN="test-only-node-token")
        raise SystemExit(subprocess.call([sys.executable,str(Path(__file__).resolve()),*sys.argv[1:]],cwd=directory,env=env))

import asyncio
from contextlib import closing
import concurrent.futures
import json
import runpy
import shutil
import threading
import time
import unittest
from unittest.mock import patch,AsyncMock
import zipfile
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
sys.path.insert(0,str(Path(__file__).resolve().parents[2]/"scripts"))
from fastapi.testclient import TestClient
from telegram_game import dev_api as api,runtime,api_server,player_auth,game_pow,persistence,mining_sessions
from telegram_game.handlers import tgwt_router,withdraw_router
from telegram_game.database.store import Store
import source_hygiene
ROOT=Path(__file__).resolve().parents[2]


class PersistenceTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory(prefix="qc-path-policy-")
        self.root=Path(self.temp.name)
    def tearDown(self):self.temp.cleanup()
    def resolve(self,env=None,**kwargs):
        return persistence.database_path({} if env is None else env,home=self.root/"home",source_root=self.root/"source",working_dir=self.root/"cwd",profile=kwargs.pop("profile","local"),**kwargs)
    def assert_same_database_path(self,actual,expected):
        actual_path=Path(actual);expected_path=Path(expected)
        self.assertEqual(actual_path.name.casefold(),expected_path.name.casefold())
        if os.name=="nt":
            self.assertTrue(actual_path.parent.samefile(expected_path.parent))
        else:
            self.assertEqual(actual_path.parent.resolve(),expected_path.parent.resolve())
    def test_default_is_stable_and_profiles_are_separate(self):
        a=self.resolve()
        b=persistence.database_path({},home=self.root/"home",source_root=self.root/"source",working_dir=self.root/"another",profile="local")
        self.assertEqual(a,b);self.assertNotEqual(a,self.resolve(profile="testnet"));self.assertNotEqual(a,self.resolve(profile="mainnet"))
        self.assertNotIn("cwd",a);self.assertNotIn("source",a)
    def test_relative_paths_and_invalid_profiles_fail(self):
        for env in ({"QC_GAME_DB":"state.db"},{"QC_GAME_DATA_DIR":"relative"}):
            with self.assertRaises(ValueError):self.resolve(env)
        with self.assertRaisesRegex(ValueError,"retired"):self.resolve({"QC_TGWT_DB":"old.sqlite3"})
        with self.assertRaises(ValueError):self.resolve(profile="../escape")
    def test_explicit_database_keeps_one_path_and_creates_parent(self):
        path=self.root/"new"/"data"/"state.sqlite3"
        path.parent.mkdir(parents=True)
        self.assert_same_database_path(self.resolve({"QC_GAME_DB":str(path)}),path)
        with closing(persistence.connect_database(str(path))) as conn:
            self.assertEqual(conn.execute("PRAGMA journal_mode").fetchone()[0],"wal")
            self.assertEqual(conn.execute("PRAGMA busy_timeout").fetchone()[0],5000)
    def test_legacy_detection_does_not_copy_merge_or_create_default(self):
        source=self.root/"source";source.mkdir();legacy=source/"dev_api_state.db";legacy.write_bytes(b"test-only-not-a-real-db")
        before=legacy.read_bytes()
        with self.assertRaisesRegex(ValueError,"Legacy"):self.resolve()
        self.assertEqual(legacy.read_bytes(),before);self.assertFalse((self.root/"home").exists())
        self.assert_same_database_path(self.resolve({"QC_GAME_DB":str(legacy)}),legacy)
    def test_custom_data_base_is_network_scoped(self):
        base=self.root/"data"
        expected=base/"testnet"/"state.sqlite3";expected.parent.mkdir(parents=True)
        self.assert_same_database_path(self.resolve({"QC_GAME_DATA_DIR":str(base)},profile="testnet"),expected)
    def test_aliases_in_distinct_processes_and_cwds_share_same_explicit_db(self):
        path=self.root/"shared"/"game.sqlite3"
        code="import json;from telegram_game import runtime,api_server,dev_api;print(json.dumps({'same':runtime.app is api_server.app is dev_api.app,'path':dev_api.APP_DB}))"
        env={**os.environ,"QC_GAME_DB":str(path),"PYTHONPATH":str(ROOT/"telegram_game")}
        for cwd in (self.root, self.root/"other"):
            cwd.mkdir(exist_ok=True)
            done=subprocess.run([sys.executable,"-c",code],cwd=cwd,env=env,capture_output=True,text=True,timeout=30)
            self.assertEqual(done.returncode,0,done.stderr);data=json.loads(done.stdout)
            self.assertTrue(data["same"]);self.assertTrue(os.path.samefile(data["path"],path))
            self.assertFalse((cwd/"dev_api_state.db").exists())
    def test_launcher_configuration_delegates_to_one_policy(self):
        for path in (ROOT/"telegram_game").glob("RUN_TELEGRAM_GAME_SERVER*.ps1"):
            text=path.read_text(encoding="utf-8-sig")
            self.assertIn("telegram_game.runtime:app",text)
            self.assertIn("--no-proxy-headers",text)
            self.assertNotIn('$env:QC_GAME_DB = Join-Path',text)
            self.assertNotIn('Set-Location -Path "C:',text)


class SourceHygieneTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory(prefix="qc-source-export-");self.root=Path(self.temp.name)
        self.git("init","--quiet");self.git("config","user.email","test-only@example.invalid");self.git("config","user.name","Source Test")
        (self.root/"README.md").write_text("test-only source\n")
        shutil.copyfile(ROOT/".gitignore",self.root/".gitignore");shutil.copyfile(ROOT/".gitattributes",self.root/".gitattributes")
        (self.root/"telegram_game").mkdir();shutil.copyfile(ROOT/"telegram_game/.gitignore",self.root/"telegram_game/.gitignore")
        (self.root/".env.example").write_text("PUBLIC_CONFIG=\n")
        self.git("add",".");self.git("commit","--quiet","-m","test-only source")
    def tearDown(self):self.temp.cleanup()
    def git(self,*args):return subprocess.run(["git",*args],cwd=self.root,check=True,capture_output=True)
    def test_actual_repo_tracks_no_runtime_databases(self):
        self.assertTrue(source_hygiene.check(ROOT)["ok"])
    def test_database_and_all_sqlite_sidecars_are_ignored_and_excluded(self):
        for suffix in (".db",".db-wal",".db-shm",".db-journal",".db.backup",".sqlite",".sqlite-wal",".sqlite3",".sqlite3-shm",".sqlite3.backup"):
            relative="telegram_game/runtime"+suffix
            (self.root/relative).write_bytes(b"test-only runtime bytes")
            self.git("check-ignore","--quiet",relative)
            self.assertIn(b"set",self.git("check-attr","export-ignore","--",relative).stdout)
        target=self.root/"safe.zip";source_hygiene.archive(self.root,target)
        with zipfile.ZipFile(target) as archive:
            self.assertIn("quantumcoin/README.md",archive.namelist());self.assertIn("quantumcoin/.env.example",archive.namelist())
            self.assertFalse(any(source_hygiene.runtime_data_path(p) for p in archive.namelist()))
    def test_forced_tracked_db_is_detected_and_archive_refused(self):
        path=self.root/"telegram_game"/"forced.db";path.write_bytes(b"test-only runtime bytes")
        self.git("add","-f",str(path))
        self.assertFalse(source_hygiene.check(self.root)["ok"])
        with self.assertRaises(ValueError):source_hygiene.archive(self.root,self.root/"refused.zip")
        self.assertFalse((self.root/"refused.zip").exists())
    def test_archive_never_overwrites_an_existing_file(self):
        target=self.root/"existing.zip";target.write_bytes(b"preserve")
        with self.assertRaises(FileExistsError):source_hygiene.archive(self.root,target)
        self.assertEqual(target.read_bytes(),b"preserve")

    def test_source_archive_refuses_binaries_raw_diagnostics_and_secrets(self):
        paths = ("bin/quantumcoin.exe", "__qcdiag/go_env.txt", ".wallet")
        for name in paths:
            path = self.root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(b"test fixture only")
            self.git("add", "-f", "--", name)
        target = self.root / "refused.zip"
        self.assertEqual(set(source_hygiene.check_source(self.root)["trackedExcludedPaths"]), set(paths))
        with self.assertRaises(ValueError):
            source_hygiene.archive(self.root, target)
        self.git("commit", "--quiet", "-m", "tracked fixture exclusions")
        self.git("rm", "--cached", "--", *paths)
        result = source_hygiene.check_source(self.root)
        self.assertEqual(result["trackedExcludedPaths"], [])
        self.assertEqual(set(result["committedExcludedPaths"]), set(paths))
        with self.assertRaises(ValueError):
            source_hygiene.archive(self.root, target)
        self.assertFalse(target.exists())


class RuntimeConcurrencyTests(unittest.TestCase):
    def setUp(self):
        self.actor="tg_91112";self.now=int(time.time())
        with closing(api._db()) as conn,conn:
            conn.execute("DELETE FROM request_limits_v1")
            api._touch_player(conn,self.actor,None);api._ensure_qc_address_column(conn)
            conn.execute("UPDATE players SET qc_address='QC_test_concurrency' WHERE player_id=?",(self.actor,))
            conn.execute("DELETE FROM game_pow_work_v1 WHERE player_id=?",(self.actor,))
            conn.execute("DELETE FROM game_mining_runs_v1 WHERE player_id=?",(self.actor,))
            conn.execute("DELETE FROM mining_sessions WHERE player_id=?",(self.actor,))
            self.session=mining_sessions.open_run(conn,self.actor,1,self.now-65)["sessionId"]
        credential=player_auth.issue_session(api._db,self.actor,api._telegram_bot_token(),self.now)
        self.headers={"Authorization":"Bearer "+credential["accessToken"]}
    def test_compatibility_app_cannot_enable_demo_economy_or_admin_default(self):
        self.assertIs(runtime.app,api_server.app);self.assertIs(runtime.app,api.app)
        self.assertEqual(tgwt_router.routes+withdraw_router.routes,[])
        with self.assertRaisesRegex(RuntimeError,"retired"):Store()
        with patch.dict(os.environ,{"TELEGRAM_GAME_ADMIN_TOKEN":"","QC_ADMIN_TOKEN":"","TGWT_OPEN_EARN":"1","ADMIN_KEY":""}),TestClient(api_server.app) as client:
            self.assertEqual(client.post("/api/v1/tgwt/earn",json={"userId":self.actor,"amount":"1","nonce":"demo"}).status_code,401)
            self.assertEqual(client.post("/api/admin/tgwt/withdraw/confirm",json={"requestId":"demo"}).status_code,401)
            response=client.get("/api/store",headers={"Origin":"https://untrusted.invalid"})
            self.assertEqual(response.status_code,200)
            self.assertNotEqual(response.headers.get("access-control-allow-origin"),"*")
    def test_module_start_is_loopback_without_reload_or_proxy_trust(self):
        with patch("uvicorn.run") as start:runpy.run_module("telegram_game.api_server",run_name="__main__")
        self.assertEqual(start.call_args.kwargs,{"host":"127.0.0.1","port":8080,"proxy_headers":False,"reload":False})
    def probe_released_db(self):
        self.assertTrue(api._lock.acquire(blocking=False));api._lock.release()
        with closing(api._db()) as conn:
            conn.execute("PRAGMA busy_timeout=100");conn.execute("BEGIN IMMEDIATE");conn.rollback()
    def assert_independent_routes(self,client):
        for method,path,kwargs in (("post","/api/presence/heartbeat",{"json":{"playerId":self.actor}}),
            ("get","/api/wallet",{"params":{"playerId":self.actor}}),
            ("get","/api/v1/tgwt/state",{"params":{"user_id":self.actor}})):
            response=getattr(client,method)(path,headers=self.headers,**kwargs)
            self.assertEqual(response.status_code,200,response.text)
            if path=="/api/wallet":
                self.assertEqual(response.json()["playerId"],self.actor)
                self.assertTrue(response.json()["real_qc_ok"],response.text)
            else:self.assertTrue(response.json()["ok"],response.text)
    def test_slow_health_bridge_releases_lock_connection_and_event_loop(self):
        entered=threading.Event();release=threading.Event()
        def health():
            self.probe_released_db();entered.set()
            if not release.wait(8):raise AssertionError("test release timed out")
            return {"ok":True}
        with TestClient(api.app) as client,patch.object(api,"_fetch_qc_node_health_sync",side_effect=health),patch.object(api,"qc_fetch_address_balance",new=AsyncMock(return_value={"ok":True,"atomsPerQc":"1000000","balanceAtoms":"0","spendableAtoms":"0","balanceQc":"0","spendableQc":"0"})),concurrent.futures.ThreadPoolExecutor(2) as pool:
            slow=pool.submit(client.get,"/api/bridge/status",headers=self.headers,params={"playerId":self.actor})
            try:
                self.assertTrue(entered.wait(3));pool.submit(self.assert_independent_routes,client).result(timeout=3)
            finally:release.set()
            self.assertTrue(slow.result(timeout=3).json()["ok"])
    def test_slow_node_job_does_not_hold_game_database(self):
        entered=threading.Event();release=threading.Event()
        async def node(*args,**kwargs):
            self.probe_released_db();entered.set();await asyncio.to_thread(release.wait,8)
            return {"ok":False,"error":"test-only-unavailable"}
        with TestClient(api.app) as client,patch.object(game_pow,"node_request",side_effect=node),patch.object(api,"qc_fetch_address_balance",new=AsyncMock(return_value={"ok":True,"atomsPerQc":"1000000","balanceAtoms":"0","spendableAtoms":"0","balanceQc":"0","spendableQc":"0"})),concurrent.futures.ThreadPoolExecutor(2) as pool:
            slow=pool.submit(client.post,"/api/mine/work",headers=self.headers,json={"playerId":self.actor,"sessionId":self.session})
            try:
                self.assertTrue(entered.wait(3));pool.submit(self.assert_independent_routes,client).result(timeout=3)
            finally:release.set()
            self.assertFalse(slow.result(timeout=3).json()["ok"])
    def test_finish_uses_stored_pow_without_global_lock_or_node_mining(self):
        with closing(api._db()) as conn,conn:
            conn.execute("INSERT INTO game_pow_work_v1(session_id,player_id,address,request_key,expires_at,status,receipt_json) VALUES(?,?,?,?,?,'ACCEPTED',?)",(self.session,self.actor,"QC_test_concurrency","test",self.now+30,json.dumps({"hash":"ab"*32,"height":2})))
        with TestClient(api.app) as client,patch.object(api,"_get_game_mining_mode",return_value="REAL_CHAIN"),patch.object(api,"qc_fetch_mine_block",side_effect=AssertionError("server mining forbidden")),concurrent.futures.ThreadPoolExecutor(1) as pool:
            api._lock.acquire()
            try:
                done=pool.submit(client.post,"/api/mine/finish",headers=self.headers,json={"playerId":self.actor,"sessionId":self.session,"siteId":1})
                result=done.result(timeout=3).json()
            finally:api._lock.release()
            self.assertTrue(result["ok"],result);self.assertEqual(result["qcEarned"],0);self.assertEqual(result["tgwtReward"],0)


if __name__=="__main__":unittest.main(verbosity=2)
