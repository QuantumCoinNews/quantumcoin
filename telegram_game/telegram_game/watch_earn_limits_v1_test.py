from watch_earn_revenue_policy import WATCH_EARN_MODE_DEV, WATCH_EARN_MODE_FINAL
from watch_earn_limits import (
    WATCH_EARN_LIMITS_VERSION,
    WATCH_EARN_LIMITS_STATUS,
    WATCH_EARN_DECISION_ALLOWED,
    WATCH_EARN_DECISION_COOLDOWN,
    WATCH_EARN_DECISION_DAILY_LIMIT,
    WATCH_EARN_DECISION_DUPLICATE_SESSION,
    WATCH_EARN_DECISION_DUPLICATE_TRANSACTION,
    WATCH_EARN_DECISION_REWARD_EVENT_REJECTED,
    watch_earn_limits_policy,
    watch_earn_limit_decision,
    watch_earn_record_reward,
    watch_earn_new_session,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def final_event(session_id="s1", transaction_id="tx1", user_id="dev_user"):
    return {
        "sessionId": session_id,
        "transactionId": transaction_id,
        "userId": user_id,
        "usesRealAd": True,
        "rewardedCompletion": True,
        "ssvVerified": True,
        "clickIncentivized": False,
        "botTraffic": False,
        "invalidTraffic": False,
        "fakeImpression": False,
        "adUnitId": "ca-app-pub-real/rewarded",
    }


def dev_event(session_id="s1", transaction_id="tx1", user_id="dev_user"):
    return {
        "sessionId": session_id,
        "transactionId": transaction_id,
        "userId": user_id,
        "devSimulation": True,
        "usesRealAd": False,
    }


def main():
    assert_equal(WATCH_EARN_LIMITS_VERSION, "WATCH_EARN_LIMITS_V1", "version")
    assert_equal(WATCH_EARN_LIMITS_STATUS, "LIMITS_ONLY_NO_AD_SDK_CALL", "status")

    dev_policy = watch_earn_limits_policy(WATCH_EARN_MODE_DEV)
    assert_equal(dev_policy["mode"], "DEV", "dev mode")
    assert_equal(dev_policy["dailyLimit"], 3, "dev daily limit")
    assert_equal(dev_policy["cooldownSeconds"], 10, "dev cooldown")

    final_policy = watch_earn_limits_policy(WATCH_EARN_MODE_FINAL)
    assert_equal(final_policy["mode"], "FINAL", "final mode")
    assert_equal(final_policy["dailyLimit"], 10, "final daily limit")
    assert_equal(final_policy["cooldownSeconds"], 120, "final cooldown")

    safe = final_policy["safeBoundary"]
    assert_true(safe["limitsOnly"] is True, "limits only")
    assert_true(safe["dailyLimitRequired"] is True, "daily limit required")
    assert_true(safe["cooldownRequired"] is True, "cooldown required")
    assert_true(safe["duplicateSessionBlocked"] is True, "duplicate session blocked")
    assert_true(safe["duplicateTransactionBlocked"] is True, "duplicate transaction blocked")
    assert_true(safe["rewardRequiresVerifiedCompletion"] is True, "verified completion required")
    assert_true(safe["serverSideVerificationRequiredFinal"] is True, "ssv required final")
    assert_true(safe["noClickIncentive"] is True, "no click incentive")
    assert_true(safe["noBotTraffic"] is True, "no bot traffic")
    assert_true(safe["noInvalidTraffic"] is True, "no invalid traffic")

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

    session = watch_earn_new_session("dev_user", "s1", 1000, WATCH_EARN_MODE_DEV)
    assert_equal(session["status"], "WATCH_SESSION_STARTED", "session started")

    history = []
    allowed_dev = watch_earn_limit_decision(
        user_id="dev_user",
        now_ts=1000,
        history=history,
        event=dev_event("s1", "tx1"),
        mode=WATCH_EARN_MODE_DEV,
    )
    assert_equal(allowed_dev["status"], WATCH_EARN_DECISION_ALLOWED, "dev allowed")
    history = watch_earn_record_reward(history, dev_event("s1", "tx1"), allowed_dev)
    assert_equal(len(history), 1, "history appended")

    cooldown = watch_earn_limit_decision(
        user_id="dev_user",
        now_ts=1005,
        history=history,
        event=dev_event("s2", "tx2"),
        mode=WATCH_EARN_MODE_DEV,
    )
    assert_equal(cooldown["status"], WATCH_EARN_DECISION_COOLDOWN, "cooldown active")
    assert_true(cooldown["remainingCooldownSeconds"] > 0, "cooldown remaining")

    duplicate_session = watch_earn_limit_decision(
        user_id="dev_user",
        now_ts=1015,
        history=history,
        event=dev_event("s1", "tx3"),
        mode=WATCH_EARN_MODE_DEV,
    )
    assert_equal(duplicate_session["status"], WATCH_EARN_DECISION_DUPLICATE_SESSION, "duplicate session")

    duplicate_tx = watch_earn_limit_decision(
        user_id="dev_user",
        now_ts=1015,
        history=history,
        event=dev_event("s3", "tx1"),
        mode=WATCH_EARN_MODE_DEV,
    )
    assert_equal(duplicate_tx["status"], WATCH_EARN_DECISION_DUPLICATE_TRANSACTION, "duplicate transaction")

    h2 = []
    for i in range(3):
        ts = 2000 + (i * 20)
        ev = dev_event(f"limit_s{i}", f"limit_tx{i}")
        decision = watch_earn_limit_decision(
            user_id="dev_user",
            now_ts=ts,
            history=h2,
            event=ev,
            mode=WATCH_EARN_MODE_DEV,
        )
        assert_equal(decision["status"], WATCH_EARN_DECISION_ALLOWED, "limit prep allowed")
        h2 = watch_earn_record_reward(h2, ev, decision)

    daily_limit = watch_earn_limit_decision(
        user_id="dev_user",
        now_ts=2100,
        history=h2,
        event=dev_event("limit_s4", "limit_tx4"),
        mode=WATCH_EARN_MODE_DEV,
    )
    assert_equal(daily_limit["status"], WATCH_EARN_DECISION_DAILY_LIMIT, "daily limit reached")

    final_ok = watch_earn_limit_decision(
        user_id="dev_user",
        now_ts=3000,
        history=[],
        event=final_event("fs1", "ftx1"),
        mode=WATCH_EARN_MODE_FINAL,
    )
    assert_equal(final_ok["status"], WATCH_EARN_DECISION_ALLOWED, "final allowed")

    final_missing_ssv = final_event("fs2", "ftx2")
    final_missing_ssv["ssvVerified"] = False
    rejected = watch_earn_limit_decision(
        user_id="dev_user",
        now_ts=4000,
        history=[],
        event=final_missing_ssv,
        mode=WATCH_EARN_MODE_FINAL,
    )
    assert_equal(rejected["status"], WATCH_EARN_DECISION_REWARD_EVENT_REJECTED, "missing ssv rejected")
    assert_true("missing_server_side_verification" in rejected["reasons"], "missing ssv reason")

    print("WATCH_EARN_LIMITS_V1_TEST_PASS")


if __name__ == "__main__":
    main()
