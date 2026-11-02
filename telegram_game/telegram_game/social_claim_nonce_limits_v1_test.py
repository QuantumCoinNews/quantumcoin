from social_claim_verification import (
    SOCIAL_CLAIM_MODE_DEV,
    SOCIAL_CLAIM_MODE_FINAL,
    SOCIAL_ACTION_FOLLOW_X_ACCOUNT,
    SOCIAL_ACTION_WEBSITE_NONCE_VERIFY,
)
from social_claim_nonce_limits import (
    SOCIAL_CLAIM_NONCE_LIMITS_VERSION,
    SOCIAL_CLAIM_NONCE_LIMITS_STATUS,
    SOCIAL_CLAIM_REWARD_AMOUNT_TGWT,
    SOCIAL_CLAIM_NONCE_ALLOWED,
    SOCIAL_CLAIM_NONCE_DEV_SIMULATION,
    SOCIAL_CLAIM_NONCE_MISSING,
    SOCIAL_CLAIM_NONCE_DUPLICATE,
    SOCIAL_CLAIM_CLAIM_ID_DUPLICATE,
    SOCIAL_CLAIM_PLATFORM_REWARD_ALREADY_GRANTED,
    SOCIAL_CLAIM_VERIFICATION_REQUIRED,
    social_claim_nonce_limits_policy,
    social_claim_apply_nonce_limits,
    social_claim_record_reward,
    social_claim_extract_nonce,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def x_claim(claim_id="x_claim_1", nonce="nonce_x_1", user_id="dev_user"):
    return {
        "userId": user_id,
        "claimId": claim_id,
        "claimNonce": nonce,
        "platform": "X",
        "action": SOCIAL_ACTION_FOLLOW_X_ACCOUNT,
        "xHandle": "@dev_user",
        "targetAccount": "@QuantumCoin",
        "providerVerified": True,
        "followVerified": True,
    }


def website_claim(claim_id="web_claim_1", nonce="web_nonce_1", user_id="dev_user"):
    return {
        "userId": user_id,
        "claimId": claim_id,
        "platform": "WEBSITE",
        "action": SOCIAL_ACTION_WEBSITE_NONCE_VERIFY,
        "websiteHost": "quantumcoin.example",
        "websiteNonce": nonce,
        "serverNonceVerified": True,
    }


def main():
    assert_equal(SOCIAL_CLAIM_NONCE_LIMITS_VERSION, "SOCIAL_CLAIM_NONCE_LIMITS_V1", "version")
    assert_equal(SOCIAL_CLAIM_NONCE_LIMITS_STATUS, "NONCE_AND_PLATFORM_LIMIT_POLICY_ONLY", "status")
    assert_equal(SOCIAL_CLAIM_REWARD_AMOUNT_TGWT, 1, "reward amount")

    policy = social_claim_nonce_limits_policy()
    assert_true(policy["nonceReplayProtection"] is True, "nonce replay protection")
    assert_true(policy["claimIdReplayProtection"] is True, "claim id replay protection")
    assert_equal(policy["rewardAmountTGWT"], 1, "policy reward amount")

    safe = policy["safeBoundary"]
    assert_true(safe["policyOnly"] is True, "policy only")
    assert_true(safe["offChainOnly"] is True, "off-chain only")
    assert_true(safe["noExternalApiCall"] is True, "no external api call")
    assert_true(safe["noBrowserAutomation"] is True, "no browser automation")
    assert_true(safe["noPasswordCollection"] is True, "no password collection")
    assert_true(safe["noOAuthTokenStorage"] is True, "no oauth token storage")
    assert_true(safe["nonceRequired"] is True, "nonce required")
    assert_true(safe["duplicateNonceBlocked"] is True, "duplicate nonce blocked")
    assert_true(safe["duplicateClaimIdBlocked"] is True, "duplicate claim id blocked")
    assert_true(safe["perUserPerPlatformOneReward"] is True, "per user per platform one reward")
    assert_equal(safe["rewardAmountTGWT"], 1, "safe reward amount")
    assert_true(safe["requiresVerifiedSocialClaim"] is True, "requires verified social claim")
    assert_true(safe["devSimulationNoReward"] is True, "dev simulation no reward")
    assert_true(safe["websiteNonceAcceptedAsClaimNonce"] is True, "website nonce accepted")

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

    dev = {
        "userId": "dev_user",
        "claimId": "dev_claim",
        "claimNonce": "dev_nonce",
        "platform": "X",
        "action": SOCIAL_ACTION_FOLLOW_X_ACCOUNT,
        "devSimulation": True,
    }
    dev_result = social_claim_apply_nonce_limits(dev, [], SOCIAL_CLAIM_MODE_DEV)
    assert_equal(dev_result["status"], SOCIAL_CLAIM_NONCE_DEV_SIMULATION, "dev simulation no reward")
    assert_true(dev_result["rewardEligible"] is False, "dev no reward")

    decision = social_claim_apply_nonce_limits(x_claim(), [], SOCIAL_CLAIM_MODE_FINAL)
    assert_equal(decision["status"], SOCIAL_CLAIM_NONCE_ALLOWED, "X first claim allowed")
    assert_true(decision["rewardEligible"] is True, "X reward eligible")
    assert_equal(decision["rewardAmountTGWT"], 1, "X reward amount")

    history = social_claim_record_reward([], x_claim(), decision, now_ts=1000)
    assert_equal(len(history), 1, "history appended")
    assert_equal(history[0]["rewardAmountTGWT"], 1, "history reward amount")

    duplicate_nonce = social_claim_apply_nonce_limits(
        x_claim(claim_id="x_claim_2", nonce="nonce_x_1", user_id="other_user"),
        history,
        SOCIAL_CLAIM_MODE_FINAL,
    )
    assert_equal(duplicate_nonce["status"], SOCIAL_CLAIM_NONCE_DUPLICATE, "duplicate nonce blocked")

    duplicate_claim_id = social_claim_apply_nonce_limits(
        x_claim(claim_id="x_claim_1", nonce="nonce_x_new", user_id="other_user"),
        history,
        SOCIAL_CLAIM_MODE_FINAL,
    )
    assert_equal(duplicate_claim_id["status"], SOCIAL_CLAIM_CLAIM_ID_DUPLICATE, "duplicate claim id blocked")

    same_platform = social_claim_apply_nonce_limits(
        x_claim(claim_id="x_claim_3", nonce="nonce_x_3", user_id="dev_user"),
        history,
        SOCIAL_CLAIM_MODE_FINAL,
    )
    assert_equal(same_platform["status"], SOCIAL_CLAIM_PLATFORM_REWARD_ALREADY_GRANTED, "same platform blocked")
    assert_true("per_user_per_platform_reward_already_granted" in same_platform["reasons"], "platform reason")

    website_decision = social_claim_apply_nonce_limits(
        website_claim(claim_id="web_claim_1", nonce="web_nonce_1", user_id="dev_user"),
        history,
        SOCIAL_CLAIM_MODE_FINAL,
    )
    assert_equal(website_decision["status"], SOCIAL_CLAIM_NONCE_ALLOWED, "website different platform allowed")
    assert_equal(social_claim_extract_nonce(website_claim()), "web_nonce_1", "website nonce extraction")

    missing_nonce_claim = x_claim(claim_id="x_claim_4", nonce="", user_id="new_user")
    missing_nonce = social_claim_apply_nonce_limits(missing_nonce_claim, history, SOCIAL_CLAIM_MODE_FINAL)
    assert_equal(missing_nonce["status"], SOCIAL_CLAIM_NONCE_MISSING, "missing nonce blocked")

    unverified = x_claim(claim_id="x_claim_5", nonce="nonce_x_5", user_id="new_user")
    unverified["providerVerified"] = False
    unverified["followVerified"] = False
    verification_required = social_claim_apply_nonce_limits(unverified, history, SOCIAL_CLAIM_MODE_FINAL)
    assert_equal(verification_required["status"], SOCIAL_CLAIM_VERIFICATION_REQUIRED, "verification required")

    print("SOCIAL_CLAIM_NONCE_LIMITS_V1_TEST_PASS")


if __name__ == "__main__":
    main()
