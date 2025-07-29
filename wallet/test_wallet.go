package wallet

import (
	"testing"
)

// TestWalletCreation cüzdanın doğru şekilde oluşturulduğunu test eder.
func TestWalletCreation(t *testing.T) {
	w := NewWallet()
	if w == nil {
		t.Fatal("Wallet oluşturulamadı: wallet nesnesi nil döndü")
	}
}

// TestWalletAddress oluşturulan cüzdandan adres alınabildiğini test eder.
func TestWalletAddress(t *testing.T) {
	w := NewWallet()
	address := w.GetAddress()
	if address == "" {
		t.Fatal("Adres boş: Cüzdandan geçerli adres alınamadı")
	}
	t.Logf("Oluşturulan adres: %s", address)
}

// TestWalletAddressValidation adresin geçerliliğini test eder.
func TestWalletAddressValidation(t *testing.T) {
	w := NewWallet()
	address := w.GetAddress()
	if !ValidateAddress(address) {
		t.Fatalf("Adres geçersiz: %s", address)
	}
}
