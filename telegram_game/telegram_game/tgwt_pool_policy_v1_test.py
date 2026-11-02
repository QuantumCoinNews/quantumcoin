from tgwt_pool_policy import (
    TGWT_POOL_POLICY_VERSION,
    TGWT_POOL_STATUS,
    TGWT_UNIT,
    TGWT_POOL_TOTAL,
    TGWT_POOL_TOTAL_U,
    tgwt_pool_policy,
    tgwt_policy_cap_ok,
    tgwt_policy_remaining_u,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def main():
    policy = tgwt_pool_policy()

    assert_equal(TGWT_POOL_POLICY_VERSION, "TGWT_POOL_POLICY_V1", "version constant")
    assert_equal(TGWT_POOL_STATUS, "POLICY_DEFINED_POOL_NOT_CREATED", "status constant")
    assert_equal(TGWT_UNIT, 1_000_000, "unit")
    assert_equal(TGWT_POOL_TOTAL, 3_000_000, "total TGWT")
    assert_equal(TGWT_POOL_TOTAL_U, 3_000_000 * 1_000_000, "total micro unit")

    assert_equal(policy["version"], "TGWT_POOL_POLICY_V1", "policy version")
    assert_equal(policy["status"], "POLICY_DEFINED_POOL_NOT_CREATED", "policy status")
    assert_equal(policy["total"], "3000000", "policy total")
    assert_equal(policy["total_u"], 3_000_000 * 1_000_000, "policy total_u")
    assert_true(policy["offChainOnly"] is True, "offChainOnly must be true")

    safe = policy["safeBoundary"]
    assert_true(safe["policyOnly"] is True, "policyOnly must be true")

    false_keys = [
        "createsPool",
        "createsLedger",
        "distributesRewards",
        "changesTGWTBalances",
        "changesQCBalances",
        "performsTransfer",
        "mintsToken",
        "burnsToken",
        "startsWithdraw",
        "approvesWithdraw",
        "bscPayout",
        "realPayout",
        "changesChain",
        "replacesChain",
        "changesConsensus",
        "changesTotalSupply",
        "startsMining",
        "stopsMining",
    ]

    for key in false_keys:
        assert_true(safe[key] is False, f"{key} must be false")

    assert_true(tgwt_policy_cap_ok(1), "1 micro unit should fit")
    assert_true(tgwt_policy_cap_ok(TGWT_POOL_TOTAL_U), "full cap should fit from zero")
    assert_true(not tgwt_policy_cap_ok(TGWT_POOL_TOTAL_U + 1), "over cap should fail")
    assert_true(not tgwt_policy_cap_ok(0), "zero amount should fail")
    assert_true(not tgwt_policy_cap_ok(-1), "negative amount should fail")
    assert_true(not tgwt_policy_cap_ok(1, TGWT_POOL_TOTAL_U), "no remaining capacity should fail")
    assert_true(tgwt_policy_cap_ok(1, TGWT_POOL_TOTAL_U - 1), "last micro unit should fit")

    assert_equal(tgwt_policy_remaining_u(0), TGWT_POOL_TOTAL_U, "remaining from zero")
    assert_equal(tgwt_policy_remaining_u(TGWT_POOL_TOTAL_U), 0, "remaining at cap")
    assert_equal(tgwt_policy_remaining_u(TGWT_POOL_TOTAL_U + 1), 0, "remaining over cap")
    assert_equal(tgwt_policy_remaining_u(-1), 0, "remaining with negative reserved")

    print("TGWT_POOL_POLICY_V1_TEST_PASS")


if __name__ == "__main__":
    main()
