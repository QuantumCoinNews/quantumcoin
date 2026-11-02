# WATCH_EARN_VERIFIED_COMPLETION_V1
#
# Verified completion requirement for Telegram Game Watch & Earn.
#
# Purpose:
# - FINAL rewards require real Google Rewarded Ads completion.
# - FINAL rewards require server-side verification.
# - FINAL rewards require a server callback and unique transaction id.
# - A client-only "I watched" signal is never enough in FINAL mode.
#
# This module does NOT:
# - show ads
# - call Google APIs
# - verify Google cryptographic signatures directly
# - create fake impressions
# - incentivize clicks
# - create bot traffic
# - send TGWT on-chain
# - create BSC transactions
# - touch QC chain/mining/consensus/balances

from __future__ import annotations

from typing import Any, Dict, List

from watch_earn_revenue_policy import (
    WATCH_EARN_REVENUE_POLICY_VERSION,
    WATCH_EARN_MODE_DEV,
    WATCH_EARN_MODE_FINAL,
    watch_earn_validate_reward_event,
)
from watch_earn_limits import (
    WATCH_EARN_LIMITS_VERSION,
    WATCH_EARN_DECISION_ALLOWED,
    watch_earn_limit_decision,
)


WATCH_EARN_VERIFIED_COMPLETION_VERSION = "WATCH_EARN_VERIFIED_COMPLETION_V1"
WATCH_EARN_VERIFIED_COMPLETION_STATUS = "VERIFIED_COMPLETION_REQUIRED_NO_AD_SDK_CALL"

WATCH_EARN_VERIFIED_ACCEPTED = "WATCH_EARN_VERIFIED_COMPLETION_ACCEPTED"
WATCH_EARN_VERIFIED_DEV_SIMULATION = "WATCH_EARN_DEV_SIMULATION_ACCEPTED_NO_REVENUE"
WATCH_EARN_VERIFIED_REJECTED = "WATCH_EARN_VERIFIED_COMPLETION_REJECTED"
WATCH_EARN_VERIFIED_DUPLICATE_TRANSACTION = "WATCH_EARN_VERIFIED_DUPLICATE_TRANSACTION"
WATCH_EARN_VERIFIED_LIMIT_REJECTED = "WATCH_EARN_VERIFIED_LIMIT_REJECTED"

WATCH_EARN_VERIFIED_COMPLETION_SAFE_BOUNDARY = {
    "verifiedCompletionOnly": True,
    "policyOnly": True,
    "noAdSdkCall": True,
    "noGoogleApiCall": True,
    "noDirectSignatureCrypto": True,
    "requiresRewardedCompletionFinal": True,
    "requiresServerSideVerificationFinal": True,
    "requiresSsvSignatureVerifiedFinal": True,
    "requiresServerCallbackFinal": True,
    "requiresUniqueTransactionId": True,
    "clientOnlyCompletionAcceptedFinal": False,
    "rewardForAdLoaded": False,
    "rewardForAdOpened": False,
    "rewardForFrontendOnlySignal": False,
    "rewardForClick": False,
    "noFakeImpressions": True,
    "noClickIncentive": True,
    "noBotTraffic": True,
    "noInvalidTraffic": True,
    "revenueLogSeparateFromRewardLog": True,
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


def watch_earn_verified_completion_policy(mode: str = WATCH_EARN_MODE_DEV) -> Dict[str, Any]:
    mode = str(mode or "").strip().upper()
    if mode != WATCH_EARN_MODE_FINAL:
        mode = WATCH_EARN_MODE_DEV

    return {
        "version": WATCH_EARN_VERIFIED_COMPLETION_VERSION,
        "status": WATCH_EARN_VERIFIED_COMPLETION_STATUS,
        "dependsOn": [
            WATCH_EARN_REVENUE_POLICY_VERSION,
            WATCH_EARN_LIMITS_VERSION,
        ],
        "mode": mode,
        "finalRequires": {
            "realGoogleRewardedAd": True,
            "rewardedCompletion": True,
            "serverSideVerification": True,
            "ssvSignatureVerified": True,
            "serverCallback": True,
            "uniqueTransactionId": True,
            "adUnitId": True,
            "userId": True,
            "customDataOrNonce": True,
        },
        "devAllowsSimulationOnly": True,
        "revenuePurpose": "OPS_COST_RECOVERY",
        "safeBoundary": dict(WATCH_EARN_VERIFIED_COMPLETION_SAFE_BOUNDARY),
    }


def watch_earn_build_verified_event(
    user_id: str,
    session_id: str,
    transaction_id: str,
    ad_unit_id: str,
    custom_data: str,
    now_ts: int,
    mode: str = WATCH_EARN_MODE_FINAL,
    rewarded_completion: bool = True,
    ssv_verified: bool = True,
    ssv_signature_verified: bool = True,
    server_callback_received: bool = True,
) -> Dict[str, Any]:
    mode = str(mode or "").strip().upper()
    if mode == WATCH_EARN_MODE_FINAL:
        return {
            "userId": str(user_id or "").strip(),
            "sessionId": str(session_id or "").strip(),
            "transactionId": str(transaction_id or "").strip(),
            "adUnitId": str(ad_unit_id or "").strip(),
            "customData": str(custom_data or "").strip(),
            "timestamp": int(now_ts or 0),
            "usesRealAd": True,
            "rewardedCompletion": bool(rewarded_completion),
            "ssvVerified": bool(ssv_verified),
            "ssvSignatureVerified": bool(ssv_signature_verified),
            "serverCallbackReceived": bool(server_callback_received),
            "clientOnlyCompletion": False,
            "clickIncentivized": False,
            "botTraffic": False,
            "invalidTraffic": False,
            "fakeImpression": False,
        }

    return {
        "userId": str(user_id or "").strip(),
        "sessionId": str(session_id or "").strip(),
        "transactionId": str(transaction_id or "").strip(),
        "timestamp": int(now_ts or 0),
        "devSimulation": True,
        "usesRealAd": False,
        "clientOnlyCompletion": False,
    }


def watch_earn_verify_completion(
    event: Dict[str, Any],
    history: List[Dict[str, Any]] | None = None,
    mode: str = WATCH_EARN_MODE_DEV,
    now_ts: int | None = None,
) -> Dict[str, Any]:
    policy = watch_earn_verified_completion_policy(mode)
    event = dict(event or {})
    history = list(history or [])

    user_id = str(event.get("userId", "")).strip()
    tx_id = str(event.get("transactionId", "")).strip()
    current_ts = int(now_ts if now_ts is not None else event.get("timestamp", 0) or 0)

    reasons: List[str] = []

    duplicate_tx = bool(tx_id) and any(
        str(x.get("transactionId", "")).strip() == tx_id for x in history
    )
    if duplicate_tx:
        return _watch_earn_verified_result(
            WATCH_EARN_VERIFIED_DUPLICATE_TRANSACTION,
            ["duplicate_transaction_id"],
            policy,
            event,
        )

    base_validation = watch_earn_validate_reward_event(event, policy["mode"])
    if str(base_validation.get("status", "")) not in {
        "FINAL_REWARD_EVENT_VERIFIED",
        "DEV_SIMULATION_ACCEPTED_NO_REVENUE",
    }:
        reasons.extend(list(base_validation.get("reasons", [])))

    if policy["mode"] == WATCH_EARN_MODE_DEV:
        if bool(event.get("devSimulation", False)) is True and bool(event.get("usesRealAd", False)) is False and not reasons:
            return _watch_earn_verified_result(
                WATCH_EARN_VERIFIED_DEV_SIMULATION,
                ["dev_simulation_no_revenue"],
                policy,
                event,
            )
        return _watch_earn_verified_result(
            WATCH_EARN_VERIFIED_REJECTED,
            reasons or ["dev_requires_simulation_only"],
            policy,
            event,
        )

    final_checks = {
        "missing_real_google_rewarded_ad": bool(event.get("usesRealAd", False)) is not True,
        "missing_rewarded_completion": bool(event.get("rewardedCompletion", False)) is not True,
        "missing_server_side_verification": bool(event.get("ssvVerified", False)) is not True,
        "missing_ssv_signature_verification": bool(event.get("ssvSignatureVerified", False)) is not True,
        "missing_server_callback": bool(event.get("serverCallbackReceived", False)) is not True,
        "client_only_completion_not_accepted": bool(event.get("clientOnlyCompletion", False)) is True,
        "missing_transaction_id": not tx_id,
        "missing_user_id": not user_id,
        "missing_ad_unit_id": not str(event.get("adUnitId", "")).strip(),
        "missing_custom_data_or_nonce": not str(event.get("customData", "")).strip(),
    }

    for reason, active in final_checks.items():
        if active and reason not in reasons:
            reasons.append(reason)

    if reasons:
        return _watch_earn_verified_result(
            WATCH_EARN_VERIFIED_REJECTED,
            reasons,
            policy,
            event,
        )

    limit_decision = watch_earn_limit_decision(
        user_id=user_id,
        now_ts=current_ts,
        history=history,
        event=event,
        mode=WATCH_EARN_MODE_FINAL,
    )

    if str(limit_decision.get("status", "")) != WATCH_EARN_DECISION_ALLOWED:
        return _watch_earn_verified_result(
            WATCH_EARN_VERIFIED_LIMIT_REJECTED,
            list(limit_decision.get("reasons", [])),
            policy,
            event,
            limit_decision,
        )

    return _watch_earn_verified_result(
        WATCH_EARN_VERIFIED_ACCEPTED,
        ["verified_completion_eligible_for_reward"],
        policy,
        event,
        limit_decision,
    )


def _watch_earn_verified_result(
    status: str,
    reasons: List[str],
    policy: Dict[str, Any],
    event: Dict[str, Any],
    limit_decision: Dict[str, Any] | None = None,
) -> Dict[str, Any]:
    return {
        "version": WATCH_EARN_VERIFIED_COMPLETION_VERSION,
        "status": status,
        "reasons": list(reasons or []),
        "mode": policy["mode"],
        "userId": str(event.get("userId", "")).strip(),
        "sessionId": str(event.get("sessionId", "")).strip(),
        "transactionId": str(event.get("transactionId", "")).strip(),
        "adUnitId": str(event.get("adUnitId", "")).strip(),
        "rewardState": "PENDING_OR_VERIFIED_BY_POLICY" if status in {
            WATCH_EARN_VERIFIED_ACCEPTED,
            WATCH_EARN_VERIFIED_DEV_SIMULATION,
        } else "NO_REWARD",
        "requiresRewardedCompletionFinal": True,
        "requiresServerSideVerificationFinal": True,
        "requiresSsvSignatureVerifiedFinal": True,
        "requiresServerCallbackFinal": True,
        "clientOnlyCompletionAcceptedFinal": False,
        "limitDecision": dict(limit_decision or {}),
        "safeBoundary": dict(WATCH_EARN_VERIFIED_COMPLETION_SAFE_BOUNDARY),
    }
