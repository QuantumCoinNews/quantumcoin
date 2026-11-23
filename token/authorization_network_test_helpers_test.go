package token

import "testing"

const testTokenNetworkMagicV1 uint32 = 0x51435431
const testTokenGenesisHashV1 = "0101010101010101010101010101010101010101010101010101010101010101"

func newTestAuthorizedToken(t *testing.T, symbol, name string, decimals uint8, maxSupply uint64, owner string) *Token {
	t.Helper()
	initialSupply := uint64(100)
	if maxSupply != 0 && initialSupply > maxSupply {
		initialSupply = maxSupply
	}
	subject, err := NewValidatedTokenWithInitialSupplyForNetworkV1(symbol, name, decimals, maxSupply, initialSupply, owner, testTokenNetworkMagicV1, testTokenGenesisHashV1)
	if err != nil {
		t.Fatalf("create test token: %v", err)
	}
	return subject
}

func newTestAuthorizedQC20Token(t *testing.T, name, symbol string, supply uint64, decimals uint8, owner string) *QC20Token {
	t.Helper()
	subject, err := NewValidatedQC20TokenForNetworkV1(name, symbol, supply, decimals, owner, testTokenNetworkMagicV1, testTokenGenesisHashV1)
	if err != nil {
		t.Fatalf("create test QC20 token: %v", err)
	}
	return subject
}

func newTestAuthorizedQC721Token(t *testing.T, name, symbol, owner string) *QC721Token {
	t.Helper()
	subject, err := NewValidatedQC721TokenForNetworkV1(name, symbol, owner, testTokenNetworkMagicV1, testTokenGenesisHashV1)
	if err != nil {
		t.Fatalf("create test QC721 token: %v", err)
	}
	return subject
}

func newTestAuthorizedQC721WithItems(t *testing.T, name, symbol, owner string, items []QC721InitialItemV1) *QC721Token {
	t.Helper()
	subject, err := NewValidatedQC721TokenWithInitialItemsForNetworkV1(name, symbol, owner, testTokenNetworkMagicV1, testTokenGenesisHashV1, items)
	if err != nil {
		t.Fatalf("create test QC721 collection: %v", err)
	}
	return subject
}

func newTestTokenWithAllocations(t *testing.T, symbol, name string, decimals uint8, maxSupply uint64, owner string, allocations map[string]uint64) *Token {
	t.Helper()
	subject, err := NewValidatedTokenWithInitialAllocationsForNetworkV1(symbol, name, decimals, maxSupply, owner, testTokenNetworkMagicV1, testTokenGenesisHashV1, allocations)
	if err != nil {
		t.Fatalf("create test token with allocations: %v", err)
	}
	return subject
}
