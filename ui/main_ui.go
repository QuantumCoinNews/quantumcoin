package ui

import (
	"quantumcoin/blockchain"
	"quantumcoin/i18n"
	"quantumcoin/wallet"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Ana UI (sekme)
func LaunchMainUI(a fyne.App, w fyne.Window, wlt *wallet.Wallet, bc *blockchain.Blockchain) {
	w.SetTitle("QuantumCoin")

	addressEntry := widget.NewEntry()
	addressEntry.Disable()
	addressEntry.SetText(wlt.GetAddress())

	walletTab := container.NewVBox(
		widget.NewLabelWithStyle(i18n.T(CurrentLang, "wallet_address"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		addressEntry,
	)

	sendTab := container.NewCenter(widget.NewButton(i18n.T(CurrentLang, "send_title"), func() {
		sendWin := a.NewWindow(i18n.T(CurrentLang, "send_title"))
		ShowSendWindow(a, sendWin, wlt, bc)
	}))

	mineTab := container.NewCenter(widget.NewButton(i18n.T(CurrentLang, "mine_title"), func() {
		mineWin := a.NewWindow(i18n.T(CurrentLang, "mine_title"))
		ShowMineWindow(a, mineWin, wlt.GetAddress(), bc)
	}))

	explorerTab := container.NewCenter(widget.NewButton(i18n.T(CurrentLang, "explorer_title"), func() {
		expWin := a.NewWindow(i18n.T(CurrentLang, "explorer_title"))
		ShowExplorerWindow(a, expWin, bc)
	}))

	settingsTab := container.NewVBox(
		widget.NewLabelWithStyle(i18n.T(CurrentLang, "settings_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewButton(i18n.T(CurrentLang, "settings_open"), func() {
			ShowSettingsWindow(a)
		}),
	)

	tabs := container.NewAppTabs(
		container.NewTabItem(i18n.T(CurrentLang, "wallet_tab"), walletTab),
		container.NewTabItem(i18n.T(CurrentLang, "send_tab"), sendTab),
		container.NewTabItem(i18n.T(CurrentLang, "mine_tab"), mineTab),
		container.NewTabItem(i18n.T(CurrentLang, "explorer_tab"), explorerTab),
		container.NewTabItem(i18n.T(CurrentLang, "settings_tab"), settingsTab),
	)

	w.SetContent(tabs)
	w.Resize(fyne.NewSize(900, 600))
	w.Show()
}
