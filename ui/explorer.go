package ui

import (
	"fmt"

	"quantumcoin/blockchain"
	"quantumcoin/i18n"
)

func PrintExplorer(bc *blockchain.Blockchain) {
	fmt.Println(i18n.T(CurrentLang, "explorer_title"))
	fmt.Println("------------------------------------------------")

	for _, block := range bc.Blocks {
		fmt.Printf(i18n.T(CurrentLang, "explorer_block")+"\n", block.Index, block.Miner, block.Hash, block.PrevHash)
		for _, tx := range block.Transactions {
			fmt.Printf(i18n.T(CurrentLang, "explorer_tx")+"\n", tx.ID)
			for _, out := range tx.Outputs {
				fmt.Printf(i18n.T(CurrentLang, "explorer_tx_out")+"\n", out.Amount)
			}
			fmt.Println("  --------------------")
		}
		fmt.Println("------------------------------------------------")
	}
}
