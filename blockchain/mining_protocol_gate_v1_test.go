package blockchain

import (
	"encoding/hex"
	"strings"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func bindMiningProtocolTestPolicyV1(
	t *testing.T,
	bc *Blockchain,
	target qconsensus.ProtocolVersion,
	activationHeight int,
) {
	t.Helper()

	if bc == nil ||
		len(bc.Blocks) == 0 ||
		bc.Blocks[0] == nil {
		t.Fatal("invalid test chain")
	}

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID: "mining-protocol-gate-test",

			LegacyGenesisHex: hex.EncodeToString(
				bc.Blocks[0].Hash,
			),

			ActivationHeight: activationHeight,
			PreserveHistory:  true,
			PreserveBalances: true,
			TargetProtocol:   target,
		},
	}

	if err := bc.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err != nil {
		t.Fatalf(
			"bind activation policy: %v",
			err,
		)
	}
}

func TestLocalMiningProtocolGateAllowsLegacyV1(
	t *testing.T,
) {
	bc := NewBlockchain(
		CanonicalInitialMiningRewardV1,
		int(QCTotalSupplyHardCap),
	)

	if err := bc.validateLocalMiningProtocolV1(1); err != nil {
		t.Fatalf(
			"legacy mining unexpectedly rejected: %v",
			err,
		)
	}
}

func TestLocalMiningProtocolGateAllowsAtomV3V1(
	t *testing.T,
) {
	bc := NewBlockchain(
		CanonicalInitialMiningRewardV1,
		int(QCTotalSupplyHardCap),
	)

	bindMiningProtocolTestPolicyV1(
		t,
		bc,
		qconsensus.ProtocolAtomV3,
		1,
	)

	if err := bc.validateLocalMiningProtocolV1(
		1,
	); err != nil {
		t.Fatalf(
			"AtomV3 local mining unexpectedly rejected: %v",
			err,
		)
	}
}

func TestLocalMiningProtocolGateRejectsContinuityV2V1(
	t *testing.T,
) {
	bc := NewBlockchain(
		CanonicalInitialMiningRewardV1,
		int(QCTotalSupplyHardCap),
	)

	bindMiningProtocolTestPolicyV1(
		t,
		bc,
		qconsensus.ProtocolContinuityV2,
		1,
	)

	err := bc.validateLocalMiningProtocolV1(1)
	if err == nil {
		t.Fatal(
			"ContinuityV2 local mining unexpectedly accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"not implemented",
	) {
		t.Fatalf(
			"unexpected ContinuityV2 rejection: %v",
			err,
		)
	}
}

func TestMineBlockSucceedsAtAtomV3ActivationV1(
	t *testing.T,
) {
	bc := NewBlockchain(
		CanonicalInitialMiningRewardV1,
		int(QCTotalSupplyHardCap),
	)
	if bc == nil {
		t.Fatal("NewBlockchain returned nil")
	}

	bindMiningProtocolTestPolicyV1(
		t,
		bc,
		qconsensus.ProtocolAtomV3,
		1,
	)

	beforeBlocks := len(bc.Blocks)
	beforeMintedQC := bc.TotalMinted()

	block, err := bc.MineBlock(
		"atom-local-miner",
		1,
	)
	if err != nil {
		t.Fatalf(
			"MineBlock rejected AtomV3 activation: %v",
			err,
		)
	}

	if block == nil {
		t.Fatal(
			"MineBlock returned nil AtomV3 block",
		)
	}

	if block.Index != 1 {
		t.Fatalf(
			"AtomV3 mined block index=%d want=1",
			block.Index,
		)
	}

	if len(bc.Blocks) != beforeBlocks+1 {
		t.Fatalf(
			"AtomV3 mining chain length=%d want=%d",
			len(bc.Blocks),
			beforeBlocks+1,
		)
	}

	if bc.GetBestHeight() != 1 {
		t.Fatalf(
			"AtomV3 mined height=%d want=1",
			bc.GetBestHeight(),
		)
	}

	if len(block.Transactions) == 0 ||
		block.Transactions[0] == nil ||
		!block.Transactions[0].IsCoinbase() {
		t.Fatal(
			"AtomV3 mined block missing canonical coinbase",
		)
	}

	coinbase := block.Transactions[0]
	if len(coinbase.Outputs) != 1 {
		t.Fatalf(
			"AtomV3 coinbase output count=%d want=1",
			len(coinbase.Outputs),
		)
	}

	if coinbase.Outputs[0].Amount != 50_000_000 {
		t.Fatalf(
			"AtomV3 coinbase amount=%d want=50000000 atoms",
			coinbase.Outputs[0].Amount,
		)
	}

	if got := bc.TotalMinted(); got != beforeMintedQC+50 {
		t.Fatalf(
			"AtomV3 minted accounting=%d want=%d whole QC",
			got,
			beforeMintedQC+50,
		)
	}

	if got := bc.ProtocolAtHeightV1(
		block.Index,
	); got != qconsensus.ProtocolAtomV3 {
		t.Fatalf(
			"mined protocol=%q want=%q",
			got,
			qconsensus.ProtocolAtomV3,
		)
	}
}
