# TGWT_CLAIM_REQUEST_V1
#
# Off-chain TGWT claim request foundation.
#
# This layer creates and validates user claim request records only.
# It does NOT create a BSC transaction, send TGWT, store private keys,
# sign transactions, use hot wallets, use treasury wallets, or pay BNB gas.
#
# Correct model:
# - User sees claimable TGWT in Telegram Game.
# - User connects BSC wallet / MetaMask.
# - Backend records claim request and review decision.
# - Later BSC claim contract will be called by user wallet.
# - User pays BNB gas.
# - QC native network and BSC claim layer remain separate.

from __future__ import annotations

import hashlib
from typing import Any, Dict, List, Tuple

from tgwt_ledger_events import tgwt_amount_from_u, tgwt_amount_to_u
from tgwt_state_flow import tgwt_state_summary, tgwt_validate_state_invariants
from tgwt_admin_review import (
    TGWT_ADMIN_REVIEW_VERSION,
    TGWT_REVIEW_APPROVED_FOR_USER_PAID_CLAIM,
    TGWT_REVIEW_PENDING_ADMIN_REVIEW,
    TGWT_REVIEW_HOLD,
    TGWT_REVIEW_REJECTED,
    tgwt_admin_review_decision,
)


TGWT_CLAIM_REQUEST_VERSION = "TGWT_CLAIM_REQUEST_V1"
TGWT_CLAIM_REQUEST_STATUS_ACTIVE = "OFF_CHAIN_CLAIM_REQUEST_ACTIVE_NO_BSC_TRANSFER"

TGWT_CLAIM_STATUS_READY_FOR_USER_PAID_CLAIM = "READY_FOR_USER_PAID_CLAIM"
TGWT_CLAIM_STATUS_PENDING_ADMIN_REVIEW = "PENDING_ADMIN_REVIEW"
TGWT_CLAIM_STATUS_HOLD = "HOLD"
TGWT_CLAIM_STATUS_REJECTED = "REJECTED"

TGWT_CLAIM_REQUEST_SAFE_BOUNDARY = {
    "offChainOnly": True,
    "recordsClaimRequestOnly": True,
    "userPaysBscFee": True,
    "requiresConnectedBscWallet": True,
    "requiresUserWalletSignatureLater": True,
    "contractAddressRequiredNow": False,
    "contractAddressRequiredLater": True,
    "tokenContractAddressRequiredLater": True,
    "claimContractAddressRequiredLater": True,
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


def tgwt_claim_request_policy() -> Dict[str, Any]:
    return {
        "version": TGWT_CLAIM_REQUEST_VERSION,
        "status": TGWT_CLAIM_REQUEST_STATUS_ACTIVE,
        "dependsOn": [TGWT_ADMIN_REVIEW_VERSION],
        "claimModel": "USER_PAID_BSC_CLAIM",
        "feeModel": "BNB_GAS_PAID_BY_USER",
        "contractAddressPolicy": {
            "requiredNow": False,
            "requiredLater": True,
            "tokenContractAddressRequiredLater": True,
            "claimContractAddressRequiredLater": True,
        },
        "networkSeparation": {
            "qcNativeNetworkOnly": True,
            "bscClaimLayerSeparate": True,
            "telegramGameIsClaimGateway": True,
            "mixesQCAndBSC": False,
        },
        "safeBoundary": dict(TGWT_CLAIM_REQUEST_SAFE_BOUNDARY),
    }


def tgwt_make_claim_id(user_id: str, bsc_address: str, amount: Any, nonce: str) -> str:
    raw = "|".join([
        "TGWT_CLAIM_REQUEST_V1",
        str(user_id or "").strip(),
        str(bsc_address or "").strip().lower(),
        str(amount or "").strip(),
        str(nonce or "").strip(),
    ])
    return "tgwt_claim_" + hashlib.sha256(raw.encode("utf-8")).hexdigest()[:32]


def tgwt_claim_request_record(
    user_id: str,
    amount: Any,
    claimable_amount: Any,
    bsc_address: str,
    nonce: str,
    previous_claim_ids: List[str] | None = None,
    risk_flags: List[str] | None = None,
    user_daily_claimed_amount: Any = "0",
    global_daily_claimed_amount: Any = "0",
) -> Dict[str, Any]:
    claim_id = tgwt_make_claim_id(user_id, bsc_address, amount, nonce)

    decision = tgwt_admin_review_decision(
        user_id=user_id,
        amount=amount,
        claimable_amount=claimable_amount,
        bsc_address=bsc_address,
        risk_flags=risk_flags or [],
        user_daily_claimed_amount=user_daily_claimed_amount,
        global_daily_claimed_amount=global_daily_claimed_amount,
        previous_claim_ids=previous_claim_ids or [],
        claim_id=claim_id,
    )

    status = _status_from_review(decision.get("status", ""))

    try:
        amount_u = tgwt_amount_to_u(amount)
    except Exception:
        amount_u = 0

    try:
        claimable_u = tgwt_amount_to_u(claimable_amount)
    except Exception:
        claimable_u = 0

    return {
        "version": TGWT_CLAIM_REQUEST_VERSION,
        "status": status,
        "claim_id": claim_id,
        "user_id": str(user_id or "").strip(),
        "bsc_address": str(bsc_address or "").strip(),
        "amount_u": amount_u,
        "amount": tgwt_amount_from_u(amount_u),
        "claimable_u": claimable_u,
        "claimable": tgwt_amount_from_u(claimable_u),
        "nonce": str(nonce or "").strip(),
        "review": dict(decision),
        "claimModel": "USER_PAID_BSC_CLAIM",
        "feeModel": "BNB_GAS_PAID_BY_USER",
        "backendPaysGas": False,
        "backendSendsToken": False,
        "createsBscTransaction": False,
        "requiresConnectedBscWallet": True,
        "requiresUserWalletSignatureLater": True,
        "contractAddressRequiredNow": False,
        "tokenContractAddressRequiredLater": True,
        "claimContractAddressRequiredLater": True,
        "safeBoundary": dict(TGWT_CLAIM_REQUEST_SAFE_BOUNDARY),
    }


def tgwt_claim_request_from_events(
    user_id: str,
    events: List[Dict[str, Any]],
    amount: Any,
    bsc_address: str,
    nonce: str,
    previous_claim_ids: List[str] | None = None,
    risk_flags: List[str] | None = None,
    user_daily_claimed_amount: Any = "0",
    global_daily_claimed_amount: Any = "0",
) -> Dict[str, Any]:
    summary = tgwt_state_summary(user_id, events)
    valid, reason = tgwt_validate_state_invariants(summary)

    if not valid:
        return {
            "version": TGWT_CLAIM_REQUEST_VERSION,
            "status": TGWT_CLAIM_STATUS_REJECTED,
            "claim_id": "",
            "user_id": str(user_id or "").strip(),
            "bsc_address": str(bsc_address or "").strip(),
            "amount_u": 0,
            "amount": "0",
            "claimable_u": int(summary.get("claimable_u", 0) or 0),
            "claimable": str(summary.get("claimable", "0")),
            "nonce": str(nonce or "").strip(),
            "review": {
                "status": TGWT_REVIEW_REJECTED,
                "risk": "HIGH",
                "reasons": [reason],
            },
            "claimModel": "USER_PAID_BSC_CLAIM",
            "feeModel": "BNB_GAS_PAID_BY_USER",
            "backendPaysGas": False,
            "backendSendsToken": False,
            "createsBscTransaction": False,
            "requiresConnectedBscWallet": True,
            "requiresUserWalletSignatureLater": True,
            "contractAddressRequiredNow": False,
            "safeBoundary": dict(TGWT_CLAIM_REQUEST_SAFE_BOUNDARY),
        }

    return tgwt_claim_request_record(
        user_id=user_id,
        amount=amount,
        claimable_amount=summary.get("claimable", "0"),
        bsc_address=bsc_address,
        nonce=nonce,
        previous_claim_ids=previous_claim_ids or [],
        risk_flags=risk_flags or [],
        user_daily_claimed_amount=user_daily_claimed_amount,
        global_daily_claimed_amount=global_daily_claimed_amount,
    )


def tgwt_claim_request_is_ready(record: Dict[str, Any]) -> bool:
    return str(record.get("status", "")) == TGWT_CLAIM_STATUS_READY_FOR_USER_PAID_CLAIM


def tgwt_claim_request_requires_admin(record: Dict[str, Any]) -> bool:
    return str(record.get("status", "")) in {
        TGWT_CLAIM_STATUS_PENDING_ADMIN_REVIEW,
        TGWT_CLAIM_STATUS_HOLD,
    }


def tgwt_claim_request_is_rejected(record: Dict[str, Any]) -> bool:
    return str(record.get("status", "")) == TGWT_CLAIM_STATUS_REJECTED


def _status_from_review(review_status: str) -> str:
    if review_status == TGWT_REVIEW_APPROVED_FOR_USER_PAID_CLAIM:
        return TGWT_CLAIM_STATUS_READY_FOR_USER_PAID_CLAIM
    if review_status == TGWT_REVIEW_PENDING_ADMIN_REVIEW:
        return TGWT_CLAIM_STATUS_PENDING_ADMIN_REVIEW
    if review_status == TGWT_REVIEW_HOLD:
        return TGWT_CLAIM_STATUS_HOLD
    return TGWT_CLAIM_STATUS_REJECTED


def tgwt_claim_request_append(
    records: List[Dict[str, Any]],
    record: Dict[str, Any],
) -> Tuple[List[Dict[str, Any]], bool, str]:
    current = list(records or [])
    claim_id = str(record.get("claim_id", "")).strip()

    if not claim_id:
        return current, False, "empty_claim_id"

    for existing in current:
        if str(existing.get("claim_id", "")).strip() == claim_id:
            return current, False, "duplicate_claim_id"

    current.append(dict(record))
    return current, True, "appended"
