# SOCIAL_CLAIM_NONCE_LIMITS_V1
#
# Social Claim nonce and per-user/per-platform reward limit policy.
#
# Purpose:
# - Block duplicate nonce replay.
# - Block duplicate claim id replay.
# - Enforce one TGWT reward per user per platform.
# - Keep X and website as separate platforms.
# - Require verified social proof from SOCIAL_CLAIM_VERIFICATION_V1 before reward eligibility.
#
# This module does NOT:
# - call X API
# - call website APIs
# - automate browser actions
# - collect passwords
# - store OAuth tokens
# - send TGWT on-chain
# - create BSC transactions
# - touch QC chain/mining/consensus/balances

from __future__ import annotations

from typing import Any, Dict, List

from social_claim_verification import (
    SOCIAL_CLAIM_VERIFICATION_VERSION,
    SOCIAL_CLAIM_MODE_DEV,
    SOCIAL_CLAIM_MODE_FINAL,
    SOCIAL_CLAIM_VERIFIED,
    SOCIAL_CLAIM_DEV_SIMULATION,
    SOCIAL_PLATFORM_WEBSITE,
    social_claim_evaluate,
    social_claim_normalize_platform,
)


SOCIAL_CLAIM_NONCE_LIMITS_VERSION = "SOCIAL_CLAIM_NONCE_LIMITS_V1"
SOCIAL_CLAIM_NONCE_LIMITS_STATUS = "NONCE_AND_PLATFORM_LIMIT_POLICY_ONLY"

SOCIAL_CLAIM_REWARD_AMOUNT_TGWT = 1

SOCIAL_CLAIM_NONCE_ALLOWED = "SOCIAL_CLAIM_NONCE_ALLOWED"
SOCIAL_CLAIM_NONCE_DEV_SIMULATION = "SOCIAL_CLAIM_NONCE_DEV_SIMULATION_NO_REWARD"
SOCIAL_CLAIM_NONCE_MISSING = "SOCIAL_CLAIM_NONCE_MISSING"
SOCIAL_CLAIM_NONCE_DUPLICATE = "SOCIAL_CLAIM_NONCE_DUPLICATE"
SOCIAL_CLAIM_CLAIM_ID_DUPLICATE = "SOCIAL_CLAIM_ID_DUPLICATE"
SOCIAL_CLAIM_PLATFORM_REWARD_ALREADY_GRANTED = "SOCIAL_CLAIM_PLATFORM_REWARD_ALREADY_GRANTED"
SOCIAL_CLAIM_VERIFICATION_REQUIRED = "SOCIAL_CLAIM_VERIFICATION_REQUIRED"

SOCIAL_CLAIM_NONCE_LIMITS_SAFE_BOUNDARY = {
    "policyOnly": True,
    "offChainOnly": True,
    "noExternalApiCall": True,
    "noBrowserAutomation": True,
    "noPasswordCollection": True,
    "noOAuthTokenStorage": True,
    "noPrivateKeyStorage": True,
    "nonceRequired": True,
    "duplicateNonceBlocked": True,
    "duplicateClaimIdBlocked": True,
    "perUserPerPlatformOneReward": True,
    "rewardAmountTGWT": SOCIAL_CLAIM_REWARD_AMOUNT_TGWT,
    "requiresVerifiedSocialClaim": True,
    "devSimulationNoReward": True,
    "websiteNonceAcceptedAsClaimNonce": True,
    "xAndWebsiteAreSeparatePlatforms": True,
    "changesQCBalances": False,
    "changesTGWTOnChainBalances": False,
    "createsBscTransaction": False,
    "sendsTGWT": False,
    "paysGas": False,
    "storesPrivateKey": False,
    "signsTransaction": False,
    "changesMempool": False,
    "changesChain": False,
    "changesConsensus": False,
    "changesTotalSupply": False,
    "startsMining": False,
    "stopsMining": False,
}


def social_claim_nonce_limits_policy() -> Dict[str, Any]:
    return {
        "version": SOCIAL_CLAIM_NONCE_LIMITS_VERSION,
        "status": SOCIAL_CLAIM_NONCE_LIMITS_STATUS,
        "dependsOn": [SOCIAL_CLAIM_VERIFICATION_VERSION],
        "rewardAmountTGWT": SOCIAL_CLAIM_REWARD_AMOUNT_TGWT,
        "rule": "ONE_TGWT_PER_USER_PER_PLATFORM_AFTER_VERIFICATION",
        "nonceReplayProtection": True,
        "claimIdReplayProtection": True,
        "safeBoundary": dict(SOCIAL_CLAIM_NONCE_LIMITS_SAFE_BOUNDARY),
    }


def social_claim_apply_nonce_limits(
    claim: Dict[str, Any],
    history: List[Dict[str, Any]] | None = None,
    mode: str = SOCIAL_CLAIM_MODE_DEV,
) -> Dict[str, Any]:
    claim = dict(claim or {})
    history = list(history or [])
    mode = str(mode or "").strip().upper()
    if mode != SOCIAL_CLAIM_MODE_FINAL:
        mode = SOCIAL_CLAIM_MODE_DEV

    evaluation = social_claim_evaluate(claim, mode)
    platform = social_claim_normalize_platform(str(claim.get("platform", "")))
    user_id = str(claim.get("userId", "")).strip()
    claim_id = str(claim.get("claimId", "")).strip()
    nonce = social_claim_extract_nonce(claim)

    if mode == SOCIAL_CLAIM_MODE_DEV:
        if str(evaluation.get("status", "")) == SOCIAL_CLAIM_DEV_SIMULATION:
            return _social_claim_nonce_result(
                SOCIAL_CLAIM_NONCE_DEV_SIMULATION,
                ["dev_simulation_no_reward"],
                claim,
                platform,
                nonce,
                evaluation,
            )
        return _social_claim_nonce_result(
            SOCIAL_CLAIM_VERIFICATION_REQUIRED,
            list(evaluation.get("reasons", [])) or ["dev_requires_simulation"],
            claim,
            platform,
            nonce,
            evaluation,
        )

    if str(evaluation.get("status", "")) != SOCIAL_CLAIM_VERIFIED:
        return _social_claim_nonce_result(
            SOCIAL_CLAIM_VERIFICATION_REQUIRED,
            list(evaluation.get("reasons", [])) or ["verified_social_claim_required"],
            claim,
            platform,
            nonce,
            evaluation,
        )

    if not nonce:
        return _social_claim_nonce_result(
            SOCIAL_CLAIM_NONCE_MISSING,
            ["missing_claim_nonce"],
            claim,
            platform,
            nonce,
            evaluation,
        )

    if claim_id and social_claim_claim_id_seen(claim_id, history):
        return _social_claim_nonce_result(
            SOCIAL_CLAIM_CLAIM_ID_DUPLICATE,
            ["duplicate_claim_id"],
            claim,
            platform,
            nonce,
            evaluation,
        )

    if social_claim_nonce_seen(nonce, history):
        return _social_claim_nonce_result(
            SOCIAL_CLAIM_NONCE_DUPLICATE,
            ["duplicate_nonce"],
            claim,
            platform,
            nonce,
            evaluation,
        )

    if social_claim_user_platform_reward_seen(user_id, platform, history):
        return _social_claim_nonce_result(
            SOCIAL_CLAIM_PLATFORM_REWARD_ALREADY_GRANTED,
            ["per_user_per_platform_reward_already_granted"],
            claim,
            platform,
            nonce,
            evaluation,
        )

    return _social_claim_nonce_result(
        SOCIAL_CLAIM_NONCE_ALLOWED,
        ["verified_unique_social_claim_eligible_for_1_tgwt"],
        claim,
        platform,
        nonce,
        evaluation,
    )


def social_claim_record_reward(
    history: List[Dict[str, Any]] | None,
    claim: Dict[str, Any],
    decision: Dict[str, Any],
    now_ts: int = 0,
) -> List[Dict[str, Any]]:
    current = list(history or [])
    claim = dict(claim or {})
    decision = dict(decision or {})

    if str(decision.get("status", "")) != SOCIAL_CLAIM_NONCE_ALLOWED:
        return current

    record = {
        "version": SOCIAL_CLAIM_NONCE_LIMITS_VERSION,
        "userId": str(decision.get("userId", "")).strip(),
        "platform": str(decision.get("platform", "")).strip().upper(),
        "claimId": str(decision.get("claimId", "")).strip(),
        "claimNonce": str(decision.get("claimNonce", "")).strip(),
        "timestamp": int(now_ts or 0),
        "rewardGranted": True,
        "rewardAmountTGWT": SOCIAL_CLAIM_REWARD_AMOUNT_TGWT,
        "rewardState": "PENDING_TGWT",
        "safeBoundary": dict(SOCIAL_CLAIM_NONCE_LIMITS_SAFE_BOUNDARY),
    }
    current.append(record)
    return current


def social_claim_extract_nonce(claim: Dict[str, Any]) -> str:
    claim = dict(claim or {})
    nonce = str(claim.get("claimNonce", "")).strip()
    if nonce:
        return nonce

    nonce = str(claim.get("nonce", "")).strip()
    if nonce:
        return nonce

    if social_claim_normalize_platform(str(claim.get("platform", ""))) == SOCIAL_PLATFORM_WEBSITE:
        nonce = str(claim.get("websiteNonce", "")).strip()
        if nonce:
            return nonce

    return ""


def social_claim_nonce_seen(nonce: str, history: List[Dict[str, Any]]) -> bool:
    nonce = str(nonce or "").strip()
    if not nonce:
        return False

    for item in history:
        if str(item.get("claimNonce", "")).strip() == nonce:
            return True
        if str(item.get("nonce", "")).strip() == nonce:
            return True
        if str(item.get("websiteNonce", "")).strip() == nonce:
            return True
    return False


def social_claim_claim_id_seen(claim_id: str, history: List[Dict[str, Any]]) -> bool:
    claim_id = str(claim_id or "").strip()
    if not claim_id:
        return False

    for item in history:
        if str(item.get("claimId", "")).strip() == claim_id:
            return True
    return False


def social_claim_user_platform_reward_seen(user_id: str, platform: str, history: List[Dict[str, Any]]) -> bool:
    user_id = str(user_id or "").strip()
    platform = social_claim_normalize_platform(platform)

    for item in history:
        if str(item.get("userId", "")).strip() != user_id:
            continue
        if social_claim_normalize_platform(str(item.get("platform", ""))) != platform:
            continue
        if bool(item.get("rewardGranted", False)) is True and int(item.get("rewardAmountTGWT", 0) or 0) >= SOCIAL_CLAIM_REWARD_AMOUNT_TGWT:
            return True
    return False


def _social_claim_nonce_result(
    status: str,
    reasons: List[str],
    claim: Dict[str, Any],
    platform: str,
    nonce: str,
    evaluation: Dict[str, Any],
) -> Dict[str, Any]:
    allowed = status == SOCIAL_CLAIM_NONCE_ALLOWED

    return {
        "version": SOCIAL_CLAIM_NONCE_LIMITS_VERSION,
        "status": status,
        "reasons": list(reasons or []),
        "userId": str(claim.get("userId", "")).strip(),
        "platform": platform,
        "claimId": str(claim.get("claimId", "")).strip(),
        "claimNonce": str(nonce or "").strip(),
        "rewardEligible": allowed,
        "rewardAmountTGWT": SOCIAL_CLAIM_REWARD_AMOUNT_TGWT if allowed else 0,
        "rewardState": "PENDING_TGWT" if allowed else "NO_REWARD",
        "verificationStatus": str(evaluation.get("status", "")).strip(),
        "nonceRequired": True,
        "duplicateNonceBlocked": True,
        "duplicateClaimIdBlocked": True,
        "perUserPerPlatformOneReward": True,
        "safeBoundary": dict(SOCIAL_CLAIM_NONCE_LIMITS_SAFE_BOUNDARY),
    }
