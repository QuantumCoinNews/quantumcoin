package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func TestUI() {
	myApp := app.New()
	w := myApp.NewWindow("Test Penceresi")
	w.SetContent(container.NewVBox(
		widget.NewLabel("Fyne çalışıyor!"),
	))
	w.Resize(fyne.NewSize(300, 200))
	w.ShowAndRun()
}
