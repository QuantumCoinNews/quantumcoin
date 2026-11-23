package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"quantumcoin/apiclient"
	"quantumcoin/consensus"
	"quantumcoin/utils"
	qwallet "quantumcoin/wallet"
	"quantumcoin/walletsigner"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/scrypt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func hideConsoleOnStartup() {
	if runtime.GOOS != "windows" {
		return
	}
	mod := syscall.NewLazyDLL("kernel32.dll")
	proc := mod.NewProc("FreeConsole")
	_, _, _ = proc.Call()
}

func startMinerVisible() error { return startNativeMining() }

func stopMinerVisible() error { return stopNativeMining() }

/* ================== Versiyon & Renkler ================== */

var appVersion = "wallet-gui v3.0.0-seed-runtime"

func qcMinerRunnerAlive() bool { return nativeMinerAlive() }

func qcResetStaleMinerState() { minerRunningState = nativeMinerAlive() }

var (
	colSuccess = color.NRGBA{R: 46, G: 204, B: 113, A: 255} // yeşil
	colInfo    = color.NRGBA{R: 52, G: 152, B: 219, A: 255} // mavi/cyan
	colError   = color.NRGBA{R: 231, G: 76, B: 60, A: 255}  // kırmızı
)

/* ================== Global Durum ================== */

var (
	apiPort               string
	walletAutoRefreshStop chan struct{}
	lastBalance           float64 // son gösterilen toplam (confirmed+pending)

	minerRunningState  = false
	walletRefreshHook  func()
	onMinerStateUpdate func(bool)
)

// --- AI sekmesi text alanları ---
var (
	aiTelemetryText *widget.Entry
	aiAlertsText    *widget.Entry
	aiAnalysisText  *widget.Entry
	aiBonusText     *widget.Entry
)

// ---- Miner komut sabitleri ----
// Miner, adresi POZİSYONEL argüman olarak ister:  quantumcoin.exe mine "ADRES"
// Bu yüzden bayrak kullanılmaz; -addr KULLANMA.
const minerSubcommand = "mine"

// (lint/unused susturucu için)
var _ = minerSubcommand

/* ============ Basit Helpers ============ */

func ui(f func()) { fyne.Do(f) }

func exeDir() string {
	self, _ := os.Executable()
	return filepath.Dir(self)
}

func isWalletTabActive(w fyne.Window) bool {
	if w == nil {
		return true
	}

	if root, ok := w.Content().(*fyne.Container); ok {
		for _, obj := range root.Objects {
			if tabs, ok := obj.(*container.AppTabs); ok {
				ti := tabs.Selected()
				if ti == nil {
					return true
				}
				t := strings.ToLower(ti.Text)
				return strings.Contains(t, "cüzdan") || strings.Contains(t, "wallet")
			}
		}
	}
	return true
}

// Lint: bazı buildlerde kullanılmayabilir; referansla sustur (top-level)
var _ = isWalletTabActive
var _ = startMinerInCmd

/* ============ Tema ============ */

// Seçili tema değeri (etiketlerden bağımsız)
var themeValue = "auto" // "auto" | "light" | "dark"

// Tema Select'i (dil değişince etiketleri yenilemek için)
var themeSel *widget.Select

// Dil Select'i (global — makeLangSelect bunu kullanıyor)
var langSelect *widget.Select

// header'da logo göstermek için tek seferlik loader
var headerIconOnce sync.Once
var headerIcon fyne.CanvasObject

// Programatik güncellemelerde OnChanged’i tetiklememek için guard
var themeSelUpdating int

// SetSelected'i callback tetiklemeden yapmak için yardımcı
func setThemeSelectSilently(label string) {
	if themeSel == nil {
		return
	}
	themeSelUpdating++
	themeSel.SetSelected(label)
	themeSelUpdating--
}

// i18n sözlüğünde tema anahtarlarını garanti altına al
func ensureThemeI18nKeys() {
	need := func(lang, k, v string) {
		if i18n[lang] == nil {
			i18n[lang] = dict{}
		}
		if _, ok := i18n[lang][k]; !ok || strings.TrimSpace(i18n[lang][k]) == "" {
			i18n[lang][k] = v
		}
	}

	// TR
	need("tr", "theme_auto", "Otomatik")
	need("tr", "theme_light", "Aydınlık")
	need("tr", "theme_dark", "Karanlık")

	// EN
	need("en", "theme_auto", "Auto")
	need("en", "theme_light", "Light")
	need("en", "theme_dark", "Dark")

	// ES
	need("es", "theme_auto", "Automático")
	need("es", "theme_light", "Claro")
	need("es", "theme_dark", "Oscuro")

	// ZH
	need("zh", "theme_auto", "自动")
	need("zh", "theme_light", "浅色")
	need("zh", "theme_dark", "深色")
}

// --- Kalıcı ayar kayıt helper'ı ---
func persistUISettings() {
	if a := fyne.CurrentApp(); a != nil {
		a.Preferences().SetString("lang", curLang)
		a.Preferences().SetString("theme_value", themeValue)
	}
}

// Girilen isimden temayı uygula (TR/EN/ES/ZH etiketlerini anlar)
func applyTheme(name string) {
	s := strings.ToLower(strings.TrimSpace(name))

	switch {
	case s == "dark" || s == "karanlık" || s == "oscuro" || s == "深色" || s == "深色模式":
		themeValue = "dark"
		//lint:ignore SA1019 explicit user-selected dark theme is intentional until a custom Fyne theme is introduced.
		fyne.CurrentApp().Settings().SetTheme(theme.DarkTheme())
	case s == "light" || s == "aydınlık" || s == "claro" || s == "浅色" || s == "亮色" || s == "浅色模式":
		themeValue = "light"
		//lint:ignore SA1019 explicit user-selected light theme is intentional until a custom Fyne theme is introduced.
		fyne.CurrentApp().Settings().SetTheme(theme.LightTheme())
	default: // auto
		themeValue = "auto"
		fyne.CurrentApp().Settings().SetTheme(theme.DefaultTheme())
	}

	// Seçimi kalıcı kaydet + menü etiketini güncelle
	persistUISettings()
	refreshThemeSelectLabels()
}

// Dil değişiminde Select etiketlerini güncelle
func refreshThemeSelectLabels() {
	if themeSel == nil {
		return
	}
	themeSel.Options = []string{T("theme_auto"), T("theme_light"), T("theme_dark")}
	switch themeValue {
	case "light":
		setThemeSelectSilently(T("theme_light"))
	case "dark":
		setThemeSelectSilently(T("theme_dark"))
	default:
		setThemeSelectSilently(T("theme_auto"))
	}
	themeSel.Refresh()
}

// Tema Select oluşturucu (yerelleştirilmiş) — **GUARD UYGULANDI**
func makeThemeSelect() *widget.Select {
	ensureThemeI18nKeys()
	themeSel = widget.NewSelect(
		[]string{T("theme_auto"), T("theme_light"), T("theme_dark")},
		func(label string) {
			// kritik: sessiz SetSelected çağrılarında geri çağrıyı yut
			if themeSelUpdating > 0 {
				return
			}
			l := strings.ToLower(strings.TrimSpace(label))
			switch l {
			case strings.ToLower(T("theme_dark")):
				applyTheme("dark")
			case strings.ToLower(T("theme_light")):
				applyTheme("light")
			default:
				applyTheme("auto")
			}
		},
	)
	refreshThemeSelectLabels()
	return themeSel
}

/* ============ Backup/Restore i18n anahtarları (garanti) ============ */
func ensureBackupRestoreI18nKeys() {
	need := func(lang, k, v string) {
		if i18n[lang] == nil {
			i18n[lang] = dict{}
		}
		if _, ok := i18n[lang][k]; !ok || strings.TrimSpace(i18n[lang][k]) == "" {
			i18n[lang][k] = v
		}
	}
	// --- Düğmeler / menü ---
	need("tr", "backup", "Cüzdanı Yedekle")
	need("tr", "restore", "Geri Yükle")
	need("en", "backup", "Backup Wallet")
	need("en", "restore", "Restore")
	need("es", "backup", "Respaldar monedero")
	need("es", "restore", "Restaurar")
	need("zh", "backup", "备份钱包")
	need("zh", "restore", "恢复")

	// --- Backup diyalogları ---
	need("tr", "backup_title", "Yedekleme")
	need("tr", "backup_select_folder", "Yedek klasörünü seçin")
	need("tr", "backup_success", "Yedekleme tamamlandı")
	need("tr", "backup_failed", "Yedekleme başarısız")
	need("en", "backup_title", "Backup")
	need("en", "backup_select_folder", "Select backup folder")
	need("en", "backup_success", "Backup completed")
	need("en", "backup_failed", "Backup failed")
	need("es", "backup_title", "Copia de seguridad")
	need("es", "backup_select_folder", "Selecciona carpeta de copia")
	need("es", "backup_success", "Copia completada")
	need("es", "backup_failed", "Copia fallida")
	need("zh", "backup_title", "备份")
	need("zh", "backup_select_folder", "选择备份文件夹")
	need("zh", "backup_success", "备份完成")
	need("zh", "backup_failed", "备份失败")

	// --- Restore diyalogları ---
	need("tr", "restore_title", "Geri Yükleme")
	need("tr", "restore_pick_file", "Yedek dosyasını seçin")
	need("tr", "restore_success", "Geri yükleme tamamlandı")
	need("tr", "restore_failed", "Geri yükleme başarısız")
	need("en", "restore_title", "Restore")
	need("en", "restore_pick_file", "Pick backup file")
	need("en", "restore_success", "Restore completed")
	need("en", "restore_failed", "Restore failed")
	need("es", "restore_title", "Restaurar")
	need("es", "restore_pick_file", "Selecciona archivo de copia")
	need("es", "restore_success", "Restauración completada")
	need("es", "restore_failed", "Fallo al restaurar")
	need("zh", "restore_title", "恢复")
	need("zh", "restore_pick_file", "选择备份文件")
	need("zh", "restore_success", "恢复完成")
	need("zh", "restore_failed", "恢复失败")
}

/* ============ AI sekmesi i18n anahtarları (garanti) ============ */
func ensureAII18nKeys() {
	need := func(lang, k, v string) {
		if i18n[lang] == nil {
			i18n[lang] = dict{}
		}
		if _, ok := i18n[lang][k]; !ok || strings.TrimSpace(i18n[lang][k]) == "" {
			i18n[lang][k] = v
		}
	}

	// TR
	need("tr", "tab_ai", "AI & Telemetry")
	need("tr", "ai_telemetry", "Ağ Telemetrisi")
	need("tr", "ai_alerts", "AI Uyarıları")
	need("tr", "ai_analysis", "AI Analizi")
	need("tr", "ai_bonus", "AI Bonus / Ödül Tahminleri")

	// EN
	need("en", "tab_ai", "AI & Telemetry")
	need("en", "ai_telemetry", "Telemetry")
	need("en", "ai_alerts", "AI Alerts")
	need("en", "ai_analysis", "AI Analysis")
	need("en", "ai_bonus", "AI Bonus / Rewards")

	// ES
	need("es", "tab_ai", "IA y Telemetría")
	need("es", "ai_telemetry", "Telemetría")
	need("es", "ai_alerts", "Alertas de IA")
	need("es", "ai_analysis", "Análisis de IA")
	need("es", "ai_bonus", "Bonos / Recompensas de IA")

	// ZH
	need("zh", "tab_ai", "AI 与遥测")
	need("zh", "ai_telemetry", "遥测数据")
	need("zh", "ai_alerts", "AI 警报")
	need("zh", "ai_analysis", "AI 分析")
	need("zh", "ai_bonus", "AI 奖励 / 奖金")
}

/* ============ Dil değiştirme yardımcıları (i18n) ============ */

// Uygulama dilini değiştir, tercihleri kaydet ve UI'yı yeniden kur
func changeLanguage(newLang string, w fyne.Window) {
	newLang = strings.TrimSpace(newLang)
	if newLang == "" || newLang == curLang {
		return
	}
	curLang = newLang
	persistUISettings()

	// Cüzdan auto-refresh loop'u temizle (yeniden kurulacak)
	if walletAutoRefreshStop != nil {
		close(walletAutoRefreshStop)
		walletAutoRefreshStop = nil
	}

	// Başlık ve tema menüsü etiketleri yeni dile göre güncellensin
	w.SetTitle(T("title") + " — " + appVersion)
	refreshThemeSelectLabels()

	// Tüm UI'ı yeni dilde yeniden inşa et
	buildUI(w)
}

/* ============ mined_balance.json path (esnek & sıralı) ============ */

// ================= Preferences & Startup Load =================

func loadPrefsAtStartup() {
	// i18n anahtarlarını tek seferde garanti et
	ensureThemeI18nKeys()
	ensureBackupRestoreI18nKeys()
	ensureAII18nKeys()

	// ---- Dil (QC_LANG > Prefs > "en") ----
	lang := strings.TrimSpace(os.Getenv("QC_LANG"))
	if lang == "" {
		if a := fyne.CurrentApp(); a != nil {
			lang = strings.TrimSpace(a.Preferences().String("lang"))
		}
	}
	switch strings.ToLower(lang) {
	case "tr", "en", "es", "zh":
		curLang = strings.ToLower(lang)
	default:
		curLang = "en"
	}

	// ---- Tema (QC_THEME / QC_THEME_VALUE > Prefs > "auto") ----
	t := strings.TrimSpace(os.Getenv("QC_THEME"))
	if t == "" {
		t = strings.TrimSpace(os.Getenv("QC_THEME_VALUE"))
	}
	if t == "" {
		if a := fyne.CurrentApp(); a != nil {
			t = a.Preferences().StringWithFallback("theme_value", "auto")
		} else {
			t = "auto"
		}
	}

	// Uygula (applyTheme TR/EN/ES/ZH eşanlamlılarını da anlar)
	applyTheme(t)

	// Seçimleri kalıcı yaz (dil + tema)
	persistUISettings()
}

func findNodeExe() string {
	// 1) EN ÖNCE: wallet-gui.exe ile aynı klasördeki quantumcoin.exe (release)
	if exePath, err := os.Executable(); err == nil && exePath != "" {
		local := filepath.Join(filepath.Dir(exePath), "quantumcoin.exe")
		if st, err := os.Stat(local); err == nil && !st.IsDir() {
			return local
		}
	}

	// 2) QC_NODE_DIR verilmişse oraya bak
	if nd := strings.TrimSpace(os.Getenv("QC_NODE_DIR")); nd != "" {
		p := filepath.Join(nd, "quantumcoin.exe")
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}

	// 3) Çalışma klasörü
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		p := filepath.Join(cwd, "quantumcoin.exe")
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}

	// 4) PATH (en son fallback)
	if p, err := exec.LookPath("quantumcoin.exe"); err == nil && p != "" {
		return p
	}

	// 5) Eski olası yollar (fallback)
	candidates := []string{
		filepath.Join(os.Getenv("APPDATA"), "QuantumCoin", "bin", "quantumcoin.exe"),
		filepath.Join(".", "quantumcoin.exe"),
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}

	return ""
}

/* ============ Miner status helpers ============ */

/* ===== Miner PID (sadece bizim başlattığımız süreçler) ===== */

/* ============ Node dir & adres dosyaları ============ */

// quantumcoin.exe’nin bulunabileceği klasörü belirle
func nodeDir() string {
	d := exeDir()

	// 1) GUI ile aynı klasörde node varsa
	if _, err := os.Stat(filepath.Join(d, "quantumcoin.exe")); err == nil {
		return d
	}
	// 2) Ortam değişkeni
	if v := strings.TrimSpace(os.Getenv("QC_NODE_DIR")); v != "" {
		if _, err := os.Stat(filepath.Join(v, "quantumcoin.exe")); err == nil {
			return v
		}
	}
	// 3) Windows sabit kurulum klasörü
	if runtime.GOOS == "windows" {
		if _, err := os.Stat(`C:\QuantumMiner\quantumcoin.exe`); err == nil {
			return `C:\QuantumMiner`
		}
	}
	// 4) Çalışma dizini
	if wd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(wd, "quantumcoin.exe")); err == nil {
			return wd
		}
	}
	// fallback: GUI exe klasörü
	return d
}

func walletAddressPath() string { return filepath.Join(nodeDir(), "wallet_address.txt") }
func minerAddressPath() string  { return filepath.Join(nodeDir(), "miner_address.txt") }
func walletPrivPath() string    { return filepath.Join(nodeDir(), "wallet_priv.hex") }

func saveText(path, s string) error { return os.WriteFile(path, []byte(strings.TrimSpace(s)), 0644) }

func loadText(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// Mevcut (ilk geçerli) adresi bulur: txt -> APPDATA\QuantumCoin\wallet.json -> regex
func detectExistingAddress() string {
	if selected, err := selectedRecoveryAddress(); err != nil || selected != "" {
		return selected
	}
	// 1) Yerel txt dosyaları (temizle + doğrula)
	if v := loadText(walletAddressPath()); v != "" {
		v = cleanBase58(v)
		if isLikelyBase58Address(v) {
			return v
		}
	}
	if v := loadText(minerAddressPath()); v != "" {
		v = cleanBase58(v)
		if isLikelyBase58Address(v) {
			return v
		}
	}

	// 2) Windows: %APPDATA%\QuantumCoin\wallet.json (tolerant okuma)
	if runtime.GOOS == "windows" {
		if app := strings.TrimSpace(os.Getenv("APPDATA")); app != "" {
			p := filepath.Join(app, "QuantumCoin", "wallet.json")
			if b, err := os.ReadFile(p); err == nil {
				// 2a) Bilinen alanlar
				var w struct {
					Address       string   `json:"address"`
					Addr          string   `json:"addr"`
					WalletAddress string   `json:"wallet_address"`
					Reward        string   `json:"reward"`
					RewardAddress string   `json:"reward_address"`
					Addresses     []string `json:"addresses"`
				}
				if json.Unmarshal(b, &w) == nil {
					cands := []string{w.Address, w.Addr, w.WalletAddress, w.Reward, w.RewardAddress}
					cands = append(cands, w.Addresses...)
					for _, c := range cands {
						a := cleanBase58(strings.TrimSpace(c))
						if a != "" && isLikelyBase58Address(a) {
							return a
						}
					}
				}
				// 2b) Genel map içinden olası anahtarlar
				var m map[string]interface{}
				if json.Unmarshal(b, &m) == nil {
					for _, k := range []string{"address", "addr", "wallet", "wallet_address", "reward", "reward_address"} {
						if s, ok := m[k].(string); ok {
							a := cleanBase58(strings.TrimSpace(s))
							if a != "" && isLikelyBase58Address(a) {
								return a
							}
						}
					}
				}
				// 2c) Regex fallback (Base58 26..50)
				re := regexp.MustCompile(`[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]{26,50}`)
				if m := re.FindString(string(b)); m != "" {
					m = cleanBase58(m)
					if isLikelyBase58Address(m) {
						return m
					}
				}
			}
		}
	}

	// bulunamadı
	return ""
}

// Sadece adres üretir (quantumcoin.exe newaddr) — kayıt yan etkisi yok
func genAddressViaNode() (string, error) {
	exe := findNodeExe()
	if _, err := os.Stat(exe); err != nil {
		return "", fmt.Errorf("quantumcoin executable not found: %w", err)
	}

	cmd := exec.Command(exe, "newaddr") // <-- DÜZGÜN KOMUT BU
	cmd.Dir = filepath.Dir(exe)
	if runtime.GOOS == "windows" {
		// küçük cmd penceresini sakla
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	if os.Getenv("QC_NODE_DIR") == "" {
		cmd.Env = append(os.Environ(), "QC_NODE_DIR="+filepath.Dir(exe))
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("newaddr error: %w (out=%s)", err, string(out))
	}
	s := strings.TrimSpace(string(out))

	// 1) Etiketli çıktılar
	for _, marker := range []string{
		"New Wallet Address:", "Wallet Address:", "New Address:", "Address:",
	} {
		if i := strings.Index(s, marker); i >= 0 {
			part := strings.TrimSpace(s[i+len(marker):])
			if j := strings.IndexByte(part, '\n'); j >= 0 {
				part = part[:j]
			}
			part = cleanBase58(part)
			if isLikelyBase58Address(part) {
				return part, nil
			}
		}
	}

	// 2) JSON olasılıkları
	var obj struct {
		Address       string `json:"address"`
		WalletAddress string `json:"wallet_address"`
		Addr          string `json:"addr"`
	}
	if json.Unmarshal([]byte(s), &obj) == nil {
		for _, c := range []string{obj.Address, obj.WalletAddress, obj.Addr} {
			a := cleanBase58(strings.TrimSpace(c))
			if a != "" && isLikelyBase58Address(a) {
				return a, nil
			}
		}
	}
	var any map[string]interface{}
	if json.Unmarshal([]byte(s), &any) == nil {
		for _, k := range []string{"address", "wallet_address", "addr"} {
			if v, ok := any[k].(string); ok {
				a := cleanBase58(strings.TrimSpace(v))
				if a != "" && isLikelyBase58Address(a) {
					return a, nil
				}
			}
		}
	}

	// 3) Regex fallback
	re := regexp.MustCompile(`[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]{26,50}`)
	if m := re.FindString(s); m != "" {
		m = cleanBase58(m)
		if isLikelyBase58Address(m) {
			return m, nil
		}
	}
	return "", fmt.Errorf("could not parse address from newaddr output: %q", s)
}

/* ================== API utils ================== */

func normalizeCanonicalAPIPort(p string) string {
	port, err := apiclient.NormalizeAPIPortV1(p)
	if err != nil {
		return ""
	}
	return port
}

func detectAPIPort() string {
	port, _ := apiclient.ConfiguredAPIPortV1()
	return port
}

func walletAPIPortV1() string {
	if apiPort != "" {
		return normalizeCanonicalAPIPort(apiPort)
	}
	return detectAPIPort()
}

func apiBase() string {
	port := walletAPIPortV1()
	if port == "" {
		return ""
	} // Invalid configuration must never form a remote URL.
	return "http://127.0.0.1:" + port
}

func canonicalAPIResponseError(status int, raw []byte) error {
	var envelope struct {
		Success *bool  `json:"success"`
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	parsed := json.Unmarshal(raw, &envelope) == nil

	message := ""
	if parsed {
		message = strings.TrimSpace(envelope.Message)
		if message == "" {
			message = strings.TrimSpace(envelope.Error)
		}
	}

	if status >= 200 && status < 300 {
		if parsed && envelope.Success != nil && !*envelope.Success {
			if message == "" {
				message = "request reported failure"
			}
			return fmt.Errorf("api request failed: %s", message)
		}
		return nil
	}

	if message != "" {
		return fmt.Errorf(
			"api request failed: http %d: %s",
			status,
			message,
		)
	}

	return fmt.Errorf("api request failed: http %d", status)
}

/* ================== i18n ================== */

type dict map[string]string

var curLang = "tr"

var i18n = map[string]dict{
	"tr": {
		"title":           "QuantumCoin Cüzdan & Madenci",
		"tab_wallet":      "Cüzdan",
		"tab_mine":        "Madenci",
		"tab_web":         "Web Cüzdan",
		"open_web_wallet": "Web Cüzdanı Aç",
		"web_wallet_url":  "Web Cüzdan URL: %s",

		"balance":         "Bakiye",
		"balance_na":      "Bakiye: (okunamadı)",
		"balance_api":     "Bakiye: (API kapalı)",
		"refresh":         "Yenile",
		"my_address":      "Cüzdan Adresim",
		"from":            "Gönderen (From)",
		"to":              "Alıcı (Adres)",
		"amount":          "Miktar (QC)",
		"fee":             "Ücret (ops.)",
		"send":            "Gönder",
		"copy":            "Kopyala",
		"paste":           "Yapıştır",
		"save":            "Kaydet",
		"start_api_bg":    "API’yi Arka Planda Başlat",
		"miner_start":     "Madenciliği Başlat",
		"miner_stop":      "Madenciliği Durdur",
		"reward_addr":     "Ödül Adresi",
		"addr_empty":      "Adres boş",
		"api_timeout":     "API’ye erişilemedi.",
		"tx_sent":         "İşlem gönderildi!",
		"ok":              "Tamam",
		"receive_copy":    "Al (Adresimi Kopyala)",
		"auto_refresh_on": "Otomatik Yenileme (3 sn)",
		"generate_wallet": "Yeni Cüzdan Oluştur",

		"miner_status":            "Durum",
		"running":                 "Çalışıyor",
		"stopped":                 "Durdu",
		"logs":                    "Kayıtlar",
		"theme_auto":              "Otomatik",
		"theme_light":             "Aydınlık",
		"theme_dark":              "Karanlık",
		"open_log_folder":         "Log Klasörünü Aç",
		"open_cmd_manual":         "CMD (manuel) aç",
		"miner_stopped":           "Madenci durduruldu.",
		"backup_wallet":           "Cüzdanı Yedekle",
		"restore_wallet":          "Cüzdanı Geri Yükle",
		"backup_password_title":   "Yedek Parolası",
		"backup_password_prompt":  "Yedek parolasını girin:",
		"restore_password_title":  "Yedek Parolası",
		"restore_password_prompt": "Parolayı girin:",
		"password_empty_error":    "Parola boş olamaz",
		"backup_created":          "Yedek oluşturuldu.",
		"restore_done_restart":    "Geri yüklendi (uygulamayı yeniden başlatın).",
		"cancel":                  "İptal",
		// --- AI sekmesi ---
		"tab_ai":       "AI & Telemetri",
		"ai_telemetry": "Düğüm Telemetrisi",
		"ai_alerts":    "AI Uyarıları",
		"ai_analysis":  "AI Analizi",
		"ai_bonus":     "AI Bonus / Ödül Tahminleri",
	},
	"en": {
		"title":           "QuantumCoin Wallet & Miner",
		"tab_wallet":      "Wallet",
		"tab_mine":        "Miner",
		"tab_web":         "Web Wallet",
		"open_web_wallet": "Open Web Wallet",
		"web_wallet_url":  "Web Wallet URL: %s",

		"balance":         "Balance",
		"balance_na":      "Balance: (unavailable)",
		"balance_api":     "Balance: (API down)",
		"refresh":         "Refresh",
		"my_address":      "My Wallet Address",
		"from":            "From (Sender)",
		"to":              "To (Address)",
		"amount":          "Amount (QC)",
		"fee":             "Fee (opt.)",
		"send":            "Send",
		"copy":            "Copy",
		"paste":           "Paste",
		"save":            "Save",
		"start_api_bg":    "Start API (Background)",
		"miner_start":     "Start Mining",
		"miner_stop":      "Stop Mining",
		"reward_addr":     "Reward Address",
		"addr_empty":      "Address is empty",
		"api_timeout":     "Could not reach API.",
		"tx_sent":         "Transaction sent!",
		"ok":              "OK",
		"receive_copy":    "Receive (Copy My Address)",
		"auto_refresh_on": "Auto Refresh (3s)",
		"generate_wallet": "Generate New Wallet",

		"miner_status":            "Status",
		"running":                 "Running",
		"stopped":                 "Stopped",
		"logs":                    "Logs",
		"theme_auto":              "Auto",
		"theme_light":             "Light",
		"theme_dark":              "Dark",
		"open_log_folder":         "Open Log Folder",
		"open_cmd_manual":         "Open CMD (manual)",
		"miner_stopped":           "Miner stopped.",
		"backup_wallet":           "Backup Wallet",
		"restore_wallet":          "Restore Wallet",
		"backup_password_title":   "Backup Password",
		"backup_password_prompt":  "Enter a password for the backup:",
		"restore_password_title":  "Backup Password",
		"restore_password_prompt": "Enter the password:",
		"password_empty_error":    "Password cannot be empty",
		"backup_created":          "Backup created.",
		"restore_done_restart":    "Restored (please restart the application).",
		"cancel":                  "Cancel",
		// --- AI tab ---
		"tab_ai":       "AI & Telemetry",
		"ai_telemetry": "Node Telemetry",
		"ai_alerts":    "AI Alerts",
		"ai_analysis":  "AI Analysis",
		"ai_bonus":     "AI Bonus / Rewards",
	},
	"es": {
		"title":           "QuantumCoin Billetera & Minero",
		"tab_wallet":      "Billetera",
		"tab_mine":        "Minero",
		"tab_web":         "Billetera Web",
		"open_web_wallet": "Abrir Billetera Web",
		"web_wallet_url":  "URL de Billetera Web: %s",

		"balance":         "Saldo",
		"balance_na":      "Saldo: (no disponible)",
		"balance_api":     "Saldo: (API caída)",
		"refresh":         "Actualizar",
		"my_address":      "Mi Dirección de Billetera",
		"from":            "Remitente (From)",
		"to":              "Destinatario (Dirección)",
		"amount":          "Monto (QC)",
		"fee":             "Tarifa (op.)",
		"send":            "Enviar",
		"copy":            "Copiar",
		"paste":           "Pegar",
		"save":            "Guardar",
		"start_api_bg":    "Iniciar API (Segundo plano)",
		"miner_start":     "Iniciar Minería",
		"miner_stop":      "Detener Minería",
		"reward_addr":     "Dirección de Recompensa",
		"addr_empty":      "La dirección está vacía",
		"api_timeout":     "No se pudo alcanzar la API.",
		"tx_sent":         "¡Transacción enviada!",
		"ok":              "OK",
		"receive_copy":    "Recibir (Copiar mi dirección)",
		"auto_refresh_on": "Actualización Automática (3s)",
		"generate_wallet": "Generar Nueva Billetera",

		"miner_status":            "Estado",
		"running":                 "En ejecución",
		"stopped":                 "Detenido",
		"logs":                    "Registros",
		"theme_auto":              "Automático",
		"theme_light":             "Claro",
		"theme_dark":              "Oscuro",
		"open_log_folder":         "Abrir carpeta de logs",
		"open_cmd_manual":         "Abrir CMD (manual)",
		"miner_stopped":           "Minero detenido.",
		"backup_wallet":           "Respaldar Billetera",
		"restore_wallet":          "Restaurar Billetera",
		"backup_password_title":   "Contraseña de Respaldo",
		"backup_password_prompt":  "Ingresa una contraseña para el respaldo:",
		"restore_password_title":  "Contraseña de Respaldo",
		"restore_password_prompt": "Ingresa la contraseña:",
		"password_empty_error":    "La contraseña no puede estar vacía",
		"backup_created":          "Respaldo creado.",
		"restore_done_restart":    "Restaurado (reinicia la aplicación).",
		"cancel":                  "Cancelar",
		"tab_ai":                  "IA y Telemetría",
		"ai_telemetry":            "Telemetría del nodo",
		"ai_alerts":               "Alertas de IA",
		"ai_analysis":             "Análisis de IA",
		"ai_bonus":                "Bonos / Recompensas de IA",
	},
	"zh": {
		"title":           "QuantumCoin 钱包与矿工",
		"tab_wallet":      "钱包",
		"tab_mine":        "矿工",
		"tab_web":         "网页版钱包",
		"open_web_wallet": "打开网页版钱包",
		"web_wallet_url":  "网页版钱包地址：%s",

		"balance":         "余额",
		"balance_na":      "余额：（不可用）",
		"balance_api":     "余额：（API 离线）",
		"refresh":         "刷新",
		"my_address":      "我的钱包地址",
		"from":            "发送方（From）",
		"to":              "接收方（地址）",
		"amount":          "数量（QC）",
		"fee":             "手续费（可选）",
		"send":            "发送",
		"copy":            "复制",
		"paste":           "粘贴",
		"save":            "保存",
		"start_api_bg":    "后台启动 API",
		"miner_start":     "开始挖矿",
		"miner_stop":      "停止挖矿",
		"reward_addr":     "奖励地址",
		"addr_empty":      "地址为空",
		"api_timeout":     "无法连接到 API。",
		"tx_sent":         "交易已发送！",
		"ok":              "确定",
		"receive_copy":    "接收（复制我的地址）",
		"auto_refresh_on": "自动刷新（3秒）",
		"generate_wallet": "生成新钱包",

		"miner_status":            "状态",
		"running":                 "运行中",
		"stopped":                 "已停止",
		"logs":                    "日志",
		"theme_auto":              "自动",
		"theme_light":             "浅色",
		"theme_dark":              "深色",
		"open_log_folder":         "打开日志文件夹",
		"open_cmd_manual":         "打开 CMD（手动）",
		"miner_stopped":           "矿工已停止。",
		"backup_wallet":           "备份钱包",
		"restore_wallet":          "恢复钱包",
		"backup_password_title":   "备份密码",
		"backup_password_prompt":  "请输入备份密码：",
		"restore_password_title":  "备份密码",
		"restore_password_prompt": "请输入密码：",
		"password_empty_error":    "密码不能为空",
		"backup_created":          "备份已创建。",
		"restore_done_restart":    "已恢复（请重启应用）。",
		"cancel":                  "取消",
		"tab_ai":                  "AI 与遥测",
		"ai_telemetry":            "节点遥测",
		"ai_alerts":               "AI 警报",
		"ai_analysis":             "AI 分析",
		"ai_bonus":                "AI 奖励 / 预测",
	},
}

func T(k string) string {
	if d, ok := i18n[curLang]; ok {
		if v, ok2 := d[k]; ok2 {
			return v
		}
	}
	return k
}

/* ===== Balance cache ===== */

func balanceCachePath() string { return filepath.Join(nodeDir(), "wallet_balance.cache.json") }

func saveBalanceCache(addr string, bal float64) {
	m := map[string]interface{}{
		"address":    addr,
		"balance":    bal,
		"updated_at": time.Now().Unix(),
	}

	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"wallet: balance cache encode failed: %v\n",
			err,
		)
		return
	}

	if err := os.WriteFile(balanceCachePath(), b, 0644); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"wallet: balance cache write failed: %v\n",
			err,
		)
	}
}

func loadBalanceCache(addr string) (float64, bool) {
	b, err := os.ReadFile(balanceCachePath())
	if err != nil {
		return 0, false
	}
	var m map[string]interface{}
	if json.Unmarshal(b, &m) != nil {
		return 0, false
	}
	a, _ := m["address"].(string)
	if strings.TrimSpace(a) == "" || !strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(addr)) {
		return 0, false
	}
	switch v := m["balance"].(type) {
	case float64:
		return v, true
	case json.Number:
		f, _ := v.Float64()
		return f, true
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

/* ===== API helpers for numbers ===== */

func getBalanceUniversal(addr string) (float64, float64, bool) {
	addr = strings.TrimSpace(addr)
	if !qwallet.ValidateAddress(addr) {
		return 0, 0, false
	}
	client := apiclient.NewNodeClient(4 * time.Second)
	defer client.CloseIdleConnections()
	resp, err := authenticatedGetV1(client, apiBase()+"/api/wallet/balance/"+url.PathEscape(addr))
	if err != nil {
		return 0, 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, false
	}
	var balance struct {
		Total     string `json:"balanceQc"`
		Spendable string `json:"spendableQc"`
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil || len(raw) > 65536 || json.Unmarshal(raw, &balance) != nil {
		return 0, 0, false
	}
	total, totalErr := consensus.ParseQCToAtomsV1(balance.Total)
	spendable, spendableErr := consensus.ParseQCToAtomsV1(balance.Spendable)
	if totalErr != nil || spendableErr != nil || spendable < 0 || total < spendable {
		return 0, 0, false
	}
	unit := float64(consensus.CanonicalAtomsPerQCV1)
	return float64(spendable) / unit, float64(total-spendable) / unit, true
}

/* ================== UI: Wallet ================== */

type canonicalSendRequest struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Amount  string `json:"amount"`
	PrivHex string `json:"-"`
}

func validateCanonicalSendResponse(statusCode int, raw []byte, expectedID string) (string, error) {
	return walletsigner.ValidateSendResponseV1(statusCode, raw, expectedID)
}

func apiSendTransactionV1(req canonicalSendRequest, timeout time.Duration) (string, error) {
	return sendWalletRequestV1(req, filepath.Join(nodeDir(), "quantumcoin.exe"), apiBase()+"/api/tx/send", os.Getenv("QC_API_TOKEN"), timeout)
}

// sendWalletRequestV1 is the same signing/submission path used by the Send button.
// A read-only or unidentified node is rejected before opening the signing pipe.
func sendWalletRequestV1(req canonicalSendRequest, executable, endpoint, token string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		return "", fmt.Errorf("send timeout must be positive")
	}
	origin, err := apiclient.ValidateNodeURL(endpoint)
	if err != nil {
		return "", err
	}
	if origin.Path != "/api/tx/send" || origin.RawQuery != "" {
		return "", fmt.Errorf("canonical send endpoint required")
	}
	origin.Path = apiclient.CanonicalHealthPathV1
	origin.RawPath = ""
	healthContext, cancelHealth := context.WithTimeout(context.Background(), timeout)
	health, err := apiclient.ProbeHealth(healthContext, origin.String(), token)
	cancelHealth()
	if err != nil {
		return "", err
	}
	if !health.Authenticated {
		return "", apiclient.ErrHealthUnauthorizedV1
	}
	if !health.ModeKnown || health.ReadOnly {
		return "", fmt.Errorf("node is read-only or its mode is unknown; use the canonical run node for signed transfers")
	}
	signContext, cancelSign := context.WithTimeout(context.Background(), 2*time.Minute)
	envelope, err := walletsigner.SignLocal(signContext, executable, req.From, req.To, req.Amount, req.PrivHex, false)
	cancelSign()
	if err != nil {
		return "", err
	}
	expectedID, err := walletsigner.SignedTransactionIDV1(envelope)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	raw, status, err := walletsigner.Submit(ctx, endpoint, token, envelope)
	if err != nil {
		return "", fmt.Errorf("submission outcome unknown for transaction %s; check its status before retrying", expectedID)
	}
	id, err := validateCanonicalSendResponse(status, raw, expectedID)
	if err != nil {
		return "", fmt.Errorf("%w (txid: %s)", err, expectedID)
	}
	return id, nil
}
func detectPrivHex() string {
	if selected, err := selectedRecovery(); err != nil || selected != nil {
		return unlockedRecoveryKey()
	}
	// 1) yerel dosya
	if v := loadText(walletPrivPath()); v != "" {
		return v
	}
	// 2) %APPDATA%\QuantumCoin\wallet.json
	if runtime.GOOS == "windows" {
		if app := strings.TrimSpace(os.Getenv("APPDATA")); app != "" {
			p := filepath.Join(app, "QuantumCoin", "wallet.json")
			if b, err := os.ReadFile(p); err == nil {
				var w struct {
					PrivHex string `json:"priv_hex"`
				}
				if json.Unmarshal(b, &w) == nil && strings.TrimSpace(w.PrivHex) != "" {
					return strings.TrimSpace(w.PrivHex)
				}
			}
		}
	}
	return ""
}

func makeWalletTab(w fyne.Window, myAddrEntry, fromEntry *widget.Entry) fyne.CanvasObject {
	toEntry := widget.NewEntry()
	toEntry.SetPlaceHolder(T("to"))
	amtEntry := widget.NewEntry()
	amtEntry.SetPlaceHolder(T("amount"))
	feeEntry := widget.NewEntry()
	feeEntry.SetPlaceHolder(T("fee"))

	balanceText := canvas.NewText(T("balance_na"), color.NRGBA{R: 140, G: 140, B: 140, A: 255})
	balanceText.TextSize = 22
	balanceText.TextStyle.Bold = true

	copyMyBtn := widget.NewButton(T("copy"), func() { fyne.CurrentApp().Clipboard().SetContent(myAddrEntry.Text) })
	receiveBtn := widget.NewButton(T("receive_copy"), func() {
		fyne.CurrentApp().Clipboard().SetContent(myAddrEntry.Text)
		dialog.ShowInformation(T("ok"), myAddrEntry.Text, w)
	})
	pasteToBtn := widget.NewButton(T("paste"), func() { toEntry.SetText(fyne.CurrentApp().Clipboard().Content()) })

	selectAddress := func(addr string) {
		myAddrEntry.SetText(addr)
		fromEntry.SetText(addr)
	}
	genBtn := widget.NewButton(T("generate_wallet"), func() {
		showCreateRecoveryWallet(w, selectAddress)
	})
	restoreWordsBtn := widget.NewButton(recoveryText("Restore words", "Kelimelerle kurtar"), func() {
		showRestoreRecoveryWallet(w, myAddrEntry.Text, selectAddress)
	})
	unlockBtn := widget.NewButton(recoveryText("Unlock", "Kilidi aç"), func() { showUnlockRecoveryWallet(w, func() {}) })
	lockBtn := widget.NewButton(recoveryText("Lock", "Kilitle"), lockRecoveryWallet)
	wordsBtn := widget.NewButton(recoveryText("Show recovery words", "Kurtarma kelimelerini göster"), func() { showRecoveryWords(w) })
	legacyBtn := widget.NewButton(recoveryText("Previous wallet / legacy backup", "Önceki cüzdan / eski yedek"), func() {
		dialog.NewConfirm(recoveryText("Switch wallet profile", "Cüzdan profilini değiştir"), recoveryText("Switch to the previous local wallet or import its old encrypted backup? All recovery vaults are kept.", "Önceki yerel cüzdana geçilsin mi? Sonrasında eski şifreli yedeğinizi de içe aktarabilirsiniz. Tüm kurtarma kasaları korunur."), func(ok bool) {
			if !ok {
				return
			}
			if err := selectLegacyRecoveryProfile(); err != nil {
				dialog.ShowError(err, w)
				return
			}
			selectAddress(detectExistingAddress())
		}, w).Show()
	})
	// ----- Bakiye yenileyici -----
	doRefresh := func() {
		addr := strings.TrimSpace(myAddrEntry.Text)
		if addr == "" {
			addr = strings.TrimSpace(fromEntry.Text)
		}
		if addr == "" {
			ui(func() {
				balanceText.Text = T("balance_na")
				balanceText.Color = color.NRGBA{R: 140, G: 140, B: 140, A: 255}
				canvas.Refresh(balanceText)
			})
			return
		}

		if spendable, locked, ok := getBalanceUniversal(addr); ok {
			total := spendable + locked
			saveBalanceCache(addr, total)

			ui(func() {
				// Renk mantığı
				if total > lastBalance {
					balanceText.Color = colSuccess
				} else if total < lastBalance {
					balanceText.Color = colError
				} else {
					balanceText.Color = colInfo
				}

				// Metin
				if locked > 1e-7 {
					balanceText.Text = fmt.Sprintf(
						"%s: %.8f QC   (spendable: %.8f • locked: %.8f)",
						T("balance"), total, spendable, locked,
					)
				} else {
					balanceText.Text = fmt.Sprintf("%s: %.8f QC", T("balance"), total)
				}

				canvas.Refresh(balanceText)
				lastBalance = total
			})
			return
		}

		// API başarısızsa cache
		if total, ok := loadBalanceCache(addr); ok {
			ui(func() {
				balanceText.Color = colInfo
				balanceText.Text = fmt.Sprintf("%s: %.8f QC (cached)", T("balance"), total)
				canvas.Refresh(balanceText)
				lastBalance = total
			})
			return
		}

		// API’den cevap bekleniyor
		ui(func() {
			balanceText.Text = T("balance_api")
			balanceText.Color = colError
			canvas.Refresh(balanceText)
		})
	}

	// dışarıdan çağrılabilsin
	walletRefreshHook = func() { go doRefresh() }

	refreshBtn := widget.NewButton(T("refresh"), func() {
		_ = startAPIBackground() // gerekirse kaldırır
		go doRefresh()
	})

	startAPIbtn := widget.NewButton(T("start_api_bg"), func() {
		if err := startAPIBackground(); err != nil {
			dialog.ShowError(fmt.Errorf("%s %v", T("api_timeout"), err), w)
			return
		}
		time.AfterFunc(800*time.Millisecond, func() { refreshBtn.Tapped(nil) })
	})

	// Auto refresh (3s)
	if walletAutoRefreshStop != nil {
		close(walletAutoRefreshStop)
	}
	walletAutoRefreshStop = make(chan struct{})
	go func(stop <-chan struct{}) {
		time.Sleep(600 * time.Millisecond)
		doRefresh()
		t := time.NewTicker(3 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				doRefresh()
			case <-stop:
				return
			}
		}
	}(walletAutoRefreshStop)

	form := widget.NewForm(
		widget.NewFormItem(T("my_address"), container.NewBorder(nil, nil, genBtn, copyMyBtn, myAddrEntry)),
		widget.NewFormItem(T("from"), fromEntry),
		widget.NewFormItem(T("to"), container.NewBorder(nil, nil, nil, pasteToBtn, toEntry)),
		widget.NewFormItem(T("amount"), amtEntry),
		widget.NewFormItem(T("fee"), feeEntry),
	)

	// ----- Cüzdan Yedek / Geri Yükle -----
	backupBtn := widget.NewButton(T("backup_wallet"), func() {
		// Select a directory, never open/truncate a wallet before password
		// verification. Each backup receives a new unique filename.
		fs := dialog.NewFolderOpen(func(folder fyne.ListableURI, err error) {
			if err != nil || folder == nil {
				return
			}
			path := filepath.Join(folder.Path(), fmt.Sprintf("qc-wallet-backup-%d.qcbak", time.Now().UnixNano()))

			pass := widget.NewEntry()
			pass.Password = true
			dialog.NewCustomConfirm(
				T("backup_password_title"),
				T("ok"),
				T("cancel"),
				container.NewVBox(widget.NewLabel(T("backup_password_prompt")), pass),
				func(ok bool) {
					secret := pass.Text
					pass.SetText("")
					if !ok {
						return
					}
					if strings.TrimSpace(secret) == "" {
						dialog.ShowError(fmt.Errorf("%s", T("password_empty_error")), w)
						return
					}
					if err := createWalletBackup(path, secret); err != nil {
						dialog.ShowError(err, w)
						return
					}
					dialog.ShowInformation(T("ok"), T("backup_created"), w)
				},
				w,
			).Show()
		}, w)
		fs.Show()
	})

	restoreBtn := widget.NewButton(T("restore_wallet"), func() {
		fo := dialog.NewFileOpen(func(ur fyne.URIReadCloser, err error) {
			if err != nil || ur == nil {
				return
			}
			path := ur.URI().Path()
			ur.Close()

			pass := widget.NewEntry()
			pass.Password = true
			dialog.NewCustomConfirm(
				T("restore_password_title"),
				T("ok"),
				T("cancel"),
				container.NewVBox(widget.NewLabel(T("restore_password_prompt")), pass),
				func(ok bool) {
					if !ok {
						return
					}
					if err := restoreWalletBackup(path, pass.Text); err != nil {
						dialog.ShowError(err, w)
						return
					}
					dialog.ShowInformation(T("ok"), T("restore_done_restart"), w)
				},
				w,
			).Show()
		}, w)
		fo.Show()
	})

	var sendBtn *widget.Button
	sendBtn = widget.NewButton(T("send"), func() {
		if err := startAPIBackground(); err != nil {
			dialog.ShowError(fmt.Errorf("%s %v", T("api_timeout"), err), w)
			return
		}
		privHex := strings.TrimSpace(detectPrivHex())
		if privHex == "" {
			if selected, err := selectedRecovery(); err != nil {
				dialog.ShowError(err, w)
				return
			} else if selected != nil {
				showUnlockRecoveryWallet(w, func() { sendBtn.OnTapped() })
				return
			}
			dialog.ShowError(fmt.Errorf("%s", "Özel anahtar bulunamadı. 'Yeni Cüzdan Oluştur' ile anahtar üretin."), w)
			return
		}
		from := strings.TrimSpace(fromEntry.Text)
		to := strings.TrimSpace(toEntry.Text)
		amount := strings.TrimSpace(amtEntry.Text)
		fee := strings.TrimSpace(feeEntry.Text)

		if from == "" || to == "" || amount == "" {
			dialog.ShowError(
				fmt.Errorf("from/to/amount gerekli"),
				w,
			)
			return
		}

		// canonical send migration:
		// The canonical /api/tx/send SendRequest does not expose a custom
		// fee field. Never silently pretend that an entered fee was applied.
		if fee != "" {
			dialog.ShowError(
				fmt.Errorf(
					"özel fee canonical /api/tx/send tarafından desteklenmiyor; fee alanını boş bırakın",
				),
				w,
			)
			return
		}

		req := canonicalSendRequest{
			From:    from,
			To:      to,
			Amount:  amount,
			PrivHex: privHex,
		}

		// canonical send migration:
		// There is one canonical transfer path. A retired transfer endpoint
		// fallback must never be used.
		txID, err := apiSendTransactionV1(
			req,
			8*time.Second,
		)
		if err != nil {
			dialog.ShowError(
				fmt.Errorf(
					"gönderilemedi: %v",
					err,
				),
				w,
			)
			return
		}

		dialog.ShowInformation(
			T("ok"),
			T("tx_sent")+"\nTxID: "+txID,
			w,
		)
		toEntry.SetText("")
		amtEntry.SetText("")
		go func() {
			deadline := time.Now().Add(10 * time.Second)
			for time.Now().Before(deadline) {
				time.Sleep(2 * time.Second)
				refreshBtn.Tapped(nil)
			}
		}()
	})
	// üstte sadece balance
	headerRow := container.NewHBox(balanceText)

	buttonsRow := container.NewHBox(
		refreshBtn,
		startAPIbtn,
		receiveBtn,
	)

	// asıl içerik artık iki satır + form + alt butonlar
	content := container.NewVBox(
		headerRow,
		buttonsRow,
		container.NewHBox(restoreWordsBtn, unlockBtn, lockBtn),
		wordsBtn,
		legacyBtn,
		form,
		container.NewHBox(sendBtn, backupBtn, restoreBtn),
	)

	// scroll'a sar ki pencere küçük kalabilsin
	sc := container.NewVScroll(content)
	sc.SetMinSize(fyne.NewSize(0, 0))

	return sc

}

/* ================== AES-GCM Yedek/Geri Yükle helpers ================== */

const (
	scryptN = 1 << 15
	scryptR = 8
	scryptP = 1
	keyLen  = 32
)

func deriveKey(pass string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(pass), salt, scryptN, scryptR, scryptP, keyLen)
}

func encryptJSONToFile(path string, data interface{}, pass string) error {
	if len(pass) < 14 || len(pass) > 1024 {
		return fmt.Errorf("use a backup password of 14 to 1024 bytes")
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	defer clear(b)
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	key, err := deriveKey(pass, salt)
	if err != nil {
		return err
	}
	defer clear(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	ct := gcm.Seal(nil, nonce, b, nil)

	blob := append(append(salt, nonce...), ct...)
	return saveEncryptedRecoveryBlobNew(path, blob)
}

func decryptLegacyRecoveryBlob(b []byte, pass string, out interface{}) error {
	if len(pass) > 1024 || len(b) > 1024*1024 {
		return fmt.Errorf("legacy backup input too large")
	}
	if len(b) < 16 {
		return fmt.Errorf("bozuk dosya")
	}
	salt := b[:16]
	key, err := deriveKey(pass, salt)
	if err != nil {
		return err
	}
	defer clear(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	ns := gcm.NonceSize()
	if len(b) < 16+ns {
		return fmt.Errorf("bozuk dosya (nonce)")
	}
	nonce := b[16 : 16+ns]
	ct := b[16+ns:]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return fmt.Errorf("legacy backup password or encrypted data is incorrect")
	}
	defer clear(pt)
	if json.Unmarshal(pt, out) != nil {
		return fmt.Errorf("invalid legacy wallet backup data")
	}
	return nil
}

func createWalletBackup(targetPath, pass string) error {
	if selection, err := selectedRecovery(); err != nil {
		return err
	} else if selection != nil {
		home, err := recoveryHome()
		if err != nil {
			return err
		}
		limit := int64(16384)
		if selection.Version == legacyRecoverySelectionVersion {
			limit = 1024 * 1024
		}
		blob, err := readRecoveryFile(filepath.Join(home, selection.File), limit)
		if selection.Version == legacyRecoverySelectionVersion {
			if err != nil {
				return err
			}
			legacy, err := openLegacyRecoveryWallet(blob, pass)
			if err != nil {
				return err
			}
			defer utils.ZeroSecpPrivateV1(legacy.PrivateKey)
			if legacy.GetAddress() != selection.Address {
				return fmt.Errorf("wallet address mismatch")
			}
			return saveEncryptedRecoveryBlobNew(targetPath, blob)
		}
		if err != nil {
			return err
		}
		material, err := qwallet.OpenRecoveryMaterial(blob, pass)
		if err != nil {
			return err
		}
		if material.Address != selection.Address {
			return fmt.Errorf("wallet address mismatch")
		}
		return qwallet.SaveRecoveryVaultNew(targetPath, blob)
	}
	dir := nodeDir()
	m := map[string]interface{}{}

	if b, err := os.ReadFile(filepath.Join(dir, "wallet_address.txt")); err == nil {
		m["wallet_address"] = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile(filepath.Join(dir, "miner_address.txt")); err == nil {
		m["miner_address"] = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile(filepath.Join(dir, "wallet_priv.hex")); err == nil {
		m["wallet_priv_hex"] = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile(filepath.Join(dir, "wallet_recovery_words.txt")); err == nil {
		m["wallet_recovery_words"] = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile(filepath.Join(dir, "wallet_recovery_manifest.json")); err == nil {
		var tmp interface{}
		_ = json.Unmarshal(b, &tmp)
		m["wallet_recovery_manifest"] = tmp
	}
	if b, err := os.ReadFile(filepath.Join(dir, "wallet_balance.cache.json")); err == nil {
		var tmp interface{}
		_ = json.Unmarshal(b, &tmp)
		m["balance_cache"] = tmp
	}
	legacy, err := legacyRecoveryWallet(m)
	if err != nil {
		return err
	}
	defer utils.ZeroSecpPrivateV1(legacy.PrivateKey)
	m["wallet_address"] = legacy.GetAddress()
	return encryptJSONToFile(targetPath, m, pass)
}
func restoreWalletBackup(sourcePath, pass string) error {
	blob, err := readRecoveryFile(sourcePath, 1024*1024)
	if err != nil {
		return err
	}
	var header struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(blob, &header) == nil && header.Version == qwallet.RecoveryVaultVersionV1 {
		material, err := qwallet.OpenRecoveryMaterial(blob, pass)
		if err != nil {
			return err
		}
		return selectRecoveryMaterial(material, pass)
	}
	if selection, err := selectedRecovery(); err != nil {
		return err
	} else if selection != nil {
		return fmt.Errorf("legacy backup retained: import it in a separate legacy wallet profile; the selected recovery wallet will not be overwritten")
	}
	return selectLegacyEncryptedBackup(blob, pass)
}

/* ================== Miner start/stop & log tail ================== */

// Base58 (BTC-style) — 0,O,I,l yasak; 26..50 uzunluk
var base58RE = regexp.MustCompile(`^[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]{26,50}$`)

func isLikelyBase58Address(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 26 || len(s) > 50 {
		return false
	}
	// yasak karakterler
	if strings.ContainsAny(s, "0OIl") {
		return false
	}
	return base58RE.MatchString(s)
}

// Yalnızca Base58 karakterlerini tutar
func cleanBase58(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= '1' && r <= '9') ||
			(r >= 'A' && r <= 'H') || (r >= 'J' && r <= 'N') || (r >= 'P' && r <= 'Z') ||
			(r >= 'a' && r <= 'k') || (r >= 'm' && r <= 'z') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

/* ================== API bootstrap ================== */

// API'yi arka planda doğrudan process olarak başlatır; logları api_out.log’a yazar.
func startAPIBackground() error {
	port := walletAPIPortV1()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return apiclient.EnsureLocalBackendV1(ctx, port, os.Getenv("QC_API_TOKEN"), func() (apiclient.BackendHandleV1, error) {
		dir := nodeDir()
		exe := filepath.Join(dir, "quantumcoin.exe")
		if _, err := os.Stat(exe); err != nil {
			return apiclient.BackendHandleV1{}, fmt.Errorf("quantumcoin.exe not found: %w", err)
		}
		f, err := os.OpenFile(filepath.Join(dir, "api_out.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return apiclient.BackendHandleV1{}, err
		}
		cmd := newWalletBackendCommandV1(exe, dir, port)
		cmd.Stdout = f
		cmd.Stderr = f
		if err := cmd.Start(); err != nil {
			_ = f.Close()
			return apiclient.BackendHandleV1{}, err
		}
		return apiclient.BackendHandleV1{
			Stop: func() {
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
				_ = cmd.Wait()
				_ = f.Close()
			},
			Release: func() error { go func() { _ = cmd.Wait(); _ = f.Close() }(); return nil },
		}, nil
	})
}

// Wallet transfers require the canonical writable node; run does not mine.
func newWalletBackendCommandV1(executable, dir, port string) *exec.Cmd {
	cmd := exec.Command(executable, "run")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "QC_NODE_DIR="+dir, "QC_HTTP_PORT=:"+port, "QC_API_LISTEN=127.0.0.1:"+port)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	return cmd
}

/* ================== Miner starter (görünür CMD) ================== */

// Miner'ı kullanıcı adresiyle başlatır.
// Windows: görünür CMD penceresi (UTF-8) + miner_out.log'a yazdırır (loop)
// macOS/Linux: arka planda, miner_out.log'a yazar
func startMinerInCmd(addr string) error {
	addr = strings.TrimSpace(addr)
	if !qwallet.ValidateAddress(addr) {
		return fmt.Errorf("valid QC reward address required")
	}
	_ = os.Setenv("ADDR", addr)
	return startMinerVisible()
}

// -------------------------------------------------------------
// Miner tab UI
func makeMinerTab(w fyne.Window, defaultAddr string) fyne.CanvasObject {
	// --- adres girişi ---
	addrEntry := widget.NewEntry()
	addrEntry.SetPlaceHolder(T("reward_addr"))
	if v := detectExistingAddress(); v != "" {
		addrEntry.SetText(v)
	} else if defaultAddr != "" {
		addrEntry.SetText(defaultAddr)
	}

	// --- durum etiketi ---
	statusLab := widget.NewLabel(T("stopped"))

	// --- log alanı ---
	logView := widget.NewRichText()
	logView.Wrapping = fyne.TextWrapWord
	logView.Segments = []widget.RichTextSegment{
		&widget.TextSegment{Text: T("logs") + ":\n"},
	}

	// log dosyası (release/nodeDir ile aynı olmalı)

	// Butonlar (onMinerStateUpdate içinde kullanacağız)
	var startBtn, stopBtn *widget.Button

	// UI state callback (TEK kez)
	onMinerStateUpdate = nativeMiningUIStateHandlerV1(ui, func(running bool) {
		if running {
			statusLab.SetText(T("running"))
			if startBtn != nil {
				startBtn.Disable()
			}
			if stopBtn != nil {
				stopBtn.Enable()
			}
		} else {
			statusLab.SetText(T("stopped"))
			if startBtn != nil {
				startBtn.Enable()
			}
			if stopBtn != nil {
				stopBtn.Disable()
			}
		}
		statusLab.Refresh()
	})

	// === START ===
	guiMining.Lock()
	var miningLog string
	guiMining.log = func(message string) {
		ui(func() {
			miningLog += "\n" + message
			if len(miningLog) > 32768 {
				miningLog = miningLog[len(miningLog)-32768:]
				if n := strings.IndexByte(miningLog, '\n'); n >= 0 {
					miningLog = miningLog[n+1:]
				}
			}
			logView.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: miningLog}}
			logView.Refresh()
		})
	}
	guiMining.Unlock()
	startBtn = widget.NewButton(T("miner_start"), func() {
		addr, err := ensureRewardAddress(addrEntry.Text)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		addrEntry.SetText(addr)

		// The selected public reward address is passed to the native PoW client.
		_ = os.Setenv("ADDR", addr)
		// Miner başlat: tek canonical flow
		if err := startMinerInCmd(addr); err != nil {
			dialog.ShowError(err, w)
			return
		}

		// gerçek miner gerçekten ayağa kalktı mı?
		if !qcMinerRunnerAlive() {
			minerRunningState = false
			if onMinerStateUpdate != nil {
				onMinerStateUpdate(false)
			}
			dialog.ShowError(fmt.Errorf("miner process crashed"), w)
			return
		}

		// UI running
		minerRunningState = true
		if onMinerStateUpdate != nil {
			onMinerStateUpdate(true)
		}

	})

	// === STOP ===
	stopBtn = widget.NewButton(T("miner_stop"), func() {
		if err := stopMinerVisible(); err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation(T("ok"), T("miner_stopped"), w)
	})

	// Reflect this GUI's native worker if the tab is rebuilt during mining.
	qcResetStaleMinerState()
	onMinerStateUpdate(minerRunningState)

	// log klasörü
	openLogBtn := widget.NewButton(T("open_log_folder"), func() {
		dir := nodeDir()
		if runtime.GOOS == "windows" {
			_ = exec.Command("explorer.exe", dir).Start()
		} else {
			_ = exec.Command("xdg-open", dir).Start()
		}
	})

	// The old batch launcher is an isolated local tool, not a network miner.
	openManualCmdBtn := widget.NewButton(T("open_cmd_manual"), func() {
		dialog.ShowInformation("Solo mining", "Use Start for device PoW on the selected network. Standalone operators can use cmd/miner with the same node origin, QC_API_TOKEN and reward address. Legacy local-release scripts do not join testnet/mainnet.", w)
	})

	top := widget.NewForm(
		widget.NewFormItem(T("reward_addr"), addrEntry),
		widget.NewFormItem(T("miner_status"), statusLab),
	)

	btnBar := container.NewHBox(startBtn, stopBtn, openLogBtn, openManualCmdBtn)

	logScroll := container.NewVScroll(logView)
	logScroll.SetMinSize(fyne.NewSize(0, 160))

	return container.NewVBox(top, btnBar, widget.NewSeparator(), logScroll)
}

func ensureRewardAddress(cur string) (string, error) {
	// 1) Kullanıcı girdisi
	a := cleanBase58(strings.TrimSpace(cur))
	if isLikelyBase58Address(a) {
		return a, nil
	}

	// 2) Mevcut dosyalardan yakala
	if v := detectExistingAddress(); isLikelyBase58Address(v) {
		_ = saveText(walletAddressPath(), v)
		_ = saveText(minerAddressPath(), v)
		return v, nil
	}

	return "", fmt.Errorf("create or restore your QC wallet in the Wallet tab before mining")
}

/* ================== Web Wallet & API bootstrap ================== */

func startAPIBackground2() error { return startAPIBackground() }

func openWebWalletBrowser() error {
	u := "http://127.0.0.1:" + detectAPIPort()
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/C", "start", "", u).Start()
	case "darwin":
		return exec.Command("open", u).Start()
	default:
		return exec.Command("xdg-open", u).Start()
	}
}

func makeWebWalletTab(w fyne.Window) fyne.CanvasObject {
	u := func() string { return "http://127.0.0.1:" + detectAPIPort() }
	urlLabel := widget.NewLabel(fmt.Sprintf(T("web_wallet_url"), u()))

	openBtn := widget.NewButton(T("open_web_wallet"), func() {
		if err := startAPIBackground2(); err != nil {
			dialog.ShowError(fmt.Errorf("%s %v", T("api_timeout"), err), w)
			return
		}
		urlLabel.SetText(fmt.Sprintf(T("web_wallet_url"), u()))
		if err := openWebWalletBrowser(); err != nil {
			dialog.ShowError(fmt.Errorf("%s: %v", "Web wallet could not open", err), w)
			return
		}
		dialog.ShowInformation(T("ok"), "Web wallet opened in browser.", w)
	})

	return container.NewVBox(urlLabel, openBtn)
}

/* ================== UI Kur & main ================== */

func makeLangSelect(w fyne.Window) *widget.Select {
	if langSelect != nil {
		return langSelect
	}

	options := []string{
		"🇹🇷 Türkçe (tr)",
		"🇺🇸 English (en)",
		"🇪🇸 Español (es)",
		"🇨🇳 中文 (zh)",
	}

	langSelect = widget.NewSelect(options, func(s string) {
		var code string
		switch {
		case strings.Contains(s, "(tr)"):
			code = "tr"
		case strings.Contains(s, "(en)"):
			code = "en"
		case strings.Contains(s, "(es)"):
			code = "es"
		case strings.Contains(s, "(zh)"):
			code = "zh"
		default:
			code = "en"
		}

		// burada artık o eski fonksiyonu kullanıyoruz
		changeLanguage(code, w)
	})

	// Açılışta seçili olanı mevcut dile göre ayarla
	switch curLang {
	case "tr":
		langSelect.SetSelected("🇹🇷 Türkçe (tr)")
	case "es":
		langSelect.SetSelected("🇪🇸 Español (es)")
	case "zh":
		langSelect.SetSelected("🇨🇳 中文 (zh)")
	default:
		langSelect.SetSelected("🇺🇸 English (en)")
	}

	return langSelect
}
func loadHeaderIcon() fyne.CanvasObject {
	headerIconOnce.Do(func() {
		exe, _ := os.Executable()
		exeDir := filepath.Dir(exe)

		// 1) çalışma klasöründeki assets
		p1 := filepath.Join(exeDir, "assets", "icon.png")
		if _, err := os.Stat(p1); err == nil {
			img := canvas.NewImageFromFile(p1)
			img.SetMinSize(fyne.NewSize(28, 28))
			img.FillMode = canvas.ImageFillContain
			headerIcon = img
			return
		}

		// 2) VSCode’dan çalışırken: projedeki cmd/wallet/assets/icon.png
		p2 := filepath.Join(exeDir, "..", "cmd", "wallet", "assets", "icon.png")
		if _, err := os.Stat(p2); err == nil {
			img := canvas.NewImageFromFile(p2)
			img.SetMinSize(fyne.NewSize(28, 28))
			img.FillMode = canvas.ImageFillContain
			headerIcon = img
			return
		}

		// 3) fallback: mavi kutu
		rect := canvas.NewRectangle(color.NRGBA{R: 20, G: 90, B: 180, A: 255})
		rect.SetMinSize(fyne.NewSize(28, 28))
		headerIcon = rect
	})

	return headerIcon
}

// === AI & Telemetry sekmesi ===
func makeAITab() *container.TabItem {
	// Çok satırlı, sadece okunur alanlar
	aiTelemetryText = widget.NewMultiLineEntry()
	aiTelemetryText.SetPlaceHolder(T("ai_telemetry"))
	// SetReadOnly bu Fyne sürümünde yok; bunun yerine disable edelim
	aiTelemetryText.Disable()

	aiAlertsText = widget.NewMultiLineEntry()
	aiAlertsText.SetPlaceHolder(T("ai_alerts"))
	aiAlertsText.Disable()

	aiAnalysisText = widget.NewMultiLineEntry()
	aiAnalysisText.SetPlaceHolder(T("ai_analysis"))
	aiAnalysisText.Disable()

	aiBonusText = widget.NewMultiLineEntry()
	aiBonusText.SetPlaceHolder(T("ai_bonus"))
	aiBonusText.Disable()

	content := container.NewVBox(
		widget.NewLabel(T("ai_telemetry")),
		aiTelemetryText,
		widget.NewSeparator(),
		widget.NewLabel(T("ai_alerts")),
		aiAlertsText,
		widget.NewSeparator(),
		widget.NewLabel(T("ai_analysis")),
		aiAnalysisText,
		widget.NewSeparator(),
		widget.NewLabel(T("ai_bonus")),
		aiBonusText,
	)

	return container.NewTabItem(T("tab_ai"), container.NewVScroll(content))
}

// AI log yardımcısı: max N satır, boş mesajı yutar
func appendAILog(e *widget.Entry, prefix, raw string, maxLines int) {
	if e == nil {
		return
	}
	txt := strings.TrimSpace(raw)
	if txt == "" {
		// Boş mesaj (ör: offline sinyali) gelirse hiçbir şey yapma
		return
	}
	if maxLines <= 0 {
		maxLines = 200
	}

	ui(func() {
		now := time.Now().Format("15:04:05")
		line := fmt.Sprintf("%s %s %s", now, prefix, txt)

		cur := strings.TrimRight(e.Text, "\r\n")
		if cur == "" {
			e.SetText(line)
			return
		}

		// mevcut satırları ayır
		lines := strings.Split(cur, "\n")
		lines = append(lines, line)

		// en fazla maxLines satır kalsın
		if len(lines) > maxLines {
			lines = lines[len(lines)-maxLines:]
		}

		e.SetText(strings.Join(lines, "\n"))
		e.CursorRow = len(lines) - 1
		e.CursorColumn = len(line)
	})
}

func buildUI(w fyne.Window) {
	qcResetStaleMinerState()
	// i18n anahtarlarını garanti altına al
	ensureThemeI18nKeys()
	ensureBackupRestoreI18nKeys()
	ensureAII18nKeys()

	// --- giriş kutuları ---
	myAddrEntry := widget.NewEntry()
	fromEntry := widget.NewEntry()
	myAddrEntry.SetPlaceHolder(T("my_address"))
	fromEntry.SetPlaceHolder(T("from"))

	// ---- Tek seferlik adres çözümü (senkron) ----
	addr := strings.TrimSpace(detectExistingAddress())
	if addr == "" {
		if a, err := genAddressViaNode(); err == nil {
			a = cleanBase58(a)
			if a != "" {
				_ = saveText(walletAddressPath(), a)
				_ = saveText(minerAddressPath(), a)
				addr = a
			}
		}
	}
	if addr != "" {
		myAddrEntry.SetText(addr)
		fromEntry.SetText(addr)

		// Ortam değişkenleri
		_ = os.Setenv("QC_NODE_DIR", nodeDir())
		_ = os.Setenv("QC_MINED_PATH", filepath.Join(nodeDir(), "mined_balance.json"))
		_ = os.Setenv("ADDR", addr) // ← EKLEDİĞİMİZ TEK SATIR

	}

	// --- SEKMEler ---
	tabs := container.NewAppTabs(
		container.NewTabItem(T("tab_wallet"), makeWalletTab(w, myAddrEntry, fromEntry)),
		container.NewTabItem(T("tab_mine"), makeMinerTab(w, addr)),
		container.NewTabItem(T("tab_web"), makeWebWalletTab(w)),
		makeAITab(), // AI & Telemetry sekmesi
	)

	// --- HEADER ---
	themeSel = makeThemeSelect()

	title := widget.NewLabelWithStyle(
		T("title")+" — "+appVersion,
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	// logo’yu yükle
	logoObj := loadHeaderIcon()

	// sol taraf: logo + boşluk + başlık
	left := container.NewHBox(
		logoObj,
		widget.NewLabel(" "),
		title,
	)

	// sağ taraf: tema + dil seçici
	right := container.NewHBox(
		themeSel,
		makeLangSelect(w),
	)

	header := container.NewBorder(
		nil, nil,
		left,
		right,
	)

	// hepsini pencereye bas
	w.SetContent(container.NewBorder(header, nil, nil, nil, tabs))
}

// === AI & Telemetry Monitors (BEGIN) ===

type AICallbacks struct {
	OnTelemetry func(raw string)
	OnAlert     func(raw string)
	OnAnalysis  func(raw string)
	OnBonus     func(raw string)
}

// Bu fonksiyon, pencereye göre UI callback'lerini hazırlar
func makeAICallbacks(win fyne.Window) AICallbacks {
	return AICallbacks{
		// Node Telemetry: height / peers / mempool özetini en üst kutuya yaz
		OnTelemetry: func(raw string) {
			ui(func() {
				if aiTelemetryText == nil {
					return
				}

				raw = strings.TrimSpace(raw)
				if raw == "" {
					aiTelemetryText.SetText("")
					return
				}

				// JSON ise temel metrikleri çıkarmayı dene
				var m map[string]any
				if err := json.Unmarshal([]byte(raw), &m); err == nil {
					getStr := func(keys ...string) string {
						for _, k := range keys {
							v, ok := m[k]
							if !ok {
								continue
							}
							switch vv := v.(type) {
							case float64:
								return fmt.Sprintf("%.0f", vv)
							case int:
								return fmt.Sprintf("%d", vv)
							case int64:
								return fmt.Sprintf("%d", vv)
							case string:
								if strings.TrimSpace(vv) != "" {
									return vv
								}
							}
						}
						return ""
					}

					height := getStr("height", "block_height", "tip_height")
					peers := getStr("peers", "peer_count", "num_peers")
					mempool := getStr("mempool", "mempool_tx", "mempool_size", "tx_count", "pending_tx")

					var b strings.Builder
					b.WriteString(time.Now().Format("2006-01-02 15:04:05"))
					b.WriteString("\n")

					hasMetric := false
					if height != "" {
						b.WriteString("Height : ")
						b.WriteString(height)
						b.WriteByte('\n')
						hasMetric = true
					}
					if peers != "" {
						b.WriteString("Peers  : ")
						b.WriteString(peers)
						b.WriteByte('\n')
						hasMetric = true
					}
					if mempool != "" {
						b.WriteString("Mempool: ")
						b.WriteString(mempool)
						b.WriteString(" tx\n")
						hasMetric = true
					}

					if hasMetric {
						aiTelemetryText.SetText(b.String())
						return
					}

					// Metrik çıkaramadıysa raw metni göster
					aiTelemetryText.SetText(raw)
					return
				}

				// JSON değilse: zaman damgası + ham çıktı
				aiTelemetryText.SetText(
					time.Now().Format("2006-01-02 15:04:05") + "\n" + raw,
				)
			})
		},

		// AI Alert geldiğinde: hem bildirim gönder, hem AI Alerts kutusuna ekle
		OnAlert: func(raw string) {
			if app := fyne.CurrentApp(); app != nil {
				app.SendNotification(&fyne.Notification{
					Title:   "QuantumCoin AI Alert",
					Content: raw,
				})
			}
			appendAILog(aiAlertsText, "[ALERT]", raw, 100)
		},

		// Analiz sonucu – AI Analysis kutusuna
		OnAnalysis: func(raw string) {
			appendAILog(aiAnalysisText, "[ANALYSIS]", raw, 100)
		},

		// Bonus tahmini – AI Bonus kutusuna
		OnBonus: func(raw string) {
			appendAILog(aiBonusText, "[BONUS]", raw, 100)
		},
	}
}

func startAIMonitors(cb AICallbacks) {
	ctx := context.Background()

	// Önce ortam değişkeninden oku; boşsa port tespiti yap
	apiBase := strings.TrimSpace(os.Getenv("QC_API_BASE"))
	if apiBase == "" {
		port := detectAPIPort()
		apiBase = "http://127.0.0.1:" + port
	}

	// Telemetry (ör: peer sayısı, block height vs.)
	if cb.OnTelemetry != nil {
		go pollLoop(ctx, apiBase+"/api/telemetry", 5*time.Second, cb.OnTelemetry)
	}

	// AI Alerts
	if cb.OnAlert != nil {
		go pollLoop(ctx, apiBase+"/api/ai/alerts", 5*time.Second, cb.OnAlert)
	}

	// AI Analysis
	if cb.OnAnalysis != nil {
		go pollLoop(ctx, apiBase+"/api/ai/analysis", 10*time.Second, cb.OnAnalysis)
	}

	// AI Bonus / ödül tahminleri
	if cb.OnBonus != nil {
		go pollLoop(ctx, apiBase+"/api/ai/bonus", 15*time.Second, cb.OnBonus)
	}
}

func pollLoop(ctx context.Context, url string, interval time.Duration, fn func(string)) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if fn == nil {
				continue
			}
			body, err := httpGetAsString(url)
			if err != nil {
				// İstersen burada log’a yazabilirsin:
				// fmt.Println("AI poll error:", url, err)
				continue
			}
			fn(body)
		}
	}
}

func httpGetAsString(url string) (string, error) {
	client := apiclient.NewNodeClient(3 * time.Second)
	defer client.CloseIdleConnections()
	resp, err := authenticatedGetV1(client, url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// JSON gelirse okunabilir olsun diye prettify etmeye çalışalım
	var pretty map[string]any
	if json.Unmarshal(b, &pretty) == nil {
		prettyBytes, err := json.MarshalIndent(pretty, "", "  ")
		if err == nil {
			return string(prettyBytes), nil
		}
	}

	return string(b), nil
}

// === AI & Telemetry Monitors (END) ===

func main() {
	// Fyne kendi kendine büyümesin
	_ = os.Setenv("FYNE_SCALE", "0.85")

	// açılışta siyah konsol gözükmesin
	hideConsoleOnStartup()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// eski büyük boyutları okumak istemiyoruz, o yüzden ID'siz
	a := app.New()

	loadPrefsAtStartup()

	title := T("title")
	if appVersion != "" {
		title += " — " + appVersion
	}

	w := a.NewWindow(title)
	w.SetOnClosed(func() { _ = stopNativeMining() })

	// daha küçük sabit pencere
	w.Resize(fyne.NewSize(760, 480))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	// mevcut UI'nı kur
	buildUI(w)

	// === AI & Telemetry monitorlerini başlat ===
	// Pencereyi verip callback'leri oluşturuyoruz, sonra poll başlıyor.
	aiCb := makeAICallbacks(w)
	go startAIMonitors(aiCb)

	w.Show()
	a.Run()
}

/*** --- BEGIN: universal balance fetch (legacy-compatible) --- ***/

// Eski kullanım için tek sayı döndüren sarmalayıcı.
// Mevcut tolerant fonksiyonu kullanır ve (confirmed+pending) toplamını verir.

func withAPIAuthV1(req *http.Request) *http.Request {
	req.Header.Set("X-QC-API-KEY", strings.TrimSpace(os.Getenv("QC_API_TOKEN")))
	return req
}
func authenticatedGetV1(client *http.Client, endpoint string) (*http.Response, error) {
	if _, err := apiclient.ValidateNodeURL(endpoint); err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(withAPIAuthV1(req))
}
