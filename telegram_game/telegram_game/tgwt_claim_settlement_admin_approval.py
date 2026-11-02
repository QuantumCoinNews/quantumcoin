# TGWT_CLAIM_SETTLEMENT_ADMIN_APPROVAL_V1
#
# Admin approval layer for Telegram Game TGWT user-paid BSC claim settlement.
#
# Purpose:
# - Convert a clean settlement precheck into an admin decision.
# - APPROVE means the user may later claim through their own BSC wallet.
# - HOLD pauses the claim for manual review.
# - REJECT blocks the claim.
#
# This module does NOT:
# - create a BSC transaction
# - send TGWT
# - pay gas
# - sign anything
# - store private keys
# - call RPC
# - call wallet providers
# - auto-approve claims
# - touch QC chain/mining/consensus/balances

from __future__ import annotations

from typing import Any, Dict, List

from tgwt_claim_settlement_precheck import (
    TGWT_CLAIM_SETTLEMENT_PRECHECK_VERSION,
    TGWT_CLAIM_SETTLEMENT_READY_FOR_ADMIN_REVIEW,
)


TGWT_CLAIM_SETTLEMENT_ADMIN_APPROVAL_VERSION = "TGWT_CLAIM_SETTLEMENT_ADMIN_APPROVAL_V1"
TGWT_CLAIM_SETTLEMENT_ADMIN_APPROVAL_STATUS = "ADMIN_APPROVAL_ONLY_NO_AUTOMATIC_TRANSFER"

TGWT_ADMIN_DECISION_APPROVE = "APPROVE"
TGWT_ADMIN_DECISION_HOLD = "HOLD"
TGWT_ADMIN_DECISION_REJECT = "REJECT"

TGWT_SETTLEMENT_ADMIN_APPROVED = "SETTLEMENT_ADMIN_APPROVED_USER_PAID_CLAIM_READY"
TGWT_SETTLEMENT_ADMIN_HOLD = "SETTLEMENT_ADMIN_HOLD_MANUAL_REVIEW"
TGWT_SETTLEMENT_ADMIN_REJECTED = "SETTLEMENT_ADMIN_REJECTED"
TGWT_SETTLEMENT_ADMIN_PRECHECK_REQUIRED = "SETTLEMENT_ADMIN_PRECHECK_REQUIRED"
TGWT_SETTLEMENT_ADMIN_INVALID_DECISION = "SETTLEMENT_ADMIN_INVALID_DECISION"

TGWT_SETTLEMENT_ADMIN_SAFE_BOUNDARY = {
    "adminApprovalOnly": True,
    "offChainOnly": True,
    "requiresPassedPrecheck": True,
    "manualAdminDecisionRequired": True,
    "approveMeansUserPaidClaimReady": True,
    "holdMeansManualReview": True,
    "rejectMeansNoClaim": True,
    "noAutomaticApproval": True,
    "noAutomaticTransfer": True,
    "noBackendTokenSend": True,
    "noBackendGasPayment": True,
    "noPrivateKeyStorage": True,
    "noSigning": True,
    "noRpcCall": True,
    "noWalletProviderCall": True,
    "noContractCall": True,
    "userPaysBnbGasLater": True,
    "userSignsClaimLater": True,
    "claimGatewayOnly": True,
    "automaticTransferAllowed": False,
    "backendPaysGas": False,
    "backendSendsToken": False,
    "createsBscTransaction": False,
    "deploysContract": False,
    "broadcastsTransaction": False,
    "storesPrivateKey": False,
    "signsTransaction": False,
    "changesQCBalances": False,
    "changesTGWTOnChainBalances": False,
    "changesMempool": False,
    "changesChain": False,
    "changesConsensus": False,
    "changesTotalSupply": False,
    "startsMining": False,
    "stopsMining": False,
}


def tgwt_claim_settlement_admin_policy() -> Dict[str, Any]:
    return {
        "version": TGWT_CLAIM_SETTLEMENT_ADMIN_APPROVAL_VERSION,
        "status": TGWT_CLAIM_SETTLEMENT_ADMIN_APPROVAL_STATUS,
        "dependsOn": [TGWT_CLAIM_SETTLEMENT_PRECHECK_VERSION],
        "allowedDecisions": [
            TGWT_ADMIN_DECISION_APPROVE,
            TGWT_ADMIN_DECISION_HOLD,
            TGWT_ADMIN_DECISION_REJECT,
        ],
        "approvalMeaning": "USER_PAID_BSC_CLAIM_CAN_BE_PREPARED_BY_UI",
        "transferMeaning": "NO_BACKEND_TRANSFER_NO_BACKEND_GAS_NO_SIGNING",
        "safeBoundary": dict(TGWT_SETTLEMENT_ADMIN_SAFE_BOUNDARY),
    }


def tgwt_claim_settlement_admin_decision(
    precheck: Dict[str, Any],
    decision: str,
    admin_id: str,
    note: str = "",
) -> Dict[str, Any]:
    precheck = dict(precheck or {})
    decision = str(decision or "").strip().upper()
    admin_id = str(admin_id or "").strip()
    note = str(note or "").strip()

    reasons: List[str] = []
    warnings: List[str] = []

    if not admin_id:
        reasons.append("missing_admin_id")

    if decision not in {
        TGWT_ADMIN_DECISION_APPROVE,
        TGWT_ADMIN_DECISION_HOLD,
        TGWT_ADMIN_DECISION_REJECT,
    }:
        return _tgwt_settlement_admin_result(
            TGWT_SETTLEMENT_ADMIN_INVALID_DECISION,
            ["invalid_admin_decision"],
            warnings,
            precheck,
            decision,
            admin_id,
            note,
        )

    precheck_status = str(precheck.get("status", "")).strip().upper()
    if precheck_status != TGWT_CLAIM_SETTLEMENT_READY_FOR_ADMIN_REVIEW:
        reasons.append("precheck_not_ready_for_admin_review")

    if bool(precheck.get("readyForAdminReview", False)) is not True:
        reasons.append("ready_for_admin_review_flag_missing")

    if bool(precheck.get("readyForAutomaticTransfer", False)) is True:
        reasons.append("automatic_transfer_flag_not_allowed")

    if bool(precheck.get("createsBscTransaction", False)) is True:
        reasons.append("precheck_must_not_create_bsc_transaction")

    if bool(precheck.get("backendPaysGas", False)) is True:
        reasons.append("backend_pays_gas_not_allowed")

    if bool(precheck.get("backendSendsToken", False)) is True:
        reasons.append("backend_sends_token_not_allowed")

    if not str(precheck.get("claimId", "")).strip():
        reasons.append("missing_claim_id")

    if not str(precheck.get("userId", "")).strip():
        reasons.append("missing_user_id")

    if not str(precheck.get("bscAddress", "")).strip():
        reasons.append("missing_bsc_address")

    if decision == TGWT_ADMIN_DECISION_APPROVE and reasons:
        return _tgwt_settlement_admin_result(
            TGWT_SETTLEMENT_ADMIN_PRECHECK_REQUIRED,
            reasons,
            warnings,
            precheck,
            decision,
            admin_id,
            note,
        )

    if decision == TGWT_ADMIN_DECISION_HOLD:
        return _tgwt_settlement_admin_result(
            TGWT_SETTLEMENT_ADMIN_HOLD,
            reasons or ["admin_hold_manual_review"],
            warnings,
            precheck,
            decision,
            admin_id,
            note,
        )

    if decision == TGWT_ADMIN_DECISION_REJECT:
        return _tgwt_settlement_admin_result(
            TGWT_SETTLEMENT_ADMIN_REJECTED,
            reasons or ["admin_rejected_claim_settlement"],
            warnings,
            precheck,
            decision,
            admin_id,
            note,
        )

    return _tgwt_settlement_admin_result(
        TGWT_SETTLEMENT_ADMIN_APPROVED,
        ["admin_approved_user_paid_claim_ready_no_transfer"],
        warnings,
        precheck,
        decision,
        admin_id,
        note,
    )


def tgwt_claim_settlement_ui_payload(admin_result: Dict[str, Any]) -> Dict[str, Any]:
    admin_result = dict(admin_result or {})
    approved = str(admin_result.get("status", "")) == TGWT_SETTLEMENT_ADMIN_APPROVED

    return {
        "version": TGWT_CLAIM_SETTLEMENT_ADMIN_APPROVAL_VERSION,
        "status": "CLAIM_UI_READY_USER_PAID" if approved else "CLAIM_UI_NOT_READY",
        "claimId": str(admin_result.get("claimId", "")).strip(),
        "userId": str(admin_result.get("userId", "")).strip(),
        "bscAddress": str(admin_result.get("bscAddress", "")).strip(),
        "amountTGWT": int(admin_result.get("amountTGWT", 0) or 0),
        "requiresUserWallet": True,
        "requiresUserSignature": True,
        "requiresUserBnbGas": True,
        "backendCreatesTransaction": False,
        "backendPaysGas": False,
        "backendSendsToken": False,
        "readyForAutomaticTransfer": False,
        "safeBoundary": dict(TGWT_SETTLEMENT_ADMIN_SAFE_BOUNDARY),
    }


def _tgwt_settlement_admin_result(
    status: str,
    reasons: List[str],
    warnings: List[str],
    precheck: Dict[str, Any],
    decision: str,
    admin_id: str,
    note: str,
) -> Dict[str, Any]:
    approved = status == TGWT_SETTLEMENT_ADMIN_APPROVED

    return {
        "version": TGWT_CLAIM_SETTLEMENT_ADMIN_APPROVAL_VERSION,
        "status": status,
        "decision": decision,
        "reasons": list(reasons or []),
        "warnings": list(warnings or []),
        "adminId": admin_id,
        "adminNote": note,
        "claimId": str(precheck.get("claimId", "")).strip(),
        "userId": str(precheck.get("userId", "")).strip(),
        "bscAddress": str(precheck.get("bscAddress", "")).strip(),
        "amountTGWT": int(precheck.get("amountTGWT", 0) or 0),
        "precheckStatus": str(precheck.get("status", "")).strip(),
        "approvedForUserPaidClaim": approved,
        "readyForAutomaticTransfer": False,
        "createsBscTransaction": False,
        "backendPaysGas": False,
        "backendSendsToken": False,
        "requiresUserSignatureLater": True,
        "requiresUserBnbGasLater": True,
        "safeBoundary": dict(TGWT_SETTLEMENT_ADMIN_SAFE_BOUNDARY),
    }
