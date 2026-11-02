package consensus

import "testing"

func TestValidateStoredPoWHashV4BindsNFTRecordDigest(t *testing.T) {
	header := PoWHeaderV4{
		PreviousHash:      make([]byte, 32),
		TransactionDigest: make([]byte, 32),
		NFTRecordDigest:   make([]byte, 32),
		Timestamp:         1,
		Difficulty:        0,
	}
	header.PreviousHash[0] = 0x11
	header.TransactionDigest[0] = 0x22
	header.NFTRecordDigest[0] = 0x33

	const nonce = uint64(7)

	hash, err := ComputePoWHashV4(header, nonce)
	if err != nil {
		t.Fatalf("compute V4 PoW hash: %v", err)
	}

	if err := ValidateStoredPoWHashV4(header, nonce, hash[:]); err != nil {
		t.Fatalf("valid V4 hash was rejected: %v", err)
	}

	mutatedHeader := header
	mutatedHeader.NFTRecordDigest = append(
		[]byte(nil),
		header.NFTRecordDigest...,
	)
	mutatedHeader.NFTRecordDigest[0] ^= 0x01

	if err := ValidateStoredPoWHashV4(
		mutatedHeader,
		nonce,
		hash[:],
	); err == nil {
		t.Fatal("V4 hash accepted after NFT record digest changed")
	}
}
