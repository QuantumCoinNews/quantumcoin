package ui

import (
	"fmt"
	"image/color"
	"math/rand"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/i18n"
	"quantumcoin/miner"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	green  = color.NRGBA{R: 46, G: 204, B: 113, A: 255}
	orange = color.NRGBA{R: 255, G: 154, B: 0, A: 255}
)

type miningUpdate struct {
	status     string
	rewardText string
	progress   float64
	animate    bool
	colorFlash bool
}

func ShowMineWindow(a fyne.App, w fyne.Window, minerAddress string, bc *blockchain.Blockchain) {
	w.SetTitle(i18n.T(CurrentLang, "mine_title"))

	statusLabel := widget.NewLabel(i18n.T(CurrentLang, "mine_status_idle"))
	rewardLabel := canvas.NewText("", green)
	rewardLabel.TextSize = 22
	rewardLabel.Alignment = fyne.TextAlignCenter

	progress := widget.NewProgressBar()
	progress.Min = 0
	progress.Max = 1

	animRect := canvas.NewRectangle(orange)
	animRect.SetMinSize(fyne.NewSize(240, 12))
	animRect.Hide()

	updateChan := make(chan miningUpdate, 16)

	// UI updater
	go func() {
		for upd := range updateChan {
			fyne.Do(func() {
				if upd.status != "" {
					statusLabel.SetText(upd.status)
				}
				if upd.rewardText != "" || rewardLabel.Text != "" {
					rewardLabel.Text = upd.rewardText
					canvas.Refresh(rewardLabel)
				}
				progress.SetValue(upd.progress)

				if upd.animate {
					animRect.Show()
				} else {
					animRect.Hide()
				}

				if upd.colorFlash {
					animRect.FillColor = color.NRGBA{
						R: uint8(rand.Intn(256)),
						G: uint8(rand.Intn(256)),
						B: uint8(rand.Intn(256)),
						A: 255,
					}
				} else {
					animRect.FillColor = orange
				}
				canvas.Refresh(animRect)
			})
		}
	}()

	startBtn := widget.NewButtonWithIcon(i18n.T(CurrentLang, "mine_start"), theme.MediaPlayIcon(), nil)
	stopBtn := widget.NewButtonWithIcon(i18n.T(CurrentLang, "mine_stop"), theme.MediaStopIcon(), nil)

	// Başlat
	startBtn.OnTapped = func() {
		if miner.IsActive() || bc == nil || minerAddress == "" {
			return
		}
		startBtn.Disable()
		stopBtn.Enable()

		updateChan <- miningUpdate{status: i18n.T(CurrentLang, "mine_status_active"), progress: 0, animate: true}

		_ = miner.Start(bc, minerAddress, 16, miner.Options{
			OnBlock: func(b *blockchain.Block, st miner.MiningStatus) {
				updateChan <- miningUpdate{
					rewardText: fmt.Sprintf("✔️ %d QC", st.Reward),
					progress:   1,
					status:     fmt.Sprintf(i18n.T(CurrentLang, "mine_last_block"), b.Index, b.Hash),
					animate:    true,
				}

				if app := fyne.CurrentApp(); app != nil {
					app.SendNotification(&fyne.Notification{
						Title:   "QuantumCoin",
						Content: fmt.Sprintf("🎉 %d QC", st.Reward),
					})
				}

				time.Sleep(900 * time.Millisecond)
				updateChan <- miningUpdate{rewardText: "", progress: 0}
				for i := 0; i < 5; i++ {
					updateChan <- miningUpdate{colorFlash: true, animate: true}
					time.Sleep(60 * time.Millisecond)
				}
				updateChan <- miningUpdate{colorFlash: false, animate: true}
			},
			OnError: func(err error) {
				updateChan <- miningUpdate{
					status:  fmt.Sprintf(i18n.T(CurrentLang, "mine_error"), err),
					animate: false,
				}
			},
		})
	}

	// Durdur
	stopBtn.OnTapped = func() {
		if miner.IsActive() {
			miner.Stop()
		}
		stopBtn.Disable()
		startBtn.Enable()
		updateChan <- miningUpdate{status: i18n.T(CurrentLang, "mine_status_idle"), progress: 0, animate: false}
	}

	// İlk buton durumları
	if bc == nil || minerAddress == "" {
		startBtn.Disable()
	}
	if !miner.IsActive() {
		stopBtn.Disable()
	}

	content := container.NewVBox(
		widget.NewLabelWithStyle(i18n.T(CurrentLang, "mine_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		statusLabel,
		progress,
		animRect,
		rewardLabel,
		container.NewHBox(startBtn, stopBtn),
	)
	w.SetContent(content)
	w.Resize(fyne.NewSize(520, 360))
	w.Show()
}
