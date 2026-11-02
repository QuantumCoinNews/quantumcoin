"""Identity/activity checks shared with freeze; retired manual selection entry.

This does not implement a fair draw or prove possession of a wallet key.
"""
import datetime
import sqlite3

from .nft_mint_bridge_v1 import NFTMintBridgeError, _wallet_owner


class SelectionRejected(ValueError):
    pass


def _require(condition, code):
    if not condition:
        raise SelectionRejected(code)


def _identity(conn, candidate, now):
    user, period = candidate["userId"], candidate["period"]
    _require(isinstance(user, str) and user and user == user.strip(), "monthly_candidate_identity_invalid")
    count = conn.execute("SELECT COUNT(*) FROM monthly_nft_candidates WHERE period=? AND TRIM(user_id)=?", (period, user)).fetchone()[0]
    _require(count == 1, "monthly_candidate_user_not_unique")
    player = conn.execute("SELECT qc_address FROM players WHERE player_id=?", (user,)).fetchone()
    _require(player is not None and isinstance(player[0], str), "linked_wallet_required")
    try:
        owner = _wallet_owner(player[0].strip())
    except NFTMintBridgeError:
        raise SelectionRejected("linked_wallet_invalid") from None
    owners = []
    for player_id, address in conn.execute("SELECT player_id,qc_address FROM players WHERE qc_address IS NOT NULL"):
        try:
            if _wallet_owner(str(address).strip()) == owner:
                owners.append(player_id)
        except NFTMintBridgeError:
            continue
    _require(owners == [user], "monthly_candidate_wallet_not_unique")
    try:
        start = datetime.datetime.strptime(period, "%Y-%m").replace(tzinfo=datetime.timezone.utc)
        _require(start.strftime("%Y-%m") == period, "monthly_candidate_period_invalid")
        end = start.replace(year=start.year+1, month=1) if start.month == 12 else start.replace(month=start.month+1)
    except (ValueError, OverflowError):
        raise SelectionRejected("monthly_candidate_period_invalid") from None
    evidence = conn.execute("SELECT COUNT(*) FROM monthly_nft_verified_game_sessions WHERE user_id=? AND period=? AND activity_verified=1 AND completed_at>=? AND completed_at<? AND completed_at<=?", (user, period, int(start.timestamp()), int(end.timestamp()), now)).fetchone()[0]
    _require(evidence > 0, "verified_game_session_required")


def select_monthly_winner_identity_v1(connect, candidate_id, allowed_periods, now, reviewer, note, ensure_tables, serialize):
    conn = None
    try:
        _require(bool(allowed_periods), "monthly_nft_program_not_active_or_period_outside")
        conn = connect()
        # Serialize writers in separate processes, as well as one API worker.
        conn.execute("BEGIN IMMEDIATE")
        ensure_tables(conn)
        columns = "candidate_id,user_id,period,nft_type,source,activity_json,score,ai_review_status,ai_risk_level,admin_status,admin_note,winner_status,minted_status,policy_version,created_at,updated_at"
        row = conn.execute("SELECT " + columns + " FROM monthly_nft_candidates WHERE candidate_id=?", (candidate_id,)).fetchone()
        _require(row is not None, "candidate_not_found")
        candidate = serialize(row)
        # QMA-0070A: after a monthly pool is committed, manual selection must
        # not bypass the future verified-draw-only route. Same BEGIN IMMEDIATE
        # serializes this guard with freeze in other processes.
        frozen_table = conn.execute("SELECT 1 FROM sqlite_master WHERE type='table' "
                                    "AND name='monthly_nft_pool_freezes_v1'").fetchone()
        if frozen_table and conn.execute("SELECT 1 FROM monthly_nft_pool_freezes_v1 "
                                         "WHERE period=?", (candidate["period"],)).fetchone():
            raise SelectionRejected("monthly_pool_frozen_manual_selection_disabled")
        _require(candidate["period"] in allowed_periods, "monthly_nft_program_not_active_or_period_outside")
        _require(candidate["nftType"] == "rare", "monthly_nft_type_invalid")
        _require(candidate["aiReviewStatus"] == "AI_REVIEW_PASSED", "ai_review_not_passed")
        _require(candidate["aiRiskLevel"] == "LOW", "ai_risk_not_low")
        _require(candidate["adminStatus"] == "APPROVED_BY_ADMIN", "admin_not_approved")
        _require(candidate["mintedStatus"] == "NOT_MINTED", "candidate_already_minted_or_locked")
        _identity(conn, candidate, now)
        existing = conn.execute("SELECT candidate_id FROM monthly_nft_candidates WHERE period=? AND (winner_status='SELECTED_MONTHLY_WINNER' OR minted_status IS NOT 'NOT_MINTED') LIMIT 1", (candidate["period"],)).fetchone()
        _require(existing is None, "monthly_winner_already_selected")
        # Identity/activity validation is not random selection authority. Keep
        # the compatibility entry fail-closed before *any* winner write, even
        # when the pool has not been frozen. No flag/admin input re-enables it.
        raise SelectionRejected("monthly_manual_selection_disabled")
    except SelectionRejected as error:
        if conn is not None:
            conn.rollback()
        return {"ok": False, "error": str(error)}
    except (sqlite3.Error, ValueError, TypeError, KeyError, OverflowError):
        if conn is not None:
            conn.rollback()
        return {"ok": False, "error": "monthly_selection_store_unavailable"}
    finally:
        if conn is not None:
            conn.close()
