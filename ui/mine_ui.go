package ui

import (
	"fmt"
	"time"

	"quantumcoin/blockchain"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Bu fonksiyonu aynen kullan:
func ShowMineWindow(a fyne.App, w fyne.Window, minerAddress string, bc *blockchain.Blockchain) {
	w.SetTitle("Madencilik")

	statusLabel := widget.NewLabel("Durum: Hazır")
	lastBlockLabel := widget.NewLabel("Henüz blok bulunamadı")

	miningActive := false
	var miningStopChan chan bool

	startBtn := widget.NewButtonWithIcon("Madenciliği Başlat", theme.MediaPlayIcon(), func() {
    if !miningActive {
        miningActive = true
        miningStopChan = make(chan bool)
        statusLabel.SetText("Madencilik aktif...")

        go func() {
            for miningActive {
                select {
                case <-miningStopChan:
                    miningActive = false
                    time.AfterFunc(0, func() {
                        statusLabel.SetText("Durdu")
                    })
                    return
                default:
                    if bc != nil {
                        block, err := bc.MineBlock(minerAddress, 16)
                        if err != nil {
                            time.AfterFunc(0, func() {
                                statusLabel.SetText(fmt.Sprintf("Hata: %v", err))
                            })
                            miningActive = false
                            return
                        }
                        time.AfterFunc(0, func() {
                            statusLabel.SetText(fmt.Sprintf("Son Blok #%d | Hash: %x", block.Index, block.Hash))
                            lastBlockLabel.SetText(fmt.Sprintf("Hash: %x", block.Hash))
                        })
                    }
                    time.Sleep(5 * time.Second)
                }
            }
        }()
    }
})
