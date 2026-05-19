package main

import (
"crypto/sha256"
"encoding/hex"
"fmt"

bip39 "github.com/tyler-smith/go-bip39"
)

func main() {

words := "suit advice north disorder use jeans noble urban note hand chest shift"

seed := bip39.NewSeed(words, "")

sum := sha256.Sum256(seed)

privHex := hex.EncodeToString(sum[:])

addrHash := sha256.Sum256([]byte(privHex))
addr := hex.EncodeToString(addrHash[:20])

fmt.Println("===================================")
fmt.Println(" QuantumCoin Deterministic Wallet")
fmt.Println("===================================")
fmt.Println()

fmt.Println("WORDS:")
fmt.Println(words)
fmt.Println()

fmt.Println("DERIVED PRIVATE KEY:")
fmt.Println(privHex)
fmt.Println()

fmt.Println("DERIVED ADDRESS:")
fmt.Println("QC" + addr)
fmt.Println()

fmt.Println("Deterministic engine phase-1 ready.")
}
