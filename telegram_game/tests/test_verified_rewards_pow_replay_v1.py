"""Authenticated rewards, durable rate limits and real JavaScript/Go block work."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile

if os.environ.get("QC_VERIFIED_REWARDS_TEST_CHILD") != "1":
    with tempfile.TemporaryDirectory(prefix="qc-verified-rewards-") as directory:
        env=os.environ.copy()
        env.update(QC_VERIFIED_REWARDS_TEST_CHILD="1",QC_PLAYER_RUNTIME_TEST_CHILD="1",
                   QC_GAME_DB=str(Path(directory)/"test-only.sqlite"),TELEGRAM_BOT_TOKEN="test-only-rewards-bot",
                   TELEGRAM_GAME_ADMIN_TOKEN="test-only-rewards-admin",QC_API_BASE="http://127.0.0.1:1",QC_API_TOKEN="test-only-no-node")
        raise SystemExit(subprocess.call([sys.executable,str(Path(__file__).resolve()),*sys.argv[1:]],cwd=directory,env=env))

import asyncio
from contextlib import closing
import concurrent.futures
import copy
import hashlib
import json
import time
import queue
import threading
import unittest
from unittest.mock import patch
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives import serialization
sys.path.insert(0,str(Path(__file__).resolve().parent))
import test_player_runtime_boundary_v1 as prior
from telegram_game import dev_api as api, reward_evidence as rewards, durable_limits as limits, game_pow


def start_browser_solver(repo):
    # Native process/module startup must not consume the node's 60-second job.
    script="""import {pathToFileURL} from 'node:url';
import {readFileSync} from 'node:fs';
const {hashMiningNonce,meetsDifficulty}=await import(pathToFileURL(process.argv[1]));
console.log('READY');
const job=JSON.parse(readFileSync(0,'utf8'));
for(let n=9007199254740993n;n<9007199255740993n;n++) {
 const hash=await hashMiningNonce(job,n);
 if(meetsDifficulty(hash,job.difficulty)){console.log(JSON.stringify({nonce:n.toString(),hash:Buffer.from(hash).toString('hex')}));process.exit(0);}
}process.exit(1);"""
    solver=subprocess.Popen(["node","--input-type=module","-e",script,str(repo/"webui/dist/canonical-pow.mjs")],
                            stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
    ready=queue.Queue()
    threading.Thread(target=lambda:ready.put(solver.stdout.readline()),daemon=True).start()
    try:
        if ready.get(timeout=60).strip()!="READY":raise RuntimeError("JavaScript solver did not become ready")
        return solver
    except BaseException:
        solver.kill();solver.communicate(timeout=10)
        raise


class VerifiedRewardPoWTests(unittest.TestCase):
    seed_player=prior.RuntimeBoundaryTests.seed_player
    opened=prior.RuntimeBoundaryTests.opened
    age_run=prior.RuntimeBoundaryTests.age_run

    def setUp(self):
        prior.RuntimeBoundaryTests.setUp(self)
        self.now=int(time.time())
        self.key=Ed25519PrivateKey.generate()
        self.admin={"X-Admin-Token":os.environ["TELEGRAM_GAME_ADMIN_TOKEN"]}
        with closing(api._db()) as conn,conn:
            for table in ("tgwt_verified_rewards_v1","tgwt_campaigns_v1","tgwt_state","tgwt_events"):
                conn.execute("DELETE FROM "+table)
            conn.execute("UPDATE tgwt_pool SET distributed_u=0,total_u=? WHERE id=1",(rewards.CAP_U,))

    def campaign(self,name="test-task",action="SOCIAL_FOLLOW",reward=1_000_000,budget=10_000_000):
        policy={"campaignId":name,"provider":"test-only-provider","publicKey":self.key.public_key().public_bytes(serialization.Encoding.Raw,serialization.PublicFormat.Raw).hex(),
                "action":action,"target":"test-only-target","reward_u":reward,"budget_u":budget,"startsAt":self.now-60,"endsAt":self.now+86400}
        result=rewards.create_campaign(api._db,policy)
        self.assertTrue(result["ok"],result)
        return policy

    def proof(self,policy,player="tg_9001",subject="test-subject",event="test-event"):
        statement={"version":rewards.VERSION,"audience":"quantumcoin.telegram-game.tgwt","campaignId":policy["campaignId"],"campaignHash":rewards.policy_hash(policy),
                   "playerId":player,"provider":policy["provider"],"subjectId":subject,"action":policy["action"],"target":policy["target"],"eventId":event,
                   "issuedAt":self.now,"expiresAt":self.now+300}
        return {"statement":statement,"signature":self.key.sign(rewards.signing_bytes(statement)).hex()}

    def claim(self,policy,proof,headers=None,**extra):
        response=self.client.post("/api/tgwt/rewards/claim",headers=headers or self.headers,
            json={"playerId":proof["statement"]["playerId"],"campaignId":policy["campaignId"],"proof":proof,**extra})
        return response

    def totals(self):
        with closing(api._db()) as conn:
            pool=conn.execute("SELECT distributed_u FROM tgwt_pool WHERE id=1").fetchone()[0]
            count=conn.execute("SELECT COUNT(*) FROM tgwt_verified_rewards_v1").fetchone()[0]
        return pool,count

    def test_unproven_legacy_routes_cannot_spend_pool(self):
        for path,body in (("/api/social/claim",{"playerId":"tg_9001","platform":"x"}),
                          ("/api/watch/claim",{"playerId":"tg_9001","videoId":"fake"}),
                          ("/api/store/buy",{"playerId":"tg_9001","shipId":"ship_05"})):
            for index in range(3):
                response=self.client.post(path,headers=self.headers,json={**body,"nonce":str(index),"amount":999999,"providerVerified":True})
                self.assertFalse(response.json()["ok"],response.text)
        for path in ("/api/v1/tgwt/earn","/api/v1/tgwt/verify"):
            response=self.client.post(path,headers=self.admin,json={"userId":"tg_9001","amount":"999999","nonce":"new"})
            self.assertEqual(response.status_code,410)
        self.assertEqual(self.totals(),(0,0))

    def test_valid_provider_receipt_credits_once_without_gas_or_token_transfer(self):
        policy=self.campaign()
        proof=self.proof(policy)
        first=self.claim(policy,proof,amount=999999,nonce="first")
        self.assertTrue(first.json()["ok"],first.text)
        self.assertEqual(first.json()["credited_u"],policy["reward_u"])
        self.assertFalse(first.json()["onChainPaid"])
        self.assertEqual(first.json()["gasPayer"],"user")
        self.assertEqual(first.json()["balances"]["verified_u"],policy["reward_u"])
        replay=self.claim(policy,proof,amount=9999999,nonce="changed")
        self.assertTrue(replay.json()["alreadyCredited"])
        self.assertEqual(replay.json()["credited_u"],0)
        self.assertEqual(self.totals(),(policy["reward_u"],1))
        self.assertEqual(api._fee_u(1_000_000),0)

    def test_each_signature_scope_and_time_field_is_enforced(self):
        policy=self.campaign()
        valid=self.proof(policy)
        for field,value in (("playerId","tg_9002"),("target","another"),("eventId","new"),("action","STORE_PURCHASE"),
                             ("campaignHash","00"*32),("audience","another-app"),("issuedAt",self.now+120),("expiresAt",self.now-1)):
            bad=copy.deepcopy(valid);bad["statement"][field]=value
            self.assertFalse(rewards.claim(api._db,"tg_9001",policy["campaignId"],bad,self.now)["ok"],field)
        forged=copy.deepcopy(valid);forged["signature"]="00"*64
        self.assertFalse(rewards.claim(api._db,"tg_9001",policy["campaignId"],forged,self.now)["ok"])
        wrong_key=copy.deepcopy(valid);wrong_key["signature"]=Ed25519PrivateKey.generate().sign(rewards.signing_bytes(wrong_key["statement"])).hex()
        self.assertFalse(rewards.claim(api._db,"tg_9001",policy["campaignId"],wrong_key,self.now)["ok"])
        self.assertEqual(self.totals(),(0,0))

    def test_subject_event_and_player_uniqueness_ignore_new_nonces(self):
        policy=self.campaign()
        self.assertTrue(rewards.claim(api._db,"tg_9001",policy["campaignId"],self.proof(policy),self.now)["ok"])
        for uid,subject,event in (("tg_9002","test-subject","another-event"),("tg_9003","another-subject","test-event")):
            result=rewards.claim(api._db,uid,policy["campaignId"],self.proof(policy,uid,subject,event),self.now)
            self.assertEqual(result["error"],"provider_account_or_event_already_rewarded")
        result=rewards.claim(api._db,"tg_9001",policy["campaignId"],self.proof(policy,event="new-event"),self.now)
        self.assertTrue(result["alreadyCredited"])
        self.assertEqual(self.totals(),(1_000_000,1))

    def test_campaign_budget_and_pool_are_atomic_across_connections(self):
        policy=self.campaign(budget=3_000_000)
        def reserve(index):
            uid="tg_"+str(10000+index)
            return rewards.claim(api._db,uid,policy["campaignId"],self.proof(policy,uid,"subject-"+str(index),"event-"+str(index)),self.now)
        with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
            results=list(pool.map(reserve,range(8)))
        self.assertEqual(sum(bool(result["ok"]) for result in results),3)
        self.assertEqual(self.totals(),(3_000_000,3))
        second=self.campaign("another-task")
        with closing(api._db()) as conn,conn:
            conn.execute("UPDATE tgwt_pool SET distributed_u=total_u-1")
        result=rewards.claim(api._db,"tg_9001",second["campaignId"],self.proof(second,event="new"),self.now)
        self.assertEqual(result["error"],"pool_exhausted_or_invalid")
        with closing(api._db()) as conn:
            self.assertEqual(conn.execute("SELECT spent_u FROM tgwt_campaigns_v1 WHERE campaign_id=?",(second["campaignId"],)).fetchone()[0],0)

    def test_campaign_budget_is_atomic_between_processes(self):
        policy=self.campaign(budget=1_000_000)
        worker="""import json,sqlite3,sys
from telegram_game.reward_evidence import claim
def db(): return sqlite3.connect(sys.argv[1],timeout=20)
request=json.loads(sys.stdin.read())
print(json.dumps(claim(db,request['player'],request['campaign'],request['proof'],request['now'])))
"""
        env=os.environ.copy();env["PYTHONPATH"]=str(Path(__file__).resolve().parents[1])
        def reserve(index):
            uid="tg_"+str(20000+index)
            request={"player":uid,"campaign":policy["campaignId"],"proof":self.proof(policy,uid,str(index),str(index)),"now":self.now}
            done=subprocess.run([sys.executable,"-c",worker,api.APP_DB],input=json.dumps(request),text=True,capture_output=True,env=env,timeout=30)
            self.assertEqual(done.returncode,0,done.stderr)
            return json.loads(done.stdout)
        with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
            results=list(pool.map(reserve,range(4)))
        self.assertEqual(sum(bool(result["ok"]) for result in results),1)
        self.assertEqual(self.totals(),(1_000_000,1))

    def test_campaign_admin_policy_is_immutable_and_can_be_disabled(self):
        policy=self.campaign()
        changed={**policy,"reward_u":2_000_000}
        self.assertFalse(rewards.create_campaign(api._db,changed)["ok"])
        self.assertEqual(self.client.post("/api/admin/tgwt/campaigns",headers=self.headers,json={"policy":policy}).status_code,401)
        response=self.client.post("/api/admin/tgwt/campaign-state",headers=self.admin,json={"campaignId":policy["campaignId"],"enabled":False})
        self.assertTrue(response.json()["ok"])
        self.assertEqual(self.claim(policy,self.proof(policy)).json()["error"],"campaign_unavailable")

    def test_claim_route_action_cannot_be_changed(self):
        policy=self.campaign()
        response=self.client.post("/api/store/buy",headers=self.headers,json={"playerId":"tg_9001","shipId":"fake","nonce":"changed","campaignId":policy["campaignId"],"proof":self.proof(policy)})
        self.assertEqual(response.json()["error"],"campaign_action_or_period_mismatch")
        self.assertEqual(self.totals(),(0,0))

    def test_validly_signed_expired_and_future_receipts_are_rejected(self):
        policy=self.campaign()
        for issued,expires in ((self.now-600,self.now-1),(self.now+120,self.now+300),(self.now-700,self.now+100)):
            proof=self.proof(policy)
            proof["statement"].update(issuedAt=issued,expiresAt=expires)
            proof["signature"]=self.key.sign(rewards.signing_bytes(proof["statement"])).hex()
            self.assertEqual(rewards.claim(api._db,"tg_9001",policy["campaignId"],proof,self.now)["error"],"provider_evidence_invalid")

    def test_failed_ledger_write_rolls_back_pool_and_entitlement(self):
        import sqlite3
        policy=self.campaign()
        with closing(api._db()) as conn,conn:
            conn.execute("CREATE TRIGGER test_only_abort_credit BEFORE INSERT ON tgwt_events BEGIN SELECT RAISE(ABORT,'test-only-ledger-failure'); END")
        try:
            with self.assertRaises(sqlite3.IntegrityError):
                rewards.claim(api._db,"tg_9001",policy["campaignId"],self.proof(policy),self.now)
        finally:
            with closing(api._db()) as conn,conn:
                conn.execute("DROP TRIGGER test_only_abort_credit")
        self.assertEqual(self.totals(),(0,0))

    def test_xff_and_request_ids_do_not_change_authenticated_actor_quota(self):
        session=self.opened()
        body={"playerId":"tg_9001","sessionId":session,"siteId":1}
        with patch.object(api,"_qc_action_burst_now_v1",return_value=self.now):
            for index in range(4):
                response=self.client.post("/api/mine/finish",headers={**self.headers,"X-Forwarded-For":"198.51.100."+str(index)},json={**body,"requestId":str(index)})
                self.assertEqual(response.status_code,200,response.text)
            response=self.client.post("/api/mine/finish",headers={**self.headers,"X-Forwarded-For":"203.0.113.99"},json={**body,"requestId":"completely-new"})
            self.assertEqual(response.status_code,429,response.text)
        self.assertEqual(api._QC_ACTION_BURST_EVENTS_V1,{})

    def test_durable_limits_are_atomic_pruned_and_capacity_bounded(self):
        dimensions=[("test-actor","tg_9001",3,60)]
        with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
            results=list(pool.map(lambda _:limits.admit(api._db,dimensions,self.now),range(8)))
        self.assertEqual(sum(results),3)
        self.assertFalse(limits.admit(api._db,dimensions,self.now))
        self.assertTrue(limits.admit(api._db,dimensions,self.now+60))
        with patch.object(limits,"MAX_ROWS",2):
            self.assertTrue(limits.admit(api._db,[("another","actor",1,60)],self.now+60))
            self.assertFalse(limits.admit(api._db,[("new","identity",1,60)],self.now+60))
        with closing(api._db()) as conn:
            self.assertEqual(conn.execute("SELECT COUNT(*) FROM request_limits_v1").fetchone()[0],2)

    def test_finish_without_actual_client_pow_never_requests_a_block(self):
        session=self.opened();self.age_run(session)
        with patch.object(api,"qc_fetch_mine_block",side_effect=AssertionError("server mined")):
            result=self.client.post("/api/mine/finish",headers=self.headers,json={"playerId":"tg_9001","sessionId":session,"siteId":1,"durationSec":1e30,"qcEarned":1e30})
        self.assertEqual(result.json()["error"],"accepted_client_pow_required")
        with closing(api._db()) as conn:
            self.assertEqual(conn.execute("SELECT status FROM game_mining_runs_v1 WHERE session_id=?",(session,)).fetchone()[0],"STARTED")

    def test_invalid_nonce_and_wrong_owner_do_not_reach_node(self):
        session=self.opened()
        header=bytes(24)
        reference=json.loads((Path(__file__).resolve().parents[2]/"testnet/genesis.json").read_text(encoding="utf-8-sig"))
        job={"network":"testnet","genesis_hash":reference["genesisHash"],"prev_hash":"01"*32,"protocol":"QC_WEB_BLOCK_POW_V1","miner":"QC_test_tg_9001","nonce_encoding":"uint64-big-endian","header":header.hex(),"nonce_offset":16,
             "max_nonce":"9223372036854775807","difficulty":255,"height":1,"expires":self.now+60,"job_id":"ab"*32}
        async def node(path,payload=None,params=None): return job
        with patch.dict(os.environ,{"QC_NETWORK":"testnet-local"}),patch.object(game_pow,"node_request",side_effect=node):
            result=asyncio.run(game_pow.issue_work(api._db,"tg_9001",session,self.now))
        self.assertTrue(result["ok"])
        with patch.object(game_pow,"node_request",side_effect=AssertionError("invalid work reached node")):
            for actor,nonce in (("tg_9001","bad"),("tg_9001","0"),("tg_9002","0")):
                self.assertFalse(asyncio.run(game_pow.submit_work(api._db,actor,session,job["job_id"],nonce,self.now))["ok"])

    def test_real_javascript_client_work_is_committed_once_by_real_go_node(self):
        repo=Path(__file__).resolve().parents[2]
        with tempfile.TemporaryDirectory(prefix="qc-client-pow-node-") as directory:
            contract=Path(directory)/"test-only-node.json";log_path=Path(directory)/"node.log"
            env=os.environ.copy();env["QC_TEST_CLIENT_POW_CONTRACT"]=str(contract)
            with log_path.open("w",encoding="utf-8") as log:
                # A cold native build is not server startup. Compile separately
                # so its duration cannot consume the fixture readiness deadline.
                binary=Path(directory)/("node-fixture.test.exe" if os.name=="nt" else "node-fixture.test")
                built=subprocess.run(["go","test","-c","-o",str(binary),"."],cwd=repo,env=env,
                                     stdout=log,stderr=subprocess.STDOUT,timeout=240)
                self.assertEqual(built.returncode,0,"Go fixture build failed: "+log_path.read_text())
                process=subprocess.Popen([str(binary),"-test.run=^TestTelegramClientPoWContractV1$","-test.timeout=180s","-test.v"],cwd=repo,env=env,stdout=log,stderr=subprocess.STDOUT)
                solver=None
                try:
                    deadline=time.monotonic()+100
                    while not contract.exists():
                        if process.poll() is not None or time.monotonic()>deadline:
                            self.fail("Go fixture did not start: "+log_path.read_text())
                        time.sleep(.05)
                    node=json.loads(contract.read_text())
                    session=self.opened()
                    with closing(api._db()) as conn,conn:
                        conn.execute("UPDATE players SET qc_address=? WHERE player_id='tg_9001'",(node["address"],))
                    with patch.dict(os.environ,{"QC_API_BASE":node["baseURL"],"QC_API_TOKEN":node["apiToken"],"QC_NETWORK":"testnet-local"}):
                        solver=start_browser_solver(repo)
                        response=self.client.post("/api/mine/work",headers=self.headers,json={"playerId":"tg_9001","sessionId":session})
                        self.assertTrue(response.json()["ok"],response.text)
                        job=response.json()["job"]
                        stdout,stderr=solver.communicate(json.dumps(job),timeout=30)
                        self.assertEqual(solver.returncode,0,stderr)
                        solution=json.loads(stdout)
                        body={"playerId":"tg_9001","sessionId":session,"jobId":job["job_id"],"nonce":solution["nonce"]}
                        accepted=self.client.post("/api/mine/work/submit",headers=self.headers,json=body)
                        self.assertTrue(accepted.json()["ok"],accepted.text)
                        self.assertEqual(accepted.json()["receipt"]["hash"],solution["hash"])
                        repeated=self.client.post("/api/mine/work/submit",headers=self.headers,json=body)
                        self.assertTrue(repeated.json()["alreadyAccepted"],repeated.text)
                        recovered=self.client.post("/api/mine/work",headers=self.headers,json={"playerId":"tg_9001","sessionId":session})
                        self.assertTrue(recovered.json()["alreadyAccepted"],recovered.text)
                        self.age_run(session)
                        finish={"playerId":"tg_9001","sessionId":session,"siteId":1,"requestId":"first"}
                        completed=self.client.post("/api/mine/finish",headers=self.headers,json=finish)
                        self.assertTrue(completed.json()["ok"],completed.text)
                        self.assertEqual(completed.json()["qcEarned"],0)
                        self.assertEqual(completed.json()["blockHash"],solution["hash"])
                        replay=self.client.post("/api/mine/finish",headers=self.headers,json={**finish,"requestId":"new"})
                        self.assertFalse(replay.json()["ok"])
                finally:
                    if solver is not None and solver.poll() is None:
                        solver.kill();solver.communicate(timeout=10)
                    Path(str(contract)+".done").touch()
                    try: process.wait(timeout=15)
                    except subprocess.TimeoutExpired: process.kill();process.wait(timeout=10)
                self.assertEqual(process.returncode,0,log_path.read_text())


if __name__=="__main__":unittest.main(verbosity=2)
