package blockchain

import (
	"testing"

	qconsensus "quantumcoin/consensus"
)

// QMA-0062:
// Burning reduces circulating/available issued supply. It must never reduce
// the canonical 25,500,000 QC hard cap itself.
func TestCanonicalCirculatingSupplySubtractsBurnV1(t *testing.T) {
	bc := NewBlockchain(
		0,
		int(qconsensus.CanonicalTotalSupplyQCV1),
	)

	if bc == nil || len(bc.Blocks) == 0 {
		t.Fatal("test blockchain has no genesis block")
	}

	issuedQC := bc.TotalMinted()
	if issuedQC <= 0 {
		t.Fatalf("test blockchain issued supply=%d want positive", issuedQC)
	}

	burnQC := 1
	if issuedQC <= burnQC {
		t.Fatalf(
			"test requires issued supply greater than burn: issued=%d burn=%d",
			issuedQC,
			burnQC,
		)
	}

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
	if err != nil {
		t.Fatalf("circulating supply accounting: %v", err)
	}

	issuedAtoms, err :=
		qconsensus.WholeQCToAtomsV1(int64(issuedQC))
	if err != nil {
		t.Fatalf("issued QC to atoms: %v", err)
	}

	burnAtoms, err :=
		qconsensus.WholeQCToAtomsV1(int64(burnQC))
	if err != nil {
		t.Fatalf("burn QC to atoms: %v", err)
	}

	want := issuedAtoms - burnAtoms

	if got != want {
		t.Fatalf(
			"circulating atoms=%d want=%d issuedAtoms=%d burnAtoms=%d",
			got,
			want,
			issuedAtoms,
			burnAtoms,
		)
	}

	if bc.TotalSupply != int(qconsensus.CanonicalTotalSupplyQCV1) {
		t.Fatalf(
			"burn changed canonical hard cap: got=%d want=%d",
			bc.TotalSupply,
			qconsensus.CanonicalTotalSupplyQCV1,
		)
	}
}
