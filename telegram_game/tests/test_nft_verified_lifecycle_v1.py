"""Real crypto + SQLite + loopback transport + Go-produced durable mint contract.

Run the node TestNFTVerifiedLifecycleContractV1 first and set
QMA_NFT_LIFECYCLE_CONTRACT_PATH. Missing contract/dependencies fail, never skip.
These test assets/keys/approvals do not constitute production NFT readiness.
"""
import concurrent.futures
import copy
import datetime
import hashlib
import hmac
import http.server
import json
import os
import sqlite3
import subprocess
import sys
import tempfile
import threading
import time
import unittest
import urllib.parse
from contextlib import closing
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec, ed25519
from py_ecc.bls import G2Basic
from telegram_game.utils import nft_wallet_proof_v1 as wallet
from telegram_game.utils import nft_draw_evidence_v1 as draw
from telegram_game.utils import nft_draw_freeze_v1 as freeze
from telegram_game.utils import nft_mint_bridge_v1 as mint
from telegram_game.utils.nft_final_readiness_v1 import VERSION as FINAL_VERSION
from test_nft_draw_freeze_v1 import ensure, serialize
from test_nft_mint_bridge_v1 import address_for_owner


def public(key):
    return key.public_key().public_bytes(serialization.Encoding.Raw, serialization.PublicFormat.Raw).hex()


def sign(key, domain, statement):
    return key.sign(domain.encode("ascii") + b"\x00" + wallet.canonical(statement).encode("ascii")).hex()


class VerifiedLifecycleTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        path = os.environ["QMA_NFT_LIFECYCLE_CONTRACT_PATH"]
        cls.contract = json.loads(Path(path).read_text(encoding="utf-8-sig"))

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="qma-verified-lifecycle-")
        self.root = Path(self.temp.name)
        self.db = self.root / "game.sqlite"
        self.program = copy.deepcopy(self.contract["program"])
        self.payload = copy.deepcopy(self.contract["request"])
        self.period = self.payload["record"]["awardPeriod"]
        self.owner = self.payload["record"]["recipientPubKeyHash"]
        start = datetime.datetime.strptime(self.period, "%Y-%m").replace(tzinfo=datetime.timezone.utc)
        end = start.replace(year=start.year + 1, month=1) if start.month == 12 else start.replace(month=start.month + 1)
        self.frozen_at = int(end.timestamp()) + 1
        self.key = ec.derive_private_key(1, ec.SECP256K1())  # TEST ONLY
        self.public = self.key.public_key().public_bytes(serialization.Encoding.X962, serialization.PublicFormat.UncompressedPoint).hex()
        self.assertEqual(hashlib.new("ripemd160", hashlib.sha256(bytes.fromhex(self.public)[1:]).digest()).hexdigest(), self.owner)
        self.witness, self.ai, self.admin = (ed25519.Ed25519PrivateKey.generate() for _ in range(3))
        self.policy = {"version": "QC_NFT_EVIDENCE_POLICY_V1", **wallet.program_scope(self.program),
            "beacon": {"chainHash": "ab" * 32, "publicKey": G2Basic.SkToPk(123).hex(),
                "scheme": "pedersen-bls-unchained", "genesisTime": self.frozen_at - 10000,
                "periodSeconds": 10, "commitLeadSeconds": 60},
            "publicationWitness": {"publicKey": public(self.witness), "urlPrefix": "https://test-only.invalid/entries/"},
            "finalApprovers": {"ai": public(self.ai), "admin": public(self.admin)}}
        with closing(self.connect()) as conn, conn:
            ensure(conn)
            conn.execute("INSERT INTO players VALUES(?,?)", ("tg_101", address_for_owner(self.owner)))
            conn.execute("INSERT INTO monthly_nft_candidates VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
                ("candidate-1", "tg_101", self.period, "rare", "server_verified_game_session", "{}", .65,
                 "AI_REVIEW_PASSED", "LOW", "APPROVED_BY_ADMIN", "", "APPROVED_CANDIDATE_NOT_WINNER", "NOT_MINTED", "test-only", 1, 1))
            conn.execute("INSERT INTO monthly_nft_verified_game_sessions VALUES(?,?,?,?,1)",
                         ("session-1", "tg_101", self.period, self.frozen_at - 100))
        self.prove_wallet()
        frozen = freeze.freeze_monthly_pool_v1(self.connect, self.period, self.program["awardPeriods"], self.frozen_at, ensure, serialize)
        self.assertTrue(frozen["ok"], frozen)
        self.frozen = frozen
        self.commitment = draw.commitment_v1(frozen, self.program, self.policy)
        self.publication = {"statement": {"version": draw.WITNESS_VERSION,
            "commitmentHash": draw.digest(draw.COMMITMENT_VERSION, self.commitment),
            "publishedAt": self.frozen_at + 1, "entryURL": "https://test-only.invalid/entries/1"}}
        self.publication["signature"] = sign(self.witness, draw.WITNESS_VERSION, self.publication["statement"])
        sig = G2Basic.Sign(123, hashlib.sha256(self.commitment["round"].to_bytes(8, "big")).digest())
        self.beacon = {"chainHash": "ab" * 32, "round": self.commitment["round"], "signature": sig.hex(), "randomness": hashlib.sha256(sig).hexdigest()}
        self.now = self.commitment["beaconAt"] + 10

    def tearDown(self):
        self.temp.cleanup()

    def connect(self):
        return sqlite3.connect(self.db, timeout=20)

    def prove_wallet(self):
        at = self.frozen_at - 200
        issued = wallet.issue_wallet_challenge_v1(self.connect, self.program, "tg_101", self.period, at)
        self.assertTrue(issued["ok"], issued)
        self.challenge = issued["challenge"]
        self.signature = self.key.sign(bytes.fromhex(issued["signingBytesHex"]), ec.ECDSA(hashes.SHA256())).hex()
        result = wallet.consume_wallet_challenge_v1(self.connect, self.program, "tg_101", self.challenge["nonce"], self.public, self.signature, at + 1)
        self.assertTrue(result["ok"], result)

    def register(self, **kwargs):
        return draw.register_draw_commitment_v1(kwargs.get("connect", self.connect), self.program, self.policy, self.period,
            kwargs.get("publication", self.publication), kwargs.get("now", self.frozen_at + 2))

    def select(self, **kwargs):
        return draw.draw_monthly_winner_v1(kwargs.get("connect", self.connect), self.program, self.policy, self.period,
            kwargs.get("beacon", self.beacon), self.now, serialize)

    def test_wallet_replay_wrong_user_and_expiration(self):
        result = wallet.consume_wallet_challenge_v1(self.connect, self.program, "tg_101", self.challenge["nonce"], self.public, self.signature, self.frozen_at)
        self.assertEqual(result["error"], "wallet_challenge_replayed")
        issued = wallet.issue_wallet_challenge_v1(self.connect, self.program, "tg_101", self.period, self.frozen_at)
        sig = self.key.sign(bytes.fromhex(issued["signingBytesHex"]), ec.ECDSA(hashes.SHA256())).hex()
        for user, now, expected in (("tg_102",self.frozen_at+1,"wallet_challenge_scope_mismatch"),
                                   ("tg_101",self.frozen_at+600,"wallet_challenge_expired")):
            self.assertEqual(wallet.consume_wallet_challenge_v1(self.connect,self.program,user,issued["challenge"]["nonce"],self.public,sig,now)["error"], expected)

    def test_wallet_proof_rechecks_signature_and_wallet_binding(self):
        with closing(self.connect()) as conn, conn:
            conn.execute("UPDATE monthly_nft_wallet_challenges_v1 SET signature=?", ("00" * 70,))
        self.assertFalse(self.register()["ok"])

    def test_external_publication_required_before_fixed_round(self):
        for when in (self.commitment["beaconAt"], self.commitment["beaconAt"] + 100):
            self.assertEqual(self.register(now=when)["error"], "commitment_registration_too_late")
        forged = copy.deepcopy(self.publication)
        forged["statement"]["commitmentHash"] = "00" * 32
        self.assertFalse(self.register(publication=forged)["ok"])
        forged = copy.deepcopy(self.publication)
        forged["signature"] = "00" * 64
        self.assertEqual(self.register(publication=forged)["error"], "invalid_external_evidence_signature")
        self.assertTrue(self.register()["ok"])
        self.assertTrue(self.register()["alreadyRegistered"])

    def test_wrong_beacon_round_network_signature_and_randomness_fail_closed(self):
        self.assertTrue(self.register()["ok"])
        for field, value in (("round",self.beacon["round"]+1),("chainHash","cd"*32),("signature","00"*96),("randomness","00"*32)):
            bad = {**self.beacon, field: value}
            if field == "signature":
                bad["randomness"] = hashlib.sha256(bytes.fromhex(value)).hexdigest()
            self.assertFalse(self.select(beacon=bad)["ok"], field)
        with closing(self.connect()) as conn:
            self.assertEqual(conn.execute("SELECT count(*) FROM monthly_nft_draw_receipts_v1").fetchone()[0],0)

    def test_real_drand_upstream_vector(self):
        # Public vector from drand/crypto/schemes_test.go TestVerifyBeacon.
        c = {"round":7601003,"beaconChainHash":"ab"*32,
             "beaconPublicKey":"8200fc249deb0148eb918d6e213980c5d01acd7fc251900d9260136da3b54836ce125172399ddc69c4e3e11429b62c11"}
        signature="af7eac5897b72401c0f248a26b612c5ef68e0ff830b4d78927988c89b5db3e997bfcdb7c24cb19f549830cd02cb854a1143fd53a1d4e0713ded471260869439060d170a77187eb6371742840e43eccfa225657c4cc2d9619f7c3d680470c9743"
        randomness=hashlib.sha256(bytes.fromhex(signature)).hexdigest()
        self.assertEqual(draw.verify_beacon_v1(c,{"round":c["round"],"chainHash":"ab"*32,"signature":signature,"randomness":randomness}),bytes.fromhex(randomness))

    def test_deterministic_unbiased_selection_supports_over_256_candidates(self):
        results = {draw.unbiased_index_v1("ab"*32,hashlib.sha256(str(i).encode()).digest(),513)[0] for i in range(1000)}
        self.assertTrue(any(i>255 for i in results))
        self.assertEqual(draw.unbiased_index_v1("ab"*32,b"x"*32,513),draw.unbiased_index_v1("ab"*32,b"x"*32,513))
        with patch.object(draw.hashlib,"sha256") as hashed:
            hashed.return_value.digest.side_effect = [b"\xff"*32, b"\x00"*32]
            self.assertEqual(draw.unbiased_index_v1("ab"*32,b"x"*32,3),(0,1))

    def test_draw_is_atomic_idempotent_and_receipt_is_immutable(self):
        self.assertTrue(self.register()["ok"])
        original = self.connect
        class Failure:
            def __init__(self): self.conn=original()
            def execute(self,sql,args=()):
                if "SET winner_status=" in sql: raise sqlite3.OperationalError("test-only-failed-write")
                return self.conn.execute(sql,args)
            def __getattr__(self,k): return getattr(self.conn,k)
        self.assertFalse(self.select(connect=Failure)["ok"])
        with closing(self.connect()) as conn:
            self.assertEqual(conn.execute("SELECT count(*) FROM monthly_nft_draw_receipts_v1").fetchone()[0],0)
        first=self.select()
        self.assertTrue(first["ok"],first)
        second=self.select()
        self.assertEqual(first["receiptHash"],second["receiptHash"])
        self.assertTrue(second["alreadySelected"])
        with closing(self.connect()) as conn:
            with self.assertRaises(sqlite3.IntegrityError), conn:
                conn.execute("DELETE FROM monthly_nft_draw_receipts_v1")

    def test_separate_processes_commit_one_identical_receipt(self):
        self.assertTrue(self.register()["ok"])
        code = '''import json,sqlite3,sys
from telegram_game.utils.nft_draw_evidence_v1 import draw_monthly_winner_v1
from telegram_game.utils.nft_draw_freeze_v1 import _COLUMNS
keys=('candidateId','userId','period','nftType','source','activity','score','aiReviewStatus','aiRiskLevel','adminStatus','adminNote','winnerStatus','mintedStatus','policyVersion','createdAt','updatedAt')
d=json.loads(sys.argv[2]); print(json.dumps(draw_monthly_winner_v1(lambda:sqlite3.connect(sys.argv[1],timeout=20),d['program'],d['policy'],d['period'],d['beacon'],d['now'],lambda row:dict(zip(keys,row)))))'''
        env=os.environ.copy()
        env["PYTHONPATH"]=str(Path(__file__).resolve().parents[1])
        data=json.dumps({"program":self.program,"policy":self.policy,"period":self.period,"beacon":self.beacon,"now":self.now})
        workers=[subprocess.Popen([sys.executable,"-c",code,str(self.db),data],env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True) for _ in range(3)]
        try:
            results=[]
            for worker in workers:
                out,err=worker.communicate(timeout=90)
                self.assertEqual(worker.returncode,0,err)
                results.append(json.loads(out))
            self.assertTrue(all(r["ok"] for r in results),results)
            self.assertEqual(len({r["receiptHash"] for r in results}),1)
            self.assertEqual(sum(not r["alreadySelected"] for r in results),1)
        finally:
            for worker in workers:
                if worker.poll() is None: worker.kill()
                worker.communicate(timeout=10)

    def final_catalog(self, draw_hash, candidate_id="candidate-1"):
        items=[]
        for i,period in enumerate(self.program["awardPeriods"]):
            image=self.root/f"images/{i}.png"
            metadata=self.root/f"metadata/{i}.json"
            image.parent.mkdir(exist_ok=True); metadata.parent.mkdir(exist_ok=True)
            image.write_bytes(b"TEST ONLY ARTWORK "+str(i).encode())
            metadata.write_bytes((self.contract["metadata"] if period==self.period else json.dumps({"TEST ONLY":i})).encode("utf-8"))
            items.append({"id":f"test-stock-{i+1}","period":period,"name":f"test-name-{i}","concept":f"test-concept-{i}","rarity":"rare","finalReady":True,
                "imagePath":str(image.relative_to(self.root)),"metadataPath":str(metadata.relative_to(self.root)),
                "finalImageSha256":hashlib.sha256(image.read_bytes()).hexdigest(),"finalMetadataSha256":hashlib.sha256(metadata.read_bytes()).hexdigest(),
                "artworkStatus":"FINAL_READY","metadataStatus":"FINAL_READY","uniquenessCheck":"PASSED","aiReviewStatus":"APPROVED","adminApprovalStatus":"APPROVED"})
        path=self.root/"nft/catalog/monthly_rare_nft_stock_2026.json"
        path.parent.mkdir(parents=True)
        path.write_text(json.dumps({"totalPlanned":12,"awardsPerMonth":1,"items":items}),encoding="utf-8")
        statement={"version":FINAL_VERSION,**wallet.program_scope(self.program),"candidateId":candidate_id,"catalogId":"test-stock-1",
            "period":self.period,"walletOwner":self.owner,"metadataHash":self.payload["record"]["metadataHash"],
            "imageHash":items[0]["finalImageSha256"],"catalogHash":hashlib.sha256(path.read_bytes()).hexdigest(),
            "drawReceiptHash":draw_hash,"approvedAt":self.now}
        evidence={"statement":statement,"approvals":[{"role":role,"signature":sign(key,FINAL_VERSION+":"+role,statement)} for role,key in (("ai",self.ai),("admin",self.admin))]}
        approval=self.root/"nft/catalog/approvals/test-stock-1.json"
        approval.parent.mkdir()
        approval.write_text(json.dumps(evidence),encoding="utf-8")
        policy_path=self.root/"policy.json"
        policy_path.write_text(json.dumps(self.policy),encoding="utf-8")
        return policy_path,approval

    def test_entire_evidence_chain_binds_transport_and_rejects_tampered_approval(self):
        self.assertTrue(self.register()["ok"])
        selected=self.select()
        self.assertTrue(selected["ok"],selected)
        policy_path,approval=self.final_catalog(selected["receiptHash"])
        owner=self
        counts={"mint":0}
        class Handler(http.server.BaseHTTPRequestHandler):
            def do_GET(self): self.respond(owner.program)
            def do_POST(self):
                request=json.loads(self.rfile.read(int(self.headers["Content-Length"])))
                if request!=owner.payload: return self.respond({},422)
                if self.path.endswith("/mint"): counts["mint"]+=1
                if not counts["mint"]: return self.respond({},404)
                self.respond(owner.contract["receipt"])
            def respond(self,value,status=200):
                self.send_response(status);self.send_header("Content-Type","application/json");self.end_headers();self.wfile.write(json.dumps(value).encode())
            def log_message(self,*args): pass
        server=http.server.ThreadingHTTPServer(("127.0.0.1",0),Handler)
        thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start()
        try:
            env={"QC_NFT_EVIDENCE_POLICY_PATH":str(policy_path),"QC_API_BASE":f"http://127.0.0.1:{server.server_port}","QC_API_TOKEN":"test-only",
                "QC_ENABLE_CHAIN_NFT_MINT":"1","QC_ENABLE_TG_NFT_MINT_BRIDGE":"1","TELEGRAM_GAME_CHAIN_MODE":"mainnet","QC_TESTNET":"","QC_PUBLIC_TESTNET":"","QC_NET_MAGIC":""}
            with patch.dict(os.environ,env), patch.object(mint.time,"time",return_value=self.now+1):
                saved=approval.read_text()
                bad=json.loads(saved);bad["statement"]["drawReceiptHash"]="00"*32
                approval.write_text(json.dumps(bad),encoding="utf-8")
                self.assertFalse(mint.controlled_monthly_nft_mint_v1(self.connect,self.root,"candidate-1","test-stock-1",self.payload)["ok"])
                self.assertEqual(counts["mint"],0)
                approval.write_text(saved,encoding="utf-8")
                result=mint.controlled_monthly_nft_mint_v1(self.connect,self.root,"candidate-1","test-stock-1",self.payload)
                self.assertTrue(result["ok"],result)
                self.assertTrue(mint.controlled_monthly_nft_mint_v1(self.connect,self.root,"candidate-1","test-stock-1",self.payload)["ok"])
                self.assertEqual(counts["mint"],1)
            with closing(self.connect()) as conn:
                self.assertEqual(conn.execute("SELECT minted_status FROM monthly_nft_candidates").fetchone()[0],"MINTED_CONFIRMED")
        finally:
            server.shutdown();server.server_close();thread.join(timeout=3)

    def test_actual_go_node_mints_after_verified_draw_and_final_approval(self):
        repo=Path(__file__).resolve().parents[2]
        live_path=self.root/"live-node.json"
        log_path=self.root/"live-node.log"
        env=os.environ.copy()
        env.update(QMA_NFT_LIFECYCLE_SERVER="1",QMA_NFT_LIFECYCLE_CONTRACT_PATH=str(live_path))
        with log_path.open("w",encoding="utf-8") as log:
            process=subprocess.Popen(["go","test","./node","-run","^TestNFTVerifiedLifecycleContractV1$","-count=1","-timeout=180s","-v"],cwd=repo,env=env,stdout=log,stderr=subprocess.STDOUT)
            try:
                deadline=time.monotonic()+90
                while not live_path.exists():
                    if process.poll() is not None or time.monotonic()>deadline:
                        self.fail("live Go node did not start: "+log_path.read_text(encoding="utf-8"))
                    time.sleep(.05)
                live=json.loads(live_path.read_text(encoding="utf-8"))
                self.assertEqual(live["program"],self.program)
                self.payload=live["request"]
                from fastapi.testclient import TestClient
                with tempfile.TemporaryDirectory(prefix="qma-full-startup-") as startup:
                    original=sqlite3.connect
                    def isolated(database,*args,**kwargs):
                        if str(database)=="dev_api_state.db": database=str(Path(startup)/"startup.sqlite")
                        return original(database,*args,**kwargs)
                    with patch.object(sqlite3,"connect",side_effect=isolated):
                        from telegram_game import dev_api as api
                policy_path=self.root/"policy.json"
                policy_path.write_text(json.dumps(self.policy),encoding="utf-8")
                self.db=self.root/"actual-game.sqlite"
                clock=[self.frozen_at-500]
                token="test-only-full-telegram"
                settings={"QC_NFT_EVIDENCE_POLICY_PATH":str(policy_path),"QC_API_BASE":live["baseURL"],"QC_API_TOKEN":"test-only-lifecycle-node",
                    "QC_ENABLE_CHAIN_NFT_MINT":"1","QC_ENABLE_TG_NFT_MINT_BRIDGE":"1","TELEGRAM_GAME_CHAIN_MODE":"mainnet","QC_TESTNET":"","QC_PUBLIC_TESTNET":"","QC_NET_MAGIC":"",
                    "TELEGRAM_BOT_TOKEN":token,"TELEGRAM_GAME_ADMIN_TOKEN":"test-only-full-admin"}
                with patch.dict(os.environ,settings),patch.object(api,"APP_DB",str(self.db)),patch.object(api,"_now",side_effect=lambda:clock[0]),patch.object(api,"_NFT_REPOSITORY_ROOT_V1",self.root),patch.object(mint.time,"time",return_value=self.now+1),TestClient(api.app) as client:
                    api._init()
                    with closing(api._db()) as conn,conn:
                        api._ensure_qc_address_column(conn)
                        api._touch_player(conn,"tg_101",None)
                        conn.execute("UPDATE players SET qc_address=? WHERE player_id='tg_101'",(address_for_owner(self.owner),))
                    api._QC_ACTION_BURST_EVENTS_V1.clear();api._QC_ACTION_BURST_BLOCKS_V1.clear()
                    values={"auth_date":str(clock[0]),"user":json.dumps({"id":101},separators=(",",":"))}
                    secret=hmac.new(b"WebAppData",token.encode(),hashlib.sha256).digest()
                    values["hash"]=hmac.new(secret,"\n".join(f"{k}={values[k]}" for k in sorted(values)).encode(),hashlib.sha256).hexdigest()
                    init_data=urllib.parse.urlencode(values)
                    def post(path,body,admin=False):
                        response=client.post(path,json=body,headers={"X-Admin-Token":"test-only-full-admin"} if admin else {})
                        self.assertEqual(response.status_code,200,response.text)
                        result=response.json();self.assertTrue(result["ok"],result)
                        return result
                    started=post("/api/v1/nft/monthly/session/start",{"playerId":"tg_101","initData":init_data})
                    clock[0]+=31
                    post("/api/mine/start",{"playerId":"tg_101","siteId":1,"initData":init_data,"nftSessionId":started["sessionId"]})
                    clock[0]+=30
                    post("/api/v1/nft/monthly/session/complete",{"playerId":"tg_101","initData":init_data,"sessionId":started["sessionId"]})
                    candidate=post("/api/v1/nft/monthly/candidate",{"userId":"tg_101","period":self.period,"source":"untrusted-client","activity":{"miningSessions":999}})["candidate"]["candidateId"]
                    post("/api/admin/nft/monthly/ai-review",{"candidateId":candidate},True)
                    post("/api/admin/nft/monthly/review",{"candidateId":candidate,"action":"APPROVE"},True)
                    issued=post("/api/nft/wallet/challenge",{"playerId":"tg_101","initData":init_data,"period":self.period})
                    signature=self.key.sign(bytes.fromhex(issued["signingBytesHex"]),ec.ECDSA(hashes.SHA256())).hex()
                    post("/api/nft/wallet/prove",{"playerId":"tg_101","initData":init_data,"nonce":issued["challenge"]["nonce"],"publicKey":self.public,"signature":signature})
                    clock[0]=self.frozen_at
                    frozen=post("/api/admin/nft/monthly/freeze-pool",{"period":self.period},True)
                    preview=client.get("/api/admin/nft/monthly/draw-commitment",params={"period":self.period},headers={"X-Admin-Token":"test-only-full-admin"}).json()
                    self.assertTrue(preview["ok"],preview)
                    publication=copy.deepcopy(self.publication)
                    publication["statement"]["commitmentHash"]=draw.digest(draw.COMMITMENT_VERSION,preview["commitment"])
                    publication["signature"]=sign(self.witness,draw.WITNESS_VERSION,publication["statement"])
                    clock[0]+=2
                    post("/api/admin/nft/monthly/register-draw-commitment",{"period":self.period,"publication":publication},True)
                    clock[0]=self.now
                    selected=post("/api/admin/nft/monthly/draw-winner",{"period":self.period,"beacon":self.beacon},True)
                    self.final_catalog(selected["receiptHash"],candidate)
                    result=post("/api/admin/nft/monthly/controlled-mint",{"candidateId":candidate,"catalogItemId":"test-stock-1","signedAward":self.payload},True)
                    self.assertTrue(result["ok"],result)
                    self.assertTrue(result["receipt"]["persisted"])
                    self.assertFalse(result["receipt"]["alreadyMinted"])
                    replay=post("/api/admin/nft/monthly/controlled-mint",{"candidateId":candidate,"catalogItemId":"test-stock-1","signedAward":self.payload},True)
                    self.assertTrue(replay["ok"],replay)
                    self.assertTrue(replay["receipt"]["alreadyMinted"])
                    self.assertEqual(result["receipt"]["tokenID"],replay["receipt"]["tokenID"])
                    output=os.getenv("QMA_NFT_EVIDENCE_OUTPUT_DIR")
                    if output:
                        directory=Path(output);directory.mkdir(parents=True,exist_ok=True)
                        for name,value in (("frozen-pool",frozen),("draw",selected),("game-mint-receipt",result)):
                            (directory/(name+".test-only.json")).write_text(json.dumps(value,indent=2),encoding="utf-8")
            finally:
                Path(str(live_path)+".done").touch()
                try:
                    process.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    process.kill();process.wait(timeout=10)
            self.assertEqual(process.returncode,0,log_path.read_text(encoding="utf-8"))
            receipt=json.loads(Path(str(live_path)+".receipt.json").read_text(encoding="utf-8"))
            self.assertEqual(receipt["tokenID"],result["receipt"]["tokenID"])

    def test_real_api_auth_wallet_challenge_publication_and_verified_draw(self):
        # dev_api creates a relative startup DB at import: redirect only that
        # connection before import, so test collection never touches user data.
        from fastapi.testclient import TestClient
        with tempfile.TemporaryDirectory(prefix="qma-api-startup-") as startup:
            original=sqlite3.connect
            def isolated(database,*args,**kwargs):
                if str(database)=="dev_api_state.db": database=str(Path(startup)/"startup.sqlite")
                return original(database,*args,**kwargs)
            with patch.object(sqlite3,"connect",side_effect=isolated):
                from telegram_game import dev_api as api
        from telegram_game.utils import nft_program_bridge_v1 as program_bridge
        clock=[self.frozen_at+2]
        token="test-only-api-telegram"
        values={"auth_date":str(clock[0]),"user":json.dumps({"id":101},separators=(",",":"))}
        secret=hmac.new(b"WebAppData",token.encode(),hashlib.sha256).digest()
        values["hash"]=hmac.new(secret,"\n".join(f"{k}={values[k]}" for k in sorted(values)).encode(),hashlib.sha256).hexdigest()
        init_data=urllib.parse.urlencode(values)
        policy_path=self.root/"api-policy.json"
        policy_path.write_text(json.dumps(self.policy),encoding="utf-8")
        settings={"TELEGRAM_BOT_TOKEN":token,"TELEGRAM_GAME_ADMIN_TOKEN":"test-only-api-admin","QC_NFT_EVIDENCE_POLICY_PATH":str(policy_path)}
        with patch.dict(os.environ,settings),patch.object(api,"APP_DB",str(self.db)),patch.object(api,"_now",side_effect=lambda:clock[0]),patch.object(program_bridge,"fetch_verified_nft_game_program_v1",return_value=self.program),TestClient(api.app) as client:
            api._QC_ACTION_BURST_EVENTS_V1.clear();api._QC_ACTION_BURST_BLOCKS_V1.clear()
            body={"playerId":"tg_101","initData":"","period":self.period}
            self.assertEqual(client.post("/api/nft/wallet/challenge",json=body).json()["error"],"telegram_identity_required")
            body["initData"]=init_data
            issued=client.post("/api/nft/wallet/challenge",json=body).json()
            self.assertTrue(issued["ok"],issued)
            signature=self.key.sign(bytes.fromhex(issued["signingBytesHex"]),ec.ECDSA(hashes.SHA256())).hex()
            proof={"playerId":"tg_101","initData":init_data,"nonce":issued["challenge"]["nonce"],"publicKey":self.public,"signature":signature}
            self.assertTrue(client.post("/api/nft/wallet/prove",json=proof).json()["ok"])
            self.assertEqual(client.post("/api/nft/wallet/prove",json=proof).json()["error"],"wallet_challenge_replayed")
            headers={"X-Admin-Token":"test-only-api-admin"}
            for path,payload in (("register-draw-commitment",{"period":self.period,"publication":self.publication}),
                                 ("draw-winner",{"period":self.period,"beacon":self.beacon})):
                uri="/api/admin/nft/monthly/"+path
                self.assertEqual(client.post(uri,json=payload).status_code,401)
                self.assertEqual(client.post(uri+"?token=test-only-api-admin",json=payload).status_code,401)
            preview=client.get("/api/admin/nft/monthly/draw-commitment",params={"period":self.period},headers=headers).json()
            self.assertEqual(preview["commitment"],self.commitment)
            self.assertFalse(preview["winnerSelected"])
            registered=client.post("/api/admin/nft/monthly/register-draw-commitment",json={"period":self.period,"publication":self.publication},headers=headers).json()
            self.assertTrue(registered["ok"],registered)
            clock[0]=self.now
            result=client.post("/api/admin/nft/monthly/draw-winner",json={"period":self.period,"beacon":self.beacon},headers=headers).json()
            self.assertTrue(result["ok"],result)
            self.assertEqual(result["receipt"]["winner"]["candidateId"],"candidate-1")


if __name__ == "__main__":
    unittest.main(verbosity=2)
