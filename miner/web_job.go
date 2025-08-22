package miner

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

// WebDifficulty: tarayıcı miner için basit model — leading HEX '0' sayısı.
var WebDifficulty = 5 // config ile besleyebilirsin

// CurrentWebChallenge: 32 baytlık rastgele challenge üretir (hex).
func CurrentWebChallenge() (challengeHex string, difficulty int) {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf), WebDifficulty
}

// doubleSHA256(ch||nonceLE)
func doubleSHA256(input []byte) [32]byte {
	h1 := sha256.Sum256(input)
	h2 := sha256.Sum256(h1[:])
	return h2
}

// meetsDifficulty: hex başındaki '0' sayısını say
func meetsDifficulty(hexHash string, diff int) bool {
	if diff <= 0 {
		return true
	}
	if len(hexHash) < diff {
		return false
	}
	for i := 0; i < diff; i++ {
		if hexHash[i] != '0' {
			return false
		}
	}
	return true
}

// VerifyWebSolution: challenge(hex) + nonce(LE uint32) ⇒ doubleSHA256
// dönen: (valid, hashHex)
func VerifyWebSolution(challengeHex string, nonce uint32, diff int) (bool, string) {
	ch, err := hex.DecodeString(challengeHex)
	if err != nil || len(ch) == 0 {
		return false, ""
	}
	in := make([]byte, 0, len(ch)+4)
	in = append(in, ch...)
	nonceLE := make([]byte, 4)
	binary.LittleEndian.PutUint32(nonceLE, nonce)
	in = append(in, nonceLE...)

	h := doubleSHA256(in)
	hashHex := hex.EncodeToString(h[:])
	ok := meetsDifficulty(hashHex, diff)
	return ok, hashHex
}
