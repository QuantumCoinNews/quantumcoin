"""Relay actual canonical block work. Never ask the node to mine for a player."""
import asyncio
import hashlib
import threading
import time
import weakref
from pathlib import Path
import json
import os
import secrets
from contextlib import closing
import httpx
from .utils.qc_api import get_api_base


def ensure_schema(conn):
    conn.execute("""CREATE TABLE IF NOT EXISTS game_pow_work_v1 (
        session_id TEXT PRIMARY KEY, player_id TEXT NOT NULL, address TEXT NOT NULL,
        request_key TEXT NOT NULL, job_id TEXT UNIQUE, job_json TEXT,
        expires_at INTEGER NOT NULL, status TEXT NOT NULL, receipt_json TEXT)""")
    conn.execute("""CREATE TABLE IF NOT EXISTS game_pow_receipts_v1 (
        block_hash TEXT PRIMARY KEY, session_id TEXT NOT NULL,
        player_id TEXT NOT NULL, receipt_json TEXT NOT NULL)""")
    conn.execute("CREATE INDEX IF NOT EXISTS pow_receipts_session_v1 ON game_pow_receipts_v1(session_id,player_id)")


async def node_request(path, payload=None, params=None):
    # Deliberately no general arbitrary-URL or /api/mine capability.
    if path not in {"/api/mine/job", "/api/mine/submit", "/api/mine/tip", "/api/mine/result"}:
        raise ValueError("unsupported mining operation")
    try:
        async with httpx.AsyncClient(timeout=15, follow_redirects=False, trust_env=False,
                headers={"X-QC-API-KEY":os.getenv("QC_API_TOKEN", "")}) as client:
            url = get_api_base().rstrip("/") + path
            async with client.stream("POST" if payload is not None else "GET",url,json=payload,params=params) as response:
                if response.status_code != 200:
                    return {"ok":False,"error":"node_work_rejected","_http_status":response.status_code}
                raw = bytearray()
                async for chunk in response.aiter_bytes():
                    if len(raw) + len(chunk) > 65536:
                        raise ValueError("node_work_response_too_large")
                    raw.extend(chunk)
                data = json.loads(raw)
            if not isinstance(data, dict):
                raise ValueError()
            return data
    except Exception:
        return {"ok":False,"error":"node_work_request_failed"}


def expected_network():
    from .testnet_connection_v1 import telegram_game_chain_mode_v1
    network = telegram_game_chain_mode_v1()
    if network not in {"testnet", "mainnet"}:
        raise ValueError("fresh_network_required")
    reference = Path(__file__).resolve().parents[2] / network / "genesis.json"
    metadata = json.loads(reference.read_text(encoding="utf-8-sig"))
    genesis = metadata["genesisHash"]
    if metadata.get("network") != network or len(bytes.fromhex(genesis)) != 32:
        raise ValueError("finalized_network_fingerprint_required")
    return network, genesis


def validate_tip(tip):
    try:
        network, genesis = expected_network()
        return (tip.get("version") == "QC_MINING_TIP_V1" and tip.get("network") == network
                and tip.get("genesis_hash") == genesis and type(tip.get("height")) is int
                and tip["height"] >= 0 and len(bytes.fromhex(tip["hash"])) == 32)
    except (OSError, KeyError, ValueError, TypeError):
        return False


_TIP_STATES = weakref.WeakKeyDictionary()
_TIP_BOOK_LOCK = threading.Lock()


async def current_tip():
    # One request per loop/origin/credential per second, shared by all players.
    loop = asyncio.get_running_loop()
    with _TIP_BOOK_LOCK:
        for old_loop in list(_TIP_STATES):
            if old_loop.is_closed():
                _TIP_STATES.pop(old_loop, None)
        state = _TIP_STATES.get(loop)
        if state is None:
            if len(_TIP_STATES) >= 128:
                return {}
            state = {"lock":asyncio.Lock(), "until":0, "key":None, "tip":{}}
            _TIP_STATES[loop] = state
    key = (get_api_base(), hashlib.sha256(os.getenv("QC_API_TOKEN", "").encode()).digest())
    async with state["lock"]:
        if key == state["key"] and time.monotonic() < state["until"]:
            return dict(state["tip"])
        tip = await node_request("/api/mine/tip")
        state.update(key=key, tip=tip if validate_tip(tip) else {}, until=time.monotonic()+1)
        return dict(state["tip"])


def validate_job(job, address, now):
    try:
        network, genesis = expected_network()
        if job.get("network") != network or job.get("genesis_hash") != genesis or len(bytes.fromhex(job["prev_hash"])) != 32:
            return False
        if job["protocol"] != "QC_WEB_BLOCK_POW_V1" or job["miner"] != address or job["nonce_encoding"] != "uint64-big-endian":
            return False
        header = bytes.fromhex(job["header"])
        return (8 <= len(header) <= 16384 and type(job["nonce_offset"]) is int and job["nonce_offset"] == len(header)-8
                and isinstance(job["job_id"],str) and len(job["job_id"]) == 64
                and type(job["difficulty"]) is int and 1 <= job["difficulty"] <= 255
                and type(job["height"]) is int and job["height"] > 0
                and type(job["expires"]) is int and now < job["expires"] <= now+90
                and isinstance(job["max_nonce"],str) and 0 <= int(job["max_nonce"]) <= 9223372036854775807)
    except (OSError, KeyError, ValueError, TypeError):
        return False


def record_receipt(conn, actor, session_id, receipt):
    conn.execute("INSERT OR IGNORE INTO game_pow_receipts_v1(block_hash,session_id,player_id,receipt_json) VALUES(?,?,?,?)",
                 (receipt["hash"], session_id, actor, json.dumps(receipt)))
    owner = conn.execute("SELECT player_id FROM game_pow_receipts_v1 WHERE block_hash=?",(receipt["hash"],)).fetchone()
    if not owner or owner[0] != actor:
        raise ValueError("receipt_owner_conflict")


async def work_status(db, actor, session_id, now):
    with closing(db()) as conn:
        row = conn.execute("""SELECT w.job_id,w.job_json,w.expires_at,w.status,w.receipt_json,w.address
            FROM game_pow_work_v1 w JOIN game_mining_runs_v1 r ON r.session_id=w.session_id
            WHERE w.session_id=? AND w.player_id=? AND r.player_id=? AND r.status='STARTED' AND r.expires_at>=?""",
            (session_id,actor,actor,now)).fetchone()
    if not row:
        return {"ok":True,"current":False}
    if row[3] == "ACCEPTED":
        return {"ok":True,"current":False,"alreadyAccepted":True,"receipt":json.loads(row[4])}
    if row[3] == "REQUESTING":
        return {"ok":False,"error":"work_already_submitted_or_pending"}
    tip = await current_tip()
    if not tip:
        return {"ok":False,"error":"node_network_or_tip_unavailable"}
    job = json.loads(row[1] or "{}")
    if job.get("genesis_hash") != tip.get("genesis_hash"):
        return {"ok":False,"error":"mining_network_changed"}
    if row[3] in {"SUBMITTING", "UNKNOWN"} and row[4]:
        attempt = json.loads(row[4])
        receipt = await node_request("/api/mine/result", params={"hash":attempt.get("hash",""),"height":job.get("height"),"address":row[5]})
        if (receipt.get("accepted") is True and receipt.get("hash") == attempt.get("hash")
                and receipt.get("height") == job.get("height") and receipt.get("miner") == row[5]
                and receipt.get("genesis_hash") == job.get("genesis_hash")):
            confirmed = {**attempt, **receipt, "accepted":True}
            with closing(db()) as conn,conn:
                record_receipt(conn,actor,session_id,confirmed)
                conn.execute("UPDATE game_pow_work_v1 SET status='ACCEPTED',receipt_json=? WHERE session_id=? AND job_id=? AND status IN ('SUBMITTING','UNKNOWN')",
                             (json.dumps(confirmed),session_id,row[0]))
            return {"ok":True,"current":False,"alreadyAccepted":True,"receipt":confirmed}
        if not (receipt.get("accepted") is False or receipt.get("_http_status") == 404):
            return {"ok":False,"error":"work_reconciliation_pending"}
    fresh = (row[2] > now and job.get("height") == tip["height"]+1 and job.get("prev_hash") == tip["hash"])
    if fresh and row[3] in {"SUBMITTING", "UNKNOWN"}:
        return {"ok":False,"error":"work_reconciliation_pending"}
    if not fresh:
        with closing(db()) as conn,conn:
            conn.execute("UPDATE game_pow_work_v1 SET status='STALE' WHERE session_id=? AND job_id=? AND status IN ('ISSUED','UNKNOWN','SUBMITTING')",(session_id,row[0]))
    return {"ok":True,"current":fresh and row[3] == "ISSUED","jobId":row[0],"tip":tip}


async def issue_work(db, actor, session_id, now, next_block=False):
    # Resolve stale/uncertain work outside any DB transaction or global lock.
    with closing(db()) as conn:
        old = conn.execute("SELECT status,expires_at,job_json,receipt_json,job_id FROM game_pow_work_v1 WHERE session_id=? AND player_id=?",(session_id,actor)).fetchone()
    if old:
        if old[0] == "ACCEPTED":
            if not next_block:
                return {"ok":True,"alreadyAccepted":True,"receipt":json.loads(old[3]),"rewardPaidByChain":True}
        elif old[0] in {"ISSUED","SUBMITTING","UNKNOWN"}:
            state = await work_status(db,actor,session_id,now)
            if not state.get("ok") or state.get("alreadyAccepted"):
                return state
            if state.get("current"):
                return {"ok":True,"job":json.loads(old[2])}
        elif old[0] == "REQUESTING" and old[1] > now:
            return {"ok":False,"error":"work_already_submitted_or_pending"}
    key = secrets.token_hex(16)
    with closing(db()) as conn,conn:
        conn.execute("BEGIN IMMEDIATE")
        run = conn.execute("""SELECT p.qc_address FROM game_mining_runs_v1 r JOIN players p ON p.player_id=r.player_id
            WHERE r.session_id=? AND r.player_id=? AND r.status='STARTED' AND r.expires_at>=?""",(session_id,actor,now)).fetchone()
        if not run or not run[0]:
            return {"ok":False,"error":"active_owned_mining_session_required"}
        current = conn.execute("SELECT status,job_id,receipt_json FROM game_pow_work_v1 WHERE session_id=?",(session_id,)).fetchone()
        if current:
            if old is None or current[1] != old[4]:
                return {"ok":False,"error":"work_already_submitted_or_pending"}
            if current[0] == "ACCEPTED" and next_block:
                record_receipt(conn,actor,session_id,json.loads(current[2]))
            elif current[0] not in {"STALE","REJECTED"} and not (current[0]=="REQUESTING" and old[1]<=now):
                return {"ok":False,"error":"work_already_submitted_or_pending"}
            conn.execute("DELETE FROM game_pow_work_v1 WHERE session_id=?",(session_id,))
        address = run[0]
        conn.execute("INSERT INTO game_pow_work_v1(session_id,player_id,address,request_key,expires_at,status) VALUES (?,?,?,?,?,'REQUESTING')",(session_id,actor,address,key,now+30))
    job = await node_request("/api/mine/job",params={"address":address})
    valid = validate_job(job,address,now)
    with closing(db()) as conn,conn:
        conn.execute("UPDATE game_pow_work_v1 SET job_id=?,job_json=?,expires_at=?,status=? WHERE session_id=? AND request_key=? AND status='REQUESTING'",
                     (job.get("job_id") if valid else None,json.dumps(job) if valid else None,job["expires"] if valid else now,"ISSUED" if valid else "REJECTED",session_id,key))
    return {"ok":True,"job":job} if valid else {"ok":False,"error":"canonical_node_work_unavailable"}


async def submit_work(db, actor, session_id, job_id, nonce, now):
    if not isinstance(nonce,str) or not nonce.isascii() or not nonce.isdecimal() or len(nonce)>19:
        return {"ok":False,"error":"decimal_nonce_required"}
    with closing(db()) as conn,conn:
        conn.execute("BEGIN IMMEDIATE")
        row = conn.execute("""SELECT w.address,w.job_json,w.expires_at,w.status,w.receipt_json
            FROM game_pow_work_v1 w JOIN game_mining_runs_v1 r ON r.session_id=w.session_id
            WHERE w.session_id=? AND w.player_id=? AND w.job_id=? AND r.player_id=? AND r.status='STARTED' AND r.expires_at>=?""",(session_id,actor,job_id,actor,now)).fetchone()
        if row and row[3] == "ACCEPTED":
            return {"ok":True,"alreadyAccepted":True,"receipt":json.loads(row[4])}
        if not row or row[3] != "ISSUED" or row[2] <= now:
            return {"ok":False,"error":"work_invalid_expired_or_consumed"}
        job = json.loads(row[1]);value=int(nonce)
        if not validate_job(job,row[0],now):
            return {"ok":False,"error":"mining_network_or_job_invalid"}
        if value>int(job["max_nonce"]):
            return {"ok":False,"error":"nonce_out_of_range"}
        digest=hashlib.sha256(bytes.fromhex(job["header"])[:-8]+value.to_bytes(8,"big")).digest()
        if int.from_bytes(digest,"big") >> (256-job["difficulty"]) != 0:
            return {"ok":False,"error":"invalid_client_pow"}
        attempt={"hash":digest.hex(),"height":job["height"],"nonce":nonce,"job_id":job_id,"genesis_hash":job["genesis_hash"]}
        conn.execute("UPDATE game_pow_work_v1 SET status='SUBMITTING',receipt_json=? WHERE session_id=? AND status='ISSUED'",(json.dumps(attempt),session_id))
    receipt=await node_request("/api/mine/submit",{"address":row[0],"job_id":job_id,"nonce":nonce})
    accepted=receipt.get("accepted") is True and receipt.get("hash")==digest.hex() and receipt.get("height")==job["height"]
    if accepted:
        receipt={**attempt,**receipt}
    else:
        receipt=attempt
    with closing(db()) as conn,conn:
        if accepted: record_receipt(conn,actor,session_id,receipt)
        conn.execute("UPDATE game_pow_work_v1 SET status=?,receipt_json=? WHERE session_id=? AND job_id=? AND status='SUBMITTING'",("ACCEPTED" if accepted else "UNKNOWN",json.dumps(receipt),session_id,job_id))
    if not accepted:
        status=await work_status(db,actor,session_id,now)
        if status.get("alreadyAccepted"): return status
        return {"ok":False,"error":"work_stale" if status.get("ok") and not status.get("current") else "node_did_not_confirm_pow","reconciliationRequired":not status.get("ok")}
    return {"ok":True,"receipt":receipt,"rewardPaidByChain":True,"gameQCCredit_u":0}
