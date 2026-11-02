"""Integration of QMA-0070A with the real FastAPI admin routes, temp SQLite only."""
import datetime
import hashlib
import os
import sqlite3
import subprocess
import sys
import tempfile
import unittest
from contextlib import closing
from pathlib import Path
from unittest.mock import patch

ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT))
if __name__ == "__main__" and os.getenv("QMA0070_FREEZE_API_CHILD") != "1":
    with tempfile.TemporaryDirectory(prefix="qma0070-freeze-api-process-") as temp:
        env=os.environ.copy()
        env.update(QMA0070_FREEZE_API_CHILD="1",QC_API_BASE="http://127.0.0.1:1",
                   QC_API_TOKEN="test-only-unreachable-node",TELEGRAM_GAME_ADMIN_TOKEN="test-only-freeze-admin",
                   TELEGRAM_BOT_TOKEN="test-only-telegram",QC_ENABLE_CHAIN_NFT_MINT="0",QC_ENABLE_TG_NFT_MINT_BRIDGE="0")
        env["PYTHONPATH"]=str(ROOT)+os.pathsep+env.get("PYTHONPATH","")
        done=subprocess.run([sys.executable,str(Path(__file__).resolve()),*sys.argv[1:]],
                            cwd=temp,env=env,timeout=120)
    raise SystemExit(done.returncode)

from fastapi.testclient import TestClient
with tempfile.TemporaryDirectory(prefix="qma0070-freeze-startup-") as startup:
    original=sqlite3.connect
    def isolated(database,*args,**kwargs):
        if str(database)=="dev_api_state.db": database=str(Path(startup)/"startup.sqlite")
        return original(database,*args,**kwargs)
    with patch.object(sqlite3,"connect",side_effect=isolated):
        from telegram_game import dev_api as api
from telegram_game.utils.nft_draw_freeze_v1 import verify_frozen_pool_v1

PERIOD="2032-01"
NOW=int(datetime.datetime(2032,2,2,tzinfo=datetime.timezone.utc).timestamp())
ACTIVATION=int(datetime.datetime(2031,12,15,tzinfo=datetime.timezone.utc).timestamp())


def wallet():
    raw=b"\x00" + b"\x01"*20
    raw+=hashlib.sha256(hashlib.sha256(raw).digest()).digest()[:4]
    chars="123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
    value=int.from_bytes(raw,"big")
    result=""
    while value:
        value, digit=divmod(value,58)
        result=chars[digit]+result
    return "1"+result


class FreezeAPIIntegration(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory(prefix="qma0070-freeze-route-")
        self.patches=[patch.object(api,"APP_DB",str(Path(self.temp.name)/"game.sqlite")),
                      patch.object(api,"_now",return_value=NOW),
                      patch.object(api,"_nft_verified_mainnet_activation_at_v1",return_value=ACTIVATION)]
        for p in self.patches: p.start()
        api._init()
        with closing(api._db()) as conn, conn:
            api._ensure_qc_address_column(conn)
            api._ensure_monthly_nft_tables(conn)
            api._ensure_monthly_nft_verified_game_sessions_v1(conn)
            api._touch_player(conn,"tg_101",None)
            conn.execute("UPDATE players SET qc_address=? WHERE player_id='tg_101'",(wallet(),))
            conn.execute("INSERT INTO monthly_nft_candidates VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
                         ("candidate-a","tg_101",PERIOD,"rare","server_verified_game_session","{}",.65,
                          "AI_REVIEW_PASSED","LOW","APPROVED_BY_ADMIN","",
                          "APPROVED_CANDIDATE_NOT_WINNER","NOT_MINTED","test",NOW-10,NOW-10))
            completed=int(datetime.datetime(2032,1,10,tzinfo=datetime.timezone.utc).timestamp())
            conn.execute("INSERT INTO monthly_nft_verified_game_sessions VALUES(?,?,?,?,1)",
                         ("session-a","tg_101",PERIOD,completed))
        api._QC_ACTION_BURST_EVENTS_V1.clear()
        api._QC_ACTION_BURST_BLOCKS_V1.clear()
        self.client=TestClient(api.app)
        self.headers={"X-Admin-Token":"test-only-freeze-admin"}

    def tearDown(self):
        self.client.close()
        for p in reversed(self.patches):p.stop()
        self.temp.cleanup()

    def test_admin_auth_freeze_read_and_old_select_block(self):
        uri="/api/admin/nft/monthly/freeze-pool"
        self.assertEqual(self.client.post(uri,json={"period":PERIOD}).status_code,401)
        frozen=self.client.post(uri,json={"period":PERIOD},headers=self.headers)
        self.assertEqual(frozen.status_code,200,frozen.text)
        self.assertTrue(frozen.json()["ok"],frozen.text)
        result=frozen.json()
        self.assertTrue(verify_frozen_pool_v1(result["manifest"],result["poolHash"]))
        retrieved=self.client.get("/api/admin/nft/monthly/frozen-pool",params={"period":PERIOD},headers=self.headers)
        self.assertEqual(retrieved.status_code,200,retrieved.text)
        self.assertEqual(retrieved.json()["poolHash"],result["poolHash"])
        selected=self.client.post("/api/admin/nft/monthly/select-winner",
                                  json={"candidateId":"candidate-a"},headers=self.headers)
        self.assertEqual(selected.status_code,200,selected.text)
        self.assertEqual(selected.json()["error"],"monthly_pool_frozen_manual_selection_disabled")
        with closing(api._db()) as conn:
            self.assertEqual(conn.execute("SELECT COUNT(*) FROM monthly_nft_candidates WHERE winner_status='SELECTED_MONTHLY_WINNER'").fetchone()[0],0)


if __name__ == "__main__": unittest.main(verbosity=2)
