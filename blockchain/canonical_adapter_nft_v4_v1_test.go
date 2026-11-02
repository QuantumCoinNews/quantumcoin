package blockchain

import (
	"strings"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestCanonicalAdapterPassesNFTV4HeaderV1(t *testing.T) {
	bc := canonicalAdapterDenominationChainV1(
		t,
		qconsensus.ProtocolNFTV4,
		1,
	)

	previous := bc.GetLastBlock()
	if previous == nil {
		t.Fatal("missing previous block")
	}

	block, err := NewBlockAtProtocolV1(
		1,
		nil,
		previous.Hash,
		"miner-test",
		0,
		nil,
		qconsensus.ProtocolNFTV4,
	)
	if err != nil {
		t.Fatalf("create valid NFT V4 block: %v", err)
	}

	_, err = bc.ValidateCanonicalBlockEnvelopeV1(
		block,
		bc.TotalMinted(),
	)
	if err == nil {
		t.Fatal("incomplete block unexpectedly passed canonical validation")
	}

	if !strings.Contains(err.Error(), "coinbase layout validation failed") {
		t.Fatalf(
			"adapter did not pass the valid NFT V4 PoW header to canonical validation: %v",
			err,
		)
	}
}
