# WATCH_EARN_REVENUE_POLICY_V1
#
# Watch & Earn revenue policy for Telegram Game.
#
# Purpose:
# - If the game has real user traffic, rewarded ads may help recover server/node/API/ops costs.
# - DEV mode must never pretend to be real ad revenue.
# - FINAL mode must use Google Rewarded Ads with verified completion.
# - Server-side verification is required for final reward verification whenever available.
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

from decimal import Decimal, ROUND_HALF_UP
from typing import Any, Dict


WATCH_EARN_REVENUE_POLICY_VERSION = "WATCH_EARN_REVENUE_POLICY_V1"
WATCH_EARN_REVENUE_POLICY_STATUS = "POLICY_ONLY_NO_AD_SDK_CALL"

WATCH_EARN_MODE_DEV = "DEV"
WATCH_EARN_MODE_FINAL = "FINAL"

WATCH_EARN_PROVIDER_NONE = "NONE"
WATCH_EARN_PROVIDER_GOOGLE_ADMOB_REWARDED = "GOOGLE_ADMOB_REWARDED_ADS"

WATCH_EARN_REVENUE_PURPOSE = "OPS_COST_RECOVERY"
WATCH_EARN_REWARD_TYPE = "TGWT"

WATCH_EARN_REVENUE_SAFE_BOUNDARY = {
    "policyOnly": True,
    "noAdSdkCall": True,
    "noGoogleApiCall": True,
    "noFakeImpressions": True,
    "noClickIncentive": True,
    "noBotTraffic": True,
    "noInvalidTraffic": True,
    "revenuePurposeOpsCostRecovery": True,
    "rewardRequiresVerifiedCompletion": True,
    "serverSideVerificationRequiredFinal": True,
    "serverSideVerificationRecommendedDev": False,
    "clickIncentive": False,
    "fakeTrafficAllowed": False,
    "botTrafficAllowed": False,
    "invalidTrafficAllowed": False,
    "autoClickAllowed": False,
    "userMustClickAd": False,
    "rewardForClick": False,
    "rewardForLoadOnly": False,
    "rewardForOpenOnly": False,
    "revenueLogSeparateFromRewardLog": True,
    "userPaysNothingToWatch": True,
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


def watch_earn_mode_policy(mode: str = WATCH_EARN_MODE_DEV) -> Dict[str, Any]:
    mode = str(mode or "").strip().upper()

    if mode == WATCH_EARN_MODE_FINAL:
        return {
            "version": WATCH_EARN_REVENUE_POLICY_VERSION,
            "status": WATCH_EARN_REVENUE_POLICY_STATUS,
            "mode": WATCH_EARN_MODE_FINAL,
            "adProvider": WATCH_EARN_PROVIDER_GOOGLE_ADMOB_REWARDED,
            "revenueEnabled": True,
            "devModeRevenue": False,
            "testAdUnitOnly": False,
            "realAdUnitRequired": True,
            "rewardRequiresGoogleRewardedAds": True,
            "rewardRequiresVerifiedCompletion": True,
            "rewardRequiresServerSideVerification": True,
            "ssvRequired": True,
            "ssvCallbackEndpointRequired": True,
            "adCompletionCallbackRequired": True,
            "onPaidEventRevenueTrackingRecommended": True,
            "rewardType": WATCH_EARN_REWARD_TYPE,
            "rewardState": "PENDING_THEN_VERIFIED",
            "revenuePurpose": WATCH_EARN_REVENUE_PURPOSE,
            "safeBoundary": dict(WATCH_EARN_REVENUE_SAFE_BOUNDARY),
        }

    return {
        "version": WATCH_EARN_REVENUE_POLICY_VERSION,
        "status": WATCH_EARN_REVENUE_POLICY_STATUS,
        "mode": WATCH_EARN_MODE_DEV,
        "adProvider": WATCH_EARN_PROVIDER_NONE,
        "revenueEnabled": False,
        "devModeRevenue": False,
        "testAdUnitOnly": True,
        "realAdUnitRequired": False,
        "rewardRequiresGoogleRewardedAds": False,
        "rewardRequiresVerifiedCompletion": False,
        "rewardRequiresServerSideVerification": False,
        "ssvRequired": False,
        "ssvCallbackEndpointRequired": False,
        "adCompletionCallbackRequired": False,
        "onPaidEventRevenueTrackingRecommended": False,
        "rewardType": WATCH_EARN_REWARD_TYPE,
        "rewardState": "DEV_SIMULATION_ONLY",
        "revenuePurpose": WATCH_EARN_REVENUE_PURPOSE,
        "safeBoundary": dict(WATCH_EARN_REVENUE_SAFE_BOUNDARY),
    }


def watch_earn_validate_reward_event(event: Dict[str, Any], mode: str = WATCH_EARN_MODE_DEV) -> Dict[str, Any]:
    policy = watch_earn_mode_policy(mode)
    event = dict(event or {})

    reasons = []
    status = "REJECTED"

    if policy["mode"] == WATCH_EARN_MODE_DEV:
        if bool(event.get("devSimulation", False)) is True and bool(event.get("usesRealAd", False)) is False:
            status = "DEV_SIMULATION_ACCEPTED_NO_REVENUE"
            reasons.append("dev_simulation_only")
        else:
            reasons.append("dev_requires_simulation_not_real_ad")

        return _watch_earn_result(status, reasons, policy)

    if bool(event.get("usesRealAd", False)) is not True:
        reasons.append("final_requires_real_rewarded_ad")

    if bool(event.get("rewardedCompletion", False)) is not True:
        reasons.append("missing_rewarded_completion")

    if bool(event.get("ssvVerified", False)) is not True:
        reasons.append("missing_server_side_verification")

    if bool(event.get("clickIncentivized", False)) is True:
        reasons.append("click_incentive_forbidden")

    if bool(event.get("botTraffic", False)) is True:
        reasons.append("bot_traffic_forbidden")

    if bool(event.get("invalidTraffic", False)) is True:
        reasons.append("invalid_traffic_forbidden")

    if bool(event.get("fakeImpression", False)) is True:
        reasons.append("fake_impression_forbidden")

    if not str(event.get("adUnitId", "")).strip():
        reasons.append("missing_ad_unit_id")

    if not str(event.get("transactionId", "")).strip():
        reasons.append("missing_transaction_id")

    if not str(event.get("userId", "")).strip():
        reasons.append("missing_user_id")

    if not reasons:
        status = "FINAL_REWARD_EVENT_VERIFIED"
        reasons.append("eligible_for_tgwt_reward_pending_verified")

    return _watch_earn_result(status, reasons, policy)


def _watch_earn_result(status: str, reasons: list[str], policy: Dict[str, Any]) -> Dict[str, Any]:
    return {
        "version": WATCH_EARN_REVENUE_POLICY_VERSION,
        "status": status,
        "reasons": list(reasons or []),
        "mode": policy["mode"],
        "adProvider": policy["adProvider"],
        "revenueEnabled": bool(policy["revenueEnabled"]),
        "rewardType": WATCH_EARN_REWARD_TYPE,
        "revenuePurpose": WATCH_EARN_REVENUE_PURPOSE,
        "clickIncentive": False,
        "invalidTrafficAllowed": False,
        "botTrafficAllowed": False,
        "fakeTrafficAllowed": False,
        "rewardRequiresVerifiedCompletion": bool(policy["rewardRequiresVerifiedCompletion"]),
        "rewardRequiresServerSideVerification": bool(policy["rewardRequiresServerSideVerification"]),
        "revenueLogSeparateFromRewardLog": True,
        "safeBoundary": dict(WATCH_EARN_REVENUE_SAFE_BOUNDARY),
    }


def watch_earn_estimate_revenue(
    daily_active_users: Any,
    ads_per_user_per_day: Any = "2",
    completion_rate: Any = "0.70",
    ecpm_usd: Any = "1.00",
    invalid_traffic_deduction_rate: Any = "0.00",
) -> Dict[str, Any]:
    dau = _decimal_non_negative(daily_active_users)
    ads_per_user = _decimal_non_negative(ads_per_user_per_day)
    completion = _clamp_decimal(_decimal_non_negative(completion_rate), Decimal("0"), Decimal("1"))
    ecpm = _decimal_non_negative(ecpm_usd)
    invalid_deduction = _clamp_decimal(_decimal_non_negative(invalid_traffic_deduction_rate), Decimal("0"), Decimal("1"))

    raw_valid_impressions = dau * ads_per_user * completion
    payable_impressions = raw_valid_impressions * (Decimal("1") - invalid_deduction)
    daily_revenue = (payable_impressions / Decimal("1000")) * ecpm
    monthly_revenue = daily_revenue * Decimal("30")

    return {
        "version": WATCH_EARN_REVENUE_POLICY_VERSION,
        "formula": "revenue ~= valid_rewarded_impressions / 1000 * eCPM",
        "dailyActiveUsers": int(dau),
        "adsPerUserPerDay": _decimal_to_str(ads_per_user),
        "completionRate": _decimal_to_str(completion),
        "invalidTrafficDeductionRate": _decimal_to_str(invalid_deduction),
        "ecpmUsd": _decimal_to_str(ecpm),
        "rawValidImpressionsPerDay": int(raw_valid_impressions),
        "payableImpressionsPerDay": int(payable_impressions),
        "estimatedDailyRevenueUsd": _money(daily_revenue),
        "estimatedMonthlyRevenueUsd": _money(monthly_revenue),
        "guaranteedRevenue": False,
        "purpose": WATCH_EARN_REVENUE_PURPOSE,
        "safeBoundary": dict(WATCH_EARN_REVENUE_SAFE_BOUNDARY),
    }


def watch_earn_estimate_revenue_table(
    daily_active_users_list: list[int] | None = None,
    ecpm_list: list[str] | None = None,
    ads_per_user_per_day: Any = "2",
    completion_rate: Any = "0.70",
) -> Dict[str, Any]:
    users = list(daily_active_users_list or [1000, 5000, 10000, 50000, 100000])
    ecpms = list(ecpm_list or ["0.50", "1.00", "3.00", "5.00"])

    rows = []
    for dau in users:
        row = {
            "dailyActiveUsers": int(dau),
            "scenarios": {},
        }
        for ecpm in ecpms:
            est = watch_earn_estimate_revenue(
                daily_active_users=dau,
                ads_per_user_per_day=ads_per_user_per_day,
                completion_rate=completion_rate,
                ecpm_usd=ecpm,
            )
            row["scenarios"][f"ecpm_{ecpm}"] = {
                "dailyUsd": est["estimatedDailyRevenueUsd"],
                "monthlyUsd": est["estimatedMonthlyRevenueUsd"],
                "payableImpressionsPerDay": est["payableImpressionsPerDay"],
            }
        rows.append(row)

    return {
        "version": WATCH_EARN_REVENUE_POLICY_VERSION,
        "status": "ESTIMATE_ONLY_NOT_GUARANTEED",
        "assumptions": {
            "adsPerUserPerDay": str(ads_per_user_per_day),
            "completionRate": str(completion_rate),
            "formula": "DAU * adsPerUser * completionRate / 1000 * eCPM",
        },
        "rows": rows,
        "safeBoundary": dict(WATCH_EARN_REVENUE_SAFE_BOUNDARY),
    }


def _decimal_non_negative(value: Any) -> Decimal:
    try:
        d = Decimal(str(value).strip())
    except Exception:
        d = Decimal("0")
    if d < 0:
        return Decimal("0")
    return d


def _clamp_decimal(value: Decimal, low: Decimal, high: Decimal) -> Decimal:
    if value < low:
        return low
    if value > high:
        return high
    return value


def _money(value: Decimal) -> str:
    q = value.quantize(Decimal("0.01"), rounding=ROUND_HALF_UP)
    return format(q, "f")


def _decimal_to_str(value: Decimal) -> str:
    normalized = value.normalize()
    if normalized == normalized.to_integral():
        return str(int(normalized))
    return format(normalized, "f")

# WATCH_EARN_TESTNET_FINAL_LOCK_V1_BRIDGE
# Runtime-safe bridge to final/testnet Watch & Earn policy guard.
def watch_earn_final_public_policy_runtime_v1():
    from telegram_game.watch_earn_final_safety_v1 import watch_earn_public_policy_from_env_v1

    return watch_earn_public_policy_from_env_v1()


def watch_earn_validate_final_public_policy_runtime_v1():
    from telegram_game.watch_earn_final_safety_v1 import watch_earn_assert_policy_safe_v1

    return watch_earn_assert_policy_safe_v1()


def watch_earn_final_frontend_public_payload_runtime_v1():
    from telegram_game.watch_earn_final_safety_v1 import watch_earn_frontend_public_payload_v1

    return watch_earn_frontend_public_payload_v1()
