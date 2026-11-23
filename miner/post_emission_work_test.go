package miner

import (
	"context"
	"path/filepath"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestQCLocalPostEmissionUsesCanonicalEmptyWork(t *testing.T) {
	clearMinerRuntimeDifficultyEnvV1(t)
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	cfg := config.Default()
	cfg.HTTPPort = ":28082"
	cfg.P2PPort = ":23001"
	cfg.ChainFile = filepath.Join(t.TempDir(), "chain_data_local.dat")
	bc := blockchain.NewBlockchain(50, 0)
	bc.TotalSupply = bc.TotalMinted()
	q := &qcLocal{cfg: cfg, bc: bc}
	work, err := q.GetWork(context.Background(), wallet.NewWallet().GetAddress())
	if err != nil {
		t.Fatal(err)
	}
	if work.Template == nil || work.Template.Transactions[0].Outputs[0].Amount != 0 {
		t.Fatal("post-emission work pays a subsidy")
	}
}
