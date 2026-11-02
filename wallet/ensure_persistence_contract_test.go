package wallet

import "testing"

func TestEnsureDefaultWalletPersistenceRoundTrip(t *testing.T) {
	t.Setenv(
		"APPDATA",
		t.TempDir(),
	)

	created, err := EnsureDefaultWallet()
	if err != nil {
		t.Fatalf(
			"EnsureDefaultWallet: %v",
			err,
		)
	}

	if created == nil {
		t.Fatal("EnsureDefaultWallet returned nil wallet")
	}

	if created.Address == "" {
		t.Fatal("EnsureDefaultWallet returned empty address")
	}

	loaded, err := LoadDefaultWallet()
	if err != nil {
		t.Fatalf(
			"LoadDefaultWallet: %v",
			err,
		)
	}

	if loaded.Address != created.Address {
		t.Fatalf(
			"loaded address=%q created=%q",
			loaded.Address,
			created.Address,
		)
	}

	if loaded.PrivateKeyHex != created.PrivateKeyHex {
		t.Fatal("persisted private key does not round-trip")
	}
}

func TestSaveDefaultWalletPersists(t *testing.T) {
	t.Setenv(
		"APPDATA",
		t.TempDir(),
	)

	want := &DefaultWallet{
		PrivateKeyHex: "test-private-key",
		Address:       "test-address",
	}

	if err := SaveDefaultWallet(want); err != nil {
		t.Fatalf(
			"SaveDefaultWallet: %v",
			err,
		)
	}

	got, err := LoadDefaultWallet()
	if err != nil {
		t.Fatalf(
			"LoadDefaultWallet: %v",
			err,
		)
	}

	if got.Address != want.Address {
		t.Fatalf(
			"address=%q want=%q",
			got.Address,
			want.Address,
		)
	}

	if got.PrivateKeyHex != want.PrivateKeyHex {
		t.Fatal("private key persistence mismatch")
	}
}
