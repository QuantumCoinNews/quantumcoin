package tests

import (
	"fmt"
	"quantumcoin/blockchain"
)

// StartTestnet: Örnek test zinciri başlatır
func StartTestnet() *blockchain.Blockchain {
	fmt.Println("[Testnet] Yerel test ağı başlatılıyor...")
	bc := blockchain.NewBlockchain(50, 25_500_000)

	// Örnek işlemler oluştur
	tx1, err1 := blockchain.NewTransaction("genesis_wallet", "alice", 10, bc)
	if err1 != nil {
		fmt.Println("[Testnet] tx1 oluşturulamadı:", err1)
		return bc
	}
	tx2, err2 := blockchain.NewTransaction("genesis_wallet", "bob", 5, bc)
	if err2 != nil {
		fmt.Println("[Testnet] tx2 oluşturulamadı:", err2)
		return bc
	}

	// İşlemleri ekle
	if err := bc.AddTransaction(tx1); err != nil {
		fmt.Println("[Testnet] tx1 eklenemedi:", err)
	}
	if err := bc.AddTransaction(tx2); err != nil {
		fmt.Println("[Testnet] tx2 eklenemedi:", err)
	}

	// Blok kaz
	_, err := bc.MineBlock("tester", 16)
	if err != nil {
		fmt.Println("[Testnet] Blok kazılamadı:", err)
	} else {
		fmt.Println("[Testnet] Başarılı: 1 blok kazıldı.")
	}

	return bc
}
