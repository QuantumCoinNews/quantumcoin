from watch_earn_revenue_policy import WATCH_EARN_MODE_DEV, WATCH_EARN_MODE_FINAL
from watch_earn_verified_completion import (
    WATCH_EARN_VERIFIED_COMPLETION_VERSION,
    WATCH_EARN_VERIFIED_COMPLETION_STATUS,
    WATCH_EARN_VERIFIED_ACCEPTED,
    WATCH_EARN_VERIFIED_DEV_SIMULATION,
    WATCH_EARN_VERIFIED_REJECTED,
    WATCH_EARN_VERIFIED_DUPLICATE_TRANSACTION,
    WATCH_EARN_VERIFIED_LIMIT_REJECTED,
    watch_earn_verified_completion_policy,
    watch_earn_build_verified_event,
    watch_earn_verify_completion,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def main():
    assert_equal(WATCH_EARN_VERIFIED_COMPLETION_VERSION, "WATCH_EARN_VERIFIED_COMPLETION_V1", "version")
    assert_equal(WATCH_EARN_VERIFIED_COMPLETION_STATUS, "VERIFIED_COMPLETION_REQUIRED_NO_AD_SDK_CALL", "status")

    dev_policy = watch_earn_verified_completion_policy(WATCH_EARN_MODE_DEV)
    assert_equal(dev_policy["mode"], "DEV", "dev mode")
    assert_true(dev_policy["devAllowsSimulationOnly"] is True, "dev simulation only")

    final_policy = watch_earn_verified_completion_policy(WATCH_EARN_MODE_FINAL)
    assert_equal(final_policy["mode"], "FINAL", "final mode")
    assert_true(final_policy["finalRequires"]["realGoogleRewardedAd"] is True, "real rewarded ad required")
    assert_true(final_policy["finalRequires"]["rewardedCompletion"] is True, "rewarded completion required")
    assert_true(final_policy["finalRequires"]["serverSideVerification"] is True, "ssv required")
    assert_true(final_policy["finalRequires"]["ssvSignatureVerified"] is True, "ssv signature required")
    assert_true(final_policy["finalRequires"]["serverCallback"] is True, "server callback required")
    assert_true(final_policy["finalRequires"]["uniqueTransactionId"] is True, "unique transaction required")
    assert_true(final_policy["finalRequires"]["customDataOrNonce"] is True, "custom data or nonce required")

    safe = final_policy["safeBoundary"]
    assert_true(safe["verifiedCompletionOnly"] is True, "verified completion only")
    assert_true(safe["requiresRewardedCompletionFinal"] is True, "safe rewarded completion")
    assert_true(safe["requiresServerSideVerificationFinal"] is True, "safe ssv")
    assert_true(safe["requiresSsvSignatureVerifiedFinal"] is True, "safe ssv signature")
    assert_true(safe["requiresServerCallbackFinal"] is True, "safe server callback")
    assert_true(safe["requiresUniqueTransactionId"] is True, "safe unique transaction")
    assert_true(safe["clientOnlyCompletionAcceptedFinal"] is False, "client only not accepted")
    assert_true(safe["rewardForAdLoaded"] is False, "no reward for ad loaded")
    assert_true(safe["rewardForAdOpened"] is False, "no reward for ad opened")
    assert_true(safe["rewardForFrontendOnlySignal"] is False, "no frontend only reward")
    assert_true(safe["rewardForClick"] is False, "no reward for click")

    false_keys = [
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

    dev_event = watch_earn_build_verified_event(
        user_id="dev_user",
        session_id="dev_s1",
        transaction_id="dev_tx1",
        ad_unit_id="",
        custom_data="",
        now_ts=1000,
        mode=WATCH_EARN_MODE_DEV,
    )
    dev_result = watch_earn_verify_completion(dev_event, [], WATCH_EARN_MODE_DEV)
    assert_equal(dev_result["status"], WATCH_EARN_VERIFIED_DEV_SIMULATION, "dev simulation accepted")
    assert_equal(dev_result["rewardState"], "PENDING_OR_VERIFIED_BY_POLICY", "dev reward state")

    final_event = watch_earn_build_verified_event(
        user_id="dev_user",
        session_id="final_s1",
        transaction_id="final_tx1",
        ad_unit_id="ca-app-pub-real/rewarded",
        custom_data="nonce_1",
        now_ts=2000,
        mode=WATCH_EARN_MODE_FINAL,
    )
    final_result = watch_earn_verify_completion(final_event, [], WATCH_EARN_MODE_FINAL)
    assert_equal(final_result["status"], WATCH_EARN_VERIFIED_ACCEPTED, "final accepted")
    assert_true("verified_completion_eligible_for_reward" in final_result["reasons"], "verified reason")
    assert_true(final_result["clientOnlyCompletionAcceptedFinal"] is False, "client only false")

    no_completion = dict(final_event)
    no_completion["transactionId"] = "final_tx2"
    no_completion["sessionId"] = "final_s2"
    no_completion["rewardedCompletion"] = False
    rejected_completion = watch_earn_verify_completion(no_completion, [], WATCH_EARN_MODE_FINAL)
    assert_equal(rejected_completion["status"], WATCH_EARN_VERIFIED_REJECTED, "missing completion rejected")
    assert_true("missing_rewarded_completion" in rejected_completion["reasons"], "missing completion reason")

    no_ssv = dict(final_event)
    no_ssv["transactionId"] = "final_tx3"
    no_ssv["sessionId"] = "final_s3"
    no_ssv["ssvVerified"] = False
    rejected_ssv = watch_earn_verify_completion(no_ssv, [], WATCH_EARN_MODE_FINAL)
    assert_equal(rejected_ssv["status"], WATCH_EARN_VERIFIED_REJECTED, "missing ssv rejected")
    assert_true("missing_server_side_verification" in rejected_ssv["reasons"], "missing ssv reason")

    no_signature = dict(final_event)
    no_signature["transactionId"] = "final_tx4"
    no_signature["sessionId"] = "final_s4"
    no_signature["ssvSignatureVerified"] = False
    rejected_signature = watch_earn_verify_completion(no_signature, [], WATCH_EARN_MODE_FINAL)
    assert_equal(rejected_signature["status"], WATCH_EARN_VERIFIED_REJECTED, "missing signature rejected")
    assert_true("missing_ssv_signature_verification" in rejected_signature["reasons"], "missing signature reason")

    no_callback = dict(final_event)
    no_callback["transactionId"] = "final_tx5"
    no_callback["sessionId"] = "final_s5"
    no_callback["serverCallbackReceived"] = False
    rejected_callback = watch_earn_verify_completion(no_callback, [], WATCH_EARN_MODE_FINAL)
    assert_equal(rejected_callback["status"], WATCH_EARN_VERIFIED_REJECTED, "missing callback rejected")
    assert_true("missing_server_callback" in rejected_callback["reasons"], "missing callback reason")

    client_only = dict(final_event)
    client_only["transactionId"] = "final_tx6"
    client_only["sessionId"] = "final_s6"
    client_only["clientOnlyCompletion"] = True
    rejected_client = watch_earn_verify_completion(client_only, [], WATCH_EARN_MODE_FINAL)
    assert_equal(rejected_client["status"], WATCH_EARN_VERIFIED_REJECTED, "client only rejected")
    assert_true("client_only_completion_not_accepted" in rejected_client["reasons"], "client only reason")

    duplicate_history = [
        {
            "userId": "dev_user",
            "transactionId": "final_tx7",
            "sessionId": "old_session",
            "timestamp": 2500,
            "rewardGranted": True,
        }
    ]
    duplicate_event = dict(final_event)
    duplicate_event["transactionId"] = "final_tx7"
    duplicate_event["sessionId"] = "final_s7"
    duplicate = watch_earn_verify_completion(duplicate_event, duplicate_history, WATCH_EARN_MODE_FINAL)
    assert_equal(duplicate["status"], WATCH_EARN_VERIFIED_DUPLICATE_TRANSACTION, "duplicate rejected")

    limit_history = []
    for i in range(10):
        limit_history.append({
            "userId": "dev_user",
            "transactionId": f"hist_tx_{i}",
            "sessionId": f"hist_s_{i}",
            "timestamp": 3000 + i,
            "rewardGranted": True,
        })

    over_limit_event = dict(final_event)
    over_limit_event["transactionId"] = "final_tx8"
    over_limit_event["sessionId"] = "final_s8"
    over_limit_event["timestamp"] = 3500
    limit_result = watch_earn_verify_completion(over_limit_event, limit_history, WATCH_EARN_MODE_FINAL)
    assert_equal(limit_result["status"], WATCH_EARN_VERIFIED_LIMIT_REJECTED, "limit rejected")

    print("WATCH_EARN_VERIFIED_COMPLETION_V1_TEST_PASS")


if __name__ == "__main__":
    main()
