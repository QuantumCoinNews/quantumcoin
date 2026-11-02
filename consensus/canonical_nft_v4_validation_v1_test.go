package consensus

import (
	"strings"
	"testing"
)

func nftV4CanonicalValidationInputV1(
	t *testing.T,
) CanonicalBlockValidationInputV1 {
	t.Helper()

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

	return CanonicalBlockValidationInputV1{
		Protocol:    ProtocolNFTV4,
		Height:      1,
		PoWHeaderV4: header,
		Nonce:       nonce,
		StoredHash:  append([]byte(nil), hash[:]...),
		Transactions: []*CoinbaseTransactionViewV1{
			{Coinbase: true, OutputAmounts: []int64{53}},
		},
		ExpectedSubsidy: 50,
		VerifiedFees:    3,
		TotalSupply:     25_500_000,
	}
}

func TestValidateCanonicalBlockV1DispatchesNFTV4PoW(t *testing.T) {
	input := nftV4CanonicalValidationInputV1(t)

	if _, err := ValidateCanonicalBlockV1(input); err != nil {
		t.Fatalf("valid NFT V4 canonical block was rejected: %v", err)
	}

	mutated := input
	mutated.PoWHeaderV4.NFTRecordDigest = append(
		[]byte(nil),
		input.PoWHeaderV4.NFTRecordDigest...,
	)
	mutated.PoWHeaderV4.NFTRecordDigest[0] ^= 0x01

	_, err := ValidateCanonicalBlockV1(mutated)
	if err == nil {
		t.Fatal("canonical validation accepted a changed NFT record digest")
	}
	if !strings.Contains(err.Error(), "proof-of-work validation failed") {
		t.Fatalf("unexpected rejection reason for changed NFT digest: %v", err)
	}
}
