package blockchain

import (
	"encoding/hex"
	"math"
	"strconv"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestCanonicalTotalBurnedAtomsRejectsAtomV3AccumulationOverflowV1(
	t *testing.T,
) {
	if strconv.IntSize < 64 {
		t.Skip("AtomV3 accumulation overflow fixture requires 64-bit int")
	}

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
			NetworkID:        "burn-accounting-atom-overflow-test",
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
						NewCanonicalBurnOutputV1(int(math.MaxInt64)),
						NewCanonicalBurnOutputV1(1),
					},
				},
			},
		},
	)

	_, err := bc.TotalBurnedAtomsCheckedV1()
	if err == nil {
		t.Fatal(
			"AtomV3 burn accumulation overflow unexpectedly accepted",
		)
	}

	if bc.TotalSupply != int(qconsensus.CanonicalTotalSupplyQCV1) {
		t.Fatalf(
			"AtomV3 overflow accounting changed hard cap: got=%d want=%d",
			bc.TotalSupply,
			qconsensus.CanonicalTotalSupplyQCV1,
		)
	}
}
