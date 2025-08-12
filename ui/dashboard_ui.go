package ui

import (
	"fmt"

	"quantumcoin/blockchain"
	"quantumcoin/i18n"
	"quantumcoin/wallet"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func ShowDashboardWindow(a fyne.App, w fyne.Window, bc *blockchain.Blockchain, wlt *wallet.Wallet) {
	w.SetTitle("QuantumCoin Dashboard")

	addressEntry := widget.NewEntry()
	addressEntry.Disable()
	addressEntry.SetText(wlt.GetAddress())

	refresh := func(balanceLabel, statusLabel *widget.Label) {
		if bc == nil {
			return
		}
		bal := bc.GetBalance(wlt.GetAddress())
		height := bc.GetBestHeight()
		balanceLabel.SetText(fmt.Sprintf("%s %d QC", i18n.T(CurrentLang, "explorer_tx_out")[:6], bal)) // "Amount:" yerine kısa kullanıyoruz
		statusLabel.SetText(fmt.Sprintf("Height: %d", height))
	}

	balanceLabel := widget.NewLabel(i18n.T(CurrentLang, "mine_status_idle"))
	statusLabel := widget.NewLabel(i18n.T(CurrentLang, "mine_last_block_none"))
	miningStatus := widget.NewLabel(i18n.T(CurrentLang, "mine_status_idle"))

	startBtn := widget.NewButton(i18n.T(CurrentLang, "mine_start"), func() {
		miningStatus.SetText(i18n.T(CurrentLang, "mine_status_active"))
		go func() {
			block, err := bc.MineBlock(wlt.GetAddress(), 16)
			if err != nil {
				fyne.Do(func() {
					miningStatus.SetText(fmt.Sprintf(i18n.T(CurrentLang, "mine_error"), err))
				})
				return
			}
			fyne.Do(func() {
				miningStatus.SetText(fmt.Sprintf(i18n.T(CurrentLang, "mine_last_block"), block.Index, block.Hash))
				refresh(balanceLabel, statusLabel)
			})
		}()
	})

	stopBtn := widget.NewButton(i18n.T(CurrentLang, "mine_stop"), func() {
		miningStatus.SetText(i18n.T(CurrentLang, "mine_status_idle"))
	})

	sendBtn := widget.NewButton(i18n.T(CurrentLang, "send_title"), func() {
		sendWin := a.NewWindow(i18n.T(CurrentLang, "send_title"))
		ShowSendWindow(a, sendWin, wlt, bc)
	})

	explorerBtn := widget.NewButton(i18n.T(CurrentLang, "explorer_title"), func() {
		expWin := a.NewWindow(i18n.T(CurrentLang, "explorer_title"))
		ShowExplorerWindow(a, expWin, bc)
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle(i18n.T(CurrentLang, "wallet_address"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		addressEntry,
		balanceLabel,
		statusLabel,
		miningStatus,
		container.NewHBox(startBtn, stopBtn),
		sendBtn,
		explorerBtn,
	)

	refresh(balanceLabel, statusLabel)
	w.SetContent(content)
	w.Resize(fyne.NewSize(600, 420))
	w.Show()
}
