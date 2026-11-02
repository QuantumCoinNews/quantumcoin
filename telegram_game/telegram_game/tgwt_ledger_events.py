# TGWT_LEDGER_EVENTS_V1
#
# Off-chain TGWT ledger/event foundation for Telegram Game.
#
# This module does NOT:
# - send TGWT
# - call BSC
# - hold private keys
# - create a blockchain transaction
# - pay BNB gas
# - mutate QC balances
# - touch QuantumCoin consensus/chain/mining
#
# Direction:
# - Telegram Game records off-chain TGWT rights.
# - User-paid BSC claim will be a later layer.
# - BNB fee/gas is paid by the user at claim time.
# - QC native network and BSC TGWT claim layer stay separate.

from __future__ import annotations

from copy import deepcopy
from decimal import Decimal, InvalidOperation
from typing import Any, Dict, List, Tuple

from tgwt_pool_policy import TGWT_POOL_TOTAL_U, TGWT_UNIT


TGWT_LEDGER_EVENTS_VERSION = "TGWT_LEDGER_EVENTS_V1"

TGWT_EVENT_EARN_PENDING = "EARN_PENDING"
TGWT_EVENT_VERIFY_PENDING = "VERIFY_PENDING"
TGWT_EVENT_CLAIM_CONFIRMED = "CLAIM_CONFIRMED"
TGWT_EVENT_ADMIN_HOLD = "ADMIN_HOLD"
TGWT_EVENT_REJECT = "REJECT"

TGWT_LEDGER_STATUS_ACTIVE = "OFF_CHAIN_LEDGER_ACTIVE_NO_BSC_TRANSFER"

TGWT_LEDGER_SAFE_BOUNDARY = {
    "offChainOnly": True,
    "recordsRightsOnly": True,
    "userPaysBscFee": True,
    "createsBscTransaction": False,
    "sendsTGWT": False,
    "autoPayout": False,
    "storesPrivateKey": False,
    "signsTransaction": False,
    "paysGas": False,
    "usesHotWallet": False,
    "usesTreasuryWallet": False,
    "mintsToken": False,
    "burnsToken": False,
    "changesQCBalances": False,
    "changesTGWTOnChainBalances": False,
    "changesMempool": False,
    "changesChain": False,
    "replacesChain": False,
    "changesConsensus": False,
    "changesTotalSupply": False,
    "startsMining": False,
    "stopsMining": False,
}


def tgwt_ledger_initial(user_id: str = "") -> Dict[str, Any]:
    return {
        "version": TGWT_LEDGER_EVENTS_VERSION,
        "status": TGWT_LEDGER_STATUS_ACTIVE,
        "user_id": str(user_id or ""),
        "earned_u": 0,
        "pending_u": 0,
        "verified_u": 0,
        "claimed_u": 0,
        "claimable_u": 0,
        "held_u": 0,
        "rejected_u": 0,
        "event_count": 0,
        "last_event_id": "",
        "safeBoundary": dict(TGWT_LEDGER_SAFE_BOUNDARY),
    }


def tgwt_amount_to_u(amount: Any) -> int:
    try:
        d = Decimal(str(amount).strip())
    except (InvalidOperation, ValueError, AttributeError):
        raise ValueError("invalid_tgwt_amount")

    if d <= 0:
        raise ValueError("amount_must_be_positive")

    micro = d * Decimal(TGWT_UNIT)

    if micro != micro.to_integral_value():
        raise ValueError("amount_has_too_many_decimals")

    return int(micro)


def tgwt_amount_from_u(amount_u: int) -> str:
    amount_u = int(amount_u)
    if amount_u < 0:
        amount_u = 0

    whole = amount_u // TGWT_UNIT
    frac = amount_u % TGWT_UNIT

    if frac == 0:
        return str(whole)

    frac_s = str(frac).rjust(6, "0").rstrip("0")
    return f"{whole}.{frac_s}"


def tgwt_ledger_event(
    event_id: str,
    user_id: str,
    event_type: str,
    amount_u: int,
    source: str,
    reason: str = "",
    meta: Dict[str, Any] | None = None,
) -> Dict[str, Any]:
    event_id = str(event_id or "").strip()
    user_id = str(user_id or "").strip()
    event_type = str(event_type or "").strip().upper()
    source = str(source or "").strip()
    reason = str(reason or "").strip()
    amount_u = int(amount_u)

    if not event_id:
        raise ValueError("empty_event_id")
    if not user_id:
        raise ValueError("empty_user_id")
    if event_type not in {
        TGWT_EVENT_EARN_PENDING,
        TGWT_EVENT_VERIFY_PENDING,
        TGWT_EVENT_CLAIM_CONFIRMED,
        TGWT_EVENT_ADMIN_HOLD,
        TGWT_EVENT_REJECT,
    }:
        raise ValueError("unsupported_event_type")
    if amount_u <= 0:
        raise ValueError("amount_must_be_positive")
    if amount_u > TGWT_POOL_TOTAL_U:
        raise ValueError("amount_exceeds_total_pool")

    return {
        "version": TGWT_LEDGER_EVENTS_VERSION,
        "id": event_id,
        "user_id": user_id,
        "type": event_type,
        "amount_u": amount_u,
        "amount": tgwt_amount_from_u(amount_u),
        "source": source,
        "reason": reason,
        "meta": dict(meta or {}),
        "safeBoundary": dict(TGWT_LEDGER_SAFE_BOUNDARY),
    }


def tgwt_recalculate_claimable(ledger: Dict[str, Any]) -> Dict[str, Any]:
    verified_u = int(ledger.get("verified_u", 0) or 0)
    claimed_u = int(ledger.get("claimed_u", 0) or 0)
    claimable_u = verified_u - claimed_u
    if claimable_u < 0:
        claimable_u = 0
    ledger["claimable_u"] = claimable_u
    return ledger


def tgwt_apply_event(
    ledger: Dict[str, Any],
    event: Dict[str, Any],
    seen_event_ids: set[str] | None = None,
) -> Tuple[Dict[str, Any], bool, str]:
    out = deepcopy(ledger or tgwt_ledger_initial(str(event.get("user_id", ""))))
    seen = seen_event_ids if seen_event_ids is not None else set()

    event_id = str(event.get("id", "")).strip()
    if not event_id:
        return tgwt_recalculate_claimable(out), False, "empty_event_id"

    if event_id in seen:
        return tgwt_recalculate_claimable(out), False, "duplicate_event"

    event_user = str(event.get("user_id", "")).strip()
    ledger_user = str(out.get("user_id", "")).strip()
    if ledger_user and event_user and ledger_user != event_user:
        return tgwt_recalculate_claimable(out), False, "user_mismatch"

    if not ledger_user:
        out["user_id"] = event_user

    event_type = str(event.get("type", "")).strip().upper()
    amount_u = int(event.get("amount_u", 0) or 0)

    if amount_u <= 0:
        return tgwt_recalculate_claimable(out), False, "invalid_amount"

    earned_u = int(out.get("earned_u", 0) or 0)
    pending_u = int(out.get("pending_u", 0) or 0)
    verified_u = int(out.get("verified_u", 0) or 0)
    claimed_u = int(out.get("claimed_u", 0) or 0)
    held_u = int(out.get("held_u", 0) or 0)
    rejected_u = int(out.get("rejected_u", 0) or 0)

    if event_type == TGWT_EVENT_EARN_PENDING:
        if earned_u + amount_u > TGWT_POOL_TOTAL_U:
            return tgwt_recalculate_claimable(out), False, "pool_cap_exceeded"
        out["earned_u"] = earned_u + amount_u
        out["pending_u"] = pending_u + amount_u

    elif event_type == TGWT_EVENT_VERIFY_PENDING:
        if amount_u > pending_u:
            return tgwt_recalculate_claimable(out), False, "insufficient_pending"
        out["pending_u"] = pending_u - amount_u
        out["verified_u"] = verified_u + amount_u

    elif event_type == TGWT_EVENT_CLAIM_CONFIRMED:
        claimable_u = max(0, verified_u - claimed_u)
        if amount_u > claimable_u:
            return tgwt_recalculate_claimable(out), False, "insufficient_claimable"
        out["claimed_u"] = claimed_u + amount_u

    elif event_type == TGWT_EVENT_ADMIN_HOLD:
        if amount_u > pending_u:
            return tgwt_recalculate_claimable(out), False, "insufficient_pending"
        out["pending_u"] = pending_u - amount_u
        out["held_u"] = held_u + amount_u

    elif event_type == TGWT_EVENT_REJECT:
        if amount_u > pending_u:
            return tgwt_recalculate_claimable(out), False, "insufficient_pending"
        out["pending_u"] = pending_u - amount_u
        out["rejected_u"] = rejected_u + amount_u

    else:
        return tgwt_recalculate_claimable(out), False, "unsupported_event_type"

    seen.add(event_id)
    out["event_count"] = int(out.get("event_count", 0) or 0) + 1
    out["last_event_id"] = event_id
    out["safeBoundary"] = dict(TGWT_LEDGER_SAFE_BOUNDARY)

    return tgwt_recalculate_claimable(out), True, "applied"


def tgwt_build_ledger_from_events(user_id: str, events: List[Dict[str, Any]]) -> Dict[str, Any]:
    ledger = tgwt_ledger_initial(user_id)
    seen: set[str] = set()

    for ev in list(events or []):
        if str(ev.get("user_id", "")).strip() != str(user_id or "").strip():
            continue
        ledger, _applied, _reason = tgwt_apply_event(ledger, ev, seen)

    return tgwt_recalculate_claimable(ledger)


def tgwt_append_event_log(
    events: List[Dict[str, Any]],
    event: Dict[str, Any],
) -> Tuple[List[Dict[str, Any]], bool, str]:
    current = list(events or [])
    event_id = str(event.get("id", "")).strip()

    if not event_id:
        return current, False, "empty_event_id"

    for ev in current:
        if str(ev.get("id", "")).strip() == event_id:
            return current, False, "duplicate_event"

    current.append(deepcopy(event))
    return current, True, "appended"


def tgwt_ledger_policy() -> Dict[str, Any]:
    return {
        "version": TGWT_LEDGER_EVENTS_VERSION,
        "status": TGWT_LEDGER_STATUS_ACTIVE,
        "poolTotalU": TGWT_POOL_TOTAL_U,
        "poolTotal": tgwt_amount_from_u(TGWT_POOL_TOTAL_U),
        "eventTypes": [
            TGWT_EVENT_EARN_PENDING,
            TGWT_EVENT_VERIFY_PENDING,
            TGWT_EVENT_CLAIM_CONFIRMED,
            TGWT_EVENT_ADMIN_HOLD,
            TGWT_EVENT_REJECT,
        ],
        "claimModel": "USER_PAID_BSC_CLAIM_LATER",
        "networkSeparation": {
            "qcNativeNetworkOnly": True,
            "bscClaimLayerSeparate": True,
            "telegramGameIsClaimGateway": True,
            "mixesQCAndBSC": False,
        },
        "safeBoundary": dict(TGWT_LEDGER_SAFE_BOUNDARY),
    }
