package ui

import (
	"fmt"
	"image/color"
	"math/rand"
	"sync/atomic"
	"time"

	"quantumcoin/blockchain"

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
	hashInfo   string
}

func ShowMineWindow(a fyne.App, w fyne.Window, minerAddress string, bc *blockchain.Blockchain) {
	w.SetTitle("Madencilik")

	statusLabel := widget.NewLabel("⛏️ Hazır")
	rewardLabel := canvas.NewText("", green)
	rewardLabel.TextSize = 22
	rewardLabel.Alignment = fyne.TextAlignCenter

	progress := widget.NewProgressBar()
	progress.Min = 0
	progress.Max = 1

	var miningActive int32
	var miningStopChan chan bool

	animRect := canvas.NewRectangle(orange)
	animRect.SetMinSize(fyne.NewSize(240, 12))
	animRect.Hide()

	updateChan := make(chan miningUpdate, 10)

	// UI updater: Thread-safe!
	go func() {
		for upd := range updateChan {
			fyne.Do(func() {
				statusLabel.SetText(upd.status)
				rewardLabel.Text = upd.rewardText
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
					canvas.Refresh(animRect)
				} else {
					animRect.FillColor = orange
					canvas.Refresh(animRect)
				}
				if upd.hashInfo != "" {
					statusLabel.SetText(upd.hashInfo)
				}
				canvas.Refresh(rewardLabel)
				canvas.Refresh(statusLabel)
			})
		}
	}()

	startBtn := widget.NewButtonWithIcon("Madenciliği Başlat", theme.MediaPlayIcon(), func() {
		if atomic.LoadInt32(&miningActive) == 1 {
			return
		}
		atomic.StoreInt32(&miningActive, 1)
		miningStopChan = make(chan bool)

		updateChan <- miningUpdate{
			status:   "⛏️ Madencilik aktif...",
			progress: 0,
			animate:  true,
		}
		rewardLabel.Text = ""

		// Progress Animasyonu
		go func() {
			animationStep := 0.0
			for atomic.LoadInt32(&miningActive) == 1 {
				select {
				case <-miningStopChan:
					atomic.StoreInt32(&miningActive, 0)
					updateChan <- miningUpdate{status: "⏸️ Durdu", progress: 0, animate: false}
					fyne.Do(func() {
						a.SendNotification(&fyne.Notification{Title: "QuantumCoin", Content: "Madencilik durduruldu!"})
					})
					return
				default:
					updateChan <- miningUpdate{progress: animationStep, animate: true}
					if animationStep < 1.0 {
						animationStep += 0.04 + rand.Float64()*0.03
					} else {
						animationStep = 0
					}
					time.Sleep(300 * time.Millisecond)
				}
			}
		}()

		// Mining Thread (gerçek PoW blok kazımı)
		go func() {
			for atomic.LoadInt32(&miningActive) == 1 {
				if bc != nil {
					block, err := bc.MineBlock(minerAddress, 16)
					reward := blockchain.GetCurrentReward()
					if err != nil {
						updateChan <- miningUpdate{status: fmt.Sprintf("Hata: %v", err)}
						fyne.Do(func() {
							a.SendNotification(&fyne.Notification{Title: "QuantumCoin", Content: fmt.Sprintf("Hata: %v", err)})
						})
						atomic.StoreInt32(&miningActive, 0)
						return
					}
					// Başarı efekti
					updateChan <- miningUpdate{
						rewardText: fmt.Sprintf("✔️  %d QC ödül kazandınız! (Block #%d)", reward, block.Index),
						progress:   1,
						status:     fmt.Sprintf("✅ Son Blok #%d | Hash: %x", block.Index, block.Hash),
						animate:    true,
					}
					fyne.Do(func() {
						a.SendNotification(&fyne.Notification{Title: "QuantumCoin", Content: fmt.Sprintf("🎉 %d QC kazandınız!", reward)})
					})

					time.Sleep(1800 * time.Millisecond)
					updateChan <- miningUpdate{rewardText: "", progress: 0}

					// Renk animasyonu
					for i := 0; i < 8; i++ {
						updateChan <- miningUpdate{colorFlash: true, animate: true}
						time.Sleep(60 * time.Millisecond)
					}
					updateChan <- miningUpdate{colorFlash: false, animate: true}
				}
			}
		}()
	})

	stopBtn := widget.NewButtonWithIcon("Durdur", theme.MediaStopIcon(), func() {
		if atomic.LoadInt32(&miningActive) == 1 && miningStopChan != nil {
			miningStopChan <- true
		}
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("⛏️ Madencilik (QuantumCoin)", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		statusLabel,
		progress,
		animRect,
		rewardLabel,
		container.NewHBox(startBtn, stopBtn),
	)
	w.SetContent(content)
	w.Resize(fyne.NewSize(520, 340))
	w.Show()
}
