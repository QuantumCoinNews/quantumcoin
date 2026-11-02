package blockchain

import (
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestCanonicalCirculatingSupplyRejectsBurnAboveIssuedV1(t *testing.T) {
	bc := NewBlockchain(
		0,
		int(qconsensus.CanonicalTotalSupplyQCV1),
	)

	if bc == nil || len(bc.Blocks) == 0 {
		t.Fatal("test blockchain has no genesis block")
	}

	issuedQC := bc.TotalMinted()
	if issuedQC <= 0 {
		t.Fatalf(
			"test blockchain issued supply=%d want positive",
			issuedQC,
		)
	}

	burnQC := issuedQC + 1

	bc.Blocks[0].Transactions = append(
		bc.Blocks[0].Transactions,
		&Transaction{
			Inputs: []TransactionInput{
				{
					TxID:     []byte{0x01},
					OutIndex: 0,
				},
			},
			Outputs: []TransactionOutput{
				NewCanonicalBurnOutputV1(burnQC),
			},
		},
	)

	got, err := bc.CirculatingSupplyAtomsCheckedV1()
	if err == nil {
		t.Fatalf(
			"expected burned-above-issued accounting error; got circulating=%d",
			got,
		)
	}

	if bc.TotalSupply != int(qconsensus.CanonicalTotalSupplyQCV1) {
		t.Fatalf(
			"failed burn accounting changed hard cap: got=%d want=%d",
			bc.TotalSupply,
			qconsensus.CanonicalTotalSupplyQCV1,
		)
	}
}
