package ui

import (
	"quantumcoin/blockchain"
	"quantumcoin/wallet"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Ana UI (sekme) fonksiyonu artık bc de alıyor!
func LaunchMainUI(a fyne.App, w fyne.Window, wlt *wallet.Wallet, bc *blockchain.Blockchain) {
	w.SetTitle("QuantumCoin")

	addressEntry := widget.NewEntry()
	addressEntry.Disable()
	addressEntry.SetText(wlt.GetAddress())

	walletTab := container.NewVBox(
		widget.NewLabelWithStyle("👛 Cüzdan Adresiniz", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		addressEntry,
	)

	sendTab := container.NewCenter(widget.NewButton("Gönderim Penceresini Aç", func() {
		sendWin := a.NewWindow("Gönder")
		ShowSendWindow(a, sendWin, wlt, bc)
	}))

	mineTab := container.NewCenter(widget.NewButton("Madencilik Penceresini Aç", func() {
		mineWin := a.NewWindow("Madencilik")
		ShowMineWindow(a, mineWin, wlt.GetAddress(), bc)
	}))

	explorerTab := container.NewCenter(widget.NewButton("Blok Gezgini", func() {
		expWin := a.NewWindow("Explorer")
		ShowExplorerWindow(a, expWin, bc)
	}))

	settingsTab := container.NewVBox(
		widget.NewLabelWithStyle("⚙️ Ayarlar", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel("(Henüz uygulanmadı)"),
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("Cüzdan", walletTab),
		container.NewTabItem("Gönder", sendTab),
		container.NewTabItem("Madencilik", mineTab),
		container.NewTabItem("Gezgin", explorerTab),
		container.NewTabItem("Ayarlar", settingsTab),
	)

	w.SetContent(tabs)
	w.Resize(fyne.NewSize(900, 600))
	w.Show()
}
