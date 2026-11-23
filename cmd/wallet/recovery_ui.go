package main

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	qwallet "quantumcoin/wallet"
)

func recoveryText(en, tr string) string {
	if curLang == "tr" {
		return tr
	}
	return en
}

func recoveryError(w fyne.Window, err error) { dialog.ShowError(err, w) }

func showCreateRecoveryWallet(w fyne.Window, selected func(string)) {
	password, confirm := widget.NewPasswordEntry(), widget.NewPasswordEntry()
	dialog.NewForm(recoveryText("Create recoverable QC wallet", "Kurtarılabilir QC cüzdanı oluştur"), T("ok"), T("cancel"),
		[]*widget.FormItem{widget.NewFormItem(recoveryText("Local password (14+ characters)", "Yerel parola (en az 14 karakter)"), password), widget.NewFormItem(recoveryText("Repeat password", "Parolayı tekrarla"), confirm)}, func(ok bool) {
			if !ok {
				password.SetText("")
				confirm.SetText("")
				return
			}
			secret := password.Text
			password.SetText("")
			if secret != confirm.Text {
				confirm.SetText("")
				recoveryError(w, errors.New(recoveryText("Passwords do not match", "Parolalar eşleşmiyor")))
				return
			}
			confirm.SetText("")
			if len(secret) < 14 {
				recoveryError(w, errors.New(recoveryText("Use a password of at least 14 characters", "En az 14 karakterlik parola kullanın")))
				return
			}
			material, err := qwallet.NewRecoveryMaterial()
			if err != nil {
				recoveryError(w, err)
				return
			}
			words := widget.NewMultiLineEntry()
			words.SetText(material.Mnemonic)
			words.Disable()
			ack := widget.NewCheck(recoveryText("I recorded these words safely", "Bu kelimeleri güvenli biçimde yedekledim"), nil)
			note := widget.NewLabel(recoveryText("These words restore this same QC address on another device. Anyone with them controls the wallet. Existing wallets are kept.", "Bu kelimeler başka cihazda aynı QC adresini geri getirir. Kelimelere sahip olan cüzdanı kullanabilir. Eski cüzdanlar korunur."))
			note.Wrapping = fyne.TextWrapWord
			var view *dialog.CustomDialog
			save := widget.NewButton(recoveryText("Save encrypted wallet", "Şifreli cüzdanı kaydet"), func() {
				if !ack.Checked {
					recoveryError(w, errors.New(recoveryText("Record the recovery words first", "Önce kurtarma kelimelerini yedekleyin")))
					return
				}
				if err := selectRecoveryMaterial(material, secret); err != nil {
					recoveryError(w, err)
					return
				}
				words.SetText("")
				secret = ""
				selected(material.Address)
				view.Hide()
			})
			view = dialog.NewCustom(recoveryText("Your recovery words", "Kurtarma kelimelerin"), T("cancel"), container.NewVBox(note, widget.NewLabel(material.Address), words, ack, save), w)
			view.SetOnClosed(func() { words.SetText(""); secret = "" })
			view.Resize(fyne.NewSize(620, 420))
			view.Show()
		}, w).Show()
}

func showRestoreRecoveryWallet(w fyne.Window, expected string, selected func(string)) {
	words := widget.NewMultiLineEntry()
	words.SetPlaceHolder(recoveryText("12 or 24 recovery words", "12 veya 24 kurtarma kelimesi"))
	address := widget.NewEntry()
	address.SetText(expected)
	password, confirm := widget.NewPasswordEntry(), widget.NewPasswordEntry()
	dialog.NewForm(recoveryText("Restore QC wallet with words", "QC cüzdanını kelimelerle geri yükle"), T("restore"), T("cancel"), []*widget.FormItem{
		widget.NewFormItem(recoveryText("Recovery words", "Kurtarma kelimeleri"), words), widget.NewFormItem(recoveryText("Old QC address, if known", "Biliniyorsa eski QC adresi"), address),
		widget.NewFormItem(recoveryText("New local password", "Yeni yerel parola"), password), widget.NewFormItem(recoveryText("Repeat password", "Parolayı tekrarla"), confirm)}, func(ok bool) {
		if !ok {
			words.SetText("")
			password.SetText("")
			confirm.SetText("")
			return
		}
		phrase, secret := words.Text, password.Text
		words.SetText("")
		password.SetText("")
		if secret != confirm.Text {
			confirm.SetText("")
			recoveryError(w, errors.New(recoveryText("Passwords do not match", "Parolalar eşleşmiyor")))
			return
		}
		confirm.SetText("")
		material, err := qwallet.RestoreRecoveryMaterial(phrase, strings.TrimSpace(address.Text))
		phrase = ""
		if err != nil {
			dialog.NewConfirm(recoveryText("Wallet could not be recovered", "Cüzdan geri yüklenemedi"), recoveryText("Words or address did not match. Existing data is unchanged. Create a NEW empty wallet instead? It will not recover the old balance.", "Kelimeler veya adres eşleşmedi. Eski veriler değişmedi. Bunun yerine YENİ ve boş bir cüzdan oluşturulsun mu? Eski bakiye kurtarılmış olmayacak."), func(create bool) {
				if create {
					showCreateRecoveryWallet(w, selected)
				}
			}, w).Show()
			return
		}
		if err = selectRecoveryMaterial(material, secret); err != nil {
			recoveryError(w, err)
			return
		}
		secret = ""
		selected(material.Address)
		dialog.ShowInformation(T("ok"), recoveryText("QC wallet restored. Verify the displayed address; balances come from the connected node.", "QC cüzdanı geri yüklendi. Görünen adresi kontrol edin; bakiye bağlı node'dan okunur."), w)
	}, w).Show()
}

func showUnlockRecoveryWallet(w fyne.Window, done func()) {
	password := widget.NewPasswordEntry()
	dialog.NewForm(recoveryText("Unlock QC wallet", "QC cüzdanının kilidini aç"), T("ok"), T("cancel"), []*widget.FormItem{widget.NewFormItem(recoveryText("Local password", "Yerel parola"), password)}, func(ok bool) {
		if !ok {
			password.SetText("")
			return
		}
		secret := password.Text
		password.SetText("")
		if err := unlockRecoveryWallet(secret); err != nil {
			recoveryError(w, err)
			return
		}
		done()
	}, w).Show()
}

func showRecoveryWords(w fyne.Window) {
	password := widget.NewPasswordEntry()
	dialog.NewForm(recoveryText("Show QC recovery words", "QC kurtarma kelimelerini göster"), T("ok"), T("cancel"), []*widget.FormItem{widget.NewFormItem(recoveryText("Local password", "Yerel parola"), password)}, func(ok bool) {
		if !ok {
			password.SetText("")
			return
		}
		secret := password.Text
		password.SetText("")
		selection, err := selectedRecovery()
		if err != nil {
			recoveryError(w, err)
			return
		}
		if selection == nil || selection.Version == legacyRecoverySelectionVersion {
			recoveryError(w, errors.New(recoveryText("This old wallet has no verified recovery words; keep its encrypted key backup", "Bu eski cüzdanın doğrulanmış kurtarma kelimeleri yok; şifreli anahtar yedeğini koruyun")))
			return
		}
		home, err := recoveryHome()
		if err != nil {
			recoveryError(w, err)
			return
		}
		blob, err := readRecoveryFile(filepath.Join(home, selection.File), 16384)
		if err != nil {
			recoveryError(w, err)
			return
		}
		material, err := qwallet.OpenRecoveryMaterial(blob, secret)
		secret = ""
		if err != nil {
			recoveryError(w, err)
			return
		}
		if material.Address != selection.Address {
			recoveryError(w, errors.New("wallet address mismatch"))
			return
		}
		words := widget.NewMultiLineEntry()
		words.SetText(material.Mnemonic)
		words.Disable()
		view := dialog.NewCustom(recoveryText("Keep these words private", "Bu kelimeleri gizli tutun"), T("ok"), container.NewVBox(widget.NewLabel(material.Address), words), w)
		timer := time.AfterFunc(time.Minute, func() { fyne.Do(func() { words.SetText(""); view.Hide() }) })
		view.SetOnClosed(func() { timer.Stop(); words.SetText("") })
		view.Resize(fyne.NewSize(620, 320))
		view.Show()
	}, w).Show()
}
