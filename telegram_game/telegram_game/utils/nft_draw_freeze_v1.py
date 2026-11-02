"""QMA-0070A: atomic, replayable monthly NFT candidate-pool commitment.

A frozen pool is *not* a draw, evidence of independently supplied randomness,
proof of wallet private-key possession, or approval to mint. No wallet writes.
"""

import datetime
import hashlib
import json
import re
import sqlite3

from .nft_mint_bridge_v1 import NFTMintBridgeError, _wallet_owner
from .nft_selection_identity_v1 import SelectionRejected, _identity

VERSION = "QMA_0070_CANDIDATE_FREEZE_V1"
_DOMAIN = b"QMA_0070_CANDIDATE_FREEZE_V1\x00"
_COLUMNS = ("candidate_id,user_id,period,nft_type,source,activity_json,score,"
            "ai_review_status,ai_risk_level,admin_status,admin_note,winner_status,"
            "minted_status,policy_version,created_at,updated_at")
_MAX_CANDIDATES = 10000


def _require(ok, code):
    if not ok:
        raise SelectionRejected(code)


def _period_end(period):
    _require(isinstance(period, str) and re.fullmatch(r"[0-9]{4}-(0[1-9]|1[0-2])", period), "invalid_freeze_period")
    try:
        year, month = map(int, period.split("-"))
        start = datetime.datetime(year, month, 1, tzinfo=datetime.timezone.utc)
        end = (start.replace(year=year + 1, month=1) if month == 12
               else start.replace(month=month + 1))
        return int(end.timestamp())
    except (ValueError, OverflowError):
        raise SelectionRejected("invalid_freeze_period") from None


def _canonical_json(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=True, allow_nan=False)


def manifest_hash_v1(manifest):
    """Verifiable with standard Python; JSON canonicalization is versioned."""
    _require(isinstance(manifest, dict) and manifest.get("version") == VERSION, "invalid_frozen_manifest")
    return hashlib.sha256(_DOMAIN + _canonical_json(manifest).encode("ascii")).hexdigest()


def verify_frozen_pool_v1(manifest, digest):
    _require(isinstance(manifest, dict) and set(manifest) == {"version", "period", "cutoffUnix", "frozenAt", "candidates"}, "invalid_frozen_manifest")
    _require(type(manifest["cutoffUnix"]) is int and manifest["cutoffUnix"] == _period_end(manifest["period"]), "invalid_frozen_manifest")
    _require(type(manifest["frozenAt"]) is int and manifest["frozenAt"] >= manifest["cutoffUnix"], "invalid_frozen_manifest")
    rows = manifest["candidates"]
    _require(isinstance(rows, list) and 0 < len(rows) <= _MAX_CANDIDATES, "invalid_frozen_manifest")
    _require(all(isinstance(c, dict) and set(c) == {"candidateId", "userId", "walletOwner"}
                 and all(isinstance(v, str) and v and v == v.strip() for v in c.values())
                 and re.fullmatch(r"[0-9a-f]{40}", c["walletOwner"])
                 for c in rows), "invalid_frozen_manifest")
    _require(rows == sorted(rows, key=lambda c: (c["userId"], c["candidateId"])), "invalid_frozen_manifest")
    _require(len({c["candidateId"] for c in rows}) == len(rows)
             and len({c["userId"] for c in rows}) == len(rows)
             and len({c["walletOwner"] for c in rows}) == len(rows), "invalid_frozen_manifest")
    _require(isinstance(digest, str) and re.fullmatch(r"[0-9a-f]{64}", digest)
             and manifest_hash_v1(manifest) == digest, "frozen_manifest_hash_mismatch")
    return True


def _ensure_freeze_tables(conn):
    conn.execute("""CREATE TABLE IF NOT EXISTS monthly_nft_pool_freezes_v1 (
        period TEXT PRIMARY KEY, version TEXT NOT NULL, pool_json TEXT NOT NULL,
        pool_sha256 TEXT NOT NULL, candidate_count INTEGER NOT NULL,
        frozen_at INTEGER NOT NULL)""")
    # Ordinary SQL updates/deletes cannot silently change a frozen commitment.
    # A privileged DB administrator can still replace data/schema; publish the
    # hash outside the DB before any external randomness is released.
    conn.execute("""CREATE TRIGGER IF NOT EXISTS qma0070_pool_immutable_update_v1
        BEFORE UPDATE ON monthly_nft_pool_freezes_v1 BEGIN
        SELECT RAISE(ABORT, 'frozen_pool_immutable'); END""")
    conn.execute("""CREATE TRIGGER IF NOT EXISTS qma0070_pool_immutable_delete_v1
        BEFORE DELETE ON monthly_nft_pool_freezes_v1 BEGIN
        SELECT RAISE(ABORT, 'frozen_pool_immutable'); END""")


def _get_existing(conn, period):
    result = conn.execute("SELECT version,pool_json,pool_sha256,candidate_count,frozen_at "
                          "FROM monthly_nft_pool_freezes_v1 WHERE period=?", (period,)).fetchone()
    if result is None:
        return None
    version, raw, digest, count, at = result
    _require(version == VERSION and isinstance(raw, str), "frozen_manifest_corrupt")
    try:
        manifest = json.loads(raw)
        _require(_canonical_json(manifest) == raw, "frozen_manifest_corrupt")
        verify_frozen_pool_v1(manifest, digest)
        _require(manifest["period"] == period and len(manifest["candidates"]) == count
                 and type(at) is int and at == manifest["frozenAt"], "frozen_manifest_corrupt")
    except (SelectionRejected, ValueError, TypeError, KeyError, OverflowError):
        raise SelectionRejected("frozen_manifest_corrupt") from None
    return {"period": period, "version": VERSION, "poolHash": digest,
            "candidateCount": count, "frozenAt": at, "manifest": manifest,
            "safeBoundary": {"minted": False, "winnerSelected": False,
                "balanceChanged": False, "chainMetadataWritten": False,
                "randomnessVerified": False,
                "note": "Frozen candidate commitment only. No randomness verification or winner draw."}}


def freeze_monthly_pool_v1(connect, period, allowed_periods, now, ensure_candidates, serialize):
    conn = None
    try:
        _require(isinstance(now, int) and type(now) is int, "invalid_freeze_time")
        cutoff = _period_end(period)
        _require(period in allowed_periods, "monthly_nft_program_not_active_or_period_outside")
        _require(now >= cutoff, "monthly_period_not_closed")
        conn = connect()
        conn.execute("BEGIN IMMEDIATE")
        ensure_candidates(conn)
        _ensure_freeze_tables(conn)
        existing = _get_existing(conn, period)
        if existing:
            conn.commit()
            return {"ok": True, "alreadyFrozen": True, **existing}
        rows = conn.execute("SELECT " + _COLUMNS + " FROM monthly_nft_candidates WHERE period=? "
                            "ORDER BY user_id,candidate_id LIMIT ?", (period, _MAX_CANDIDATES + 1)).fetchall()
        _require(len(rows) <= _MAX_CANDIDATES, "monthly_candidate_pool_too_large")
        selected = []
        for row in rows:
            cand = serialize(row)
            _require(cand["mintedStatus"] == "NOT_MINTED"
                     and cand["winnerStatus"] != "SELECTED_MONTHLY_WINNER", "monthly_winner_already_selected")
            if (cand["nftType"] != "rare" or cand["aiReviewStatus"] != "AI_REVIEW_PASSED"
                    or cand["aiRiskLevel"] != "LOW" or cand["adminStatus"] != "APPROVED_BY_ADMIN"):
                continue
            _identity(conn, cand, now)
            wallet_row = conn.execute("SELECT qc_address FROM players WHERE player_id=?", (cand["userId"],)).fetchone()
            _require(wallet_row is not None, "linked_wallet_required")
            try:
                owner = _wallet_owner(wallet_row[0].strip())
            except (NFTMintBridgeError, AttributeError):
                raise SelectionRejected("linked_wallet_invalid") from None
            selected.append({"candidateId": cand["candidateId"], "userId": cand["userId"], "walletOwner": owner})
        _require(bool(selected), "monthly_no_eligible_candidates")
        selected.sort(key=lambda c: (c["userId"], c["candidateId"]))
        manifest = {"version": VERSION, "period": period, "cutoffUnix": cutoff, "frozenAt": now, "candidates": selected}
        digest = manifest_hash_v1(manifest)
        verify_frozen_pool_v1(manifest, digest)
        conn.execute("INSERT INTO monthly_nft_pool_freezes_v1 VALUES(?,?,?,?,?,?)",
                     (period, VERSION, _canonical_json(manifest), digest, len(selected), now))
        conn.commit()
        return {"ok": True, "alreadyFrozen": False, **_get_frozen_after_commit(manifest, digest, now)}
    except SelectionRejected as error:
        if conn is not None and conn.in_transaction:
            conn.rollback()
        return {"ok": False, "error": str(error)}
    except (sqlite3.Error, ValueError, TypeError, KeyError, OverflowError):
        if conn is not None and conn.in_transaction:
            conn.rollback()
        return {"ok": False, "error": "monthly_freeze_store_unavailable"}
    finally:
        if conn is not None:
            conn.close()


def _get_frozen_after_commit(manifest, digest, now):
    return {"period": manifest["period"], "version": VERSION, "poolHash": digest,
            "candidateCount": len(manifest["candidates"]), "frozenAt": now,
            "manifest": manifest, "safeBoundary": {
                "minted": False, "winnerSelected": False, "balanceChanged": False,
                "chainMetadataWritten": False, "randomnessVerified": False,
                "note": "Frozen candidate commitment only. No randomness verification or winner draw."}}


def read_monthly_pool_v1(connect, period):
    conn = None
    try:
        _period_end(period)
        conn = connect()
        exists = conn.execute("SELECT 1 FROM sqlite_master WHERE type='table' "
                              "AND name='monthly_nft_pool_freezes_v1'").fetchone()
        if not exists:
            return {"ok": False, "error": "monthly_pool_not_frozen"}
        row = _get_existing(conn, period)
        return {"ok": True, **row} if row else {"ok": False, "error": "monthly_pool_not_frozen"}
    except SelectionRejected as error:
        return {"ok": False, "error": str(error)}
    except (sqlite3.Error, ValueError, TypeError, KeyError):
        return {"ok": False, "error": "monthly_freeze_store_unavailable"}
    finally:
        if conn is not None:
            conn.close()
