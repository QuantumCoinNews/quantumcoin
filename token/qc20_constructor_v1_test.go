package token

import "testing"

func TestQC20ConstructorV1OwnerlessHasNoPhantomSupply(
	t *testing.T,
) {
	const requestedSupply uint64 = 1_000_000

	subject := NewQC20Token(
		"Legacy Token",
		"LGC",
		requestedSupply,
		8,
	)

	if subject == nil {
		t.Fatal("ownerless constructor returned nil")
	}
	if got := subject.Owner(); got != "" {
		t.Fatalf("owner = %q, want empty", got)
	}
	if got := subject.MaxSupply(); got != requestedSupply {
		t.Fatalf(
			"max supply = %d, want %d",
			got,
			requestedSupply,
		)
	}
	if got := subject.TotalSupply(); got != 0 {
		t.Fatalf(
			"ownerless total supply = %d, want 0",
			got,
		)
	}
	if got := len(subject.BalancesSnapshotV1()); got != 0 {
		t.Fatalf(
			"ownerless balance entry count = %d, want 0",
			got,
		)
	}

	err := subject.MintByOwner(
		"",
		"recipient",
		1,
	)
	if err != ErrTokenOwnerNotConfigured {
		t.Fatalf(
			"ownerless mint error = %v, want %v",
			err,
			ErrTokenOwnerNotConfigured,
		)
	}
	if got := subject.TotalSupply(); got != 0 {
		t.Fatalf(
			"failed ownerless mint changed total supply to %d",
			got,
		)
	}
}

func TestQC20ConstructorV1ExplicitEmptyOwnerHasNoPhantomSupply(
	t *testing.T,
) {
	const requestedSupply uint64 = 750_000

	subject := NewQC20TokenWithOwner(
		"Empty Owner Token",
		"EOT",
		requestedSupply,
		8,
		"",
	)

	if subject == nil {
		t.Fatal("explicit empty-owner constructor returned nil")
	}
	if got := subject.MaxSupply(); got != requestedSupply {
		t.Fatalf(
			"max supply = %d, want %d",
			got,
			requestedSupply,
		)
	}
	if got := subject.TotalSupply(); got != 0 {
		t.Fatalf(
			"empty-owner total supply = %d, want 0",
			got,
		)
	}
	if got := len(subject.BalancesSnapshotV1()); got != 0 {
		t.Fatalf(
			"empty-owner balance entry count = %d, want 0",
			got,
		)
	}
}

func TestQC20ConstructorV1OwnerReceivesFixedInitialSupply(
	t *testing.T,
) {
	const (
		owner         = "token-owner"
		initialSupply = uint64(1_000_000)
	)

	subject := NewQC20TokenWithOwner(
		"Fixed Token",
		"FIX",
		initialSupply,
		8,
		owner,
	)

	if subject == nil {
		t.Fatal("owner constructor returned nil")
	}
	if got := subject.Owner(); got != owner {
		t.Fatalf(
			"owner = %q, want %q",
			got,
			owner,
		)
	}
	if got := subject.MaxSupply(); got != initialSupply {
		t.Fatalf(
			"max supply = %d, want %d",
			got,
			initialSupply,
		)
	}
	if got := subject.TotalSupply(); got != initialSupply {
		t.Fatalf(
			"total supply = %d, want %d",
			got,
			initialSupply,
		)
	}
	if got := subject.BalanceOf(owner); got != initialSupply {
		t.Fatalf(
			"owner balance = %d, want %d",
			got,
			initialSupply,
		)
	}

	balances := subject.BalancesSnapshotV1()
	if len(balances) != 1 {
		t.Fatalf(
			"balance entry count = %d, want 1",
			len(balances),
		)
	}

	if err := subject.MintByOwner(
		owner,
		"other",
		1,
	); err == nil {
		t.Fatal("mint above fixed initial supply unexpectedly succeeded")
	}

	if got := subject.TotalSupply(); got != initialSupply {
		t.Fatalf(
			"rejected mint changed total supply to %d",
			got,
		)
	}
	if got := subject.BalanceOf("other"); got != 0 {
		t.Fatalf(
			"rejected mint created recipient balance %d",
			got,
		)
	}
}
