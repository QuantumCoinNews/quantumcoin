package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"quantumcoin/blockchain"
	"quantumcoin/wallet"
)

var bc *blockchain.Blockchain
var wlt *wallet.Wallet

func handleSend(w http.ResponseWriter, r *http.Request) {
	type SendReq struct {
		To     string  `json:"to"`
		Amount float64 `json:"amount"`
	}
	var req SendReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz JSON", http.StatusBadRequest)
		return
	}
	if req.To == "" || req.Amount <= 0 {
		http.Error(w, "Adres ve miktar zorunludur", http.StatusBadRequest)
		return
	}
	tx, err := blockchain.NewTransaction(wlt.GetAddress(), req.To, int(req.Amount), bc)
	if err != nil {
		http.Error(w, "İşlem oluşturulamadı: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = bc.AddTransaction(tx)
	if err != nil {
		http.Error(w, "İşlem havuza eklenemedi: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, `{"status": "pending", "to": "%s", "amount": %.2f}`, req.To, req.Amount)
}
