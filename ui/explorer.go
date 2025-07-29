package ui

import (
	"fmt"

	"quantumcoin/blockchain"
	"quantumcoin/i18n"
	"quantumcoin/wallet"
)

// PrintExplorer: Terminal blockchain explorer
func PrintExplorer(bc *blockchain.Blockchain, lang string) {
	fmt.Println(i18n.T(lang, "welcome"))
	fmt.Println("------------------------------------------------")

	for _, block := range bc.Blocks {
		fmt.Printf("⛓️  %s #%d\n", i18n.T(lang, "block"), block.Index)
		fmt.Printf("🔐 %s: %x\n", i18n.T(lang, "hash"), block.Hash)
		fmt.Printf("🔗 %s: %x\n", i18n.T(lang, "prev_hash"), block.PrevHash)
		fmt.Printf("⛏️  %s: %s\n", i18n.T(lang, "miner"), block.Miner)
		fmt.Printf("📦 %s:\n", i18n.T(lang, "transactions"))

		for _, tx := range block.Transactions {
			fmt.Printf("🧾 %s: %x\n", i18n.T(lang, "txid"), tx.ID)
			// Girişler
			for _, input := range tx.Inputs {
				if len(input.PubKey) > 0 {
					address := wallet.HashAndEncode(input.PubKey)
					fmt.Printf(" 🔻 %s: %s\n", i18n.T(lang, "from"), address)
				} else {
					fmt.Printf(" 🔻 %s: COINBASE\n", i18n.T(lang, "from"))
				}
			}
			// Çıkışlar
			for _, output := range tx.Outputs {
				address := wallet.HashAndEncode(output.PubKeyHash)
				fmt.Printf(" 🔺 %s: %s (%d QC)\n", i18n.T(lang, "to"), address, output.Amount)
			}
			fmt.Println("  --------------------")
		}
		fmt.Println("------------------------------------------------")
	}
}
