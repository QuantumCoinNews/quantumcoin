"""Atomic off-chain TGWT reservations. BSC confirmation is a separate boundary.

earned is lifetime credit; pending/verified/reserved/withdrawn are disjoint
buckets. Legacy records are preserved and never silently certified or repaired.
"""
from contextlib import closing
from decimal import Decimal, InvalidOperation
import hashlib
import json
import re
import secrets
from .exact_numbers import decimal_atoms

UNIT = 1_000_000


def schema(conn):
    conn.execute("""CREATE TABLE IF NOT EXISTS tgwt_withdraw_reservations_v1 (
        request_id TEXT PRIMARY KEY, player_id TEXT NOT NULL, nonce TEXT NOT NULL,
        fingerprint TEXT NOT NULL, amount_u INTEGER NOT NULL CHECK(amount_u>0),
        status TEXT NOT NULL, updated_at INTEGER NOT NULL,
        UNIQUE(player_id,nonce))""")
    conn.execute("""CREATE TABLE IF NOT EXISTS tgwt_withdraw_confirmations_v1 (
        request_id TEXT PRIMARY KEY, chain_id INTEGER NOT NULL, tx_hash TEXT NOT NULL,
        log_index INTEGER NOT NULL, evidence_json TEXT NOT NULL,
        UNIQUE(chain_id,tx_hash,log_index))""")


def amounts(conn, actor):
    schema(conn)
    row = conn.execute("SELECT earned_u,pending_u,verified_u,withdrawn_u,rev FROM tgwt_state WHERE user_id=?",(actor,)).fetchone() or (0,0,0,0,0)
    result = dict(zip(("earned_u","pending_u","verified_u","withdrawn_u","rev"),map(int,row)))
    result["reserved_u"] = conn.execute("SELECT COALESCE(SUM(amount_u),0) FROM tgwt_withdraw_reservations_v1 WHERE player_id=? AND status='ADMIN_REVIEW'",(actor,)).fetchone()[0]
    result["legacyUnreconciledRequests"] = conn.execute("""SELECT COUNT(*) FROM withdraw_requests r
        LEFT JOIN tgwt_withdraw_reservations_v1 h ON h.request_id=r.id
        WHERE r.user_id=? AND h.request_id IS NULL AND r.status NOT IN ('REJECTED','RELEASED','CONFIRMED')""",(actor,)).fetchone()[0]
    result["accountingConsistent"] = (all(type(v) is int and 0<=v<=3_000_000*UNIT for v in row[:4])
        and result["earned_u"] == sum(result[k] for k in ("pending_u","verified_u","reserved_u","withdrawn_u")))
    return result


def event(conn, actor, kind, amount, request, now):
    conn.execute("INSERT INTO tgwt_events VALUES(?,?,?,?,?,?)",(secrets.token_hex(16),actor,kind,amount,json.dumps({"requestId":request}),now))


def reserve(db, actor, destination, amount, nonce, now):
    try:
        if not re.fullmatch(r"0x[0-9a-fA-F]{40}",destination or "") or int(destination[2:],16)==0:
            raise ValueError("invalid_to_address")
        if not isinstance(nonce,str) or not re.fullmatch(r"[A-Za-z0-9_.:-]{8,128}",nonce):
            raise ValueError("invalid_request_nonce")
        quantity = decimal_atoms(amount)
        if not 0 < quantity <= 3_000_000*UNIT:
            raise ValueError("invalid_micro_unit_amount")
        destination = destination.lower()
        fingerprint = hashlib.sha256((destination+":"+str(quantity)).encode()).hexdigest()
        with closing(db()) as conn, conn:
            conn.execute("BEGIN IMMEDIATE")
            schema(conn)
            prior = conn.execute("SELECT request_id,fingerprint,status FROM tgwt_withdraw_reservations_v1 WHERE player_id=? AND nonce=?",(actor,nonce)).fetchone()
            if prior:
                if prior[1]!=fingerprint:
                    raise ValueError("request_nonce_conflict")
                return {"ok":True,"requestId":prior[0],"status":prior[2],"alreadyRecorded":True,"balances":amounts(conn,actor),"onChainPaid":prior[2]=="CONFIRMED"}
            # New accounts must come through the canonical verified reward engine.
            balance = amounts(conn,actor)
            if not balance["accountingConsistent"] or balance["legacyUnreconciledRequests"]:
                raise ValueError("legacy_ledger_reconciliation_required")
            if conn.execute("UPDATE tgwt_state SET verified_u=verified_u-?,rev=rev+1,updated_at=? WHERE user_id=? AND verified_u>=?",(quantity,now,actor,quantity)).rowcount!=1:
                raise ValueError("insufficient_verified")
            request = secrets.token_hex(16)
            conn.execute("INSERT INTO withdraw_requests(id,user_id,asset,amount_u,to_address,status,tx_hash,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)",(request,actor,"TGWT",quantity,destination,"ADMIN_REVIEW",None,now,now))
            conn.execute("INSERT INTO tgwt_withdraw_reservations_v1 VALUES(?,?,?,?,?,?,?)",(request,actor,nonce,fingerprint,quantity,"ADMIN_REVIEW",now))
            event(conn,actor,"WITHDRAW_RESERVED",quantity,request,now)
            return {"ok":True,"requestId":request,"status":"ADMIN_REVIEW","onChainPaid":False,"balances":amounts(conn,actor),
                    "gasPayer":"user","gasAsset":"BNB","fee":"0","bscClaimEnabled":False}
    except (InvalidOperation,ValueError) as error:
        return {"ok":False,"error":str(error) if isinstance(error,ValueError) else "invalid_amount"}


def release(db, request, actor, now, operator=False, reject=False):
    try:
        with closing(db()) as conn, conn:
            conn.execute("BEGIN IMMEDIATE")
            schema(conn)
            row = conn.execute("SELECT player_id,amount_u,status FROM tgwt_withdraw_reservations_v1 WHERE request_id=?",(request,)).fetchone()
            if not row or (not operator and actor!=row[0]):
                raise ValueError("owned_reservation_required")
            terminal = "REJECTED" if reject else "RELEASED"
            if row[2]==terminal:
                return {"ok":True,"status":terminal,"alreadyRecorded":True}
            if row[2]!="ADMIN_REVIEW":
                raise ValueError("withdraw_transition_not_allowed")
            conn.execute("UPDATE tgwt_state SET verified_u=verified_u+?,rev=rev+1,updated_at=? WHERE user_id=?",(row[1],now,row[0]))
            conn.execute("UPDATE tgwt_withdraw_reservations_v1 SET status=?,updated_at=? WHERE request_id=?",(terminal,now,request))
            conn.execute("UPDATE withdraw_requests SET status=?,updated_at=? WHERE id=?",(terminal,now,request))
            event(conn,row[0],"WITHDRAW_"+terminal,row[1],request,now)
            return {"ok":True,"status":terminal,"onChainPaid":False,"balances":amounts(conn,row[0])}
    except ValueError as error:
        return {"ok":False,"error":str(error)}


def confirm(db, request, tx_hash, verify_chain, now):
    """verify_chain is trusted server code, never a body-supplied receipt/flag.

    A production HTTP adapter remains disabled until the user-paid claim
    contract and its request-bound event are actually deployed and configured.
    """
    # TODO(mainnet-final): Integrate real BSC receipt/finality verification in
    # the final mainnet phase, as requested on 2026-11-04; fixtures cannot enable it.
    try:
        if not isinstance(tx_hash,str) or not re.fullmatch(r"0x[0-9a-f]{64}",tx_hash):
            raise ValueError("invalid_bsc_transaction_hash")
        with closing(db()) as conn:
            schema(conn)
            row = conn.execute("SELECT r.user_id,r.to_address,r.amount_u,h.status FROM withdraw_requests r JOIN tgwt_withdraw_reservations_v1 h ON h.request_id=r.id WHERE r.id=?",(request,)).fetchone()
            if not row:
                raise ValueError("managed_reservation_required")
        evidence = verify_chain(request,row[1],row[2],tx_hash)
        if not isinstance(evidence,dict) or evidence.get("requestId")!=request or evidence.get("recipient")!=row[1] or evidence.get("amount_u")!=row[2] or evidence.get("txHash")!=tx_hash or evidence.get("chainId")!=56 or evidence.get("finalized") is not True or type(evidence.get("logIndex")) is not int or evidence["logIndex"]<0:
            raise ValueError("verified_bsc_claim_evidence_required")
        with closing(db()) as conn, conn:
            conn.execute("BEGIN IMMEDIATE")
            current = conn.execute("SELECT status FROM tgwt_withdraw_reservations_v1 WHERE request_id=?",(request,)).fetchone()[0]
            if current=="CONFIRMED":
                previous = conn.execute("SELECT tx_hash FROM tgwt_withdraw_confirmations_v1 WHERE request_id=?",(request,)).fetchone()
                if previous[0]!=tx_hash:
                    raise ValueError("confirmation_conflict")
                return {"ok":True,"alreadyRecorded":True,"status":"CONFIRMED"}
            if current!="ADMIN_REVIEW":
                raise ValueError("withdraw_transition_not_allowed")
            conn.execute("INSERT INTO tgwt_withdraw_confirmations_v1 VALUES(?,?,?,?,?)",(request,56,tx_hash,evidence["logIndex"],json.dumps(evidence,sort_keys=True)))
            conn.execute("UPDATE tgwt_state SET withdrawn_u=withdrawn_u+?,rev=rev+1,updated_at=? WHERE user_id=?",(row[2],now,row[0]))
            conn.execute("UPDATE tgwt_withdraw_reservations_v1 SET status='CONFIRMED',updated_at=? WHERE request_id=?",(now,request))
            conn.execute("UPDATE withdraw_requests SET status='CONFIRMED',tx_hash=?,updated_at=? WHERE id=?",(tx_hash,now,request))
            event(conn,row[0],"WITHDRAW_CONFIRMED",row[2],request,now)
            return {"ok":True,"status":"CONFIRMED","onChainPaid":True,"balances":amounts(conn,row[0])}
    except ValueError as error:
        return {"ok":False,"error":str(error)}
