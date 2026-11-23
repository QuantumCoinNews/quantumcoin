"""Isolated work refresh, durable proof reconciliation and continuous solo mining."""
import asyncio
from contextlib import closing
import hashlib
import json
from pathlib import Path
import sqlite3
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from telegram_game import game_pow, mining_sessions


class MiningLifecycleTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self):
        self.directory=tempfile.TemporaryDirectory(prefix="qc-work-lifecycle-")
        self.addCleanup(self.directory.cleanup)
        self.path=Path(self.directory.name)/"test-only.sqlite"
        self.now=int(time.time())
        self.genesis="ab"*32
        self.patch=patch.object(game_pow,"expected_network",return_value=("testnet",self.genesis))
        self.patch.start();self.addCleanup(self.patch.stop)
        with closing(self.db()) as conn,conn:
            conn.execute("CREATE TABLE players(player_id TEXT PRIMARY KEY,qc_address TEXT)")
            conn.execute("CREATE TABLE mining_sessions(player_id TEXT PRIMARY KEY,site_id INTEGER,started_at INTEGER)")
            mining_sessions.ensure_schema(conn);game_pow.ensure_schema(conn)
            conn.execute("INSERT INTO players VALUES ('tg_test','test-wallet')")
            self.session=mining_sessions.open_run(conn,"tg_test",1,self.now-65)["sessionId"]
        self.job={"network":"testnet","genesis_hash":self.genesis,"prev_hash":"cd"*32,
                  "protocol":"QC_WEB_BLOCK_POW_V1","miner":"test-wallet","nonce_encoding":"uint64-big-endian",
                  "header":"00"*24,"nonce_offset":16,"max_nonce":"9223372036854775807",
                  "difficulty":8,"height":6,"expires":self.now+60,"job_id":"ef"*32}

    def db(self):
        return sqlite3.connect(self.path)

    def tip(self,**extra):
        return {"version":"QC_MINING_TIP_V1","network":"testnet","genesis_hash":self.genesis,
                "height":5,"hash":self.job["prev_hash"],**extra}

    async def issue(self):
        async def node(path,payload=None,params=None):return self.job
        with patch.object(game_pow,"node_request",side_effect=node):
            result=await game_pow.issue_work(self.db,"tg_test",self.session,self.now)
        self.assertTrue(result["ok"],result)

    async def test_equal_height_reorg_replaces_unexpired_work(self):
        await self.issue()
        tip=self.tip(hash="12"*32)
        replacement={**self.job,"job_id":"34"*32,"prev_hash":tip["hash"]}
        async def node(path,payload=None,params=None):
            return tip if path=="/api/mine/tip" else replacement
        with patch.object(game_pow,"node_request",side_effect=node):
            state=await game_pow.work_status(self.db,"tg_test",self.session,self.now)
            self.assertFalse(state["current"])
            fresh=await game_pow.issue_work(self.db,"tg_test",self.session,self.now)
        self.assertEqual(fresh["job"]["prev_hash"],tip["hash"])
        self.assertNotEqual(fresh["job"]["job_id"],self.job["job_id"])

    async def test_status_is_bound_to_owned_active_session(self):
        await self.issue()
        with patch.object(game_pow,"node_request",side_effect=AssertionError("foreign session reached node")):
            state=await game_pow.work_status(self.db,"tg_other",self.session,self.now)
        self.assertFalse(state["current"])

    async def test_unknown_submit_is_reconciled_from_chain_without_reposting(self):
        await self.issue()
        attempt={"hash":"12"*32,"height":6,"nonce":"1","genesis_hash":self.genesis}
        with closing(self.db()) as conn,conn:
            conn.execute("UPDATE game_pow_work_v1 SET status='UNKNOWN',receipt_json=?",(json.dumps(attempt),))
        paths=[]
        async def node(path,payload=None,params=None):
            paths.append(path)
            if path=="/api/mine/tip":return self.tip(height=6,hash=attempt["hash"])
            self.assertEqual(path,"/api/mine/result")
            return {**attempt,"accepted":True,"miner":"test-wallet"}
        with patch.object(game_pow,"node_request",side_effect=node):
            result=await game_pow.work_status(self.db,"tg_test",self.session,self.now)
        self.assertTrue(result["alreadyAccepted"])
        self.assertNotIn("/api/mine/submit",paths)
        with closing(self.db()) as conn:
            self.assertEqual(conn.execute("SELECT COUNT(*) FROM game_pow_receipts_v1").fetchone()[0],1)

    async def test_next_block_keeps_previous_proof_for_single_game_finish(self):
        await self.issue()
        receipt={"hash":"12"*32,"height":6,"accepted":True}
        with closing(self.db()) as conn,conn:
            conn.execute("UPDATE game_pow_work_v1 SET status='ACCEPTED',receipt_json=?",(json.dumps(receipt),))
        next_job={**self.job,"job_id":"34"*32,"height":7,"prev_hash":receipt["hash"]}
        async def node(path,payload=None,params=None):return next_job
        with patch.object(game_pow,"node_request",side_effect=node):
            result=await game_pow.issue_work(self.db,"tg_test",self.session,self.now,next_block=True)
        self.assertEqual(result["job"]["height"],7)
        reserved=mining_sessions.reserve_run(self.db,"tg_test",1,self.session,self.now,require_pow=True)
        self.assertTrue(reserved["ok"],reserved)
        self.assertEqual(reserved["powReceipt"]["hash"],receipt["hash"])
        self.assertFalse(mining_sessions.reserve_run(self.db,"tg_test",1,self.session,self.now,require_pow=True)["ok"])

    async def test_shared_tip_cache_coalesces_players(self):
        calls=0
        async def node(path,payload=None,params=None):
            nonlocal calls
            calls+=1;await asyncio.sleep(.01);return self.tip()
        with patch.object(game_pow,"node_request",side_effect=node):
            tips=await asyncio.gather(*(game_pow.current_tip() for _ in range(75)))
        self.assertEqual(calls,1)
        self.assertTrue(all(t["genesis_hash"]==self.genesis for t in tips))

    async def test_foreign_genesis_work_is_never_issued(self):
        foreign={**self.job,"genesis_hash":"00"*32}
        async def node(path,payload=None,params=None):return foreign
        with patch.object(game_pow,"node_request",side_effect=node):
            result=await game_pow.issue_work(self.db,"tg_test",self.session,self.now)
        self.assertFalse(result["ok"])


if __name__=="__main__":unittest.main(verbosity=2)
