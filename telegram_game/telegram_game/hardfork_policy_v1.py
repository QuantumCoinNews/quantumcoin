# QC_HARDFORK_POLICY_V1
# Hardfork policy and emergency network upgrade rules.
#
# Boundary:
# - Control/validation only.
# - Does not execute hardfork.
# - Does not change consensus now.
# - Does not mint, change max supply, replace chain, or change balances.

from typing import Any, Dict, List

QC_HARDFORK_POLICY_VERSION = "QC_HARDFORK_POLICY_V1"

ALLOWED_REASONS = [
    "CRITICAL_CONSENSUS_BUG",
    "CHAIN_HALT_OR_FINALITY_FAILURE",
    "CRITICAL_SECURITY_VULNERABILITY",
    "CRYPTOGRAPHIC_ALGORITHM_BREAK_OR_URGENT_ROTATION",
    "STATE_CORRUPTION_REPAIR_WITH_AUDITED_SNAPSHOT",
    "NETWORK_WIDE_PROTOCOL_UPGRADE_WITH_PUBLIC_NOTICE",
]

FORBIDDEN_REASONS = [
    "CHANGE_MAX_SUPPLY",
    "MINT_EXTRA_QC",
    "RESET_FOUNDER_VESTING",
    "BYPASS_BURN_RULES",
    "STEAL_OR_REASSIGN_BALANCES",
    "CHANGE_ECONOMY_FOR_CONVENIENCE",
    "ADMIN_SHORTCUT",
    "AI_AUTOMATION_ONLY",
    "GAME_OR_TGWT_REQUEST_ONLY",
]

SAFE_BOUNDARY = {
    "doesNotExecuteHardfork": True,
    "doesNotChangeConsensusNow": True,
    "doesNotMint": True,
    "doesNotChangeMaxSupply": True,
    "doesNotChangeBalances": True,
    "doesNotBypassFounderVesting": True,
    "doesNotBypassBurnRules": True,
    "doesNotStartOrStopNodes": True,
    "doesNotReplaceChain": True,
    "requiresFounderApproval": True,
    "requiresAdminMultistepApproval": True,
    "requiresPublicNotice": True,
    "requiresTestnetRehearsal": True,
    "requiresMigrationHeight": True,
    "requiresSnapshotHash": True,
    "requiresRollbackPlan": True,
    "requiresNodeUpgradeWindow": True,
    "requiresReleaseBinaryHashes": True,
    "aiCannotRunHardfork": True,
    "workerCannotRunHardfork": True,
    "gameCannotRunHardfork": True,
    "adminPanelAloneCannotRunHardfork": True,
    "controlOnly": True,
}

def default_hardfork_policy_v1() -> Dict[str, Any]:
    return {
        "version": QC_HARDFORK_POLICY_VERSION,
        "mode": "HARDFORK_POLICY_CONTROL_ONLY_NO_FORK_EXECUTION",
        "hardforkAllowed": True,
        "hardforkExecutionNow": False,
        "allowedReasons": list(ALLOWED_REASONS),
        "forbiddenReasons": list(FORBIDDEN_REASONS),
        "requiresFounderApproval": True,
        "requiresAdminMultistepApproval": True,
        "requiresPublicNotice": True,
        "requiresTestnetRehearsal": True,
        "requiresMigrationHeight": True,
        "requiresSnapshotHash": True,
        "requiresRollbackPlan": True,
        "requiresNodeUpgradeWindow": True,
        "requiresReleaseBinaryHashes": True,
        "requiresPostForkAudit": True,
        "hardforkCannotChangeMaxSupply": True,
        "hardforkCannotMintExtraQc": True,
        "hardforkCannotBypassFounderVesting": True,
        "hardforkCannotBypassBurnRules": True,
        "hardforkCannotStealOrReassignBalances": True,
        "hardforkCannotAutoRunByAi": True,
        "hardforkCannotAutoRunByWorker": True,
        "hardforkCannotAutoRunByGame": True,
        "hardforkCannotAutoRunByAdminPanelOnly": True,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def validate_hardfork_policy_v1(policy: Dict[str, Any]) -> Dict[str, Any]:
    errors: List[str] = []

    if policy.get("version") != QC_HARDFORK_POLICY_VERSION:
        errors.append("invalid_version")
    if policy.get("mode") != "HARDFORK_POLICY_CONTROL_ONLY_NO_FORK_EXECUTION":
        errors.append("invalid_mode")
    if policy.get("hardforkAllowed") is not True:
        errors.append("hardfork_must_remain_available_for_emergency")
    if policy.get("hardforkExecutionNow") is not False:
        errors.append("hardfork_execution_now_forbidden")
    if len(policy.get("allowedReasons") or []) < 6:
        errors.append("allowed_reasons_missing")
    if len(policy.get("forbiddenReasons") or []) < 9:
        errors.append("forbidden_reasons_missing")
    for key in [
        "requiresFounderApproval",
        "requiresAdminMultistepApproval",
        "requiresPublicNotice",
        "requiresTestnetRehearsal",
        "requiresMigrationHeight",
        "requiresSnapshotHash",
        "requiresRollbackPlan",
        "requiresNodeUpgradeWindow",
        "requiresReleaseBinaryHashes",
        "requiresPostForkAudit",
        "hardforkCannotChangeMaxSupply",
        "hardforkCannotMintExtraQc",
        "hardforkCannotBypassFounderVesting",
        "hardforkCannotBypassBurnRules",
        "hardforkCannotStealOrReassignBalances",
        "hardforkCannotAutoRunByAi",
        "hardforkCannotAutoRunByWorker",
        "hardforkCannotAutoRunByGame",
        "hardforkCannotAutoRunByAdminPanelOnly",
    ]:
        if policy.get(key) is not True:
            errors.append(key + "_required")

    safe = policy.get("safeBoundary") or {}
    for key in [
        "doesNotExecuteHardfork",
        "doesNotChangeConsensusNow",
        "doesNotMint",
        "doesNotChangeMaxSupply",
        "doesNotChangeBalances",
        "doesNotBypassFounderVesting",
        "doesNotBypassBurnRules",
        "doesNotStartOrStopNodes",
        "doesNotReplaceChain",
        "requiresFounderApproval",
        "requiresAdminMultistepApproval",
        "requiresPublicNotice",
        "requiresTestnetRehearsal",
        "requiresMigrationHeight",
        "requiresSnapshotHash",
        "requiresRollbackPlan",
        "requiresNodeUpgradeWindow",
        "requiresReleaseBinaryHashes",
        "aiCannotRunHardfork",
        "workerCannotRunHardfork",
        "gameCannotRunHardfork",
        "adminPanelAloneCannotRunHardfork",
        "controlOnly",
    ]:
        if safe.get(key) is not True:
            errors.append("safe_boundary_missing:" + key)

    return {
        "ok": len(errors) == 0,
        "version": QC_HARDFORK_POLICY_VERSION,
        "errors": errors,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def hardfork_policy_summary_v1() -> Dict[str, Any]:
    policy = default_hardfork_policy_v1()
    return {
        "ok": True,
        "version": QC_HARDFORK_POLICY_VERSION,
        "policy": policy,
        "validation": validate_hardfork_policy_v1(policy),
    }
