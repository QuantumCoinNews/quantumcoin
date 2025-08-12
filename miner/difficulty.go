package miner

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// DifficultyWidget: görsel zorluk göstergesi (demo)
func DifficultyWidget(currentDifficulty float64) fyne.CanvasObject {
	title := canvas.NewText("⛏️ Mining Difficulty", theme.PrimaryColor())
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	difficultyLabel := widget.NewLabel(fmt.Sprintf("Current: %.4f", currentDifficulty))
	difficultyLabel.Alignment = fyne.TextAlignCenter

	progress := widget.NewProgressBar()
	progress.Min = 0.0
	progress.Max = 10.0
	progress.Value = currentDifficulty

	go func() {
		for {
			time.Sleep(5 * time.Second)
			progress.Value += 0.01
			if progress.Value > progress.Max {
				progress.Value = progress.Min
			}
			progress.Refresh()
			difficultyLabel.SetText(fmt.Sprintf("Current: %.4f", progress.Value))
		}
	}()

	return container.NewVBox(
		title,
		canvas.NewLine(color.Gray{Y: 100}),
		difficultyLabel,
		progress,
	)
}
