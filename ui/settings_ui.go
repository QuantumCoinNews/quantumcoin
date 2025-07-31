package ui

import (
	"quantumcoin/i18n"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var CurrentLang = "tr"
var CurrentTheme = "dark"

func ShowSettingsWindow(a fyne.App) {
	win := a.NewWindow(i18n.T(CurrentLang, "settings_title"))

	// Tema seçimi
	themeLabel := widget.NewLabel(i18n.T(CurrentLang, "settings_theme_label"))
	themeOptions := []string{
		i18n.T(CurrentLang, "settings_theme_dark"),
		i18n.T(CurrentLang, "settings_theme_light"),
	}
	themeSelect := widget.NewSelect(themeOptions, func(value string) {
		if value == i18n.T(CurrentLang, "settings_theme_dark") {
			a.Settings().SetTheme(theme.DarkTheme())
			CurrentTheme = "dark"
		} else {
			a.Settings().SetTheme(theme.LightTheme())
			CurrentTheme = "light"
		}
	})
	if CurrentTheme == "dark" {
		themeSelect.SetSelected(themeOptions[0])
	} else {
		themeSelect.SetSelected(themeOptions[1])
	}

	// Dil seçimi
	langLabel := widget.NewLabel(i18n.T(CurrentLang, "settings_lang_label"))
	langs := []struct {
		Code, Name, Emoji string
	}{
		{"en", "English", "🇬🇧"},
		{"tr", "Türkçe", "🇹🇷"},
		{"es", "Español", "🇪🇸"},
		{"zh", "中文", "🇨🇳"},
	}
	langBtns := container.NewHBox()
	for _, lang := range langs {
		langCode := lang.Code
		btn := widget.NewButton(lang.Emoji+" "+lang.Name, func() {
			// Dil değiştiğinde pencereyi kapatıp tekrar aç!
			CurrentLang = langCode
			win.Close()
			ShowSettingsWindow(a)
		})
		langBtns.Add(btn)
	}

	saveBtn := widget.NewButton(i18n.T(CurrentLang, "settings_save_button"), func() {
		win.Close()
	})

	form := container.NewVBox(
		widget.NewLabelWithStyle(i18n.T(CurrentLang, "settings_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		themeLabel, themeSelect,
		langLabel, langBtns,
		saveBtn,
	)

	win.SetContent(form)
	win.Resize(fyne.NewSize(400, 300))
	win.Show()
}
