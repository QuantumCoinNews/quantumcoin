from watch_earn_revenue_policy import (
    WATCH_EARN_REVENUE_POLICY_VERSION,
    WATCH_EARN_REVENUE_POLICY_STATUS,
    WATCH_EARN_MODE_DEV,
    WATCH_EARN_MODE_FINAL,
    WATCH_EARN_PROVIDER_NONE,
    WATCH_EARN_PROVIDER_GOOGLE_ADMOB_REWARDED,
    WATCH_EARN_REVENUE_PURPOSE,
    watch_earn_mode_policy,
    watch_earn_validate_reward_event,
    watch_earn_estimate_revenue,
    watch_earn_estimate_revenue_table,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def main():
    assert_equal(WATCH_EARN_REVENUE_POLICY_VERSION, "WATCH_EARN_REVENUE_POLICY_V1", "version")
    assert_equal(WATCH_EARN_REVENUE_POLICY_STATUS, "POLICY_ONLY_NO_AD_SDK_CALL", "status")
    assert_equal(WATCH_EARN_REVENUE_PURPOSE, "OPS_COST_RECOVERY", "revenue purpose")

    dev = watch_earn_mode_policy(WATCH_EARN_MODE_DEV)
    assert_equal(dev["mode"], "DEV", "dev mode")
    assert_equal(dev["adProvider"], WATCH_EARN_PROVIDER_NONE, "dev provider")
    assert_true(dev["revenueEnabled"] is False, "dev revenue disabled")
    assert_true(dev["devModeRevenue"] is False, "dev mode revenue false")
    assert_true(dev["testAdUnitOnly"] is True, "dev test ad unit only")
    assert_true(dev["rewardRequiresGoogleRewardedAds"] is False, "dev no google reward required")
    assert_true(dev["rewardRequiresServerSideVerification"] is False, "dev no ssv required")

    final = watch_earn_mode_policy(WATCH_EARN_MODE_FINAL)
    assert_equal(final["mode"], "FINAL", "final mode")
    assert_equal(final["adProvider"], WATCH_EARN_PROVIDER_GOOGLE_ADMOB_REWARDED, "final provider")
    assert_true(final["revenueEnabled"] is True, "final revenue enabled")
    assert_true(final["realAdUnitRequired"] is True, "real ad unit required")
    assert_true(final["rewardRequiresGoogleRewardedAds"] is True, "final requires Google rewarded ads")
    assert_true(final["rewardRequiresVerifiedCompletion"] is True, "final requires verified completion")
    assert_true(final["rewardRequiresServerSideVerification"] is True, "final requires ssv")
    assert_true(final["ssvCallbackEndpointRequired"] is True, "ssv callback required")
    assert_true(final["onPaidEventRevenueTrackingRecommended"] is True, "paid event revenue tracking recommended")

    safe = final["safeBoundary"]
    assert_true(safe["policyOnly"] is True, "policy only")
    assert_true(safe["noAdSdkCall"] is True, "no ad sdk call")
    assert_true(safe["noGoogleApiCall"] is True, "no google api call")
    assert_true(safe["noFakeImpressions"] is True, "no fake impressions")
    assert_true(safe["noClickIncentive"] is True, "no click incentive")
    assert_true(safe["noBotTraffic"] is True, "no bot traffic")
    assert_true(safe["noInvalidTraffic"] is True, "no invalid traffic")
    assert_true(safe["revenuePurposeOpsCostRecovery"] is True, "ops cost recovery")
    assert_true(safe["serverSideVerificationRequiredFinal"] is True, "ssv required final")
    assert_true(safe["revenueLogSeparateFromRewardLog"] is True, "separate logs")

    false_keys = [
        "clickIncentive",
        "fakeTrafficAllowed",
        "botTrafficAllowed",
        "invalidTrafficAllowed",
        "autoClickAllowed",
        "userMustClickAd",
        "rewardForClick",
        "rewardForLoadOnly",
        "rewardForOpenOnly",
        "changesQCBalances",
        "changesTGWTOnChainBalances",
        "createsBscTransaction",
        "sendsTGWT",
        "paysGas",
        "storesPrivateKey",
        "signsTransaction",
        "changesMempool",
        "changesChain",
        "changesConsensus",
        "changesTotalSupply",
        "startsMining",
        "stopsMining",
    ]

    for key in false_keys:
        assert_true(safe[key] is False, f"{key} must be false")

    dev_ok = watch_earn_validate_reward_event(
        {
            "devSimulation": True,
            "usesRealAd": False,
        },
        WATCH_EARN_MODE_DEV,
    )
    assert_equal(dev_ok["status"], "DEV_SIMULATION_ACCEPTED_NO_REVENUE", "dev simulation ok")
    assert_true(dev_ok["revenueEnabled"] is False, "dev result revenue false")

    final_ok = watch_earn_validate_reward_event(
        {
            "usesRealAd": True,
            "rewardedCompletion": True,
            "ssvVerified": True,
            "clickIncentivized": False,
            "botTraffic": False,
            "invalidTraffic": False,
            "fakeImpression": False,
            "adUnitId": "ca-app-pub-real/rewarded",
            "transactionId": "tx1",
            "userId": "dev_user",
        },
        WATCH_EARN_MODE_FINAL,
    )
    assert_equal(final_ok["status"], "FINAL_REWARD_EVENT_VERIFIED", "final verified")
    assert_true(final_ok["revenueEnabled"] is True, "final result revenue true")
    assert_true(final_ok["rewardRequiresVerifiedCompletion"] is True, "final result requires completion")
    assert_true(final_ok["rewardRequiresServerSideVerification"] is True, "final result requires ssv")
    assert_true(final_ok["clickIncentive"] is False, "no click incentive")
    assert_true(final_ok["invalidTrafficAllowed"] is False, "invalid traffic disallowed")
    assert_true(final_ok["botTrafficAllowed"] is False, "bot traffic disallowed")
    assert_true(final_ok["fakeTrafficAllowed"] is False, "fake traffic disallowed")

    final_bad = watch_earn_validate_reward_event(
        {
            "usesRealAd": True,
            "rewardedCompletion": True,
            "ssvVerified": False,
            "clickIncentivized": True,
            "botTraffic": False,
            "invalidTraffic": False,
            "fakeImpression": False,
            "adUnitId": "ca-app-pub-real/rewarded",
            "transactionId": "tx2",
            "userId": "dev_user",
        },
        WATCH_EARN_MODE_FINAL,
    )
    assert_equal(final_bad["status"], "REJECTED", "final bad rejected")
    assert_true("missing_server_side_verification" in final_bad["reasons"], "missing ssv reason")
    assert_true("click_incentive_forbidden" in final_bad["reasons"], "click incentive reason")

    estimate = watch_earn_estimate_revenue(
        daily_active_users=10000,
        ads_per_user_per_day="2",
        completion_rate="0.70",
        ecpm_usd="3.00",
    )
    assert_equal(estimate["payableImpressionsPerDay"], 14000, "payable impressions")
    assert_equal(estimate["estimatedDailyRevenueUsd"], "42.00", "daily estimate")
    assert_equal(estimate["estimatedMonthlyRevenueUsd"], "1260.00", "monthly estimate")
    assert_true(estimate["guaranteedRevenue"] is False, "not guaranteed")

    table = watch_earn_estimate_revenue_table()
    assert_equal(table["status"], "ESTIMATE_ONLY_NOT_GUARANTEED", "table status")
    assert_equal(len(table["rows"]), 5, "default rows")
    assert_true("ecpm_3.00" in table["rows"][2]["scenarios"], "ecpm scenario exists")

    print("WATCH_EARN_REVENUE_POLICY_V1_TEST_PASS")


if __name__ == "__main__":
    main()
