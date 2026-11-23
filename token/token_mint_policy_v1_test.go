package token

import (
	"errors"
	"testing"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestTokenPublicMintClosesAfterConstructorIssuanceV1(t *testing.T) {
	owner, recipient := wallet.NewWallet(), wallet.NewWallet()
	subject, err := NewValidatedTokenWithInitialSupplyForNetworkV1("INIT", "Initial", 8, 100, 40, owner.GetAddress(), testTokenNetworkMagicV1, testTokenGenesisHashV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := subject.Mint(owner.GetAddress(), recipient.GetAddress(), 1); !errors.Is(err, ErrTokenMintDisabled) {
		t.Fatalf("owner post-creation mint error = %v", err)
	}
	if err := subject.Mint(recipient.GetAddress(), recipient.GetAddress(), 1); err == nil {
		t.Fatal("non-owner mint succeeded")
	}
	if subject.TotalSupply() != 40 || subject.BalanceOf(recipient.GetAddress()) != 0 {
		t.Fatal("rejected mint changed token state")
	}
}

func TestTokenInitialAllocationsAreBoundedAndAtomicV1(t *testing.T) {
	owner, holder := wallet.NewWallet(), wallet.NewWallet()
	if _, err := NewValidatedTokenWithInitialAllocationsForNetworkV1("INIT", "Initial", 8, 10, owner.GetAddress(), testTokenNetworkMagicV1, testTokenGenesisHashV1, map[string]uint64{owner.GetAddress(): 7, holder.GetAddress(): 4}); !errors.Is(err, ErrTokenMaxSupplyExceeded) {
		t.Fatalf("over-cap initial allocation error = %v", err)
	}
	subject, err := NewValidatedTokenWithInitialAllocationsForNetworkV1("INIT", "Initial", 8, 11, owner.GetAddress(), testTokenNetworkMagicV1, testTokenGenesisHashV1, map[string]uint64{owner.GetAddress(): 7, holder.GetAddress(): 4})
	if err != nil {
		t.Fatal(err)
	}
	if subject.TotalSupply() != 11 || subject.BalanceOf(owner.GetAddress()) != 7 || subject.BalanceOf(holder.GetAddress()) != 4 {
		t.Fatal("initial allocations were not applied atomically")
	}
}

func TestTokenMinerBridgeCannotBeEnabledOnTestnetV1(t *testing.T) {
	for _, key := range []string{"QC_TESTNET", "QC_PUBLIC_TESTNET", "QC_API_PUBLIC_TESTNET", "QC_TESTNET_PUBLIC_API", "QC_MAINNET", "QC_NET_MAGIC", "QC_NETWORK", "QC_NETWORK_MODE", "QC_CHAIN_MODE", "QC_MODE"} {
		t.Setenv(key, "")
	}
	t.Setenv("QC_ENABLE_TOKEN_MINER_BRIDGE", "1")
	t.Setenv("QC_TESTNET", "1")
	if TokenMinerBridgeEnabled() {
		t.Fatal("testnet enabled token mint bridge")
	}

	t.Setenv("QC_TESTNET", "")
	t.Setenv("QC_NET_MAGIC", config.QCTestnetMagicV1)
	if TokenMinerBridgeEnabled() {
		t.Fatal("testnet network magic enabled token mint bridge")
	}
}
