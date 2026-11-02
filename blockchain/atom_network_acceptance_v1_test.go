package blockchain

import (
	"strings"
	"testing"
)

func TestAtomV3PeerEntryPassesProtocolGateAndRejectsBadPoWV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(t, 1)

	if got := bc.GetBestHeight(); got != 0 {
		t.Fatalf(
			"unexpected initial height=%d want=0",
			got,
		)
	}

	beforeBlocks := len(bc.Blocks)
	beforeTip := append(
		[]byte(nil),
		bc.GetLastBlock().Hash...,
	)

	prev := bc.GetLastBlock()
	if prev == nil {
		t.Fatal("missing genesis block")
	}

	blk, err := NewBlock(
		prev.Index+1,
		[]*Transaction{
			testPeerCoinbaseTxV2(50_000_000),
		},
		prev.Hash,
		"atom-peer-miner",
		testExpectedNextDifficultyV1(t, bc),
	)
	if err != nil {
		t.Fatalf(
			"mine AtomV3 peer fixture: %v",
			err,
		)
	}

	// Deliberately corrupt the stored PoW hash after producing an otherwise
	// structurally valid candidate block.
	blk.Hash = []byte("bad-atom-peer-pow")

	err = bc.AddBlockFromPeerStrictV2(blk)
	if err == nil {
		t.Fatal(
			"bad-PoW AtomV3 peer block unexpectedly accepted",
		)
	}

	if strings.Contains(
		err.Error(),
		"unsupported protocol",
	) {
		t.Fatalf(
			"AtomV3 peer block stopped at unsupported protocol gate: %v",
			err,
		)
	}

	if strings.Contains(
		err.Error(),
		"not implemented",
	) {
		t.Fatalf(
			"AtomV3 peer block stopped at unimplemented protocol gate: %v",
			err,
		)
	}

	if !strings.Contains(
		err.Error(),
		"proof-of-work validation failed",
	) {
		t.Fatalf(
			"AtomV3 peer block did not reach canonical PoW validation: %v",
			err,
		)
	}

	if len(bc.Blocks) != beforeBlocks {
		t.Fatalf(
			"rejected AtomV3 peer block mutated chain length: before=%d after=%d",
			beforeBlocks,
			len(bc.Blocks),
		)
	}

	if got := bc.GetBestHeight(); got != 0 {
		t.Fatalf(
			"rejected AtomV3 peer block changed height: got=%d want=0",
			got,
		)
	}

	afterTip := bc.GetLastBlock()
	if afterTip == nil {
		t.Fatal("chain lost tip after rejected AtomV3 peer block")
	}

	if string(afterTip.Hash) != string(beforeTip) {
		t.Fatal(
			"rejected AtomV3 peer block changed chain tip hash",
		)
	}
}

func TestAtomV3ReplacementPassesProtocolGateAndRejectsBadPoWV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(t, 1)

	if got := bc.GetBestHeight(); got != 0 {
		t.Fatalf(
			"unexpected initial height=%d want=0",
			got,
		)
	}

	beforeBlocks := len(bc.Blocks)
	beforeTip := append(
		[]byte(nil),
		bc.GetLastBlock().Hash...,
	)

	genesis := bc.GetLastBlock()
	if genesis == nil {
		t.Fatal("missing genesis block")
	}

	next, err := NewBlock(
		1,
		[]*Transaction{
			testPeerCoinbaseTxV2(50_000_000),
		},
		genesis.Hash,
		"atom-replacement-miner",
		testExpectedNextDifficultyV1(t, bc),
	)
	if err != nil {
		t.Fatalf(
			"mine AtomV3 replacement fixture: %v",
			err,
		)
	}

	// Force rejection after the AtomV3 network protocol gate.
	next.Hash = []byte("bad-atom-replacement-pow")

	incoming := append(
		[]*Block(nil),
		bc.GetAllBlocks()...,
	)
	incoming = append(
		incoming,
		next,
	)

	err = bc.ReplaceChainFromPeerStrictV1(incoming)
	if err == nil {
		t.Fatal(
			"bad-PoW AtomV3 replacement chain unexpectedly accepted",
		)
	}

	if strings.Contains(
		err.Error(),
		"unsupported protocol",
	) {
		t.Fatalf(
			"AtomV3 replacement stopped at unsupported protocol gate: %v",
			err,
		)
	}

	if strings.Contains(
		err.Error(),
		"not implemented",
	) {
		t.Fatalf(
			"AtomV3 replacement stopped at unimplemented protocol gate: %v",
			err,
		)
	}

	if !strings.Contains(
		err.Error(),
		"proof-of-work validation failed",
	) {
		t.Fatalf(
			"AtomV3 replacement did not reach canonical PoW validation: %v",
			err,
		)
	}

	if len(bc.Blocks) != beforeBlocks {
		t.Fatalf(
			"rejected AtomV3 replacement mutated chain length: before=%d after=%d",
			beforeBlocks,
			len(bc.Blocks),
		)
	}

	if got := bc.GetBestHeight(); got != 0 {
		t.Fatalf(
			"rejected AtomV3 replacement changed height: got=%d want=0",
			got,
		)
	}

	afterTip := bc.GetLastBlock()
	if afterTip == nil {
		t.Fatal(
			"chain lost tip after rejected AtomV3 replacement",
		)
	}

	if string(afterTip.Hash) != string(beforeTip) {
		t.Fatal(
			"rejected AtomV3 replacement changed chain tip hash",
		)
	}
}
