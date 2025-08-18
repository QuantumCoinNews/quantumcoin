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
	if wlt != nil {
		addressEntry.SetText(wlt.GetAddress())
	}

	balanceLabel := widget.NewLabel("QC Balance: —")
	statusLabel := widget.NewLabel(i18n.T(CurrentLang, "mine_last_block_none"))
	miningStatus := widget.NewLabel(i18n.T(CurrentLang, "mine_status_idle"))

	refresh := func() {
		if bc == nil || wlt == nil {
			return
		}
		bal := bc.GetBalance(wlt.GetAddress())
		height := bc.GetBestHeight()
		balanceLabel.SetText(fmt.Sprintf("QC Balance: %d", bal))
		statusLabel.SetText(fmt.Sprintf("Height: %d", height))
	}

	startBtn := widget.NewButton(i18n.T(CurrentLang, "mine_start"), func() {
		miningStatus.SetText(i18n.T(CurrentLang, "mine_status_active"))
		// not: gerçek worker bağlandığında burası değişecek
		go func() {
			if bc == nil || wlt == nil {
				return
			}
			block, err := bc.MineBlock(wlt.GetAddress(), 16)
			if err != nil {
				miningStatus.SetText(fmt.Sprintf(i18n.T(CurrentLang, "mine_error"), err))
				return
			}
			miningStatus.SetText(fmt.Sprintf(i18n.T(CurrentLang, "mine_last_block"), block.Index, fmt.Sprintf("%x", block.Hash)))
			refresh()
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

	refresh()
	w.SetContent(content)
	w.Resize(fyne.NewSize(600, 420))
	w.Show()
}
