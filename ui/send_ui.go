package ui

import (
	"fmt"
	"quantumcoin/blockchain"
	"quantumcoin/wallet"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func ShowSendWindow(a fyne.App, w fyne.Window, wlt *wallet.Wallet, bc *blockchain.Blockchain) {
	w.SetTitle("Transfer Yap")
	toEntry := widget.NewEntry()
	toEntry.SetPlaceHolder("Alıcı Adresi")
	amountEntry := widget.NewEntry()
	amountEntry.SetPlaceHolder("Miktar (QC)")

	sendBtn := widget.NewButton("Gönder", func() {
		to := toEntry.Text
		amountStr := amountEntry.Text
		if to == "" || amountStr == "" {
			dialog.ShowError(fmt.Errorf("Alanları doldurun!"), w)
			return
		}
		amount, err := strconv.Atoi(amountStr)
		if err != nil || amount <= 0 {
			dialog.ShowError(fmt.Errorf("Geçersiz miktar!"), w)
			return
		}
		if bc == nil {
			dialog.ShowError(fmt.Errorf("Blockchain bağlı değil!"), w)
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
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("QC Gönder", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		toEntry,
		amountEntry,
		sendBtn,
	)

	w.SetContent(container.NewCenter(content))
	w.Resize(fyne.NewSize(400, 250))
	w.Show()
}
