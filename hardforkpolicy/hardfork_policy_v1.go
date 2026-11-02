// Package hardforkpolicy contains QuantumCoin project code for this module.
package hardforkpolicy

import (
	"errors"
	"strings"
)

const HardforkPolicyVersionV1 = "QC_HARDFORK_POLICY_V1"

type HardforkPolicy struct {
	Version                               string          `json:"version"`
	Mode                                  string          `json:"mode"`
	HardforkAllowed                       bool            `json:"hardforkAllowed"`
	HardforkExecutionNow                  bool            `json:"hardforkExecutionNow"`
	AllowedReasons                        []string        `json:"allowedReasons"`
	ForbiddenReasons                      []string        `json:"forbiddenReasons"`
	RequiresFounderApproval               bool            `json:"requiresFounderApproval"`
	RequiresAdminMultistepApproval        bool            `json:"requiresAdminMultistepApproval"`
	RequiresPublicNotice                  bool            `json:"requiresPublicNotice"`
	RequiresTestnetRehearsal              bool            `json:"requiresTestnetRehearsal"`
	RequiresMigrationHeight               bool            `json:"requiresMigrationHeight"`
	RequiresSnapshotHash                  bool            `json:"requiresSnapshotHash"`
	RequiresRollbackPlan                  bool            `json:"requiresRollbackPlan"`
	RequiresNodeUpgradeWindow             bool            `json:"requiresNodeUpgradeWindow"`
	RequiresReleaseBinaryHashes           bool            `json:"requiresReleaseBinaryHashes"`
	RequiresPostForkAudit                 bool            `json:"requiresPostForkAudit"`
	HardforkCannotChangeMaxSupply         bool            `json:"hardforkCannotChangeMaxSupply"`
	HardforkCannotMintExtraQC             bool            `json:"hardforkCannotMintExtraQc"`
	HardforkCannotBypassFounderVesting    bool            `json:"hardforkCannotBypassFounderVesting"`
	HardforkCannotBypassBurnRules         bool            `json:"hardforkCannotBypassBurnRules"`
	HardforkCannotStealOrReassignBalances bool            `json:"hardforkCannotStealOrReassignBalances"`
	HardforkCannotAutoRunByAI             bool            `json:"hardforkCannotAutoRunByAi"`
	HardforkCannotAutoRunByWorker         bool            `json:"hardforkCannotAutoRunByWorker"`
	HardforkCannotAutoRunByGame           bool            `json:"hardforkCannotAutoRunByGame"`
	HardforkCannotAutoRunByAdminPanelOnly bool            `json:"hardforkCannotAutoRunByAdminPanelOnly"`
	SafeBoundary                          map[string]bool `json:"safeBoundary"`
}

func DefaultAllowedReasonsV1() []string {
	return []string{
		"CRITICAL_CONSENSUS_BUG",
		"CHAIN_HALT_OR_FINALITY_FAILURE",
		"CRITICAL_SECURITY_VULNERABILITY",
		"CRYPTOGRAPHIC_ALGORITHM_BREAK_OR_URGENT_ROTATION",
		"STATE_CORRUPTION_REPAIR_WITH_AUDITED_SNAPSHOT",
		"NETWORK_WIDE_PROTOCOL_UPGRADE_WITH_PUBLIC_NOTICE",
	}
}

func DefaultForbiddenReasonsV1() []string {
	return []string{
		"CHANGE_MAX_SUPPLY",
		"MINT_EXTRA_QC",
		"RESET_FOUNDER_VESTING",
		"BYPASS_BURN_RULES",
		"STEAL_OR_REASSIGN_BALANCES",
		"CHANGE_ECONOMY_FOR_CONVENIENCE",
		"ADMIN_SHORTCUT",
		"AI_AUTOMATION_ONLY",
		"GAME_OR_TGWT_REQUEST_ONLY",
	}
}

func DefaultPolicyV1() HardforkPolicy {
	return HardforkPolicy{
		Version:                               HardforkPolicyVersionV1,
		Mode:                                  "HARDFORK_POLICY_CONTROL_ONLY_NO_FORK_EXECUTION",
		HardforkAllowed:                       true,
		HardforkExecutionNow:                  false,
		AllowedReasons:                        DefaultAllowedReasonsV1(),
		ForbiddenReasons:                      DefaultForbiddenReasonsV1(),
		RequiresFounderApproval:               true,
		RequiresAdminMultistepApproval:        true,
		RequiresPublicNotice:                  true,
		RequiresTestnetRehearsal:              true,
		RequiresMigrationHeight:               true,
		RequiresSnapshotHash:                  true,
		RequiresRollbackPlan:                  true,
		RequiresNodeUpgradeWindow:             true,
		RequiresReleaseBinaryHashes:           true,
		RequiresPostForkAudit:                 true,
		HardforkCannotChangeMaxSupply:         true,
		HardforkCannotMintExtraQC:             true,
		HardforkCannotBypassFounderVesting:    true,
		HardforkCannotBypassBurnRules:         true,
		HardforkCannotStealOrReassignBalances: true,
		HardforkCannotAutoRunByAI:             true,
		HardforkCannotAutoRunByWorker:         true,
		HardforkCannotAutoRunByGame:           true,
		HardforkCannotAutoRunByAdminPanelOnly: true,
		SafeBoundary:                          SafeBoundaryV1(),
	}
}

func SafeBoundaryV1() map[string]bool {
	return map[string]bool{
		"doesNotExecuteHardfork":           true,
		"doesNotChangeConsensusNow":        true,
		"doesNotMint":                      true,
		"doesNotChangeMaxSupply":           true,
		"doesNotChangeBalances":            true,
		"doesNotBypassFounderVesting":      true,
		"doesNotBypassBurnRules":           true,
		"doesNotStartOrStopNodes":          true,
		"doesNotReplaceChain":              true,
		"requiresFounderApproval":          true,
		"requiresAdminMultistepApproval":   true,
		"requiresPublicNotice":             true,
		"requiresTestnetRehearsal":         true,
		"requiresMigrationHeight":          true,
		"requiresSnapshotHash":             true,
		"requiresRollbackPlan":             true,
		"requiresNodeUpgradeWindow":        true,
		"requiresReleaseBinaryHashes":      true,
		"aiCannotRunHardfork":              true,
		"workerCannotRunHardfork":          true,
		"gameCannotRunHardfork":            true,
		"adminPanelAloneCannotRunHardfork": true,
		"controlOnly":                      true,
	}
}

func ValidatePolicyV1(policy HardforkPolicy) error {
	if strings.TrimSpace(policy.Version) != HardforkPolicyVersionV1 {
		return errors.New("invalid hardfork policy version")
	}
	if strings.TrimSpace(policy.Mode) != "HARDFORK_POLICY_CONTROL_ONLY_NO_FORK_EXECUTION" {
		return errors.New("invalid hardfork policy mode")
	}
	if !policy.HardforkAllowed {
		return errors.New("hardfork must remain possible for emergencies")
	}
	if policy.HardforkExecutionNow {
		return errors.New("this policy step must not execute a hardfork")
	}
	if err := validateReasonSetsV1(policy.AllowedReasons, policy.ForbiddenReasons); err != nil {
		return err
	}
	if !policy.RequiresFounderApproval || !policy.RequiresAdminMultistepApproval || !policy.RequiresPublicNotice {
		return errors.New("approval and public notice requirements missing")
	}
	if !policy.RequiresTestnetRehearsal || !policy.RequiresMigrationHeight || !policy.RequiresSnapshotHash || !policy.RequiresRollbackPlan {
		return errors.New("testnet rehearsal, migration, snapshot, and rollback requirements missing")
	}
	if !policy.RequiresNodeUpgradeWindow || !policy.RequiresReleaseBinaryHashes || !policy.RequiresPostForkAudit {
		return errors.New("upgrade window, binary hash, and post-fork audit requirements missing")
	}
	if !policy.HardforkCannotChangeMaxSupply || !policy.HardforkCannotMintExtraQC {
		return errors.New("hardfork must not change max supply or mint extra QC")
	}
	if !policy.HardforkCannotBypassFounderVesting || !policy.HardforkCannotBypassBurnRules {
		return errors.New("hardfork must not bypass founder vesting or burn rules")
	}
	if !policy.HardforkCannotStealOrReassignBalances {
		return errors.New("hardfork must not steal or reassign balances")
	}
	if !policy.HardforkCannotAutoRunByAI || !policy.HardforkCannotAutoRunByWorker || !policy.HardforkCannotAutoRunByGame || !policy.HardforkCannotAutoRunByAdminPanelOnly {
		return errors.New("automation must not run hardfork")
	}
	if err := validateExactSafeBoundaryMapV1(
		policy.SafeBoundary,
		SafeBoundaryV1(),
	); err != nil {
		return err
	}

	for _, k := range []string{
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
	} {
		if !policy.SafeBoundary[k] {
			return errors.New("safe boundary missing: " + k)
		}
	}

	return nil
}

func validateExactSafeBoundaryMapV1(
	boundary map[string]bool,
	expected map[string]bool,
) error {
	if boundary == nil {
		return errors.New("hardfork policy safe boundary required")
	}

	if len(boundary) != len(expected) {
		return errors.New("hardfork policy safe boundary key count mismatch")
	}

	for key, want := range expected {
		got, exists := boundary[key]
		if !exists {
			return errors.New("safe boundary missing: " + key)
		}
		if got != want {
			return errors.New("safe boundary value mismatch: " + key)
		}
	}

	for key := range boundary {
		if _, exists := expected[key]; !exists {
			return errors.New("unexpected safe boundary key: " + key)
		}
	}

	return nil
}

// validateReasonSetsV1 validates sets, not just their lengths. Order is not
// meaningful for this control-only policy; spelling and membership are exact.
func validateReasonSetsV1(allowed, forbidden []string) error {
	forbiddenSet := make(map[string]bool, len(forbidden))
	for _, reason := range forbidden {
		forbiddenSet[reason] = true
	}
	for _, reason := range allowed {
		if forbiddenSet[reason] {
			return errors.New("conflicting allowed and forbidden reason: " + reason)
		}
	}
	if err := validateExactReasonSetV1(allowed, DefaultAllowedReasonsV1(), "allowed"); err != nil {
		return err
	}
	return validateExactReasonSetV1(forbidden, DefaultForbiddenReasonsV1(), "forbidden")
}

func validateExactReasonSetV1(reasons, canonical []string, kind string) error {
	expected := make(map[string]bool, len(canonical))
	for _, reason := range canonical {
		expected[reason] = true
	}
	seen := make(map[string]bool, len(reasons))
	for _, reason := range reasons {
		if !expected[reason] {
			return errors.New("unknown " + kind + " reason: " + reason)
		}
		if seen[reason] {
			return errors.New("duplicate " + kind + " reason: " + reason)
		}
		seen[reason] = true
	}
	if len(seen) != len(expected) {
		return errors.New("incomplete " + kind + " reason set")
	}
	return nil
}
