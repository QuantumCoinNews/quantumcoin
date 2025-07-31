package ui

import (
	"fmt"
	"quantumcoin/blockchain"
	"quantumcoin/wallet"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Gönder penceresi, Fyne thread-safe & kullanıcı dostu!
func ShowSendWindow(a fyne.App, w fyne.Window, wlt *wallet.Wallet, bc *blockchain.Blockchain) {
	w.SetTitle("Transfer Yap")
	toEntry := widget.NewEntry()
	toEntry.SetPlaceHolder("Alıcı Adresi")
	amountEntry := widget.NewEntry()
	amountEntry.SetPlaceHolder("Miktar (QC)")

	// YARDIMCI: Adres minimum uzunluk kontrolü (isteğe bağlı, QC adres standardına göre ayarlayabilirsin)
	isAddressValid := func(addr string) bool {
		return len(strings.TrimSpace(addr)) >= 20 // örnek: min 20 karakter
	}

	sendBtn := widget.NewButton("Gönder", func() {
		to := strings.TrimSpace(toEntry.Text)
		amountStr := strings.TrimSpace(amountEntry.Text)

		if to == "" || amountStr == "" {
			dialog.ShowError(fmt.Errorf("Lütfen tüm alanları doldurun!"), w)
			return
		}
		if !isAddressValid(to) {
			dialog.ShowError(fmt.Errorf("Geçersiz alıcı adresi!"), w)
			return
		}
		amount, err := strconv.Atoi(amountStr)
		if err != nil || amount <= 0 {
			dialog.ShowError(fmt.Errorf("Geçersiz miktar!"), w)
			return
		}
		if amount > 1000000000 {
			dialog.ShowError(fmt.Errorf("Çok yüksek miktar!"), w)
			return
		}
		if bc == nil {
			dialog.ShowError(fmt.Errorf("Blockchain bağlı değil!"), w)
			return
		}
		// Varlık/bakiye kontrolü örnek (opsiyonel)
		balance := bc.GetBalance(wlt.GetAddress())
		if amount > balance {
			dialog.ShowError(fmt.Errorf("Yetersiz bakiye! Mevcut: %d QC", balance), w)
			return
		}
		tx, err := blockchain.NewTransaction(wlt.GetAddress(), to, amount, bc)
		if err != nil {
			dialog.ShowError(fmt.Errorf("İşlem hatası: %v", err), w)
			return
		}
		err = bc.AddTransaction(tx)
		if err != nil {
			dialog.ShowError(fmt.Errorf("İşlem havuza eklenemedi: %v", err), w)
			return
		}
		dialog.ShowInformation("Başarılı", "Transfer gönderildi!", w)
		toEntry.SetText("")
		amountEntry.SetText("")
		// w.Close() // istersen pencereyi otomatik kapat
	})

	form := container.NewVBox(
		widget.NewLabelWithStyle("QC Gönder", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		toEntry,
		amountEntry,
		sendBtn,
	)

	w.SetContent(container.NewCenter(form))
	w.Resize(fyne.NewSize(400, 260))
	w.Show()
}
