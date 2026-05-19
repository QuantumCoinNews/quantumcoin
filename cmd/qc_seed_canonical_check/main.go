package main

import (
"crypto/ecdsa"
"crypto/sha256"
"encoding/hex"
"fmt"
"math/big"

bip39 "github.com/tyler-smith/go-bip39"
"github.com/decred/dcrd/dcrec/secp256k1/v4"

"quantumcoin/utils"
"quantumcoin/wallet"
)

func main() {

words := "bachelor repair step various violin around garment renew fog genius retire guard"

seed := bip39.NewSeed(words, "")
sum := sha256.Sum256(seed)

d := new(big.Int).SetBytes(sum[:])

curve := secp256k1.S256()

n := curve.Params().N

d.Mod(d, new(big.Int).Sub(n, big.NewInt(1)))
d.Add(d, big.NewInt(1))

priv := new(ecdsa.PrivateKey)
priv.PublicKey.Curve = curve
priv.D = d

priv.PublicKey.X, priv.PublicKey.Y =
curve.ScalarBaseMult(d.Bytes())

w := &wallet.Wallet{
PrivateKey: priv,
PublicKey:  utils.MakeUncompressedPub(priv),
}

fmt.Println("===================================")
fmt.Println(" QuantumCoin Canonical Seed Wallet")
fmt.Println("===================================")
fmt.Println()

fmt.Println("WORDS:")
fmt.Println(words)
fmt.Println()

fmt.Println("PRIVATE KEY HEX:")
fmt.Println(hex.EncodeToString(d.Bytes()))
fmt.Println()

fmt.Println("CANONICAL ADDRESS:")
fmt.Println(w.GetAddress())
}
