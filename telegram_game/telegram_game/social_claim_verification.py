# SOCIAL_CLAIM_VERIFICATION_V1
#
# Social Claim verification policy for Telegram Game.
#
# Purpose:
# - Verify X / website / social follow claims before TGWT eligibility.
# - A user-side "I followed" claim is never enough in FINAL mode.
# - Screenshots or URLs alone become pending/admin-review evidence, not final reward.
# - FINAL mode requires provider/server verification marker.
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


SOCIAL_CLAIM_VERIFICATION_VERSION = "SOCIAL_CLAIM_VERIFICATION_V1"
SOCIAL_CLAIM_VERIFICATION_STATUS = "VERIFICATION_POLICY_ONLY_NO_EXTERNAL_API_CALL"

SOCIAL_CLAIM_MODE_DEV = "DEV"
SOCIAL_CLAIM_MODE_FINAL = "FINAL"

SOCIAL_PLATFORM_X = "X"
SOCIAL_PLATFORM_WEBSITE = "WEBSITE"
SOCIAL_PLATFORM_TELEGRAM = "TELEGRAM"
SOCIAL_PLATFORM_YOUTUBE = "YOUTUBE"
SOCIAL_PLATFORM_INSTAGRAM = "INSTAGRAM"

SOCIAL_ACTION_FOLLOW_X_ACCOUNT = "FOLLOW_X_ACCOUNT"
SOCIAL_ACTION_WEBSITE_NONCE_VERIFY = "WEBSITE_NONCE_VERIFY"
SOCIAL_ACTION_JOIN_TELEGRAM = "JOIN_TELEGRAM"
SOCIAL_ACTION_SUBSCRIBE_YOUTUBE = "SUBSCRIBE_YOUTUBE"
SOCIAL_ACTION_FOLLOW_INSTAGRAM = "FOLLOW_INSTAGRAM"

SOCIAL_CLAIM_VERIFIED = "SOCIAL_CLAIM_VERIFIED"
SOCIAL_CLAIM_PENDING_PROVIDER_VERIFICATION = "SOCIAL_CLAIM_PENDING_PROVIDER_VERIFICATION"
SOCIAL_CLAIM_HOLD_ADMIN_REVIEW = "SOCIAL_CLAIM_HOLD_ADMIN_REVIEW"
SOCIAL_CLAIM_REJECTED = "SOCIAL_CLAIM_REJECTED"
SOCIAL_CLAIM_DEV_SIMULATION = "SOCIAL_CLAIM_DEV_SIMULATION_NO_REWARD"

SOCIAL_CLAIM_SAFE_BOUNDARY = {
    "policyOnly": True,
    "offChainOnly": True,
    "noExternalApiCall": True,
    "noBrowserAutomation": True,
    "noPasswordCollection": True,
    "noOAuthTokenStorage": True,
    "noPrivateKeyStorage": True,
    "productionRequiresProviderVerification": True,
    "productionRequiresServerVerification": True,
    "userClaimOnlyNotEnough": True,
    "screenshotOnlyNotEnough": True,
    "urlOnlyNotEnough": True,
    "adminReviewForUnverifiedEvidence": True,
    "rewardRequiresVerifiedProof": True,
    "duplicateNonceHandledByNextLayer": True,
    "perPlatformOneRewardHandledByNextLayer": True,
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


def social_claim_policy(platform: str = "") -> Dict[str, Any]:
    normalized = social_claim_normalize_platform(platform)

    requirements = {
        SOCIAL_PLATFORM_X: {
            "action": SOCIAL_ACTION_FOLLOW_X_ACCOUNT,
            "requiresUserHandle": True,
            "requiresTargetAccount": True,
            "requiresProviderVerified": True,
            "providerAdapter": "X_API_OR_APPROVED_VERIFICATION_ADAPTER_LATER",
        },
        SOCIAL_PLATFORM_WEBSITE: {
            "action": SOCIAL_ACTION_WEBSITE_NONCE_VERIFY,
            "requiresWebsiteHost": True,
            "requiresWebsiteNonce": True,
            "requiresServerNonceVerified": True,
            "providerAdapter": "WEBSITE_SERVER_NONCE_VERIFIER_LATER",
        },
        SOCIAL_PLATFORM_TELEGRAM: {
            "action": SOCIAL_ACTION_JOIN_TELEGRAM,
            "requiresUserId": True,
            "requiresProviderVerified": True,
            "providerAdapter": "TELEGRAM_BOT_MEMBERSHIP_VERIFIER_LATER",
        },
        SOCIAL_PLATFORM_YOUTUBE: {
            "action": SOCIAL_ACTION_SUBSCRIBE_YOUTUBE,
            "requiresChannelId": True,
            "requiresProviderVerified": True,
            "providerAdapter": "YOUTUBE_API_OR_APPROVED_VERIFICATION_ADAPTER_LATER",
        },
        SOCIAL_PLATFORM_INSTAGRAM: {
            "action": SOCIAL_ACTION_FOLLOW_INSTAGRAM,
            "requiresProfile": True,
            "requiresProviderVerified": True,
            "providerAdapter": "APPROVED_VERIFICATION_ADAPTER_LATER",
        },
    }

    return {
        "version": SOCIAL_CLAIM_VERIFICATION_VERSION,
        "status": SOCIAL_CLAIM_VERIFICATION_STATUS,
        "platform": normalized,
        "supported": normalized in requirements,
        "requirements": dict(requirements.get(normalized, {})),
        "rewardModel": "TGWT_PENDING_ONLY_AFTER_VERIFIED_SOCIAL_PROOF",
        "safeBoundary": dict(SOCIAL_CLAIM_SAFE_BOUNDARY),
    }


def social_claim_normalize_platform(platform: str) -> str:
    p = str(platform or "").strip().upper()
    if p in {"TWITTER", "X_COM", "X.COM"}:
        return SOCIAL_PLATFORM_X
    if p in {"WEB", "SITE", "WEBSITE"}:
        return SOCIAL_PLATFORM_WEBSITE
    if p in {"TG", "TELEGRAM"}:
        return SOCIAL_PLATFORM_TELEGRAM
    if p in {"YT", "YOUTUBE"}:
        return SOCIAL_PLATFORM_YOUTUBE
    if p in {"IG", "INSTAGRAM"}:
        return SOCIAL_PLATFORM_INSTAGRAM
    return p


def social_claim_evaluate(
    claim: Dict[str, Any],
    mode: str = SOCIAL_CLAIM_MODE_DEV,
) -> Dict[str, Any]:
    claim = dict(claim or {})
    mode = str(mode or "").strip().upper()
    if mode != SOCIAL_CLAIM_MODE_FINAL:
        mode = SOCIAL_CLAIM_MODE_DEV

    platform = social_claim_normalize_platform(str(claim.get("platform", "")))
    policy = social_claim_policy(platform)
    reasons: List[str] = []

    if not policy["supported"]:
        return _social_claim_result(
            SOCIAL_CLAIM_REJECTED,
            ["unsupported_platform"],
            claim,
            mode,
            policy,
        )

    user_id = str(claim.get("userId", "")).strip()
    action = str(claim.get("action", "")).strip().upper()
    expected_action = str(policy["requirements"].get("action", "")).strip().upper()

    if not user_id:
        reasons.append("missing_user_id")

    if action != expected_action:
        reasons.append("wrong_social_action")

    if mode == SOCIAL_CLAIM_MODE_DEV:
        if bool(claim.get("devSimulation", False)) is True and not reasons:
            return _social_claim_result(
                SOCIAL_CLAIM_DEV_SIMULATION,
                ["dev_simulation_no_reward"],
                claim,
                mode,
                policy,
            )
        return _social_claim_result(
            SOCIAL_CLAIM_REJECTED,
            reasons or ["dev_requires_simulation"],
            claim,
            mode,
            policy,
        )

    platform_reasons = _social_claim_platform_reasons(platform, claim)
    reasons.extend(platform_reasons)

    if reasons:
        evidence_url = str(claim.get("evidenceUrl", "")).strip()
        screenshot_ref = str(claim.get("screenshotRef", "")).strip()
        if evidence_url or screenshot_ref:
            return _social_claim_result(
                SOCIAL_CLAIM_PENDING_PROVIDER_VERIFICATION,
                reasons + ["evidence_requires_provider_or_admin_review"],
                claim,
                mode,
                policy,
            )

        return _social_claim_result(
            SOCIAL_CLAIM_REJECTED,
            reasons,
            claim,
            mode,
            policy,
        )

    return _social_claim_result(
        SOCIAL_CLAIM_VERIFIED,
        ["verified_social_claim_eligible_for_pending_tgwt"],
        claim,
        mode,
        policy,
    )


def _social_claim_platform_reasons(platform: str, claim: Dict[str, Any]) -> List[str]:
    reasons: List[str] = []

    if platform == SOCIAL_PLATFORM_X:
        if not str(claim.get("xHandle", "")).strip():
            reasons.append("missing_x_handle")
        if not str(claim.get("targetAccount", "")).strip():
            reasons.append("missing_target_account")
        if bool(claim.get("providerVerified", False)) is not True:
            reasons.append("missing_provider_verification")
        if bool(claim.get("followVerified", False)) is not True:
            reasons.append("missing_follow_verification")

    elif platform == SOCIAL_PLATFORM_WEBSITE:
        if not str(claim.get("websiteHost", "")).strip():
            reasons.append("missing_website_host")
        if not str(claim.get("websiteNonce", "")).strip():
            reasons.append("missing_website_nonce")
        if bool(claim.get("serverNonceVerified", False)) is not True:
            reasons.append("missing_server_nonce_verification")

    elif platform == SOCIAL_PLATFORM_TELEGRAM:
        if not str(claim.get("telegramUserId", "")).strip():
            reasons.append("missing_telegram_user_id")
        if bool(claim.get("providerVerified", False)) is not True:
            reasons.append("missing_provider_verification")

    elif platform == SOCIAL_PLATFORM_YOUTUBE:
        if not str(claim.get("channelId", "")).strip():
            reasons.append("missing_channel_id")
        if bool(claim.get("providerVerified", False)) is not True:
            reasons.append("missing_provider_verification")

    elif platform == SOCIAL_PLATFORM_INSTAGRAM:
        if not str(claim.get("profile", "")).strip():
            reasons.append("missing_profile")
        if bool(claim.get("providerVerified", False)) is not True:
            reasons.append("missing_provider_verification")

    return reasons


def _social_claim_result(
    status: str,
    reasons: List[str],
    claim: Dict[str, Any],
    mode: str,
    policy: Dict[str, Any],
) -> Dict[str, Any]:
    reward_eligible = status == SOCIAL_CLAIM_VERIFIED

    return {
        "version": SOCIAL_CLAIM_VERIFICATION_VERSION,
        "status": status,
        "reasons": list(reasons or []),
        "mode": mode,
        "platform": policy["platform"],
        "userId": str(claim.get("userId", "")).strip(),
        "claimId": str(claim.get("claimId", "")).strip(),
        "action": str(claim.get("action", "")).strip().upper(),
        "rewardEligible": reward_eligible,
        "rewardState": "PENDING_TGWT_ALLOWED_AFTER_VERIFICATION" if reward_eligible else "NO_REWARD",
        "requiresProviderVerificationFinal": True,
        "requiresServerVerificationFinal": True,
        "userClaimOnlyNotEnough": True,
        "screenshotOnlyNotEnough": True,
        "urlOnlyNotEnough": True,
        "duplicateNonceHandledByNextLayer": True,
        "perPlatformOneRewardHandledByNextLayer": True,
        "safeBoundary": dict(SOCIAL_CLAIM_SAFE_BOUNDARY),
    }
