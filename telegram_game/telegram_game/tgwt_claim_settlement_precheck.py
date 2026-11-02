# TGWT_CLAIM_SETTLEMENT_PRECHECK_V1
#
# Claim settlement precheck for Telegram Game TGWT user-paid BSC claim.
#
# Purpose:
# - Check whether an off-chain TGWT claim request is structurally ready for later BSC user-paid claim.
# - Validate user, claim id, claim status, amount, claimable balance, BSC address and BSC config.
# - Keep the result as a precheck/admin-review signal only.
#
# This module does NOT:
# - create a BSC transaction
# - send TGWT
# - pay gas
# - sign anything
# - store private keys
# - call RPC
# - call wallet providers
# - touch QC chain/mining/consensus/balances

from __future__ import annotations

import re
from typing import Any, Dict, List


TGWT_CLAIM_SETTLEMENT_PRECHECK_VERSION = "TGWT_CLAIM_SETTLEMENT_PRECHECK_V1"
TGWT_CLAIM_SETTLEMENT_PRECHECK_STATUS = "PRECHECK_ONLY_NO_BSC_TRANSACTION"

TGWT_CLAIM_SETTLEMENT_READY_FOR_ADMIN_REVIEW = "CLAIM_SETTLEMENT_READY_FOR_ADMIN_REVIEW"
TGWT_CLAIM_SETTLEMENT_CONFIG_REQUIRED = "CLAIM_SETTLEMENT_CONFIG_REQUIRED"
TGWT_CLAIM_SETTLEMENT_REJECTED = "CLAIM_SETTLEMENT_REJECTED"
TGWT_CLAIM_SETTLEMENT_HOLD = "CLAIM_SETTLEMENT_HOLD"

TGWT_CLAIM_READY_STATUS = "READY_FOR_USER_PAID_CLAIM"
TGWT_CLAIM_PENDING_ADMIN_STATUS = "PENDING_ADMIN_REVIEW"

TGWT_BSC_TESTNET_CHAIN_ID = 97
TGWT_BSC_MAINNET_CHAIN_ID = 56

TGWT_CLAIM_SETTLEMENT_SAFE_BOUNDARY = {
    "precheckOnly": True,
    "offChainOnly": True,
    "adminReviewSignalOnly": True,
    "requiresUserPaidClaim": True,
    "requiresBscAddress": True,
    "requiresClaimStatusReady": True,
    "requiresAmountWithinClaimable": True,
    "requiresChainConfigWhenEnabled": True,
    "requiresTokenContractWhenEnabled": True,
    "requiresClaimContractWhenEnabled": True,
    "doesNotCreateTransaction": True,
    "doesNotSendToken": True,
    "doesNotPayGas": True,
    "doesNotSign": True,
    "doesNotStorePrivateKey": True,
    "doesNotCallRpc": True,
    "doesNotCallWalletProvider": True,
    "userPaysBnbGasLater": True,
    "adminApprovalHandledByNextLayer": True,
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


def tgwt_claim_settlement_precheck_policy() -> Dict[str, Any]:
    return {
        "version": TGWT_CLAIM_SETTLEMENT_PRECHECK_VERSION,
        "status": TGWT_CLAIM_SETTLEMENT_PRECHECK_STATUS,
        "dependsOn": [
            "TGWT_CLAIM_REQUEST_V1",
            "TGWT_BSC_CLAIM_CONFIG_V1",
            "TGWT_ADMIN_REVIEW_V1",
        ],
        "resultUse": "ADMIN_REVIEW_SIGNAL_ONLY",
        "settlementModel": "USER_PAID_BSC_CLAIM_LATER",
        "adminApprovalHandledByNextLayer": True,
        "safeBoundary": dict(TGWT_CLAIM_SETTLEMENT_SAFE_BOUNDARY),
    }


def tgwt_claim_settlement_precheck(
    claim_request: Dict[str, Any],
    bsc_config: Dict[str, Any],
    mode: str = "TESTNET",
) -> Dict[str, Any]:
    claim_request = dict(claim_request or {})
    bsc_config = dict(bsc_config or {})
    mode = str(mode or "TESTNET").strip().upper()

    reasons: List[str] = []
    warnings: List[str] = []

    user_id = str(claim_request.get("userId", "")).strip()
    claim_id = str(claim_request.get("claimId", "")).strip()
    bsc_address = str(claim_request.get("bscAddress", "")).strip()
    claim_status = str(claim_request.get("claimStatus", "")).strip().upper()
    admin_status = str(claim_request.get("adminStatus", TGWT_CLAIM_PENDING_ADMIN_STATUS)).strip().upper()

    amount = _tgwt_int(claim_request.get("amountTGWT", 0))
    claimable = _tgwt_int(claim_request.get("claimableTGWT", 0))

    if not user_id:
        reasons.append("missing_user_id")

    if not claim_id:
        reasons.append("missing_claim_id")

    if claim_status != TGWT_CLAIM_READY_STATUS:
        reasons.append("claim_status_not_ready_for_user_paid_claim")

    if admin_status not in {TGWT_CLAIM_PENDING_ADMIN_STATUS, "APPROVED_BY_ADMIN"}:
        reasons.append("invalid_admin_status_for_precheck")

    if not _is_valid_bsc_address(bsc_address):
        reasons.append("invalid_bsc_address")

    if amount <= 0:
        reasons.append("invalid_claim_amount")

    if claimable <= 0:
        reasons.append("invalid_claimable_amount")

    if amount > claimable:
        reasons.append("claim_amount_exceeds_claimable")

    config_reasons, config_warnings = _tgwt_claim_config_reasons(bsc_config, mode)
    reasons.extend(config_reasons)
    warnings.extend(config_warnings)

    if any(r in reasons for r in {
        "claim_config_disabled",
        "missing_token_contract_address",
        "missing_claim_contract_address",
        "invalid_chain_id",
        "token_and_claim_contract_must_differ",
    }):
        status = TGWT_CLAIM_SETTLEMENT_CONFIG_REQUIRED
    elif reasons:
        status = TGWT_CLAIM_SETTLEMENT_REJECTED
    elif warnings:
        status = TGWT_CLAIM_SETTLEMENT_HOLD
    else:
        status = TGWT_CLAIM_SETTLEMENT_READY_FOR_ADMIN_REVIEW
        reasons.append("precheck_passed_ready_for_admin_review")

    return {
        "version": TGWT_CLAIM_SETTLEMENT_PRECHECK_VERSION,
        "status": status,
        "reasons": list(reasons),
        "warnings": list(warnings),
        "mode": mode,
        "userId": user_id,
        "claimId": claim_id,
        "bscAddress": bsc_address,
        "amountTGWT": amount,
        "claimableTGWT": claimable,
        "adminStatus": admin_status,
        "readyForAdminReview": status == TGWT_CLAIM_SETTLEMENT_READY_FOR_ADMIN_REVIEW,
        "readyForAutomaticTransfer": False,
        "createsBscTransaction": False,
        "backendPaysGas": False,
        "backendSendsToken": False,
        "requiresUserSignatureLater": True,
        "requiresUserBnbGasLater": True,
        "safeBoundary": dict(TGWT_CLAIM_SETTLEMENT_SAFE_BOUNDARY),
    }


def _tgwt_claim_config_reasons(bsc_config: Dict[str, Any], mode: str) -> tuple[List[str], List[str]]:
    reasons: List[str] = []
    warnings: List[str] = []

    enabled = bool(bsc_config.get("claimEnabled", False))
    if enabled is not True:
        reasons.append("claim_config_disabled")
        return reasons, warnings

    chain_id = _tgwt_int(bsc_config.get("chainId", 0))
    expected_chain = TGWT_BSC_MAINNET_CHAIN_ID if mode == "MAINNET" else TGWT_BSC_TESTNET_CHAIN_ID

    token_contract = str(bsc_config.get("tokenContractAddress", "")).strip()
    claim_contract = str(bsc_config.get("claimContractAddress", "")).strip()

    if chain_id != expected_chain:
        reasons.append("invalid_chain_id")

    if not _is_valid_bsc_address(token_contract):
        reasons.append("missing_token_contract_address")

    if not _is_valid_bsc_address(claim_contract):
        reasons.append("missing_claim_contract_address")

    if token_contract and claim_contract and token_contract.lower() == claim_contract.lower():
        reasons.append("token_and_claim_contract_must_differ")

    if bool(bsc_config.get("backendPaysGas", False)) is True:
        reasons.append("backend_pays_gas_not_allowed")

    if bool(bsc_config.get("backendSendsToken", False)) is True:
        reasons.append("backend_sends_token_not_allowed")

    if bool(bsc_config.get("createsBscTransaction", False)) is True:
        reasons.append("backend_transaction_creation_not_allowed")

    if bool(bsc_config.get("requiresUserSignature", True)) is not True:
        warnings.append("user_signature_requirement_should_be_enabled")

    if bool(bsc_config.get("userPaysGas", True)) is not True:
        warnings.append("user_pays_gas_requirement_should_be_enabled")

    return reasons, warnings


def _is_valid_bsc_address(value: str) -> bool:
    text = str(value or "").strip()
    return bool(re.fullmatch(r"0x[a-fA-F0-9]{40}", text))


def _tgwt_int(value: Any) -> int:
    try:
        return int(value)
    except Exception:
        return 0
