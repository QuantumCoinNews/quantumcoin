package main

import (
	"net/http"
	"quantumcoin/blockchain"
	"quantumcoin/consensus"
	qnode "quantumcoin/node"
)

func handleRuntimePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "method not allowed")
		return
	}
	chain, configuration := nodeRuntime.Snapshot()
	if chain == nil || configuration == nil {
		writeError(w, 503, "runtime unavailable")
		return
	}
	context, err := chain.RuntimePolicySnapshot()
	if err != nil {
		writeError(w, 503, "runtime policy unavailable")
		return
	}
	declarations := qnode.DefaultRuntimePolicySetV1()
	writeOK(w, map[string]any{
		"version": "NODE_RUNTIME_POLICY_V1", "source": "quantumcoin_go_node", "context": context,
		"sections": map[string]any{
			"mainnet-before": declarations.Economy, "burn-sink": declarations.Burn,
			"network-fee": declarations.NetworkFee, "hardfork-policy": declarations.Hardfork,
			"founder-vesting": map[string]any{"declaration": declarations.FounderVesting,
				"configuredCliffMonths":    configuration.FounderVestingCliffMonths,
				"configuredDurationMonths": configuration.FounderVestingDurationMonths,
				"configuredLiquidAddress":  configuration.PremineLiquidAddress, "configuredLockedAddress": configuration.PremineLockedAddress},
			"mining-transition": map[string]any{"halvingIntervalBlocks": consensus.LegacyHalvingIntervalBlocks,
				"minimumRewardQc": blockchain.CanonicalMinimumMiningRewardV1, "remainingSupplyDirectDistribution": false,
				"nextSubsidyProtocolUnits": context["nextSubsidyProtocolUnits"]},
		},
		"authority": "Chain context describes the running node. Policy sections are Go startup declarations, not proof that every declared feature is active. Effective denomination and hardfork state are in context.",
	})
}
