package blockchain

import (
	"encoding/hex"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestCanonicalTotalBurnedAtomsUsesAtomV3UnitsV1(t *testing.T) {
	bc := NewBlockchain(
		0,
		int(qconsensus.CanonicalTotalSupplyQCV1),
	)

	if bc == nil || len(bc.Blocks) == 0 || bc.Blocks[0] == nil {
		t.Fatal("test blockchain has no genesis block")
	}

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID:        "burn-accounting-atom-v3-test",
			LegacyGenesisHex: hex.EncodeToString(bc.Blocks[0].Hash),
			ActivationHeight: 1,
			PreserveHistory:  true,
			PreserveBalances: true,
			TargetProtocol:   qconsensus.ProtocolAtomV3,
		},
	}

	if err := bc.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err != nil {
		t.Fatalf("bind AtomV3 activation: %v", err)
	}

	if got := bc.ProtocolAtHeightV1(1); got != qconsensus.ProtocolAtomV3 {
		t.Fatalf(
			"protocol at height 1=%q want=%q",
			got,
			qconsensus.ProtocolAtomV3,
		)
	}

	bc.Blocks = append(
		bc.Blocks,
		&Block{
			Index: 1,
			Transactions: []*Transaction{
				{
					Inputs: []TransactionInput{
						{
							TxID:     []byte{0x01},
							OutIndex: 0,
						},
					},
					Outputs: []TransactionOutput{
						NewCanonicalBurnOutputV1(1),
					},
				},
			},
		},
	)

	got, err := bc.TotalBurnedAtomsCheckedV1()
	if err != nil {
		t.Fatalf("AtomV3 burn accounting: %v", err)
	}

	const want int64 = 1

	if got != want {
		t.Fatalf(
			"AtomV3 burned atoms=%d want=%d",
			got,
			want,
		)
	}

	if bc.TotalSupply != int(qconsensus.CanonicalTotalSupplyQCV1) {
		t.Fatalf(
			"AtomV3 burn accounting changed hard cap: got=%d want=%d",
			bc.TotalSupply,
			qconsensus.CanonicalTotalSupplyQCV1,
		)
	}
}
