package token

import (
	"errors"
	"testing"

	"quantumcoin/security"
)

func TestMinerTokenBridgeAIObservabilityV1IsReadOnly(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)

	testCases := []struct {
		outcome string
		err     error
		typeV1  string
	}{
		{"PENDING_RECORDED", nil, TokenMinerRewardPendingEventV1},
		{"FINALIZED", nil, TokenMinerRewardFinalizedEventV1},
		{"DUPLICATE_COMMITTED", nil, TokenMinerRewardDuplicateEventV1},
		{"REORG_REJECTED", ErrMinerTokenBlockIdentityV1, TokenMinerRewardReorgRejectedEventV1},
		{"REJECTED", ErrMinerTokenFinalizedHeightConflictV1, TokenMinerRewardRejectedEventV1},
		{"PERSISTENCE_REJECTED", ErrMinerTokenPersistenceV1, TokenMinerRewardPersistenceEventV1},
	}

	for _, testCase := range testCases {
		observeMinerTokenRewardV1(
			testCase.outcome,
			"MINE",
			7,
			"0123456789abcdef",
			"miner-address",
			25,
			6,
			testCase.err,
		)
	}

	events := security.ReadEvents(
		directory,
		len(testCases),
	).Events
	if len(events) != len(testCases) {
		t.Fatalf("event count = %d, want %d", len(events), len(testCases))
	}

	for index, testCase := range testCases {
		event := events[index]
		if event.Type != testCase.typeV1 {
			t.Fatalf("event %d type = %q, want %q", index, event.Type, testCase.typeV1)
		}
		if event.Metadata["operation"] != TokenOperationMinerRewardV1 {
			t.Fatalf("event %d operation = %v", index, event.Metadata["operation"])
		}
		for _, key := range []string{
			"aiReadOnly",
			"advisoryOnly",
		} {
			if event.Metadata[key] != true {
				t.Fatalf("event %d %s = %v, want true", index, key, event.Metadata[key])
			}
		}
		for _, key := range []string{
			"authorizesOperation",
			"changesTokenState",
			"changesRegistry",
			"changesChain",
			"changesConsensus",
		} {
			if event.Metadata[key] != false {
				t.Fatalf("event %d %s = %v, want false", index, key, event.Metadata[key])
			}
		}
	}

	subject := New("OBS", "Observer Test", 8, 100, "owner")
	beforeSupply := subject.TotalSupply()
	observeMinerTokenRewardV1(
		"REJECTED",
		"MINE",
		0,
		"",
		"",
		0,
		0,
		errors.New("advisory test"),
	)
	if afterSupply := subject.TotalSupply(); afterSupply != beforeSupply {
		t.Fatal("observer changed token state")
	}
}

func TestMinerTokenBridgeAIObservabilityV1IsBoundToBridge(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)
	chain := minerBridgeTestChain(1, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, directory)
	block := chain.GetBlockByIndex(1)

	if err := bridge.processMinerTokenBlockV1(
		block,
		minerBridgeTestStatus(block),
	); err != nil {
		t.Fatalf("record pending reward: %v", err)
	}

	appendCanonicalMinerBridgeTestBlock(chain, 7, "miner", "canonical")
	head := chain.GetBlockByIndex(7)
	if err := bridge.processMinerTokenBlockV1(
		head,
		minerBridgeTestStatus(head),
	); err != nil {
		t.Fatalf("finalize reward: %v", err)
	}

	counts := map[string]int{}
	for _, event := range security.ReadEvents(directory, 100).Events {
		counts[event.Type]++
	}

	if counts[TokenMinerRewardPendingEventV1] == 0 {
		t.Fatal("bridge did not emit pending observation")
	}
	if counts[TokenMinerRewardFinalizedEventV1] != 1 {
		t.Fatalf(
			"finalized observation count = %d, want 1",
			counts[TokenMinerRewardFinalizedEventV1],
		)
	}
}
