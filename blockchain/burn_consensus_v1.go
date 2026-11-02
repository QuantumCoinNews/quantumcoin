package blockchain

import (
	"bytes"

	qconsensus "quantumcoin/consensus"
)

const BurnConsensusVersionV1 = "QC_BURN_CONSENSUS_V1"

// canonicalBurnOutputMarkerV1 is a consensus marker, not a wallet
// public-key hash. No private key owns this value.
var canonicalBurnOutputMarkerV1 = []byte{
	0x51, 0x43, 0x5f, 0x42, 0x55, 0x52, 0x4e, 0x5f,
	0x53, 0x49, 0x4e, 0x4b, 0x5f, 0x56, 0x31,
}

// IsCanonicalBurnAddressV1 recognizes the canonical burn alias.
// The alias is a consensus identifier and must never be decoded as
// a normal wallet address.
func IsCanonicalBurnAddressV1(address string) bool {
	return address == qconsensus.CanonicalBurnSinkAddressV1
}

// NewCanonicalBurnOutputV1 constructs the canonical unspendable
// burn output representation.
func NewCanonicalBurnOutputV1(amount int) TransactionOutput {
	return TransactionOutput{
		Amount:     amount,
		PubKeyHash: append([]byte(nil), canonicalBurnOutputMarkerV1...),
	}
}

// IsCanonicalBurnOutputV1 deterministically classifies a burn output.
func IsCanonicalBurnOutputV1(out TransactionOutput) bool {
	return out.Amount > 0 &&
		bytes.Equal(
			out.PubKeyHash,
			canonicalBurnOutputMarkerV1,
		)
}
