package tests

import (
	"fmt"
	"quantumcoin/wallet"
)

func TestSignature() {
	privateKey, publicKey := wallet.GenerateKeyPair()
	message := []byte("QuantumCoin: Yapılmayanı yapmak!")

	r, s, err := wallet.SignData(message, privateKey)
	if err != nil {
		fmt.Println("[Signature Test] İmzalama hatası:", err)
		return
	}

	valid := wallet.VerifySignature(publicKey, message, r, s)
	if !valid {
		fmt.Println("[Signature Test] İmza doğrulanamadı!")
	} else {
		fmt.Println("[Signature Test] İmza doğrulandı ✓")
	}
}
