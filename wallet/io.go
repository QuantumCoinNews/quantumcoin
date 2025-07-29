package wallet

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"quantumcoin/utils"
)

const walletFile = "wallet_data.json"

// SaveWalletToFile: Cüzdanı JSON olarak dosyaya yaz
func SaveWalletToFile(w *Wallet) {
	data, err := json.Marshal(w)
	if err != nil {
		log.Fatalf("Cüzdan serileştirme hatası: %v", err)
	}
	err = ioutil.WriteFile(walletFile, data, 0644)
	if err != nil {
		log.Fatalf("Cüzdan dosyası yazılamadı: %v", err)
	}
}

// LoadWalletFromFile: Cüzdanı JSON dosyasından yükle, yoksa yenisini oluştur
func LoadWalletFromFile() *Wallet {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return NewWallet()
	}
	data, err := ioutil.ReadFile(walletFile)
	if err != nil {
		log.Fatalf("Cüzdan dosyası okunamadı: %v", err)
	}
	var w Wallet
	err = json.Unmarshal(data, &w)
	if err != nil {
		log.Fatalf("Cüzdan parse edilemedi: %v", err)
	}
	return &w
}

// Base58DecodeAddress: QC adresinden pubKeyHash'i çıkart
func Base58DecodeAddress(address string) []byte {
	decoded, err := utils.Base58Decode([]byte(address))
	if err != nil {
		log.Fatalf("Adres çözümlenemedi: %v", err)
	}
	if len(decoded) < 5 {
		log.Fatal("Adres formatı geçersiz")
	}
	// decoded: [version][pubKeyHash][checksum]
	return decoded[1 : len(decoded)-4]
}
