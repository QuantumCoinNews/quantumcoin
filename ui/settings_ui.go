package ui

import (
	"quantumcoin/i18n"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var currentLang = "en"
var currentTheme = "dark"

func ShowSettingsWindow(a fyne.App) {
	win := a.NewWindow("⚙️ " + i18n.T(currentLang, "settings_title"))

	themeLabel := widget.NewLabel(i18n.T(currentLang, "settings_theme_label"))
	themeOptions := []string{i18n.T(currentLang, "settings_theme_dark"), i18n.T(currentLang, "settings_theme_light")}
	themeSelect := widget.NewSelect(themeOptions, func(value string) {
		if value == i18n.T(currentLang, "settings_theme_dark") {
			a.Settings().SetTheme(theme.DarkTheme())
			currentTheme = "dark"
		} else {
			a.Settings().SetTheme(theme.LightTheme())
			currentTheme = "light"
		}
	})
	if currentTheme == "dark" {
		themeSelect.SetSelected(themeOptions[0])
	} else {
		themeSelect.SetSelected(themeOptions[1])
	}

	langLabel := widget.NewLabel(i18n.T(currentLang, "settings_lang_label"))
	langOptions := []string{"en", "tr", "es", "zh"}
	langSelect := widget.NewSelect(langOptions, func(value string) {
		currentLang = value
		// Burada, aktif pencereleri yeniden çizmek gerekebilir
	})
	langSelect.SetSelected(currentLang)

	saveBtn := widget.NewButton(i18n.T(currentLang, "settings_save_button"), func() {
		win.Close()
	})

	form := container.NewVBox(
		widget.NewLabelWithStyle(i18n.T(currentLang, "settings_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		themeLabel, themeSelect,
		langLabel, langSelect,
		saveBtn,
	)

	win.SetContent(form)
	win.Resize(fyne.NewSize(400, 300))
	win.Show()
}
