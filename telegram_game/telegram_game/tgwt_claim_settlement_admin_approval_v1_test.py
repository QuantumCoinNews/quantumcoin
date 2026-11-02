from tgwt_claim_settlement_precheck import (
    TGWT_CLAIM_SETTLEMENT_READY_FOR_ADMIN_REVIEW,
    TGWT_CLAIM_SETTLEMENT_CONFIG_REQUIRED,
)
from tgwt_claim_settlement_admin_approval import (
    TGWT_CLAIM_SETTLEMENT_ADMIN_APPROVAL_VERSION,
    TGWT_CLAIM_SETTLEMENT_ADMIN_APPROVAL_STATUS,
    TGWT_ADMIN_DECISION_APPROVE,
    TGWT_ADMIN_DECISION_HOLD,
    TGWT_ADMIN_DECISION_REJECT,
    TGWT_SETTLEMENT_ADMIN_APPROVED,
    TGWT_SETTLEMENT_ADMIN_HOLD,
    TGWT_SETTLEMENT_ADMIN_REJECTED,
    TGWT_SETTLEMENT_ADMIN_PRECHECK_REQUIRED,
    TGWT_SETTLEMENT_ADMIN_INVALID_DECISION,
    tgwt_claim_settlement_admin_policy,
    tgwt_claim_settlement_admin_decision,
    tgwt_claim_settlement_ui_payload,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def ready_precheck(**overrides):
    data = {
        "status": TGWT_CLAIM_SETTLEMENT_READY_FOR_ADMIN_REVIEW,
        "reasons": ["precheck_passed_ready_for_admin_review"],
        "warnings": [],
        "userId": "dev_user",
        "claimId": "claim_1",
        "bscAddress": "0x1111111111111111111111111111111111111111",
        "amountTGWT": 100,
        "claimableTGWT": 100,
        "readyForAdminReview": True,
        "readyForAutomaticTransfer": False,
        "createsBscTransaction": False,
        "backendPaysGas": False,
        "backendSendsToken": False,
    }
    data.update(overrides)
    return data


def main():
    assert_equal(TGWT_CLAIM_SETTLEMENT_ADMIN_APPROVAL_VERSION, "TGWT_CLAIM_SETTLEMENT_ADMIN_APPROVAL_V1", "version")
    assert_equal(TGWT_CLAIM_SETTLEMENT_ADMIN_APPROVAL_STATUS, "ADMIN_APPROVAL_ONLY_NO_AUTOMATIC_TRANSFER", "status")

    policy = tgwt_claim_settlement_admin_policy()
    assert_true(TGWT_ADMIN_DECISION_APPROVE in policy["allowedDecisions"], "approve decision allowed")
    assert_true(TGWT_ADMIN_DECISION_HOLD in policy["allowedDecisions"], "hold decision allowed")
    assert_true(TGWT_ADMIN_DECISION_REJECT in policy["allowedDecisions"], "reject decision allowed")
    assert_equal(policy["approvalMeaning"], "USER_PAID_BSC_CLAIM_CAN_BE_PREPARED_BY_UI", "approval meaning")
    assert_equal(policy["transferMeaning"], "NO_BACKEND_TRANSFER_NO_BACKEND_GAS_NO_SIGNING", "transfer meaning")

    safe = policy["safeBoundary"]
    assert_true(safe["adminApprovalOnly"] is True, "admin approval only")
    assert_true(safe["offChainOnly"] is True, "off-chain only")
    assert_true(safe["requiresPassedPrecheck"] is True, "requires passed precheck")
    assert_true(safe["manualAdminDecisionRequired"] is True, "manual admin decision required")
    assert_true(safe["approveMeansUserPaidClaimReady"] is True, "approve means user paid claim ready")
    assert_true(safe["noAutomaticApproval"] is True, "no automatic approval")
    assert_true(safe["noAutomaticTransfer"] is True, "no automatic transfer")
    assert_true(safe["noBackendTokenSend"] is True, "no backend token send")
    assert_true(safe["noBackendGasPayment"] is True, "no backend gas payment")
    assert_true(safe["noPrivateKeyStorage"] is True, "no private key storage")
    assert_true(safe["noSigning"] is True, "no signing")
    assert_true(safe["noRpcCall"] is True, "no rpc call")
    assert_true(safe["noWalletProviderCall"] is True, "no wallet provider call")
    assert_true(safe["noContractCall"] is True, "no contract call")
    assert_true(safe["userPaysBnbGasLater"] is True, "user pays bnb gas later")
    assert_true(safe["userSignsClaimLater"] is True, "user signs claim later")
    assert_true(safe["claimGatewayOnly"] is True, "claim gateway only")

    false_keys = [
        "automaticTransferAllowed",
        "backendPaysGas",
        "backendSendsToken",
        "createsBscTransaction",
        "deploysContract",
        "broadcastsTransaction",
        "storesPrivateKey",
        "signsTransaction",
        "changesQCBalances",
        "changesTGWTOnChainBalances",
        "changesMempool",
        "changesChain",
        "changesConsensus",
        "changesTotalSupply",
        "startsMining",
        "stopsMining",
    ]

    for key in false_keys:
        assert_true(safe[key] is False, f"{key} must be false")

    approved = tgwt_claim_settlement_admin_decision(
        ready_precheck(),
        TGWT_ADMIN_DECISION_APPROVE,
        "admin_1",
        "checked",
    )
    assert_equal(approved["status"], TGWT_SETTLEMENT_ADMIN_APPROVED, "approved")
    assert_true(approved["approvedForUserPaidClaim"] is True, "approved flag")
    assert_true(approved["readyForAutomaticTransfer"] is False, "no auto transfer")
    assert_true(approved["createsBscTransaction"] is False, "no bsc transaction")
    assert_true(approved["backendPaysGas"] is False, "backend no gas")
    assert_true(approved["backendSendsToken"] is False, "backend no send")
    assert_true("admin_approved_user_paid_claim_ready_no_transfer" in approved["reasons"], "approval reason")

    payload = tgwt_claim_settlement_ui_payload(approved)
    assert_equal(payload["status"], "CLAIM_UI_READY_USER_PAID", "ui ready")
    assert_true(payload["requiresUserWallet"] is True, "requires user wallet")
    assert_true(payload["requiresUserSignature"] is True, "requires user signature")
    assert_true(payload["requiresUserBnbGas"] is True, "requires user bnb gas")
    assert_true(payload["backendCreatesTransaction"] is False, "ui no backend tx")
    assert_true(payload["backendPaysGas"] is False, "ui no backend gas")
    assert_true(payload["backendSendsToken"] is False, "ui no backend send")
    assert_true(payload["readyForAutomaticTransfer"] is False, "ui no automatic transfer")

    hold = tgwt_claim_settlement_admin_decision(
        ready_precheck(),
        TGWT_ADMIN_DECISION_HOLD,
        "admin_1",
        "manual review",
    )
    assert_equal(hold["status"], TGWT_SETTLEMENT_ADMIN_HOLD, "hold")
    assert_true(hold["approvedForUserPaidClaim"] is False, "hold not approved")

    rejected = tgwt_claim_settlement_admin_decision(
        ready_precheck(),
        TGWT_ADMIN_DECISION_REJECT,
        "admin_1",
        "bad claim",
    )
    assert_equal(rejected["status"], TGWT_SETTLEMENT_ADMIN_REJECTED, "rejected")
    assert_true(rejected["approvedForUserPaidClaim"] is False, "reject not approved")

    invalid_decision = tgwt_claim_settlement_admin_decision(
        ready_precheck(),
        "SEND_NOW",
        "admin_1",
        "bad decision",
    )
    assert_equal(invalid_decision["status"], TGWT_SETTLEMENT_ADMIN_INVALID_DECISION, "invalid decision")
    assert_true("invalid_admin_decision" in invalid_decision["reasons"], "invalid decision reason")

    not_ready = ready_precheck(status=TGWT_CLAIM_SETTLEMENT_CONFIG_REQUIRED, readyForAdminReview=False)
    precheck_required = tgwt_claim_settlement_admin_decision(
        not_ready,
        TGWT_ADMIN_DECISION_APPROVE,
        "admin_1",
        "not ready",
    )
    assert_equal(precheck_required["status"], TGWT_SETTLEMENT_ADMIN_PRECHECK_REQUIRED, "precheck required")
    assert_true("precheck_not_ready_for_admin_review" in precheck_required["reasons"], "precheck reason")

    unsafe_precheck = ready_precheck(readyForAutomaticTransfer=True)
    unsafe = tgwt_claim_settlement_admin_decision(
        unsafe_precheck,
        TGWT_ADMIN_DECISION_APPROVE,
        "admin_1",
        "unsafe",
    )
    assert_equal(unsafe["status"], TGWT_SETTLEMENT_ADMIN_PRECHECK_REQUIRED, "unsafe precheck required")
    assert_true("automatic_transfer_flag_not_allowed" in unsafe["reasons"], "automatic transfer reason")

    no_admin = tgwt_claim_settlement_admin_decision(
        ready_precheck(),
        TGWT_ADMIN_DECISION_APPROVE,
        "",
        "missing admin",
    )
    assert_equal(no_admin["status"], TGWT_SETTLEMENT_ADMIN_PRECHECK_REQUIRED, "missing admin precheck required")
    assert_true("missing_admin_id" in no_admin["reasons"], "missing admin reason")

    print("TGWT_CLAIM_SETTLEMENT_ADMIN_APPROVAL_V1_TEST_PASS")


if __name__ == "__main__":
    main()
