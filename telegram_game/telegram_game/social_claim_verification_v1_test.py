from social_claim_verification import (
    SOCIAL_CLAIM_VERIFICATION_VERSION,
    SOCIAL_CLAIM_VERIFICATION_STATUS,
    SOCIAL_CLAIM_MODE_DEV,
    SOCIAL_CLAIM_MODE_FINAL,
    SOCIAL_PLATFORM_X,
    SOCIAL_PLATFORM_WEBSITE,
    SOCIAL_ACTION_FOLLOW_X_ACCOUNT,
    SOCIAL_ACTION_WEBSITE_NONCE_VERIFY,
    SOCIAL_CLAIM_VERIFIED,
    SOCIAL_CLAIM_PENDING_PROVIDER_VERIFICATION,
    SOCIAL_CLAIM_REJECTED,
    SOCIAL_CLAIM_DEV_SIMULATION,
    social_claim_policy,
    social_claim_normalize_platform,
    social_claim_evaluate,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def main():
    assert_equal(SOCIAL_CLAIM_VERIFICATION_VERSION, "SOCIAL_CLAIM_VERIFICATION_V1", "version")
    assert_equal(SOCIAL_CLAIM_VERIFICATION_STATUS, "VERIFICATION_POLICY_ONLY_NO_EXTERNAL_API_CALL", "status")

    assert_equal(social_claim_normalize_platform("twitter"), SOCIAL_PLATFORM_X, "twitter maps to X")
    assert_equal(social_claim_normalize_platform("site"), SOCIAL_PLATFORM_WEBSITE, "site maps to website")

    x_policy = social_claim_policy("X")
    assert_true(x_policy["supported"] is True, "X supported")
    assert_equal(x_policy["requirements"]["action"], SOCIAL_ACTION_FOLLOW_X_ACCOUNT, "X action")
    assert_true(x_policy["requirements"]["requiresProviderVerified"] is True, "X provider verification required")

    website_policy = social_claim_policy("website")
    assert_true(website_policy["supported"] is True, "website supported")
    assert_equal(website_policy["requirements"]["action"], SOCIAL_ACTION_WEBSITE_NONCE_VERIFY, "website action")
    assert_true(website_policy["requirements"]["requiresServerNonceVerified"] is True, "website nonce verification required")

    safe = x_policy["safeBoundary"]
    assert_true(safe["policyOnly"] is True, "policy only")
    assert_true(safe["offChainOnly"] is True, "off-chain only")
    assert_true(safe["noExternalApiCall"] is True, "no external api call")
    assert_true(safe["noBrowserAutomation"] is True, "no browser automation")
    assert_true(safe["noPasswordCollection"] is True, "no password collection")
    assert_true(safe["noOAuthTokenStorage"] is True, "no oauth token storage")
    assert_true(safe["productionRequiresProviderVerification"] is True, "provider verification required")
    assert_true(safe["productionRequiresServerVerification"] is True, "server verification required")
    assert_true(safe["userClaimOnlyNotEnough"] is True, "user claim only not enough")
    assert_true(safe["screenshotOnlyNotEnough"] is True, "screenshot only not enough")
    assert_true(safe["urlOnlyNotEnough"] is True, "url only not enough")
    assert_true(safe["rewardRequiresVerifiedProof"] is True, "reward requires verified proof")
    assert_true(safe["duplicateNonceHandledByNextLayer"] is True, "duplicate nonce next layer")
    assert_true(safe["perPlatformOneRewardHandledByNextLayer"] is True, "per platform next layer")

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

    dev_claim = {
        "userId": "dev_user",
        "claimId": "dev_claim_1",
        "platform": "X",
        "action": SOCIAL_ACTION_FOLLOW_X_ACCOUNT,
        "devSimulation": True,
    }
    dev_result = social_claim_evaluate(dev_claim, SOCIAL_CLAIM_MODE_DEV)
    assert_equal(dev_result["status"], SOCIAL_CLAIM_DEV_SIMULATION, "dev simulation accepted")
    assert_true(dev_result["rewardEligible"] is False, "dev simulation no final reward")

    x_verified = {
        "userId": "dev_user",
        "claimId": "x_claim_1",
        "platform": "X",
        "action": SOCIAL_ACTION_FOLLOW_X_ACCOUNT,
        "xHandle": "@dev_user",
        "targetAccount": "@QuantumCoin",
        "providerVerified": True,
        "followVerified": True,
    }
    x_result = social_claim_evaluate(x_verified, SOCIAL_CLAIM_MODE_FINAL)
    assert_equal(x_result["status"], SOCIAL_CLAIM_VERIFIED, "X verified")
    assert_true(x_result["rewardEligible"] is True, "X reward eligible")
    assert_true("verified_social_claim_eligible_for_pending_tgwt" in x_result["reasons"], "X verified reason")

    x_evidence_only = {
        "userId": "dev_user",
        "claimId": "x_claim_2",
        "platform": "X",
        "action": SOCIAL_ACTION_FOLLOW_X_ACCOUNT,
        "xHandle": "@dev_user",
        "targetAccount": "@QuantumCoin",
        "evidenceUrl": "https://x.com/dev_user",
        "providerVerified": False,
        "followVerified": False,
    }
    x_pending = social_claim_evaluate(x_evidence_only, SOCIAL_CLAIM_MODE_FINAL)
    assert_equal(x_pending["status"], SOCIAL_CLAIM_PENDING_PROVIDER_VERIFICATION, "X pending evidence")
    assert_true(x_pending["rewardEligible"] is False, "X pending no reward")

    website_verified = {
        "userId": "dev_user",
        "claimId": "web_claim_1",
        "platform": "WEBSITE",
        "action": SOCIAL_ACTION_WEBSITE_NONCE_VERIFY,
        "websiteHost": "quantumcoin.example",
        "websiteNonce": "nonce_1",
        "serverNonceVerified": True,
    }
    web_result = social_claim_evaluate(website_verified, SOCIAL_CLAIM_MODE_FINAL)
    assert_equal(web_result["status"], SOCIAL_CLAIM_VERIFIED, "website verified")
    assert_true(web_result["rewardEligible"] is True, "website reward eligible")

    website_missing_nonce = {
        "userId": "dev_user",
        "claimId": "web_claim_2",
        "platform": "WEBSITE",
        "action": SOCIAL_ACTION_WEBSITE_NONCE_VERIFY,
        "websiteHost": "quantumcoin.example",
        "serverNonceVerified": False,
    }
    web_rejected = social_claim_evaluate(website_missing_nonce, SOCIAL_CLAIM_MODE_FINAL)
    assert_equal(web_rejected["status"], SOCIAL_CLAIM_REJECTED, "website missing nonce rejected")
    assert_true("missing_website_nonce" in web_rejected["reasons"], "missing website nonce reason")

    wrong_action = dict(x_verified)
    wrong_action["claimId"] = "x_claim_3"
    wrong_action["action"] = "LIKE_POST"
    wrong_result = social_claim_evaluate(wrong_action, SOCIAL_CLAIM_MODE_FINAL)
    assert_equal(wrong_result["status"], SOCIAL_CLAIM_REJECTED, "wrong action rejected")
    assert_true("wrong_social_action" in wrong_result["reasons"], "wrong action reason")

    unsupported = {
        "userId": "dev_user",
        "claimId": "bad_claim",
        "platform": "UNKNOWN",
        "action": "FOLLOW",
    }
    unsupported_result = social_claim_evaluate(unsupported, SOCIAL_CLAIM_MODE_FINAL)
    assert_equal(unsupported_result["status"], SOCIAL_CLAIM_REJECTED, "unsupported platform rejected")
    assert_true("unsupported_platform" in unsupported_result["reasons"], "unsupported reason")

    print("SOCIAL_CLAIM_VERIFICATION_V1_TEST_PASS")


if __name__ == "__main__":
    main()
