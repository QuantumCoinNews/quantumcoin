// gui-smoke is an optional interactive Fyne check, separate from the node.
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	w := myApp.NewWindow("QuantumCoin GUI check")
	w.SetContent(container.NewVBox(widget.NewLabel("Fyne is working.")))
	w.Resize(fyne.NewSize(300, 200))
	w.ShowAndRun()
}
