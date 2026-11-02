# TGWT_ADMIN_REVIEW_V1
#
# Admin review / suspicious claim filter for Telegram Game TGWT.
#
# This layer reviews off-chain claim eligibility only.
# It does NOT create BSC transactions, send TGWT, store private keys,
# sign transactions, use hot wallets, use treasury wallets, or pay BNB gas.
#
# Correct model:
# - Telegram Game records TGWT claim rights.
# - User pays BNB gas when claiming later on BSC.
# - Backend only classifies/recommends claim status.
# - QC native network and BSC claim layer stay separate.

from __future__ import annotations

from typing import Any, Dict, List

from tgwt_pool_policy import TGWT_POOL_TOTAL_U, TGWT_UNIT
from tgwt_ledger_events import tgwt_amount_from_u, tgwt_amount_to_u
from tgwt_state_flow import TGWT_STATE_FLOW_VERSION


TGWT_ADMIN_REVIEW_VERSION = "TGWT_ADMIN_REVIEW_V1"
TGWT_ADMIN_REVIEW_STATUS_ACTIVE = "OFF_CHAIN_ADMIN_REVIEW_ACTIVE_NO_BSC_TRANSFER"

TGWT_REVIEW_APPROVED_FOR_USER_PAID_CLAIM = "APPROVED_FOR_USER_PAID_CLAIM"
TGWT_REVIEW_PENDING_ADMIN_REVIEW = "PENDING_ADMIN_REVIEW"
TGWT_REVIEW_HOLD = "HOLD"
TGWT_REVIEW_REJECTED = "REJECTED"

TGWT_REVIEW_RISK_LOW = "LOW"
TGWT_REVIEW_RISK_MEDIUM = "MEDIUM"
TGWT_REVIEW_RISK_HIGH = "HIGH"

TGWT_AUTO_APPROVE_MAX_U = 50 * TGWT_UNIT
TGWT_SINGLE_CLAIM_REVIEW_U = 500 * TGWT_UNIT
TGWT_USER_DAILY_REVIEW_U = 1_000 * TGWT_UNIT
TGWT_GLOBAL_DAILY_REVIEW_U = 50_000 * TGWT_UNIT

TGWT_ADMIN_REVIEW_SAFE_BOUNDARY = {
    "offChainOnly": True,
    "recordsDecisionOnly": True,
    "userPaysBscFee": True,
    "requiresUserWalletSignatureLater": True,
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


def tgwt_is_bsc_address(value: Any) -> bool:
    s = str(value or "").strip()
    if not (s.startswith("0x") and len(s) == 42):
        return False

    body = s[2:]
    for ch in body:
        if ch not in "0123456789abcdefABCDEF":
            return False

    return True


def tgwt_admin_review_policy() -> Dict[str, Any]:
    return {
        "version": TGWT_ADMIN_REVIEW_VERSION,
        "status": TGWT_ADMIN_REVIEW_STATUS_ACTIVE,
        "dependsOn": [TGWT_STATE_FLOW_VERSION],
        "autoApproveMax": tgwt_amount_from_u(TGWT_AUTO_APPROVE_MAX_U),
        "singleClaimReviewAt": tgwt_amount_from_u(TGWT_SINGLE_CLAIM_REVIEW_U),
        "userDailyReviewAt": tgwt_amount_from_u(TGWT_USER_DAILY_REVIEW_U),
        "globalDailyReviewAt": tgwt_amount_from_u(TGWT_GLOBAL_DAILY_REVIEW_U),
        "claimModel": "USER_PAID_BSC_CLAIM",
        "feeModel": "BNB_GAS_PAID_BY_USER",
        "networkSeparation": {
            "qcNativeNetworkOnly": True,
            "bscClaimLayerSeparate": True,
            "telegramGameIsClaimGateway": True,
            "mixesQCAndBSC": False,
        },
        "safeBoundary": dict(TGWT_ADMIN_REVIEW_SAFE_BOUNDARY),
    }


def tgwt_admin_review_decision(
    user_id: str,
    amount: Any,
    claimable_amount: Any,
    bsc_address: str,
    risk_flags: List[str] | None = None,
    user_daily_claimed_amount: Any = "0",
    global_daily_claimed_amount: Any = "0",
    previous_claim_ids: List[str] | None = None,
    claim_id: str = "",
) -> Dict[str, Any]:
    reasons: List[str] = []
    flags = [str(x).strip() for x in list(risk_flags or []) if str(x).strip()]
    previous = set(str(x).strip() for x in list(previous_claim_ids or []) if str(x).strip())
    claim_id = str(claim_id or "").strip()
    user_id = str(user_id or "").strip()
    bsc_address = str(bsc_address or "").strip()

    if not user_id:
        reasons.append("empty_user_id")
        return _tgwt_review_result(
            user_id,
            claim_id,
            bsc_address,
            0,
            0,
            TGWT_REVIEW_REJECTED,
            TGWT_REVIEW_RISK_HIGH,
            reasons,
            flags,
        )

    try:
        amount_u = tgwt_amount_to_u(amount)
    except Exception:
        reasons.append("invalid_amount")
        return _tgwt_review_result(
            user_id,
            claim_id,
            bsc_address,
            0,
            0,
            TGWT_REVIEW_REJECTED,
            TGWT_REVIEW_RISK_HIGH,
            reasons,
            flags,
        )

    try:
        claimable_u = tgwt_amount_to_u(claimable_amount)
    except Exception:
        claimable_u = 0

    try:
        user_daily_u = tgwt_amount_to_u(user_daily_claimed_amount)
    except Exception:
        user_daily_u = 0

    try:
        global_daily_u = tgwt_amount_to_u(global_daily_claimed_amount)
    except Exception:
        global_daily_u = 0

    if not tgwt_is_bsc_address(bsc_address):
        reasons.append("invalid_bsc_address")

    if claim_id and claim_id in previous:
        reasons.append("duplicate_claim_id")

    if amount_u > claimable_u:
        reasons.append("insufficient_claimable")

    if amount_u > TGWT_POOL_TOTAL_U:
        reasons.append("amount_exceeds_total_pool")

    if flags:
        reasons.append("risk_flags_present")

    if user_daily_u + amount_u > TGWT_USER_DAILY_REVIEW_U:
        reasons.append("user_daily_limit_review")

    if global_daily_u + amount_u > TGWT_GLOBAL_DAILY_REVIEW_U:
        reasons.append("global_daily_limit_review")

    if amount_u >= TGWT_SINGLE_CLAIM_REVIEW_U:
        reasons.append("large_single_claim_review")

    if any(r in reasons for r in [
        "invalid_bsc_address",
        "duplicate_claim_id",
        "insufficient_claimable",
        "amount_exceeds_total_pool",
    ]):
        status = TGWT_REVIEW_REJECTED
        risk = TGWT_REVIEW_RISK_HIGH
    elif any(r in reasons for r in [
        "risk_flags_present",
        "user_daily_limit_review",
        "global_daily_limit_review",
        "large_single_claim_review",
    ]):
        status = TGWT_REVIEW_HOLD
        risk = TGWT_REVIEW_RISK_HIGH
    elif amount_u > TGWT_AUTO_APPROVE_MAX_U:
        reasons.append("amount_requires_admin_review")
        status = TGWT_REVIEW_PENDING_ADMIN_REVIEW
        risk = TGWT_REVIEW_RISK_MEDIUM
    else:
        reasons.append("eligible_for_user_paid_claim")
        status = TGWT_REVIEW_APPROVED_FOR_USER_PAID_CLAIM
        risk = TGWT_REVIEW_RISK_LOW

    return _tgwt_review_result(
        user_id,
        claim_id,
        bsc_address,
        amount_u,
        claimable_u,
        status,
        risk,
        reasons,
        flags,
    )


def _tgwt_review_result(
    user_id: str,
    claim_id: str,
    bsc_address: str,
    amount_u: int,
    claimable_u: int,
    status: str,
    risk: str,
    reasons: List[str],
    flags: List[str],
) -> Dict[str, Any]:
    return {
        "version": TGWT_ADMIN_REVIEW_VERSION,
        "status": status,
        "risk": risk,
        "user_id": user_id,
        "claim_id": claim_id,
        "bsc_address": bsc_address,
        "amount_u": int(amount_u or 0),
        "amount": tgwt_amount_from_u(int(amount_u or 0)),
        "claimable_u": int(claimable_u or 0),
        "claimable": tgwt_amount_from_u(int(claimable_u or 0)),
        "reasons": list(reasons or []),
        "riskFlags": list(flags or []),
        "claimModel": "USER_PAID_BSC_CLAIM",
        "feeModel": "BNB_GAS_PAID_BY_USER",
        "backendPaysGas": False,
        "backendSendsToken": False,
        "requiresConnectedBscWallet": True,
        "requiresUserWalletSignatureLater": True,
        "createsBscTransaction": False,
        "safeBoundary": dict(TGWT_ADMIN_REVIEW_SAFE_BOUNDARY),
    }


def tgwt_admin_review_is_allowed(decision: Dict[str, Any]) -> bool:
    return str(decision.get("status", "")) == TGWT_REVIEW_APPROVED_FOR_USER_PAID_CLAIM


def tgwt_admin_review_requires_admin(decision: Dict[str, Any]) -> bool:
    return str(decision.get("status", "")) in {
        TGWT_REVIEW_PENDING_ADMIN_REVIEW,
        TGWT_REVIEW_HOLD,
    }


def tgwt_admin_review_is_rejected(decision: Dict[str, Any]) -> bool:
    return str(decision.get("status", "")) == TGWT_REVIEW_REJECTED
