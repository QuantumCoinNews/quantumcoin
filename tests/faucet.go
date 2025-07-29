package tests

import (
	"fmt"
	"quantumcoin/blockchain"
)

// Faucet: Testnet için adrese QC gönderimi yapar
func Faucet(bc *blockchain.Blockchain, to string, amount int) {
	tx, err := blockchain.NewTransaction("faucet", to, amount, bc)
	if err != nil {
		fmt.Printf("[Faucet] İşlem oluşturulamadı: %v\n", err)
		return
	}

	err = bc.AddTransaction(tx)
	if err != nil {
		fmt.Printf("[Faucet] İşlem havuza eklenemedi: %v\n", err)
		return
	}

	_, err = bc.MineBlock("faucet", 16)
	if err != nil {
		fmt.Printf("[Faucet] Blok kazılamadı: %v\n", err)
		return
	}

	fmt.Printf("[Faucet] %d QC başarıyla gönderildi! => %s\n", amount, to)
}
