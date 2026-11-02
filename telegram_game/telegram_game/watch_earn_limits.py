# WATCH_EARN_LIMITS_V1
#
# Watch & Earn daily limit / cooldown policy for Telegram Game.
#
# Purpose:
# - Protect Google Rewarded Ads revenue quality.
# - Protect the game from repeated reward abuse.
# - Keep user rewards tied to verified rewarded completion.
# - Keep revenue policy separate from TGWT reward ledger.
#
# This module does NOT:
# - show ads
# - call Google APIs
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


WATCH_EARN_LIMITS_VERSION = "WATCH_EARN_LIMITS_V1"
WATCH_EARN_LIMITS_STATUS = "LIMITS_ONLY_NO_AD_SDK_CALL"

WATCH_EARN_DEFAULT_DAILY_LIMIT_DEV = 3
WATCH_EARN_DEFAULT_DAILY_LIMIT_FINAL = 10
WATCH_EARN_DEFAULT_COOLDOWN_SECONDS_DEV = 10
WATCH_EARN_DEFAULT_COOLDOWN_SECONDS_FINAL = 120

WATCH_EARN_DECISION_ALLOWED = "WATCH_EARN_ALLOWED"
WATCH_EARN_DECISION_COOLDOWN = "WATCH_EARN_COOLDOWN_ACTIVE"
WATCH_EARN_DECISION_DAILY_LIMIT = "WATCH_EARN_DAILY_LIMIT_REACHED"
WATCH_EARN_DECISION_DUPLICATE_SESSION = "WATCH_EARN_DUPLICATE_SESSION"
WATCH_EARN_DECISION_DUPLICATE_TRANSACTION = "WATCH_EARN_DUPLICATE_TRANSACTION"
WATCH_EARN_DECISION_REWARD_EVENT_REJECTED = "WATCH_EARN_REWARD_EVENT_REJECTED"

WATCH_EARN_LIMITS_SAFE_BOUNDARY = {
    "limitsOnly": True,
    "policyOnly": True,
    "noAdSdkCall": True,
    "noGoogleApiCall": True,
    "noFakeImpressions": True,
    "noClickIncentive": True,
    "noBotTraffic": True,
    "noInvalidTraffic": True,
    "dailyLimitRequired": True,
    "cooldownRequired": True,
    "duplicateSessionBlocked": True,
    "duplicateTransactionBlocked": True,
    "rewardRequiresVerifiedCompletion": True,
    "serverSideVerificationRequiredFinal": True,
    "revenueLogSeparateFromRewardLog": True,
    "clickIncentive": False,
    "fakeTrafficAllowed": False,
    "botTrafficAllowed": False,
    "invalidTrafficAllowed": False,
    "autoClickAllowed": False,
    "userMustClickAd": False,
    "rewardForClick": False,
    "rewardForLoadOnly": False,
    "rewardForOpenOnly": False,
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


def watch_earn_limits_policy(mode: str = WATCH_EARN_MODE_DEV) -> Dict[str, Any]:
    mode = str(mode or "").strip().upper()

    if mode == WATCH_EARN_MODE_FINAL:
        daily_limit = WATCH_EARN_DEFAULT_DAILY_LIMIT_FINAL
        cooldown_seconds = WATCH_EARN_DEFAULT_COOLDOWN_SECONDS_FINAL
    else:
        mode = WATCH_EARN_MODE_DEV
        daily_limit = WATCH_EARN_DEFAULT_DAILY_LIMIT_DEV
        cooldown_seconds = WATCH_EARN_DEFAULT_COOLDOWN_SECONDS_DEV

    return {
        "version": WATCH_EARN_LIMITS_VERSION,
        "status": WATCH_EARN_LIMITS_STATUS,
        "dependsOn": [WATCH_EARN_REVENUE_POLICY_VERSION],
        "mode": mode,
        "dailyLimit": daily_limit,
        "cooldownSeconds": cooldown_seconds,
        "decisionOrder": [
            "duplicate_session",
            "duplicate_transaction",
            "daily_limit",
            "cooldown",
            "reward_event_validation",
        ],
        "rewardModel": "VERIFIED_REWARDED_COMPLETION_ONLY",
        "revenuePurpose": "OPS_COST_RECOVERY",
        "safeBoundary": dict(WATCH_EARN_LIMITS_SAFE_BOUNDARY),
    }


def watch_earn_new_session(
    user_id: str,
    session_id: str,
    now_ts: int,
    mode: str = WATCH_EARN_MODE_DEV,
) -> Dict[str, Any]:
    policy = watch_earn_limits_policy(mode)
    return {
        "version": WATCH_EARN_LIMITS_VERSION,
        "userId": str(user_id or "").strip(),
        "sessionId": str(session_id or "").strip(),
        "mode": policy["mode"],
        "startedAt": int(now_ts or 0),
        "status": "WATCH_SESSION_STARTED",
        "safeBoundary": dict(WATCH_EARN_LIMITS_SAFE_BOUNDARY),
    }


def watch_earn_limit_decision(
    user_id: str,
    now_ts: int,
    history: List[Dict[str, Any]] | None = None,
    event: Dict[str, Any] | None = None,
    mode: str = WATCH_EARN_MODE_DEV,
    daily_limit_override: int | None = None,
    cooldown_seconds_override: int | None = None,
) -> Dict[str, Any]:
    policy = watch_earn_limits_policy(mode)
    history = list(history or [])
    event = dict(event or {})
    user_id = str(user_id or "").strip()
    now_ts = int(now_ts or 0)

    daily_limit = int(daily_limit_override if daily_limit_override is not None else policy["dailyLimit"])
    cooldown_seconds = int(cooldown_seconds_override if cooldown_seconds_override is not None else policy["cooldownSeconds"])

    session_id = str(event.get("sessionId", "")).strip()
    transaction_id = str(event.get("transactionId", "")).strip()

    user_events_today = _watch_earn_user_events_today(user_id, now_ts, history)
    completed_today = len([x for x in user_events_today if bool(x.get("rewardGranted", False)) is True])

    last_reward_ts = _watch_earn_last_reward_ts(user_id, history)
    next_allowed_ts = int(last_reward_ts + cooldown_seconds) if last_reward_ts > 0 else now_ts
    remaining_cooldown = max(0, next_allowed_ts - now_ts)

    duplicate_session = bool(session_id) and any(
        str(x.get("sessionId", "")).strip() == session_id for x in history
    )

    duplicate_transaction = bool(transaction_id) and any(
        str(x.get("transactionId", "")).strip() == transaction_id for x in history
    )

    if duplicate_session:
        return _watch_earn_decision(
            WATCH_EARN_DECISION_DUPLICATE_SESSION,
            ["duplicate_session"],
            user_id,
            now_ts,
            completed_today,
            daily_limit,
            remaining_cooldown,
            next_allowed_ts,
            policy,
        )

    if duplicate_transaction:
        return _watch_earn_decision(
            WATCH_EARN_DECISION_DUPLICATE_TRANSACTION,
            ["duplicate_transaction"],
            user_id,
            now_ts,
            completed_today,
            daily_limit,
            remaining_cooldown,
            next_allowed_ts,
            policy,
        )

    if completed_today >= daily_limit:
        return _watch_earn_decision(
            WATCH_EARN_DECISION_DAILY_LIMIT,
            ["daily_limit_reached"],
            user_id,
            now_ts,
            completed_today,
            daily_limit,
            remaining_cooldown,
            next_allowed_ts,
            policy,
        )

    if remaining_cooldown > 0:
        return _watch_earn_decision(
            WATCH_EARN_DECISION_COOLDOWN,
            ["cooldown_active"],
            user_id,
            now_ts,
            completed_today,
            daily_limit,
            remaining_cooldown,
            next_allowed_ts,
            policy,
        )

    validation = watch_earn_validate_reward_event(event, policy["mode"])
    if str(validation.get("status", "")) not in {
        "FINAL_REWARD_EVENT_VERIFIED",
        "DEV_SIMULATION_ACCEPTED_NO_REVENUE",
    }:
        return _watch_earn_decision(
            WATCH_EARN_DECISION_REWARD_EVENT_REJECTED,
            list(validation.get("reasons", [])),
            user_id,
            now_ts,
            completed_today,
            daily_limit,
            remaining_cooldown,
            next_allowed_ts,
            policy,
        )

    return _watch_earn_decision(
        WATCH_EARN_DECISION_ALLOWED,
        ["eligible_within_daily_limit_and_cooldown"],
        user_id,
        now_ts,
        completed_today,
        daily_limit,
        remaining_cooldown,
        next_allowed_ts,
        policy,
    )


def watch_earn_record_reward(history: List[Dict[str, Any]] | None, event: Dict[str, Any], decision: Dict[str, Any]) -> List[Dict[str, Any]]:
    current = list(history or [])
    event = dict(event or {})
    decision = dict(decision or {})

    if str(decision.get("status", "")) != WATCH_EARN_DECISION_ALLOWED:
        return current

    record = {
        "version": WATCH_EARN_LIMITS_VERSION,
        "userId": str(decision.get("userId", "")).strip(),
        "sessionId": str(event.get("sessionId", "")).strip(),
        "transactionId": str(event.get("transactionId", "")).strip(),
        "timestamp": int(decision.get("nowTs", 0) or 0),
        "mode": str(decision.get("mode", WATCH_EARN_MODE_DEV)).strip(),
        "rewardGranted": True,
        "rewardState": "PENDING_OR_VERIFIED_BY_POLICY",
        "revenueLogSeparateFromRewardLog": True,
        "safeBoundary": dict(WATCH_EARN_LIMITS_SAFE_BOUNDARY),
    }
    current.append(record)
    return current


def _watch_earn_decision(
    status: str,
    reasons: List[str],
    user_id: str,
    now_ts: int,
    completed_today: int,
    daily_limit: int,
    remaining_cooldown: int,
    next_allowed_ts: int,
    policy: Dict[str, Any],
) -> Dict[str, Any]:
    return {
        "version": WATCH_EARN_LIMITS_VERSION,
        "status": status,
        "reasons": list(reasons or []),
        "userId": user_id,
        "nowTs": int(now_ts or 0),
        "mode": policy["mode"],
        "completedToday": int(completed_today or 0),
        "dailyLimit": int(daily_limit or 0),
        "remainingToday": max(0, int(daily_limit or 0) - int(completed_today or 0)),
        "cooldownSeconds": int(policy["cooldownSeconds"]),
        "remainingCooldownSeconds": int(remaining_cooldown or 0),
        "nextAllowedTs": int(next_allowed_ts or 0),
        "rewardRequiresVerifiedCompletion": True,
        "serverSideVerificationRequiredFinal": True,
        "safeBoundary": dict(WATCH_EARN_LIMITS_SAFE_BOUNDARY),
    }


def _watch_earn_user_events_today(user_id: str, now_ts: int, history: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    day_start = now_ts - (now_ts % 86400)
    out = []
    for item in history:
        if str(item.get("userId", "")).strip() != user_id:
            continue
        ts = int(item.get("timestamp", 0) or 0)
        if ts >= day_start and ts <= now_ts:
            out.append(item)
    return out


def _watch_earn_last_reward_ts(user_id: str, history: List[Dict[str, Any]]) -> int:
    last = 0
    for item in history:
        if str(item.get("userId", "")).strip() != user_id:
            continue
        if bool(item.get("rewardGranted", False)) is not True:
            continue
        ts = int(item.get("timestamp", 0) or 0)
        if ts > last:
            last = ts
    return last
