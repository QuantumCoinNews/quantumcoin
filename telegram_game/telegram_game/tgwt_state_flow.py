# TGWT_STATE_FLOW_V1
#
# TGWT state-flow layer for Telegram Game.
#
# This layer summarizes and validates off-chain TGWT rights:
# earned -> pending -> verified -> claimable -> claimed
#
# Important wording:
# - "withdrawn" is intentionally treated as a legacy UI word.
# - The correct model is user-paid BSC claim.
# - Backend does not send TGWT, sign transactions, store private keys, or pay BNB gas.
# - QC native network and BSC claim layer remain separate.

from __future__ import annotations

from copy import deepcopy
from typing import Any, Dict, List, Tuple

from tgwt_pool_policy import TGWT_POOL_TOTAL_U
from tgwt_ledger_events import (
    TGWT_LEDGER_EVENTS_VERSION,
    TGWT_EVENT_EARN_PENDING,
    TGWT_EVENT_VERIFY_PENDING,
    TGWT_EVENT_CLAIM_CONFIRMED,
    TGWT_EVENT_ADMIN_HOLD,
    TGWT_EVENT_REJECT,
    tgwt_amount_from_u,
    tgwt_amount_to_u,
    tgwt_ledger_event,
    tgwt_build_ledger_from_events,
    tgwt_append_event_log,
)


TGWT_STATE_FLOW_VERSION = "TGWT_STATE_FLOW_V1"
TGWT_STATE_FLOW_STATUS = "OFF_CHAIN_STATE_FLOW_ACTIVE_USER_PAID_BSC_CLAIM"

TGWT_STATE_EARNED = "earned"
TGWT_STATE_PENDING = "pending"
TGWT_STATE_VERIFIED = "verified"
TGWT_STATE_CLAIMABLE = "claimable"
TGWT_STATE_CLAIMED = "claimed"
TGWT_STATE_HELD = "held"
TGWT_STATE_REJECTED = "rejected"

TGWT_WITHDRAWN_LEGACY_ALIAS = "claimed"

TGWT_STATE_FLOW_SAFE_BOUNDARY = {
    "offChainOnly": True,
    "recordsRightsOnly": True,
    "userPaysBscFee": True,
    "usesClaimModel": True,
    "withdrawnIsLegacyAliasForClaimed": True,
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


def tgwt_state_flow_policy() -> Dict[str, Any]:
    return {
        "version": TGWT_STATE_FLOW_VERSION,
        "status": TGWT_STATE_FLOW_STATUS,
        "dependsOn": [TGWT_LEDGER_EVENTS_VERSION],
        "states": [
            TGWT_STATE_EARNED,
            TGWT_STATE_PENDING,
            TGWT_STATE_VERIFIED,
            TGWT_STATE_CLAIMABLE,
            TGWT_STATE_CLAIMED,
            TGWT_STATE_HELD,
            TGWT_STATE_REJECTED,
        ],
        "legacyAliases": {
            "withdrawn": TGWT_WITHDRAWN_LEGACY_ALIAS,
        },
        "claimModel": "USER_PAID_BSC_CLAIM",
        "feeModel": "BNB_GAS_PAID_BY_USER",
        "networkSeparation": {
            "qcNativeNetworkOnly": True,
            "bscClaimLayerSeparate": True,
            "telegramGameIsClaimGateway": True,
            "mixesQCAndBSC": False,
        },
        "safeBoundary": dict(TGWT_STATE_FLOW_SAFE_BOUNDARY),
    }


def tgwt_state_summary(user_id: str, events: List[Dict[str, Any]]) -> Dict[str, Any]:
    ledger = tgwt_build_ledger_from_events(user_id, events)

    earned_u = int(ledger.get("earned_u", 0) or 0)
    pending_u = int(ledger.get("pending_u", 0) or 0)
    verified_u = int(ledger.get("verified_u", 0) or 0)
    claimed_u = int(ledger.get("claimed_u", 0) or 0)
    claimable_u = int(ledger.get("claimable_u", 0) or 0)
    held_u = int(ledger.get("held_u", 0) or 0)
    rejected_u = int(ledger.get("rejected_u", 0) or 0)

    if earned_u > TGWT_POOL_TOTAL_U:
        raise ValueError("pool_cap_exceeded")

    return {
        "version": TGWT_STATE_FLOW_VERSION,
        "status": TGWT_STATE_FLOW_STATUS,
        "user_id": str(user_id or ""),
        "earned_u": earned_u,
        "pending_u": pending_u,
        "verified_u": verified_u,
        "claimable_u": claimable_u,
        "claimed_u": claimed_u,
        "withdrawn_u": claimed_u,
        "held_u": held_u,
        "rejected_u": rejected_u,
        "earned": tgwt_amount_from_u(earned_u),
        "pending": tgwt_amount_from_u(pending_u),
        "verified": tgwt_amount_from_u(verified_u),
        "claimable": tgwt_amount_from_u(claimable_u),
        "claimed": tgwt_amount_from_u(claimed_u),
        "withdrawn": tgwt_amount_from_u(claimed_u),
        "held": tgwt_amount_from_u(held_u),
        "rejected": tgwt_amount_from_u(rejected_u),
        "canClaim": claimable_u > 0,
        "requiresConnectedBscWallet": claimable_u > 0,
        "gasPaidBy": "USER",
        "backendPaysGas": False,
        "backendSendsToken": False,
        "safeBoundary": dict(TGWT_STATE_FLOW_SAFE_BOUNDARY),
    }


def tgwt_validate_state_invariants(summary: Dict[str, Any]) -> Tuple[bool, str]:
    earned_u = int(summary.get("earned_u", 0) or 0)
    pending_u = int(summary.get("pending_u", 0) or 0)
    verified_u = int(summary.get("verified_u", 0) or 0)
    claimable_u = int(summary.get("claimable_u", 0) or 0)
    claimed_u = int(summary.get("claimed_u", 0) or 0)
    held_u = int(summary.get("held_u", 0) or 0)
    rejected_u = int(summary.get("rejected_u", 0) or 0)

    values = [earned_u, pending_u, verified_u, claimable_u, claimed_u, held_u, rejected_u]
    if any(v < 0 for v in values):
        return False, "negative_state_value"

    if earned_u > TGWT_POOL_TOTAL_U:
        return False, "pool_cap_exceeded"

    if claimable_u != max(0, verified_u - claimed_u):
        return False, "claimable_mismatch"

    if pending_u + verified_u + held_u + rejected_u > earned_u:
        return False, "state_total_exceeds_earned"

    return True, "ok"


def tgwt_create_earn_pending_event(
    event_id: str,
    user_id: str,
    amount: Any,
    source: str,
    reason: str = "",
    meta: Dict[str, Any] | None = None,
) -> Dict[str, Any]:
    return tgwt_ledger_event(
        event_id,
        user_id,
        TGWT_EVENT_EARN_PENDING,
        tgwt_amount_to_u(amount),
        source,
        reason,
        meta or {},
    )


def tgwt_create_verify_event(
    event_id: str,
    user_id: str,
    amount: Any,
    source: str = "ADMIN_REVIEW",
    reason: str = "verified for future user-paid BSC claim",
    meta: Dict[str, Any] | None = None,
) -> Dict[str, Any]:
    return tgwt_ledger_event(
        event_id,
        user_id,
        TGWT_EVENT_VERIFY_PENDING,
        tgwt_amount_to_u(amount),
        source,
        reason,
        meta or {},
    )


def tgwt_create_claim_confirmed_event(
    event_id: str,
    user_id: str,
    amount: Any,
    bsc_tx_hash: str,
    bsc_address: str,
    meta: Dict[str, Any] | None = None,
) -> Dict[str, Any]:
    tx = str(bsc_tx_hash or "").strip()
    addr = str(bsc_address or "").strip()

    if not tx:
        raise ValueError("empty_bsc_tx_hash")
    if not (addr.startswith("0x") and len(addr) == 42):
        raise ValueError("invalid_bsc_address")

    m = dict(meta or {})
    m["bscTxHash"] = tx
    m["bscAddress"] = addr
    m["gasPaidBy"] = "USER"
    m["backendPaidGas"] = False

    return tgwt_ledger_event(
        event_id,
        user_id,
        TGWT_EVENT_CLAIM_CONFIRMED,
        tgwt_amount_to_u(amount),
        "BSC_USER_PAID_CLAIM_RECORD",
        "record confirmed user-paid BSC claim; backend did not send token or pay gas",
        m,
    )


def tgwt_create_admin_hold_event(
    event_id: str,
    user_id: str,
    amount: Any,
    reason: str = "admin hold",
    meta: Dict[str, Any] | None = None,
) -> Dict[str, Any]:
    return tgwt_ledger_event(
        event_id,
        user_id,
        TGWT_EVENT_ADMIN_HOLD,
        tgwt_amount_to_u(amount),
        "ADMIN_REVIEW",
        reason,
        meta or {},
    )


def tgwt_create_reject_event(
    event_id: str,
    user_id: str,
    amount: Any,
    reason: str = "admin reject",
    meta: Dict[str, Any] | None = None,
) -> Dict[str, Any]:
    return tgwt_ledger_event(
        event_id,
        user_id,
        TGWT_EVENT_REJECT,
        tgwt_amount_to_u(amount),
        "ADMIN_REVIEW",
        reason,
        meta or {},
    )


def tgwt_apply_state_event_log(
    user_id: str,
    events: List[Dict[str, Any]],
    event: Dict[str, Any],
) -> Tuple[List[Dict[str, Any]], Dict[str, Any], bool, str]:
    new_events, ok, reason = tgwt_append_event_log(events, event)
    summary = tgwt_state_summary(user_id, new_events)

    if not ok:
        return events, tgwt_state_summary(user_id, events), False, reason

    valid, valid_reason = tgwt_validate_state_invariants(summary)
    if not valid:
        return events, tgwt_state_summary(user_id, events), False, valid_reason

    return new_events, summary, True, "applied"
