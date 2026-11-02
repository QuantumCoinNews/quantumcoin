package miner

import (
	"time"

	"quantumcoin/blockchain"
)

// Basit/metinlik stublar — main.go bunları doğrudan kullanmıyor,
// derlemeyi kolaylaştırmak için uyumlu tutuldu.

//lint:ignore U1000 retained compatibility stub for candidate block assembly experiments
func assembleCoinbaseAndMetadata(
	height int64,
	headerHash []byte,
	txs []*blockchain.Transaction,
	totalMintedQC int64,
) (*blockchain.Transaction, map[string]string) {
	// Bu örnekte coinbase'i blockchain.MineBlock zaten oluşturuyor.
	return nil, map[string]string{}
}

func BuildCandidateBlock(
	prev *blockchain.Block,
	txs []*blockchain.Transaction,
	_ int64, // totalMintedQC
) *blockchain.Block {
	if prev == nil {
		return nil
	}
	return &blockchain.Block{
		Index:        prev.Index + 1,
		PrevHash:     prev.Hash,
		Timestamp:    time.Now().Unix(),
		Transactions: txs,
		Miner:        "candidate",
		Difficulty:   1,
		Metadata:     map[string]string{},
	}
}

// SumFeesAtoms retains the legacy name while delegating to verified UTXO fee
// accounting. The current ledger stores integer QC amounts; atomic-unit
// migration is a separate mainnet compatibility decision.
func SumFeesAtoms(
	bc *blockchain.Blockchain,
	txs []*blockchain.Transaction,
) (int64, error) {
	fees, err := bc.CalculateBlockFeesSnapshot(txs)
	if err != nil {
		return 0, err
	}

	return int64(fees), nil
}
