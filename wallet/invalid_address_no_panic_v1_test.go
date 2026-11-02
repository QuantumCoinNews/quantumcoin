package wallet

import "testing"

func TestBase58DecodeAddressInvalidDoesNotPanicV1(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Base58DecodeAddress must not panic on invalid address: %v", r)
		}
	}()

	got := Base58DecodeAddress("not-a-valid-qc-address@@@")
	if got != nil {
		t.Fatalf("expected nil pubKeyHash for invalid address, got %x", got)
	}
}

func TestBase58DecodeAddressSafeInvalidReturnsErrorV1(t *testing.T) {
	if _, err := Base58DecodeAddressSafe("not-a-valid-qc-address@@@"); err == nil {
		t.Fatalf("expected safe decoder to reject invalid address")
	}
}

func TestValidateAddressInvalidReturnsFalseV1(t *testing.T) {
	if ValidateAddress("not-a-valid-qc-address@@@") {
		t.Fatalf("expected invalid address to be false")
	}
}
