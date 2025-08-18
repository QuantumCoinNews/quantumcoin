package ui

import (
	"fmt"
	"strconv"
	"strings"

	"quantumcoin/blockchain"
	"quantumcoin/i18n"
	"quantumcoin/wallet"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func ShowSendWindow(a fyne.App, w fyne.Window, wlt *wallet.Wallet, bc *blockchain.Blockchain) {
	w.SetTitle(i18n.T(CurrentLang, "send_title"))
	toEntry := widget.NewEntry()
	toEntry.SetPlaceHolder(i18n.T(CurrentLang, "send_to_placeholder"))
	amountEntry := widget.NewEntry()
	amountEntry.SetPlaceHolder(i18n.T(CurrentLang, "send_amount_placeholder"))

	sendBtn := widget.NewButton(i18n.T(CurrentLang, "send_button"), func() {
		to := strings.TrimSpace(toEntry.Text)
		amountStr := strings.TrimSpace(amountEntry.Text)

		if to == "" || amountStr == "" {
			dialog.ShowError(fmt.Errorf(i18n.T(CurrentLang, "error_fill_fields")), w)
			return
		}
		if !wallet.ValidateAddress(to) {
			dialog.ShowError(fmt.Errorf(i18n.T(CurrentLang, "error_invalid_address")), w)
			return
		}
		amount, err := strconv.Atoi(amountStr)
		if err != nil || amount <= 0 {
			dialog.ShowError(fmt.Errorf(i18n.T(CurrentLang, "error_invalid_amount")), w)
			return
		}
		if bc == nil || wlt == nil {
			dialog.ShowError(fmt.Errorf(i18n.T(CurrentLang, "error_blockchain_not_connected")), w)
			return
		}
		balance := bc.GetBalance(wlt.GetAddress())
		if amount > balance {
			dialog.ShowError(fmt.Errorf(i18n.T(CurrentLang, "error_insufficient_balance")), w)
			return
		}
		tx, err := blockchain.NewTransaction(wlt.GetAddress(), to, amount, bc)
		if err != nil {
			dialog.ShowError(fmt.Errorf("%s: %v", i18n.T(CurrentLang, "error_tx_create"), err), w)
			return
		}
		if err := bc.AddTransaction(tx); err != nil {
			dialog.ShowError(fmt.Errorf("%s: %v", i18n.T(CurrentLang, "error_tx_add"), err), w)
			return
		}
		dialog.ShowInformation(i18n.T(CurrentLang, "success"), i18n.T(CurrentLang, "send_success"), w)
		toEntry.SetText("")
		amountEntry.SetText("")
	})

	form := container.NewVBox(
		widget.NewLabelWithStyle(i18n.T(CurrentLang, "send_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		toEntry,
		amountEntry,
		sendBtn,
	)

	w.SetContent(container.NewCenter(form))
	w.Resize(fyne.NewSize(420, 280))
	w.Show()
}
