package ui

import (
	"fmt"

	"quantumcoin/blockchain"
	"quantumcoin/miner"
	"quantumcoin/wallet"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// DashboardUI gösterimi, tüm temel bilgileri gösterir
func ShowDashboardWindow(a fyne.App, w fyne.Window, bc *blockchain.Blockchain, wlt *wallet.Wallet) {
	w.SetTitle("QuantumCoin Dashboard")

	addressEntry := widget.NewEntry()
	addressEntry.Disable()
	addressEntry.SetText(wlt.GetAddress())

	bcLocal := bc
	balance := bcLocal.GetBalance(wlt.GetAddress())
	height := bcLocal.GetBestHeight()

	statusLabel := widget.NewLabel(fmt.Sprintf("Son Blok Yüksekliği: %d", height))
	balanceLabel := widget.NewLabel(fmt.Sprintf("Bakiyeniz: %d QC", balance))

	miningStatus := widget.NewLabel("Madencilik Durumu: Pasif")

	startBtn := widget.NewButton("Madenciliği Başlat", func() {
		if !miner.IsMiningActive() {
			miner.StartMining(wlt.GetAddress(), func(status miner.MiningStatus) {
				// UI güncellemesi için main thread gerekebilir
				statusLabel.SetText(fmt.Sprintf("Son Blok Yüksekliği: %d", status.BlockHeight))
				miningStatus.SetText("Madencilik Durumu: Aktif")
			})
		}
	})

	stopBtn := widget.NewButton("Madenciliği Durdur", func() {
		if miner.IsMiningActive() {
			miner.StopMining()
			miningStatus.SetText("Madencilik Durumu: Pasif")
		}
	})

	sendBtn := widget.NewButton("Gönderim Penceresi", func() {
		sendWin := a.NewWindow("Gönder")
		ShowSendWindow(a, sendWin, wlt, bc)
	})

	explorerBtn := widget.NewButton("Blockchain Gezgini", func() {
		expWin := a.NewWindow("Explorer")
		ShowExplorerWindow(a, expWin, bc)
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("👛 Cüzdan Adresi", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		addressEntry,
		balanceLabel,
		statusLabel,
		miningStatus,
		container.NewHBox(startBtn, stopBtn),
		sendBtn,
		explorerBtn,
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(600, 400))
	w.Show()
}
