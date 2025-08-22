package ui

import (
	"encoding/hex"
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

// Renkler
var (
	green  = color.NRGBA{R: 46, G: 204, B: 113, A: 255}
	orange = color.NRGBA{R: 255, G: 154, B: 0, A: 255}
)

// UI güncellemeleri için mesaj
type miningUpdate struct {
	status      string
	rewardText  string
	progress    float64
	animate     bool
	colorFlash  bool
	appendFound *canvas.Text
}

// --- UI thread güvenli çağrı (Fyne 2.x) ---
func runOnMain(f func()) { fyne.Do(f) }

// []byte hash -> kısa hex "xxxxxxxxxxxx..."
func safeHashBytes(b []byte) string {
	h := hex.EncodeToString(b)
	if len(h) > 12 {
		return h[:12] + "..."
	}
	return h
}

func ShowMineWindow(a fyne.App, w fyne.Window, minerAddress string, bc *blockchain.Blockchain) {
	w.SetTitle(i18n.T(CurrentLang, "mine_title"))

	// Üst kısım: durum + ödül + progress
	statusLabel := widget.NewLabel(i18n.T(CurrentLang, "mine_status_idle"))

	rewardLabel := canvas.NewText("", green)
	rewardLabel.TextSize = 22
	rewardLabel.Alignment = fyne.TextAlignCenter

	progress := widget.NewProgressBar()
	progress.Min, progress.Max = 0, 1

	// Animasyon şeridi
	animRect := canvas.NewRectangle(orange)
	animRect.SetMinSize(fyne.NewSize(240, 12))
	animRect.Hide()

	// Bulunan bloklar listesi (scrollable)
	foundBox := container.NewVBox()
	foundScroll := container.NewVScroll(foundBox)
	foundScroll.SetMinSize(fyne.NewSize(520, 140))

	// UI güncellemeleri kanalı
	updateChan := make(chan miningUpdate, 32)

	// UI updater goroutine (her değişikliği ana threade kuyruklar)
	go func() {
		for upd := range updateChan {
			u := upd
			runOnMain(func() {
				if u.status != "" {
					statusLabel.SetText(u.status)
				}
				if u.rewardText != "" || rewardLabel.Text != "" {
					rewardLabel.Text = u.rewardText
					canvas.Refresh(rewardLabel)
				}
				progress.SetValue(u.progress)

				if u.animate {
					animRect.Show()
				} else {
					animRect.Hide()
				}

				if u.colorFlash {
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

				if u.appendFound != nil {
					foundBox.Add(u.appendFound)
					canvas.Refresh(foundBox)
				}
			})
		}
	}()

	// Butonlar
	startBtn := widget.NewButtonWithIcon(i18n.T(CurrentLang, "mine_start"), theme.MediaPlayIcon(), nil)
	stopBtn := widget.NewButtonWithIcon(i18n.T(CurrentLang, "mine_stop"), theme.MediaStopIcon(), nil)

	// Başlat
	startBtn.OnTapped = func() {
		if miner.IsActive() || bc == nil || minerAddress == "" {
			return
		}
		startBtn.Disable()
		stopBtn.Enable()

		updateChan <- miningUpdate{
			status:   i18n.T(CurrentLang, "mine_status_active"),
			progress: 0,
			animate:  true,
		}

		_ = miner.Start(bc, minerAddress, 16, miner.Options{
			OnBlock: func(b *blockchain.Block, st miner.MiningStatus) {
				// UI: son blok + ödül
				updateChan <- miningUpdate{
					rewardText: fmt.Sprintf("✔️ %.8f QC", float64(st.Reward)),
					progress:   1,
					status:     fmt.Sprintf(i18n.T(CurrentLang, "mine_last_block"), b.Index, safeHashBytes(b.Hash)),
					animate:    true,
					appendFound: func() *canvas.Text {
						line := fmt.Sprintf("[BLOCK FOUND] h=%d hash=%s reward=%.8f QC",
							b.Index, safeHashBytes(b.Hash), float64(st.Reward))
						t := canvas.NewText(line, green)
						t.TextSize = 14
						return t
					}(),
				}

				// Sistem bildirimi
				if app := fyne.CurrentApp(); app != nil {
					app.SendNotification(&fyne.Notification{
						Title:   "QuantumCoin",
						Content: fmt.Sprintf("🎉 %.8f QC", float64(st.Reward)),
					})
				}

				// Küçük animasyon
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
		updateChan <- miningUpdate{
			status:   i18n.T(CurrentLang, "mine_status_idle"),
			progress: 0,
			animate:  false,
		}
	}

	// İlk buton durumları
	if bc == nil || minerAddress == "" {
		startBtn.Disable()
	}
	if !miner.IsActive() {
		stopBtn.Disable()
	}

	// Layout
	content := container.NewVBox(
		widget.NewLabelWithStyle(
			i18n.T(CurrentLang, "mine_title"),
			fyne.TextAlignCenter,
			fyne.TextStyle{Bold: true},
		),
		statusLabel,
		progress,
		animRect,
		rewardLabel,
		container.NewHBox(startBtn, stopBtn),
		widget.NewSeparator(),
		widget.NewLabel(i18n.T(CurrentLang, "mine_found_blocks")),
		foundScroll,
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(640, 480))
	w.Show()
}
