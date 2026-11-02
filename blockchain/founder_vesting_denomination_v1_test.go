package blockchain

import (
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestFounderVestingAtomV3NormalizesLegacyLockedInputV1(
	t *testing.T,
) {
	const activationHeight = 1

	bc := atomDenominationTestChainV1(
		t,
		activationHeight,
	)

	lockedPubKeyHash := []byte{
		0x11, 0x22, 0x33, 0x44,
	}
	releasedPubKeyHash := []byte{
		0xaa, 0xbb, 0xcc, 0xdd,
	}

	funding := &Transaction{
		ID: []byte{
			0xf0, 0x60, 0x01,
		},
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{
				Amount: FounderLockedAllocationV1,
				PubKeyHash: append(
					[]byte(nil),
					lockedPubKeyHash...,
				),
			},
		},
		Sender: "COINBASE",
	}

	bc.Blocks[0].Transactions = []*Transaction{
		funding,
	}

	atomsPerQC := int(
		qconsensus.CanonicalAtomsPerQCV1,
	)

	lockedAllocationAtoms :=
		FounderLockedAllocationV1 *
			atomsPerQC

	releaseAtoms := atomsPerQC

	spend := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID: append(
					[]byte(nil),
					funding.ID...,
				),
				OutIndex: 0,
			},
		},
		Outputs: []TransactionOutput{
			{
				Amount: lockedAllocationAtoms -
					releaseAtoms,
				PubKeyHash: append(
					[]byte(nil),
					lockedPubKeyHash...,
				),
			},
			{
				Amount: releaseAtoms,
				PubKeyHash: append(
					[]byte(nil),
					releasedPubKeyHash...,
				),
			},
		},
	}

	if got := bc.ProtocolAtHeightV1(
		activationHeight,
	); got != qconsensus.ProtocolAtomV3 {
		t.Fatalf(
			"protocol at activation=%q want=%q",
			got,
			qconsensus.ProtocolAtomV3,
		)
	}

	got, err := bc.founderNetReleaseForTxV1(
		spend,
		lockedPubKeyHash,
	)
	if err != nil {
		t.Fatalf(
			"founder AtomV3 release accounting: %v",
			err,
		)
	}

	if got != releaseAtoms {
		t.Fatalf(
			"founder AtomV3 release=%d atoms want=%d atoms",
			got,
			releaseAtoms,
		)
	}
}
