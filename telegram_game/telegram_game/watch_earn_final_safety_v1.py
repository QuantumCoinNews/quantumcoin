# WATCH_EARN_TESTNET_FINAL_LOCK_V1
# Final/mainnet safety boundary for Watch & Earn.
#
# This module does NOT call Google APIs, does NOT call an ad SDK,
# does NOT create fake ad impressions, and does NOT pay rewards by itself.
# It only validates whether Watch & Earn policy is safe.

import os
from typing import Any, Dict, Tuple, List

WATCH_EARN_FINAL_LOCK_VERSION = "WATCH_EARN_TESTNET_FINAL_LOCK_V1"

WATCH_MODE_DEV = "dev"
WATCH_MODE_TESTNET = "testnet"
WATCH_MODE_FINAL = "final"

WATCH_PROVIDER_NONE = "none"
WATCH_PROVIDER_GOOGLE_REWARDED_ADS = "google_rewarded_ads"
WATCH_PROVIDER_ADMOB_REWARDED = "admob_rewarded"

WATCH_FINAL_APPROVAL = "I_APPROVE_WATCH_EARN_FINAL_VERIFIED_AD_REWARDS"

WATCH_SAFE_BOUNDARY_V1 = {
    "rewardAsset": "TGWT",
    "rewardLedgerState": "pending",
    "revenueLogSeparateFromRewardLog": True,
    "requiresVerifiedRewardedCompletion": True,
    "requiresServerSideVerificationInFinal": True,
    "allowsDevSimulationOnlyOutsideFinal": True,
    "noRewardForAdLoadOnly": True,
    "noRewardForAdOpenOnly": True,
    "noRewardForClickOnly": True,
    "noFakeImpressions": True,
    "noBotTraffic": True,
    "doesNotCreateBscTransaction": True,
    "doesNotPayBscGas": True,
    "doesNotChangeQCConsensus": True,
    "doesNotChangeQCSupply": True,
    "doesNotChangeMiningReward": True,
}

WATCH_FORBIDDEN_FINAL_COMPLETION_TYPES = {
    "ad_loaded",
    "ad_opened",
    "clicked",
    "timer_only",
    "local_timer",
    "simulated",
    "dev_simulation",
}


class WatchEarnFinalSafetyError(ValueError):
    pass


def _env(name: str, default: str = "") -> str:
    return str(os.getenv(name) or default).strip()


def _env_bool(name: str) -> bool:
    return _env(name).lower() in {"1", "true", "yes", "on", "enabled"}


def watch_earn_resolve_mode_v1(mode: str = "") -> str:
    raw = (
        str(mode or "").strip().lower()
        or _env("WATCH_EARN_MODE").lower()
        or _env("WATCH_EARN_NETWORK_MODE").lower()
    )

    if raw in {"final", "mainnet", "production", "prod"}:
        return WATCH_MODE_FINAL

    if raw in {"testnet", "test"}:
        return WATCH_MODE_TESTNET

    project_mode = " ".join([
        _env("QC_TESTNET"),
        _env("QC_CHAIN_MODE"),
        _env("QC_NETWORK"),
        _env("TELEGRAM_GAME_CHAIN_MODE"),
        _env("TELEGRAM_GAME_NETWORK"),
    ]).lower()

    if "testnet" in project_mode or _env_bool("QC_TESTNET"):
        return WATCH_MODE_TESTNET

    return WATCH_MODE_DEV


def watch_earn_public_policy_from_env_v1() -> Dict[str, Any]:
    mode = watch_earn_resolve_mode_v1()

    return {
        "version": WATCH_EARN_FINAL_LOCK_VERSION,
        "mode": mode,
        "finalEnabled": _env_bool("WATCH_EARN_FINAL_ENABLED"),
        "provider": (_env("WATCH_EARN_PROVIDER") or WATCH_PROVIDER_NONE).lower(),
        "serverSideVerificationRequired": _env_bool("WATCH_EARN_SERVER_SIDE_VERIFICATION_REQUIRED"),
        "rewardAsset": _env("WATCH_EARN_REWARD_ASSET", "TGWT"),
        "rewardLedgerState": _env("WATCH_EARN_REWARD_LEDGER_STATE", "pending"),
        "revenueLogSeparateFromRewardLog": True,
        "finalApproval": _env("WATCH_EARN_FINAL_APPROVAL"),
        "safeBoundary": dict(WATCH_SAFE_BOUNDARY_V1),
    }


def watch_earn_validate_policy_v1(policy: Dict[str, Any]) -> Tuple[bool, str]:
    p = dict(policy or {})
    mode = watch_earn_resolve_mode_v1(str(p.get("mode", "")))
    final_enabled = bool(p.get("finalEnabled", False))
    provider = str(p.get("provider", WATCH_PROVIDER_NONE) or WATCH_PROVIDER_NONE).lower()
    ssv_required = bool(p.get("serverSideVerificationRequired", False))

    if str(p.get("rewardAsset", "")) != "TGWT":
        return False, "watch_reward_asset_must_be_tgwt"

    if str(p.get("rewardLedgerState", "")) != "pending":
        return False, "watch_reward_must_enter_pending_ledger"

    if p.get("revenueLogSeparateFromRewardLog") is not True:
        return False, "watch_revenue_log_must_stay_separate"

    if mode == WATCH_MODE_FINAL or final_enabled:
        if p.get("finalApproval") != WATCH_FINAL_APPROVAL:
            return False, "watch_final_requires_explicit_manual_approval"

        if provider not in {WATCH_PROVIDER_GOOGLE_REWARDED_ADS, WATCH_PROVIDER_ADMOB_REWARDED}:
            return False, "watch_final_requires_rewarded_ads_provider"

        if not ssv_required:
            return False, "watch_final_requires_server_side_verification"

    return True, "ok"


def watch_earn_assert_policy_safe_v1(policy: Dict[str, Any] = None) -> Dict[str, Any]:
    p = watch_earn_public_policy_from_env_v1() if policy is None else dict(policy)
    valid, reason = watch_earn_validate_policy_v1(p)
    if not valid:
        raise WatchEarnFinalSafetyError(reason)
    return p


def watch_earn_validate_completion_event_v1(event: Dict[str, Any], policy: Dict[str, Any]) -> Tuple[bool, str]:
    e = dict(event or {})
    p = watch_earn_assert_policy_safe_v1(policy)
    mode = watch_earn_resolve_mode_v1(str(p.get("mode", "")))

    completion_type = str(e.get("completionType", "") or "").lower()
    verified = bool(e.get("verifiedRewardedCompletion", False))
    ssv = bool(e.get("serverSideVerificationValid", False))
    dev_sim = bool(e.get("devSimulation", False))

    if mode == WATCH_MODE_FINAL:
        if dev_sim:
            return False, "watch_final_rejects_dev_simulation"

        if completion_type in WATCH_FORBIDDEN_FINAL_COMPLETION_TYPES:
            return False, "watch_final_rejects_non_rewarded_completion_type"

        if not verified:
            return False, "watch_final_requires_verified_rewarded_completion"

        if not ssv:
            return False, "watch_final_requires_valid_ssv_event"

    else:
        if completion_type in {"ad_loaded", "ad_opened", "clicked"}:
            return False, "watch_reward_not_allowed_for_load_open_or_click"

    return True, "ok"


def watch_earn_frontend_public_payload_v1(policy: Dict[str, Any] = None) -> Dict[str, Any]:
    p = watch_earn_assert_policy_safe_v1(policy)
    return {
        "version": WATCH_EARN_FINAL_LOCK_VERSION,
        "mode": p.get("mode", WATCH_MODE_DEV),
        "finalEnabled": bool(p.get("finalEnabled", False)),
        "provider": p.get("provider", WATCH_PROVIDER_NONE),
        "rewardAsset": "TGWT",
        "rewardLedgerState": "pending",
        "finalMode": "GOOGLE_REWARDED_ADS_VERIFIED_COMPLETION",
        "devMode": "DEV_TEST_TIMER_ONLY_NO_REVENUE",
        "serverSideVerificationRequired": bool(p.get("serverSideVerificationRequired", False)),
        "revenueLogSeparateFromRewardLog": True,
        "safeBoundary": dict(WATCH_SAFE_BOUNDARY_V1),
    }


def watch_earn_core_todo_v1() -> List[str]:
    return [
        "wire_google_rewarded_ads_sdk_later",
        "wire_server_side_verification_later",
        "keep_dev_timer_no_revenue",
        "keep_final_rewards_verified_only",
        "keep_tgwt_reward_pending_ledger_only",
        "keep_revenue_log_separate_from_reward_log",
        "keep_mainnet_activation_manual_approval",
    ]
