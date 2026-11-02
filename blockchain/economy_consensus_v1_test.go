package blockchain

import (
	"bytes"
	"testing"
	"time"
)

func TestCanonicalGenesisDeterministicV1(
	t *testing.T,
) {
	first := NewBlockchain(1, 0)
	second := NewBlockchain(1, 0)

	if first == nil || second == nil {
		t.Fatalf("canonical genesis chain is nil")
	}

	if len(first.Blocks) != 1 ||
		len(second.Blocks) != 1 {
		t.Fatalf("expected one genesis block")
	}

	if first.Blocks[0].Timestamp !=
		CanonicalGenesisUnixV1 {
		t.Fatalf(
			"genesis timestamp: got %d want %d",
			first.Blocks[0].Timestamp,
			CanonicalGenesisUnixV1,
		)
	}

	if !bytes.Equal(
		first.Blocks[0].Hash,
		second.Blocks[0].Hash,
	) {
		t.Fatalf(
			"canonical genesis hash is not deterministic:\n%x\n%x",
			first.Blocks[0].Hash,
			second.Blocks[0].Hash,
		)
	}

	if first.Blocks[0].Transactions[0].
		Timestamp.Unix() != CanonicalGenesisUnixV1 {
		t.Fatalf(
			"genesis transaction timestamp is not canonical",
		)
	}
}

func TestRewardAtHeightV1HalvingBoundaries(
	t *testing.T,
) {
	tests := []struct {
		height int
		want   int
	}{
		{1, 50},
		{CanonicalBlocksPerHalvingV1, 50},
		{CanonicalBlocksPerHalvingV1 + 1, 25},
		{2*CanonicalBlocksPerHalvingV1 + 1, 12},
		{3*CanonicalBlocksPerHalvingV1 + 1, 6},
		{4*CanonicalBlocksPerHalvingV1 + 1, 3},
		{5*CanonicalBlocksPerHalvingV1 + 1, 1},
		{20*CanonicalBlocksPerHalvingV1 + 1, 1},
	}

	for _, test := range tests {
		if got := RewardAtHeightV1(
			test.height,
		); got != test.want {
			t.Fatalf(
				"height %d reward: got %d want %d",
				test.height,
				got,
				test.want,
			)
		}
	}
}

func TestFinalPartialRewardV1(
	t *testing.T,
) {
	if got := ExpectedSubsidyFromStateV1(
		1,
		25_499_993,
		25_500_000,
	); got != 7 {
		t.Fatalf(
			"partial reward: got %d want 7",
			got,
		)
	}

	if got := ExpectedSubsidyFromStateV1(
		1,
		25_500_000,
		25_500_000,
	); got != 0 {
		t.Fatalf(
			"completed emission reward: got %d want 0",
			got,
		)
	}
}

func TestValidateBlockSubsidyV1(
	t *testing.T,
) {
	chain := NewBlockchain(1, 0)
	if chain == nil {
		t.Fatalf("chain is nil")
	}

	previous := chain.GetLastBlock()
	reward := chain.ExpectedSubsidyForHeightV1(1)

	coinbase := &Transaction{
		ID:     nil,
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{
				Amount:     reward,
				PubKeyHash: []byte("subsidy-test-miner"),
			},
		},
		Timestamp: time.Unix(
			CanonicalGenesisUnixV1+30,
			0,
		).UTC(),
		Sender: "COINBASE",
		Amount: float64(reward),
	}
	coinbase.ID = coinbase.Hash()

	block, err := NewBlock(
		1,
		[]*Transaction{coinbase},
		previous.Hash,
		"subsidy-test-miner",
		1,
	)
	if err != nil {
		t.Fatalf("mine subsidy test block: %v", err)
	}

	if err := chain.ValidateBlockSubsidyV1(
		block,
	); err != nil {
		t.Fatalf(
			"valid subsidy rejected: %v",
			err,
		)
	}

	block.Transactions[0].Outputs[0].Amount++
	block.Transactions[0].Amount++

	if err := chain.ValidateBlockSubsidyV1(
		block,
	); err == nil {
		t.Fatalf(
			"expected excessive subsidy rejection",
		)
	}
}

func TestFounderUnlockedAtUnixV1(
	t *testing.T,
) {
	genesis := CanonicalGenesisTimeV1()

	beforeFirstUnlock := genesis.AddDate(
		1,
		0,
		0,
	).Unix()

	if got := FounderUnlockedAtUnixV1(
		genesis.Unix(),
		beforeFirstUnlock,
	); got != 0 {
		t.Fatalf(
			"cliff unlock: got %d want 0",
			got,
		)
	}

	firstUnlock := genesis.AddDate(
		1,
		1,
		0,
	).Unix()

	if got := FounderUnlockedAtUnixV1(
		genesis.Unix(),
		firstUnlock,
	); got != FounderMonthlyUnlockQCV1 {
		t.Fatalf(
			"first tranche: got %d want %d",
			got,
			FounderMonthlyUnlockQCV1,
		)
	}

	fullyVested := genesis.AddDate(
		5,
		0,
		0,
	).Unix()

	if got := FounderUnlockedAtUnixV1(
		genesis.Unix(),
		fullyVested,
	); got != FounderLockedAllocationV1 {
		t.Fatalf(
			"full vesting: got %d want %d",
			got,
			FounderLockedAllocationV1,
		)
	}
}

func TestExpectedSubsidyFromStateV1ConsensusBridge(
	t *testing.T,
) {
	tests := []struct {
		name        string
		height      int
		totalMinted int
		totalSupply int
		want        int
	}{
		{
			name:        "ilk blok 50 QC",
			height:      1,
			totalMinted: 0,
			totalSupply: 0,
			want:        50,
		},
		{
			name:        "legacy son 50 QC blogu",
			height:      CanonicalBlocksPerHalvingV1,
			totalMinted: 0,
			totalSupply: 0,
			want:        50,
		},
		{
			name:        "legacy ilk 25 QC blogu",
			height:      CanonicalBlocksPerHalvingV1 + 1,
			totalMinted: 0,
			totalSupply: 0,
			want:        25,
		},
		{
			name:        "son kismi blok odulu",
			height:      1,
			totalMinted: 25_499_993,
			totalSupply: 25_500_000,
			want:        7,
		},
		{
			name:        "arz tamamlandi",
			height:      1,
			totalMinted: 25_500_000,
			totalSupply: 25_500_000,
			want:        0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ExpectedSubsidyFromStateV1(
				test.height,
				test.totalMinted,
				test.totalSupply,
			)

			if got != test.want {
				t.Fatalf(
					"ExpectedSubsidyFromStateV1(%d, %d, %d) = %d, want %d",
					test.height,
					test.totalMinted,
					test.totalSupply,
					got,
					test.want,
				)
			}
		})
	}
}
