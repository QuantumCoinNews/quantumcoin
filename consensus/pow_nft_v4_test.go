package consensus

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func TestPoWHeaderV4BindsNFTRecordDigest(t *testing.T) {
	txDigest := sha256.Sum256([]byte("transaction-set"))
	nftDigestA := sha256.Sum256([]byte("nft-record-set-a"))

	header := PoWHeaderV4{
		PreviousHash:      bytes.Repeat([]byte{0x11}, sha256.Size),
		TransactionDigest: txDigest[:],
		NFTRecordDigest:   nftDigestA[:],
		Timestamp:         1700,
		Difficulty:        8,
	}

	first, err := SerializePoWHeaderV4(header, 9)
	if err != nil {
		t.Fatalf("serialize first V4 header: %v", err)
	}

	repeated, err := SerializePoWHeaderV4(header, 9)
	if err != nil {
		t.Fatalf("serialize repeated V4 header: %v", err)
	}
	if !bytes.Equal(first, repeated) {
		t.Fatal("same V4 header produced different bytes")
	}

	nftDigestB := sha256.Sum256([]byte("nft-record-set-b"))
	header.NFTRecordDigest = nftDigestB[:]

	changed, err := SerializePoWHeaderV4(header, 9)
	if err != nil {
		t.Fatalf("serialize changed V4 header: %v", err)
	}
	if bytes.Equal(first, changed) {
		t.Fatal("changing the NFT record digest did not change V4 PoW bytes")
	}
}
