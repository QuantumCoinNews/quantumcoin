package node

import (
	"reflect"
	"testing"

	"quantumcoin/backuprestore"
	"quantumcoin/blockchain"
	"quantumcoin/burncontrol"
	"quantumcoin/economycontrol"
	"quantumcoin/foundervesting"
	"quantumcoin/hardforkpolicy"
	"quantumcoin/miningtransition"
	"quantumcoin/networkfee"
	"quantumcoin/persistence"
	"quantumcoin/queueworker"
)

type qma0059ChainObservationV1 struct {
	State        blockchain.BlockchainStateSnapshot
	Height       int
	TotalMinted  int
	TotalSupply  int
	PendingCount int
}

func qma0059ObserveChainV1(
	chain *blockchain.Blockchain,
) qma0059ChainObservationV1 {
	return qma0059ChainObservationV1{
		State:        chain.StateSnapshot(),
		Height:       chain.HeightSnapshot(),
		TotalMinted:  chain.TotalMintedSnapshot(),
		TotalSupply:  chain.TotalSupplySnapshot(),
		PendingCount: chain.PendingTransactionCountSnapshot(),
	}
}

func qma0059RequireChainUnchangedV1(
	t *testing.T,
	before qma0059ChainObservationV1,
	after qma0059ChainObservationV1,
) {
	t.Helper()

	if before.Height != after.Height {
		t.Fatalf(
			"policy operation changed chain height: before=%d after=%d",
			before.Height,
			after.Height,
		)
	}

	if before.TotalMinted != after.TotalMinted {
		t.Fatalf(
			"policy operation changed total minted: before=%d after=%d",
			before.TotalMinted,
			after.TotalMinted,
		)
	}

	if before.TotalSupply != after.TotalSupply {
		t.Fatalf(
			"policy operation changed total supply: before=%d after=%d",
			before.TotalSupply,
			after.TotalSupply,
		)
	}

	if before.PendingCount != after.PendingCount {
		t.Fatalf(
			"policy operation changed mempool count: before=%d after=%d",
			before.PendingCount,
			after.PendingCount,
		)
	}

	if !reflect.DeepEqual(before.State, after.State) {
		t.Fatal("policy operation changed detached blockchain state snapshot")
	}
}

func TestQMA0059PolicyValidationIsBlockchainSideEffectFreeV1(
	t *testing.T,
) {
	chain := blockchain.NewBlockchain(1, 0)

	before := qma0059ObserveChainV1(chain)

	if err := burncontrol.ValidatePolicyV1(
		burncontrol.DefaultPolicyV1(),
	); err != nil {
		t.Fatalf("burn policy validation failed: %v", err)
	}

	if err := economycontrol.ValidatePolicyV1(
		economycontrol.DefaultPolicyV1(),
	); err != nil {
		t.Fatalf("economy policy validation failed: %v", err)
	}

	if err := foundervesting.ValidatePolicyV1(
		foundervesting.DefaultPolicyV1(),
	); err != nil {
		t.Fatalf("founder vesting policy validation failed: %v", err)
	}

	if err := hardforkpolicy.ValidatePolicyV1(
		hardforkpolicy.DefaultPolicyV1(),
	); err != nil {
		t.Fatalf("hardfork policy validation failed: %v", err)
	}

	if err := miningtransition.ValidatePolicyV1(
		miningtransition.DefaultPolicyV1(),
	); err != nil {
		t.Fatalf("mining transition policy validation failed: %v", err)
	}

	if err := networkfee.ValidatePolicyV1(
		networkfee.DefaultPolicyV1(),
	); err != nil {
		t.Fatalf("network fee policy validation failed: %v", err)
	}

	if err := backuprestore.ValidateRunbookV1(
		backuprestore.DefaultRunbookV1(),
	); err != nil {
		t.Fatalf("backup/restore runbook validation failed: %v", err)
	}

	if err := persistence.ValidatePlanV1(
		persistence.DefaultPlanV1(),
	); err != nil {
		t.Fatalf("persistence plan validation failed: %v", err)
	}

	if err := queueworker.ValidatePlanV1(
		queueworker.DefaultPlanV1(),
	); err != nil {
		t.Fatalf("queue worker plan validation failed: %v", err)
	}

	after := qma0059ObserveChainV1(chain)

	qma0059RequireChainUnchangedV1(t, before, after)
}

func TestQMA0059MutableDeclarationsDoNotAuthorizeBlockchainMutationV1(
	t *testing.T,
) {
	chain := blockchain.NewBlockchain(1, 0)

	before := qma0059ObserveChainV1(chain)

	burn := burncontrol.DefaultPolicyV1()
	burn.SafeBoundary["doesNotMint"] = false
	burn.BurnDoesNotMintReplacement = false

	economy := economycontrol.DefaultPolicyV1()
	economy.SafeBoundary["doesNotChangeChain"] = false

	founder := foundervesting.DefaultPolicyV1()
	founder.SafeBoundary["doesNotExecuteUnlock"] = false
	founder.DoesNotMintFounderAllocation = false

	hardfork := hardforkpolicy.DefaultPolicyV1()
	hardfork.SafeBoundary["doesNotReplaceChain"] = false
	hardfork.HardforkCannotMintExtraQC = false

	mining := miningtransition.DefaultPolicyV1()
	mining.SafeBoundary["doesNotMint"] = false
	mining.AutoMintRemainingForbidden = false

	fee := networkfee.DefaultPolicyV1()
	fee.SafeBoundary["doesNotChangeBalances"] = false
	fee.AICannotChangeFee = false

	runbook := backuprestore.DefaultRunbookV1()
	runbook.SafeBoundary["doesNotChangeChain"] = false

	dbPlan := persistence.DefaultPlanV1()
	dbPlan.SafeBoundary["doesNotChangeBalances"] = false

	queuePlan := queueworker.DefaultPlanV1()
	queuePlan.SafeBoundary["doesNotMint"] = false
	queuePlan.Queues[0].CanMint = true

	// These mutations deliberately corrupt declarative policy metadata.
	// They must not themselves mutate blockchain state or grant a
	// blockchain write capability.
	afterDeclarationMutation := qma0059ObserveChainV1(chain)

	qma0059RequireChainUnchangedV1(
		t,
		before,
		afterDeclarationMutation,
	)

	// Validators must reject the corrupted declarations.
	for name, validate := range map[string]func() error{
		"burn": func() error {
			return burncontrol.ValidatePolicyV1(burn)
		},
		"economy": func() error {
			return economycontrol.ValidatePolicyV1(economy)
		},
		"founder": func() error {
			return foundervesting.ValidatePolicyV1(founder)
		},
		"hardfork": func() error {
			return hardforkpolicy.ValidatePolicyV1(hardfork)
		},
		"mining": func() error {
			return miningtransition.ValidatePolicyV1(mining)
		},
		"networkfee": func() error {
			return networkfee.ValidatePolicyV1(fee)
		},
		"backuprestore": func() error {
			return backuprestore.ValidateRunbookV1(runbook)
		},
		"persistence": func() error {
			return persistence.ValidatePlanV1(dbPlan)
		},
		"queueworker": func() error {
			return queueworker.ValidatePlanV1(queuePlan)
		},
	} {
		if err := validate(); err == nil {
			t.Fatalf(
				"%s corrupted declaration unexpectedly validated",
				name,
			)
		}
	}

	afterValidation := qma0059ObserveChainV1(chain)

	qma0059RequireChainUnchangedV1(
		t,
		before,
		afterValidation,
	)
}

func TestQMA0059SafeBoundaryFactoriesReturnIndependentMapsV1(
	t *testing.T,
) {
	tests := []struct {
		name string
		new  func() map[string]bool
	}{
		{"burn", burncontrol.SafeBoundaryV1},
		{"economy", economycontrol.SafeBoundaryV1},
		{"founder", foundervesting.SafeBoundaryV1},
		{"hardfork", hardforkpolicy.SafeBoundaryV1},
		{"mining", miningtransition.SafeBoundaryV1},
		{"networkfee", networkfee.SafeBoundaryV1},
		{"backuprestore", backuprestore.SafeBoundaryV1},
		{"persistence", persistence.SafeBoundaryV1},
		{"queueworker", queueworker.SafeBoundaryV1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first := test.new()
			second := test.new()

			if first == nil || second == nil {
				t.Fatal("safe-boundary factory returned nil")
			}

			var key string
			for candidate := range first {
				key = candidate
				break
			}

			if key == "" {
				t.Fatal("safe-boundary factory returned empty map")
			}

			original := second[key]

			first[key] = !first[key]

			if second[key] != original {
				t.Fatalf(
					"safe-boundary factory shares mutable map state for key %q",
					key,
				)
			}
		})
	}
}
