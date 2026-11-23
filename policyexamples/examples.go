// Package policyexamples generates documentation fixtures from Go authority.
// Generated examples are not node configuration files or activation approvals.
package policyexamples

import (
	"encoding/json"
	"quantumcoin/backuprestore"
	"quantumcoin/burncontrol"
	"quantumcoin/consensus"
	"quantumcoin/economycontrol"
	"quantumcoin/foundervesting"
	"quantumcoin/miningtransition"
	"quantumcoin/networkfee"
)

func Files() (map[string][]byte, error) {
	values := map[string]any{
		"network_fee_policy.example.json":             networkfee.DefaultPolicyV1(),
		"burn_sink_control.example.json":              burncontrol.DefaultPolicyV1(),
		"founder_vesting_policy.example.json":         foundervesting.DefaultPolicyV1(),
		"mainnet_before_economy_control.example.json": economycontrol.DefaultPolicyV1(),
		"mining_node_transition.example.json":         miningtransition.DefaultPolicyV1(),
		"backup_restore.runbook.example.json":         backuprestore.DefaultRunbookV1(),
		"testnet.config.skeleton.json": map[string]any{
			"version": "QC_TESTNET_CONFIG_SKELETON_V1", "network": "testnet", "network_magic": "QC_TESTNET_V1",
			"chain_file": "chain_data_testnet.dat", "http_port": ":18082", "p2p_port": ":13001",
			"total_supply": consensus.CanonicalTotalSupplyQCV1, "default_difficulty_bits": 8,
			"genesis_unix": consensus.CanonicalGenesisUnixV1, "public_api_requires_token": true,
			"public_cors_requires_allowlist": true, "dev_miner_endpoints_disabled": true, "separate_chain_file_required": true,
			"faucet_enabled": false, "premine_enabled": false,
			"notes": "Generated documentation skeleton only; not a runtime config or public testnet launch.",
		},
	}
	result := map[string][]byte{}
	for name, value := range values {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return nil, err
		}
		result[name] = append(data, '\n')
	}
	return result, nil
}
