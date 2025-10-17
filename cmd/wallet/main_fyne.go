// cmd/wallet/main_fyne.go
package main

import (
	"bufio"
	"bytes"
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

// Windows paylaşım kilidi metnini yakalamak için basit helper
func isSharingViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "being used by another process") ||
		strings.Contains(s, "sharing violation") ||
		strings.Contains(s, "used by another process")
}

/* ================== Versiyon & Renkler ================== */

var appVersion = "wallet-gui v1.6"

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

	// Miner log & state
	minerLogPath                     = ""
	minerTailStop      chan struct{} = nil
	minerTailMu        sync.Mutex
	minerRunningState  = false
	walletRefreshHook  func()
	onMinerStateUpdate func(bool)

	// Loga bağlı otomatik bakiye yenileme için throttle
	minerRefreshMu   sync.Mutex
	minerLastRefresh time.Time

	langSelect *widget.Select
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

// ---- UI helper: Cüzdan sekmesi aktif mi?
func isWalletTabActive(w fyne.Window) bool {
	if w == nil {
		return true
	}
	// buildUI() içinde AppTabs kullanılıyor; seçili sekmeyi bulalım
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

/* ============ Tema ============ */

// Seçili tema değeri (etiketlerden bağımsız)
var themeValue = "auto" // "auto" | "light" | "dark"

// Tema Select'i (dil değişince etiketleri yenilemek için)
var themeSel *widget.Select

// i18n sözlüğünde tema anahtarları yoksa ekle (TR/EN/ES/ZH)
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

// Girilen isimden temayı uygula (TR/EN/ES/ZH etiketlerini anlar)
func applyTheme(name string) {
	s := strings.ToLower(strings.TrimSpace(name))

	switch {
	// — DARK — (TR/EN/ES/ZH)
	case s == "dark" || s == "karanlık" || s == "oscuro" ||
		s == "深色" || s == "深色模式":
		themeValue = "dark"
		fyne.CurrentApp().Settings().SetTheme(theme.DarkTheme())

	// — LIGHT — (TR/EN/ES/ZH)
	case s == "light" || s == "aydınlık" || s == "claro" ||
		s == "浅色" || s == "亮色" || s == "浅色模式":
		themeValue = "light"
		fyne.CurrentApp().Settings().SetTheme(theme.LightTheme())

	// — AUTO/DEFAULT — (TR/EN/ES/ZH)
	case s == "auto" || s == "automatic" || s == "otomatik" ||
		s == "automático" || s == "automatico" || s == "自动" || s == "自動":
		themeValue = "auto"
		fyne.CurrentApp().Settings().SetTheme(theme.DefaultTheme())

	// — fallback —
	default:
		themeValue = "auto"
		fyne.CurrentApp().Settings().SetTheme(theme.DefaultTheme())
	}
}
if a := fyne.CurrentApp(); a != nil {
    a.Preferences().SetString("lang", curLang)
}
refreshThemeSelectLabels() // tema seçenekleri yeni dile anında çevirinsin

    a.Preferences().SetString("theme_value", themeValue)
}

// Dil değişiminde Select etiketlerini güncelle
func refreshThemeSelectLabels() {
	if themeSel == nil {
		return
	}
	themeSel.Options = []string{T("theme_auto"), T("theme_light"), T("theme_dark")}
	switch themeValue {
	case "light":
		themeSel.SetSelected(T("theme_light"))
	case "dark":
		themeSel.SetSelected(T("theme_dark"))
	default:
		themeSel.SetSelected(T("theme_auto"))
	}
	themeSel.Refresh()
}

// Tema Select oluşturucu (yerelleştirilmiş)
func makeThemeSelect() *widget.Select {
	ensureThemeI18nKeys() // etiketleri garanti altına al
	themeSel = widget.NewSelect(
		[]string{T("theme_auto"), T("theme_light"), T("theme_dark")},
		func(label string) {
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

/* === Entegrasyon notları (tek satırlık değişiklikler) ===

1) buildUI(...) içinde, ESKİ tema select satırını kaldır:
   // ESKİ:
   // themeSel := widget.NewSelect([]string{"Otomatik", "Aydınlık", "Karanlık"}, func(s string){ applyTheme(s) })

   // YENİ:
   themeSel = makeThemeSelect()

2) Header kurulumunda global themeSel’i kullanmaya devam edin:
   header := container.NewBorder(
       nil, nil,
       widget.NewLabelWithStyle(T("title"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
       container.NewHBox(themeSel, makeLangSelect(w)),
   )

3) Dil select’inin onChange'inde (dil değiştirince) buildUI(...) çağırmıyorsanız,
   şu satırı ekleyin ki tema menüsü yeni dile çevrilsin:
   refreshThemeSelectLabels()

*/
/* ============ mined_balance.json path (esnek) ============ */

func minedJSONPath() string {
	candidates := []string{}
	if v := strings.TrimSpace(os.Getenv("QC_MINED_PATH")); v != "" {
		candidates = append(candidates, v)
	}
	if v := strings.TrimSpace(os.Getenv("QC_NODE_DIR")); v != "" {
		candidates = append(candidates,
			filepath.Join(v, "mined_balance.json"),
			filepath.Join(v, "miner", "mined_balance.json"),
			filepath.Join(v, "data", "mined_balance.json"),
		)
	}
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			`C:\QuantumMiner\mined_balance.json`,
			`C:\QuantumMiner\miner\mined_balance.json`,
			`C:\QuantumMiner\data\mined_balance.json`,
		)
	}
	d := exeDir()
	candidates = append(candidates,
		filepath.Join(d, "mined_balance.json"),
		filepath.Join(d, "miner", "mined_balance.json"),
	)
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "mined_balance.json"),
			filepath.Join(wd, "miner", "mined_balance.json"),
		)
	}
	altNames := []string{"mined.json", "mining_balance.json", "rewards.json", "miner_balance.json"}
	baseDirs := []string{d}
	if v := strings.TrimSpace(os.Getenv("QC_NODE_DIR")); v != "" {
		baseDirs = append(baseDirs, v)
	}
	if runtime.GOOS == "windows" {
		baseDirs = append(baseDirs, `C:\QuantumMiner`, `C:\QuantumMiner\data`, `C:\QuantumMiner\miner`)
	}
	for _, bd := range baseDirs {
		for _, name := range altNames {
			candidates = append(candidates, filepath.Join(bd, name))
		}
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(exeDir(), "mined_balance.json")
}

/* ============ Miner status helpers ============ */

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if runtime.GOOS == "windows" {
		out, _ := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid)).CombinedOutput()
		return strings.Contains(string(out), fmt.Sprintf("%d", pid))
	}
	return exec.Command("kill", "-0", fmt.Sprint(pid)).Run() == nil
}

/* ===== Miner PID (sadece bizim başlattığımız süreçler) ===== */

func minerPIDPath() string { return filepath.Join(exeDir(), "miner_pid.txt") }

func writeMinerPID(pid int) { _ = os.WriteFile(minerPIDPath(), []byte(strconv.Itoa(pid)), 0644) }
func readMinerPID() (int, error) {
	b, e := os.ReadFile(minerPIDPath())
	if e != nil {
		return 0, e
	}
	return strconv.Atoi(strings.TrimSpace(string(b)))
}
func clearMinerPID() { _ = os.Remove(minerPIDPath()) }

func minerActiveViaAPI() bool {
	if apiPort == "" {
		apiPort = detectAPIPort()
	}
	if !portHasAPI(apiPort) {
		return false
	}
	client := &http.Client{Timeout: 700 * time.Millisecond}
	for _, p := range []string{"/api/miner/status", "/api/miner"} {
		resp, err := client.Get(apiBase() + p)
		if err != nil {
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		var m map[string]interface{}
		if json.Unmarshal(b, &m) == nil {
			for _, k := range []string{"active", "running", "mining", "is_active"} {
				switch v := m[k].(type) {
				case bool:
					if v {
						return true
					}
				case string:
					if strings.EqualFold(v, "true") || strings.EqualFold(v, "running") || strings.EqualFold(v, "active") {
						return true
					}
				}
			}
		}
	}
	return false
}

func minerActiveHeuristic() bool {
	if pid, err := readMinerPID(); err == nil && isProcessAlive(pid) {
		return true
	}
	return minerActiveViaAPI()
}

/* ============ Node dir & adres dosyaları ============ */

// quantumcoin.exe’nin bulunduğu klasörü bul
func nodeDir() string {
	d := exeDir()
	// 1) GUI ile aynı klasörde miner varsa doğrudan kullan
	if _, err := os.Stat(filepath.Join(d, "quantumcoin.exe")); err == nil {
		return d
	}
	// 2) Ortam değişkeni
	if v := strings.TrimSpace(os.Getenv("QC_NODE_DIR")); v != "" {
		if _, err := os.Stat(filepath.Join(v, "quantumcoin.exe")); err == nil {
			return v
		}
	}
	// 3) Windows sabit konum
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
	// fallback (GUI klasörü)
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

// Mevcut (ilk geçerli) adresi bulur: txt -> APPDATA\wallet.json -> regex fallback
func detectExistingAddress() string {
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

	// 2) Windows: %APPDATA%\QuantumCoin\wallet.json (tolerant)
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
					cands := []string{
						w.Address, w.Addr, w.WalletAddress, w.Reward, w.RewardAddress,
					}
					cands = append(cands, w.Addresses...)
					for _, cand := range cands {
						a := cleanBase58(strings.TrimSpace(cand))
						if a != "" && isLikelyBase58Address(a) {
							return a
						}
					}
				}

				// 2b) Genel map içinden olası anahtarlar
				var m map[string]interface{}
				if json.Unmarshal(b, &m) == nil {
					for _, k := range []string{
						"address", "addr", "wallet", "wallet_address", "reward", "reward_address",
					} {
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

// Sadece adres üretir (quantumcoin.exe newaddr)
func genAddressViaNode() (string, error) {
	exe := filepath.Join(nodeDir(), "quantumcoin.exe")
	if _, err := os.Stat(exe); err != nil {
		return "", fmt.Errorf("quantumcoin.exe not found: %w", err)
	}
	cmd := exec.Command(exe, "newaddr")
	cmd.Dir = nodeDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("newaddr error: %w (out=%s)", err, string(out))
	}
	s := string(out)

	// Önce bilinen biçimi tara
	if i := strings.Index(s, "New Wallet Address:"); i >= 0 {
		part := strings.TrimSpace(s[i+len("New Wallet Address:"):])
		if j := strings.IndexByte(part, '\n'); j >= 0 {
			part = part[:j]
		}
		part = cleanBase58(part)
		if isLikelyBase58Address(part) {
			return part, nil
		}
	}

	// Regex fallback (Base58 26..50)
	re := regexp.MustCompile(`[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]{26,50}`)
	if m := re.FindString(s); m != "" {
		m = cleanBase58(m)
		if isLikelyBase58Address(m) {
			return m, nil
		}
	}
	return "", fmt.Errorf("address not found in newaddr output")
}

// Adres + private key (hex) birlikte üretir (quantumcoin.exe newaddr-priv)
func genAddressPrivViaNode() (addr, privHex string, err error) {
	exe := filepath.Join(nodeDir(), "quantumcoin.exe")
	if _, e := os.Stat(exe); e != nil {
		err = fmt.Errorf("quantumcoin.exe not found: %w", e)
		return
	}
	cmd := exec.Command(exe, "newaddr-priv")
	cmd.Dir = nodeDir()
	out, e := cmd.CombinedOutput()
	if e != nil {
		err = fmt.Errorf("newaddr-priv error: %w (out=%s)", e, string(out))
		return
	}
	s := string(out)

	// Address (etiketli kısım)
	if i := strings.Index(s, "New Wallet Address:"); i >= 0 {
		part := strings.TrimSpace(s[i+len("New Wallet Address:"):])
		if j := strings.IndexByte(part, '\n'); j >= 0 {
			part = part[:j]
		}
		part = cleanBase58(part)
		if isLikelyBase58Address(part) {
			addr = part
		}
	}

	// Private key hex (etiketli kısım)
	if i := strings.Index(s, "PrivateKey (hex):"); i >= 0 {
		part := strings.TrimSpace(s[i+len("PrivateKey (hex):"):])
		if j := strings.IndexByte(part, '\n'); j >= 0 {
			part = part[:j]
		}
		privHex = strings.TrimSpace(part)
	}

	// Address fallback (regex)
	if addr == "" {
		re := regexp.MustCompile(`[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]{26,50}`)
		if m := re.FindString(s); m != "" {
			m = cleanBase58(m)
			if isLikelyBase58Address(m) {
				addr = m
			}
		}
	}

	if addr == "" || privHex == "" {
		err = fmt.Errorf("could not parse address/private key")
		return
	}
	return
}

/* ================== API utils ================== */

func ping(u string, t time.Duration) error {
	client := &http.Client{Timeout: t}
	req, _ := http.NewRequest("GET", noCacheURL(u), nil)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Expires", "0")
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	return nil
}

func portHasAPI(p string) bool {
	candidates := []string{
		"http://127.0.0.1:" + p + "/health",
		"http://127.0.0.1:" + p + "/api/health",
		"http://127.0.0.1:" + p + "/api/status",
		"http://127.0.0.1:" + p + "/api/version",
		"http://127.0.0.1:" + p + "/",
	}
	for _, u := range candidates {
		if ping(u, 400*time.Millisecond) == nil {
			return true
		}
	}
	return false
}

func detectAPIPort() string {
	// Önce env
	for _, k := range []string{"QC_HTTP_PORT", "HTTP_PORT"} {
		if p := strings.TrimSpace(os.Getenv(k)); p != "" && portHasAPI(p) {
			return p
		}
	}
	// Yaygın portlar
	for _, p := range []string{"8082", "8081", "3001", "8080", "9090", "9091", "3000"} {
		if portHasAPI(p) {
			return p
		}
	}
	return "8081"
}

func apiBase() string {
	if apiPort == "" {
		apiPort = detectAPIPort()
	}
	return "http://127.0.0.1:" + apiPort
}

func apiPost(path string, body interface{}, to time.Duration) error {
	b := []byte("{}")
	if body != nil {
		var err error
		b, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	client := &http.Client{Timeout: to}
	resp, err := client.Post(apiBase()+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return nil
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

		"miner_status":    "Durum",
		"running":         "Çalışıyor",
		"stopped":         "Durdu",
		"logs":            "Kayıtlar",
		"theme_auto":      "Otomatik",
		"theme_light":     "Aydınlık",
		"theme_dark":      "Karanlık",
		"open_log_folder": "Log Klasörünü Aç",
		"open_cmd_manual": "CMD (manuel) aç",
		"miner_stopped":   "Madenci durduruldu.",
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

		"miner_status":    "Status",
		"running":         "Running",
		"stopped":         "Stopped",
		"logs":            "Logs",
		"theme_auto":      "Auto",
		"theme_light":     "Light",
		"theme_dark":      "Dark",
		"open_log_folder": "Open Log Folder",
		"open_cmd_manual": "Open CMD (manual)",
		"miner_stopped":   "Miner stopped.",
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

		"miner_status":    "Estado",
		"running":         "En ejecución",
		"stopped":         "Detenido",
		"logs":            "Registros",
		"theme_auto":      "Automático",
		"theme_light":     "Claro",
		"theme_dark":      "Oscuro",
		"open_log_folder": "Abrir carpeta de logs",
		"open_cmd_manual": "Abrir CMD (manual)",
		"miner_stopped":   "Minero detenido.",
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

		"miner_status":    "状态",
		"running":         "运行中",
		"stopped":         "已停止",
		"logs":            "日志",
		"theme_auto":      "自动",
		"theme_light":     "浅色",
		"theme_dark":      "深色",
		"open_log_folder": "打开日志文件夹",
		"open_cmd_manual": "打开 CMD（手动）",
		"miner_stopped":   "矿工已停止。",
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

/* ========== mined_balance.json tolerant reader & pending ========== */

type minedBalance struct {
	Addr      string  `json:"addr"`
	Balance   float64 `json:"balance"`
	UpdatedAt int64   `json:"updated_at"`
}

func readLocalMined() (minedBalance, error) {
	var m minedBalance
	p := minedJSONPath()

	// Dosya kilitlenmiş olabileceği için birden fazla kez dene
	var b []byte
	var err error
	for i := 0; i < 12; i++ { // ~12 * 80ms ≈ 1s
		b, err = os.ReadFile(p)
		if err == nil && len(bytes.TrimSpace(b)) > 0 {
			break
		}
		// Paylaşım kilidi veya boş/yarım yazılmış içerikse biraz bekleyip tekrar dene
		if err != nil && isSharingViolation(err) {
			time.Sleep(80 * time.Millisecond)
			continue
		}
		if err == nil && len(b) == 0 {
			time.Sleep(60 * time.Millisecond)
			continue
		}
		// Başka bir hata ise küçük bir gecikmeyle tekrar dene
		time.Sleep(60 * time.Millisecond)
	}
	if err != nil {
		return m, err
	}

	// Normal parse akışı (sizin mevcut mantığınız)
	if err := json.Unmarshal(b, &m); err == nil && (m.Addr != "" || m.Balance != 0) {
		return m, nil
	}

	var m2 map[string]interface{}
	if err := json.Unmarshal(b, &m2); err != nil {
		return m, err
	}
	if s, ok := m2["addr"].(string); ok && s != "" {
		m.Addr = strings.TrimSpace(s)
	}
	if s, ok := m2["address"].(string); ok && s != "" && m.Addr == "" {
		m.Addr = strings.TrimSpace(s)
	}

	candidates := []string{
		"balance", "confirmed_balance", "confirmed", "spendable",
		"available", "total", "mined_total", "mined", "reward", "rewards", "amount", "pending",
	}
	parseNum := func(v interface{}) (float64, bool) {
		switch t := v.(type) {
		case float64:
			return t, true
		case json.Number:
			if f, err := t.Float64(); err == nil {
				return f, true
			}
		case string:
			if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
				return f, true
			}
		}
		return 0, false
	}
	for _, k := range candidates {
		if v, ok := m2[k]; ok {
			if f, ok := parseNum(v); ok {
				m.Balance = f
				break
			}
		}
	}
	if v, ok := m2["updated_at"].(float64); ok {
		m.UpdatedAt = int64(v)
	} else if v, ok := m2["updated"].(float64); ok {
		m.UpdatedAt = int64(v)
	}

	if m.Balance == 0 {
		re := regexp.MustCompile(`[-+]?\d+(\.\d+)?`)
		if g := re.FindString(string(b)); g != "" {
			if f, err := strconv.ParseFloat(g, 64); err == nil {
				m.Balance = f
			}
		}
	}
	if m.Balance == 0 && m.Addr == "" && m.UpdatedAt == 0 {
		return m, fmt.Errorf("unsupported mined_balance.json")
	}
	return m, nil
}

func getPendingForAddr(addr string) float64 {
	if m, err := readLocalMined(); err == nil {
		if m.Addr == "" || strings.EqualFold(strings.TrimSpace(m.Addr), strings.TrimSpace(addr)) {
			if m.Balance < 0 {
				return 0
			}
			return m.Balance
		}
	}
	return 0
}

/* ===== Balance cache ===== */

func balanceCachePath() string { return filepath.Join(nodeDir(), "wallet_balance.cache.json") }

func saveBalanceCache(addr string, bal float64) {
	m := map[string]interface{}{"address": addr, "balance": bal, "updated_at": time.Now().Unix()}
	b, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(balanceCachePath(), b, 0644)
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

func numericFromAny(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return f, true
		}
	case string:
		s := strings.TrimSpace(t)
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, true
		}
		re := regexp.MustCompile(`[-+]?\d+(\.\d+)?`)
		if g := re.FindString(s); g != "" {
			if f, err := strconv.ParseFloat(g, 64); err == nil {
				return f, true
			}
		}
	}
	return 0, false
}

func getStringAnyKey(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func getAnyKey(m map[string]interface{}, keys ...string) interface{} {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}

/* ========== API balance fetcher (confirmed + pending) - tolerant ========== */

func findBalancesInMapTolerant(m map[string]interface{}) (confirmed float64, pending float64) {
	if f, ok := numericFromAny(getAnyKey(m, "confirmed_balance", "confirmed", "spendable", "available")); ok {
		confirmed = f
	}
	if confirmed == 0 {
		if f, ok := numericFromAny(getAnyKey(m, "balance", "Balance", "total", "Total")); ok {
			confirmed = f
		}
	}
	if f, ok := numericFromAny(getAnyKey(m, "pending", "unconfirmed", "immature", "pending_balance")); ok {
		pending = f
	}
	if bsub, ok := m["balance"].(map[string]interface{}); ok {
		if f, ok := numericFromAny(bsub["confirmed"]); ok {
			confirmed = f
		}
		if f, ok := numericFromAny(bsub["pending"]); ok {
			pending = f
		}
		if confirmed == 0 {
			if tot, ok := numericFromAny(bsub["total"]); ok {
				if pending > 0 {
					confirmed = tot - pending
				} else {
					confirmed = tot
				}
			}
		}
	}
	return
}

func getBalanceUniversal(addr string) (float64, float64, bool) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return 0, 0, false
	}

	// 1) Denenecek portlar: önce mevcut/tespit edilen, sonra yaygınlar (tekrarları at)
	cands := []string{}
	if apiPort != "" {
		cands = append(cands, apiPort)
	}
	if p := detectAPIPort(); p != "" {
		cands = append(cands, p)
	}
	cands = append(cands, "8082", "8081", "3001", "8080", "9090", "9091", "3000")
	seen := map[string]bool{}
	ports := []string{}
	for _, p := range cands {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		ports = append(ports, p)
	}

	// 2) Olası endpoint desenleri
	patterns := []string{
		"/api/address/balance?addr=%s",
		"/api/wallet/balance/%s",
		"/api/balance?addr=%s",
		"/api/address/%s/balance",
		"/api/address/%s",
		"/address/%s/balance",
		"/wallet/balance/%s",
		"/balance/%s",
	}

	client := &http.Client{Timeout: 4 * time.Second}

	for _, p := range ports {
		base := "http://127.0.0.1:" + p

		for _, pat := range patterns {
			var esc string
			if strings.Contains(pat, "?addr=%s") {
				esc = url.QueryEscape(addr)
			} else {
				esc = url.PathEscape(addr)
			}
			u := base + fmt.Sprintf(pat, esc)

			// -- no-cache istek --
			req, _ := http.NewRequest("GET", noCacheURL(u), nil)
			req.Header.Set("Cache-Control", "no-cache")
			req.Header.Set("Pragma", "no-cache")
			req.Header.Set("Expires", "0")

			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			ct := strings.ToLower(resp.Header.Get("Content-Type"))
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			trim := strings.TrimSpace(string(body))
			if len(trim) == 0 || strings.HasPrefix(trim, "<!doctype") || strings.HasPrefix(trim, "<html") || strings.Contains(ct, "text/html") {
				continue
			}

			// JSON parse et; değilse metinden sayı çekmeyi dene
			var anyval interface{}
			if json.Unmarshal(body, &anyval) != nil {
				re := regexp.MustCompile(`[-+]?\d+(\.\d+)?`)
				if g := re.FindString(trim); g != "" {
					if f, err := strconv.ParseFloat(g, 64); err == nil {
						return f, 0, true
					}
				}
				continue
			}

			switch v := anyval.(type) {
			case map[string]interface{}:
				// Adres alanını toleranslı kontrol et
				gotAddr := getStringAnyKey(v, "address", "addr", "account", "wallet")
				if gotAddr != "" && !strings.EqualFold(strings.TrimSpace(gotAddr), addr) {
					break
				}
				c, pnd := findBalancesInMapTolerant(v)
				if c > 0 || pnd > 0 {
					return c, pnd, true
				}
				// Bazı API'ler "data": {...} içinde döner
				if data, ok := v["data"].(map[string]interface{}); ok {
					c, pnd := findBalancesInMapTolerant(data)
					if c > 0 || pnd > 0 {
						return c, pnd, true
					}
				}

			case []interface{}:
				for _, it := range v {
					if m, ok := it.(map[string]interface{}); ok {
						gotAddr := getStringAnyKey(m, "address", "addr", "account", "wallet")
						if gotAddr != "" && !strings.EqualFold(strings.TrimSpace(gotAddr), addr) {
							continue
						}
						c, pnd := findBalancesInMapTolerant(m)
						if c > 0 || pnd > 0 {
							return c, pnd, true
						}
					}
				}

			default:
				if f, ok := numericFromAny(v); ok {
					return f, 0, true
				}
			}
		}
	}

	return 0, 0, false
}

/* ================== Tek adresli bonus_store helper ================== */

func ensureBonusStore(addr string) {
	p := filepath.Join(nodeDir(), "bonus_store.json")
	if _, err := os.Stat(p); err == nil {
		return // varsa dokunma
	}
	m := map[string]interface{}{
		"community_address": addr,
		"dev_fund_address":  addr,
		"premine_address":   addr,
		"reward_percentages": map[string]int{
			"miner":     95,
			"community": 5,
			"dev":       0,
			"premine":   0,
		},
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(p, b, 0644)
}

/* ================== UI: Wallet ================== */

func detectPrivHex() string {
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

	copyMyBtn := widget.NewButton(T("copy"), func() { w.Clipboard().SetContent(myAddrEntry.Text) })
	receiveBtn := widget.NewButton(T("receive_copy"), func() {
		w.Clipboard().SetContent(myAddrEntry.Text)
		dialog.ShowInformation(T("ok"), myAddrEntry.Text, w)
	})
	pasteToBtn := widget.NewButton(T("paste"), func() { toEntry.SetText(w.Clipboard().Content()) })

	genBtn := widget.NewButton(T("generate_wallet"), func() {
		addr, priv, err := genAddressPrivViaNode()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		_ = saveText(walletAddressPath(), addr)
		_ = saveText(minerAddressPath(), addr)
		_ = saveText(walletPrivPath(), priv)
		myAddrEntry.SetText(addr)
		fromEntry.SetText(addr)
		dialog.ShowInformation(T("ok"), addr, w)
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

		pendingLocal := getPendingForAddr(addr)

		if confirmed, pendingAPI, ok := getBalanceUniversal(addr); ok {
			// API + local pending birleşimi
			pending := pendingAPI
			if pendingLocal > pending {
				pending = pendingLocal
			}
			total := confirmed + pending
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
				if pending > 1e-7 {
					balanceText.Text = fmt.Sprintf(
						"%s: %.8f QC   (confirmed: %.8f • pending: %.8f)",
						T("balance"), total, confirmed, pending,
					)
				} else {
					balanceText.Text = fmt.Sprintf("%s: %.8f QC", T("balance"), total)
				}

				canvas.Refresh(balanceText)
				lastBalance = total
			})
			return
		}

		// >>> ADD: Legacy HTTP fallback (modern uç başarısızsa)
		if n, err := getBalanceUniversalLegacy(addr); err == nil && n > 0 {
			total := float64(n) // legacy toplam (confirmed+pending) kabulü
			ui(func() {
				if total > lastBalance {
					balanceText.Color = colSuccess
				} else if total < lastBalance {
					balanceText.Color = colError
				} else {
					balanceText.Color = colInfo
				}
				// Yerel pending bilgisi varsa ek bilgi olarak göster
				if pendingLocal > 1e-7 {
					balanceText.Text = fmt.Sprintf(
						"%s: %.8f QC   (pending≈ %.8f)",
						T("balance"), total, pendingLocal,
					)
				} else {
					balanceText.Text = fmt.Sprintf("%s: %.8f QC", T("balance"), total)
				}
				canvas.Refresh(balanceText)
				lastBalance = total
			})
			saveBalanceCache(addr, total)
			return
		}
		// <<< END ADD

		// API başarısızsa cache
		if total, ok := loadBalanceCache(addr); ok {
			ui(func() {
				balanceText.Color = colInfo
				balanceText.Text = fmt.Sprintf("%s: %.8f QC", T("balance"), total)
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
	backupBtn := widget.NewButton("Cüzdanı Yedekle", func() {
		fs := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
			if err != nil || uc == nil {
				return
			}
			path := uc.URI().Path()
			uc.Close()

			pass := widget.NewEntry()
			pass.Password = true
			dialog.NewCustomConfirm("Yedek Parolası", "Tamam", "İptal",
				container.NewVBox(widget.NewLabel("Yedek parolasını girin:"), pass),
				func(ok bool) {
					if !ok {
						return
					}
					if strings.TrimSpace(pass.Text) == "" {
						dialog.ShowError(fmt.Errorf("Parola boş olamaz"), w)
						return
					}
					if err := createWalletBackup(path, pass.Text); err != nil {
						dialog.ShowError(err, w)
						return
					}
					dialog.ShowInformation("Tamam", "Yedek oluşturuldu.", w)
				}, w).Show()
		}, w)
		fs.SetFileName("qc_wallet_backup.qcbak")
		fs.Show()
	})

	restoreBtn := widget.NewButton("Cüzdanı Geri Yükle", func() {
		fo := dialog.NewFileOpen(func(ur fyne.URIReadCloser, err error) {
			if err != nil || ur == nil {
				return
			}
			path := ur.URI().Path()
			ur.Close()

			pass := widget.NewEntry()
			pass.Password = true
			dialog.NewCustomConfirm("Yedek Parolası", "Tamam", "İptal",
				container.NewVBox(widget.NewLabel("Parolayı girin:"), pass),
				func(ok bool) {
					if !ok {
						return
					}
					if err := restoreWalletBackup(path, pass.Text); err != nil {
						dialog.ShowError(err, w)
						return
					}
					dialog.ShowInformation("Tamam", "Geri yüklendi (uygulamayı yeniden başlatın).", w)
				}, w).Show()
		}, w)
		fo.Show()
	})

	sendBtn := widget.NewButton(T("send"), func() {
		if err := startAPIBackground(); err != nil {
			dialog.ShowError(fmt.Errorf("%s %v", T("api_timeout"), err), w)
			return
		}
		privHex := strings.TrimSpace(detectPrivHex())
		if privHex == "" {
			dialog.ShowError(fmt.Errorf("Özel anahtar bulunamadı. 'Yeni Cüzdan Oluştur' ile anahtar üretin."), w)
			return
		}
		body := map[string]string{
			"from":     strings.TrimSpace(fromEntry.Text),
			"to":       strings.TrimSpace(toEntry.Text),
			"amount":   strings.TrimSpace(amtEntry.Text),
			"fee":      strings.TrimSpace(feeEntry.Text),
			"priv_hex": privHex,
		}
		if body["from"] == "" || body["to"] == "" || body["amount"] == "" {
			dialog.ShowError(fmt.Errorf("from/to/amount gerekli"), w)
			return
		}
		// /api/tx/send dene; olmazsa /api/send fallback
		if err := apiPost("/api/tx/send", body, 8*time.Second); err != nil {
			if err2 := apiPost("/api/send", body, 8*time.Second); err2 != nil {
				dialog.ShowError(fmt.Errorf("Gönderilemedi: %v / %v", err, err2), w)
				return
			}
		}
		dialog.ShowInformation(T("ok"), T("tx_sent"), w)
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

	return container.NewVBox(
		container.NewHBox(balanceText, refreshBtn, startAPIbtn, receiveBtn),
		form,
		container.NewHBox(sendBtn, backupBtn, restoreBtn),
	)
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
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	key, err := deriveKey(pass, salt)
	if err != nil {
		return err
	}
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

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(salt); err != nil {
		return err
	}
	if _, err := f.Write(nonce); err != nil {
		return err
	}
	if _, err := f.Write(ct); err != nil {
		return err
	}
	return nil
}

func decryptJSONFromFile(path string, pass string, out interface{}) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(b) < 16 {
		return fmt.Errorf("bozuk dosya")
	}
	salt := b[:16]
	key, err := deriveKey(pass, salt)
	if err != nil {
		return err
	}
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
		return err
	}
	return json.Unmarshal(pt, out)
}

func createWalletBackup(targetPath, pass string) error {
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
	if b, err := os.ReadFile(filepath.Join(dir, "wallet_balance.cache.json")); err == nil {
		var tmp interface{}
		_ = json.Unmarshal(b, &tmp)
		m["balance_cache"] = tmp
	}
	if b, err := os.ReadFile(filepath.Join(dir, "bonus_store.json")); err == nil {
		var tmp interface{}
		_ = json.Unmarshal(b, &tmp)
		m["bonus_store"] = tmp
	}
	return encryptJSONToFile(targetPath, m, pass)
}

func restoreWalletBackup(sourcePath, pass string) error {
	var m map[string]interface{}
	if err := decryptJSONFromFile(sourcePath, pass, &m); err != nil {
		return err
	}
	dir := nodeDir()
	if v, ok := m["wallet_address"].(string); ok {
		_ = os.WriteFile(filepath.Join(dir, "wallet_address.txt"), []byte(strings.TrimSpace(v)), 0644)
	}
	if v, ok := m["miner_address"].(string); ok {
		_ = os.WriteFile(filepath.Join(dir, "miner_address.txt"), []byte(strings.TrimSpace(v)), 0644)
	}
	if v, ok := m["wallet_priv_hex"].(string); ok {
		_ = os.WriteFile(filepath.Join(dir, "wallet_priv.hex"), []byte(strings.TrimSpace(v)), 0644)
	}
	if v, ok := m["balance_cache"]; ok {
		if b, err := json.MarshalIndent(v, "", "  "); err == nil {
			_ = os.WriteFile(filepath.Join(dir, "wallet_balance.cache.json"), b, 0644)
		}
	}
	if v, ok := m["bonus_store"]; ok {
		if b, err := json.MarshalIndent(v, "", "  "); err == nil {
			_ = os.WriteFile(filepath.Join(dir, "bonus_store.json"), b, 0644)
		}
	}
	return nil
}

/* ================== Miner start/stop & log tail ================== */

// panic / invalid base58 satırlarını kırmızı göstermek için
var rePanic = regexp.MustCompile(`(?i)\b(panic|invalid base58)\b`)
var (
	// EN + TR varyasyonları: "block found", "block mined", "new block",
	// "yeni blok", "blok bulundu", "blok kazıldı"
	reBlock = regexp.MustCompile(`(?i)\b(block\s*(found|mined)|new\s*block|yeni\s*blok|blok\s*(bulundu|kazıldı))\b`)
	reHash  = regexp.MustCompile(`(?i)\b([a-f0-9]{16,})\b`)
)

// --- Blok sonrası birkaç kez aralıklı yenile (eventual consistency için)
func burstRefreshAfterBlock() {
	if walletRefreshHook == nil {
		return
	}
	for _, d := range []time.Duration{0, 1200 * time.Millisecond, 3 * time.Second, 6 * time.Second} {
		delay := d
		go func() {
			time.Sleep(delay)
			walletRefreshHook()
		}()
	}
}

// Base58 (BTC-style) — 0,O,I,l yasak; 26..50 uzunluk
var base58RE = regexp.MustCompile(`^[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]{26,50}$`)

func isLikelyBase58Address(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 26 || len(s) > 50 {
		return false
	}
	// yasak karakterler
	if strings.IndexAny(s, "0OIl") >= 0 {
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

// Miner log dosyasını RichText’e canlı akıt
func startMinerTailInto(rich *widget.RichText) {
	minerTailMu.Lock()
	defer minerTailMu.Unlock()

	if minerTailStop != nil {
		close(minerTailStop)
		minerTailStop = nil
	}
	if minerLogPath == "" {
		return
	}
	f, err := os.Open(minerLogPath)
	if err != nil {
		return
	}
	// dosyanın sonundan tail et
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		_ = f.Close()
		return
	}
	stop := make(chan struct{})
	minerTailStop = stop

	go func() {
		defer f.Close()
		reader := bufio.NewReader(f)
		for {
			select {
			case <-stop:
				return
			default:
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						time.Sleep(200 * time.Millisecond)
						continue
					}
					return
				}

				ui(func() { appendColoredLog(rich, line) })

				// Loga bağlı otomatik bakiye yenileme (throttle'lı + burst)
				if walletRefreshHook != nil {
					if reBlock.MatchString(line) {
						// Blok bulundu: aralıklı birkaç kez yenile (eventual consistency)
						go burstRefreshAfterBlock()
					} else {
						// Diğer loglarda en fazla ~1.5 sn'de bir yenile
						minerRefreshMu.Lock()
						if time.Since(minerLastRefresh) > 1500*time.Millisecond {
							minerLastRefresh = time.Now()
							go walletRefreshHook()
						}
						minerRefreshMu.Unlock()
					}
				}
			}
		}
	}()
}

func stopMinerTail() {
	minerTailMu.Lock()
	defer minerTailMu.Unlock()
	if minerTailStop != nil {
		close(minerTailStop)
		minerTailStop = nil
	}
}

// Log satırı renklendirme
func appendColoredLog(rt *widget.RichText, line string) {
	// panic/invalid base58 => kırmızı
	if rePanic.MatchString(line) {
		rt.Segments = append(rt.Segments, &widget.TextSegment{
			Text: line,
			Style: widget.RichTextStyle{
				ColorName: theme.ColorNameError,
			},
		})
		rt.Refresh()
		return
	}

	// "block found/mined" kısımlarını yeşil yap (tema success)
	segs := []widget.RichTextSegment{}
	idx := 0
	for _, loc := range reBlock.FindAllStringIndex(line, -1) {
		if loc[0] > idx {
			segs = append(segs, &widget.TextSegment{Text: line[idx:loc[0]]})
		}
		segs = append(segs, &widget.TextSegment{
			Text: line[loc[0]:loc[1]],
			Style: widget.RichTextStyle{
				ColorName: theme.ColorNameSuccess,
			},
		})
		idx = loc[1]
	}
	if idx < len(line) {
		segs = append(segs, &widget.TextSegment{Text: line[idx:]})
	}

	// içindeki hash’leri (hex) mavi renge boya
	colored := []widget.RichTextSegment{}
	for _, s := range segs {
		if txt, ok := s.(*widget.TextSegment); ok {
			chunks := splitByRegex(txt.Text, reHash)
			for _, c := range chunks {
				if reHash.MatchString(c) {
					colored = append(colored, &widget.TextSegment{
						Text: c,
						Style: widget.RichTextStyle{
							ColorName: theme.ColorNamePrimary,
						},
					})
				} else {
					colored = append(colored, &widget.TextSegment{Text: c})
				}
			}
		} else {
			colored = append(colored, s)
		}
	}

	rt.Segments = append(rt.Segments, colored...)
	rt.Refresh()
}

func splitByRegex(s string, r *regexp.Regexp) []string {
	out := []string{}
	last := 0
	for _, loc := range r.FindAllStringIndex(s, -1) {
		if last < loc[0] {
			out = append(out, s[last:loc[0]])
		}
		out = append(out, s[loc[0]:loc[1]])
		last = loc[1]
	}
	if last < len(s) {
		out = append(out, s[last:])
	}
	return out
}

/* ================== API bootstrap ================== */

// API'yi arka planda doğrudan process olarak başlatır; logları api_out.log’a yazar.
func startAPIBackground() error {
	if apiPort == "" {
		apiPort = detectAPIPort()
	}
	if portHasAPI(apiPort) {
		return nil
	}

	dir := nodeDir()
	exe := filepath.Join(dir, "quantumcoin.exe")
	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("quantumcoin.exe not found: %v", err)
	}

	apiLog := filepath.Join(dir, "api_out.log")
	f, err := os.OpenFile(apiLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	cmd := exec.Command(exe, "api")
	cmd.Dir = dir
	// QC_* adreslerini burada set etmiyoruz; sadece node-dir
	cmd.Env = append(os.Environ(), "QC_NODE_DIR="+dir)
	cmd.Stdout = f
	cmd.Stderr = f
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	if err := cmd.Start(); err != nil {
		_ = f.Close()
		return err
	}
	go func() { _ = cmd.Wait(); _ = f.Close() }()

	// Sağlık kontrolü
	health := apiBase() + "/health"
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if ping(health, 500*time.Millisecond) == nil {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}

	// ADD: /health çalışmasa da API'yı bulmak için ek port taraması
	// portHasAPI() zaten /health, /api/health, /api/status, /api/version ve /'i dener.
	{
		tryPorts := []string{}
		if apiPort != "" {
			tryPorts = append(tryPorts, apiPort) // mevcut portu öncele
		}
		tryPorts = append(tryPorts, "8082", "8081", "3001", "8080", "9090", "9091", "3000")

		seen := map[string]bool{}
		deadline2 := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline2) {
			// Mevcut apiPort'u kontrol et
			if apiPort != "" && portHasAPI(apiPort) {
				return nil
			}

			// detectAPIPort() tekrar dene (ortam değişmiş olabilir)
			if p := detectAPIPort(); p != "" && !seen[p] && portHasAPI(p) {
				apiPort = p
				return nil
			}

			// Yaygın portları tara
			for _, p := range tryPorts {
				if p == "" || seen[p] {
					continue
				}
				if portHasAPI(p) {
					apiPort = p
					return nil
				}
				seen[p] = true
			}
			time.Sleep(300 * time.Millisecond)
		}
	}

	return fmt.Errorf("API timeout")
}

/* ================== Miner starter (görünür CMD) ================== */

// Miner'ı kullanıcı adresiyle başlatır.
// Windows: görünür CMD penceresi (UTF-8) + miner_out.log'a yazdırır (loop)
// macOS/Linux: arka planda, miner_out.log'a yazar
func startMinerInCmd(addr string) error {
	addr = cleanBase58(strings.TrimSpace(addr))
	if !isLikelyBase58Address(addr) {
		return fmt.Errorf("invalid reward address (Base58)")
	}

	dir := nodeDir()
	exe := filepath.Join(dir, "quantumcoin.exe")
	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("quantumcoin.exe not found: %w", err)
	}

	// Tek-adres env’leri ve yapı
	_ = os.Setenv("QC_NODE_DIR", dir)
	_ = os.Setenv("QC_COMMUNITY_ADDRESS", addr)
	_ = os.Setenv("QC_DEV_FUND_ADDRESS", addr)
	_ = os.Setenv("QC_PREMINE_ADDRESS", addr)
	ensureBonusStore(addr)

	// --- WINDOWS ---
	if runtime.GOOS == "windows" {
		// POZİSYONEL argüman: mine "ADDR"  ( -addr kullanma )
		argLine := fmt.Sprintf(`%s "%s"`, minerSubcommand, addr)

		runBat := filepath.Join(dir, "run_miner.cmd")
		bat := fmt.Sprintf(`@echo off
setlocal
chcp 65001 >NUL
cd /d "%s"
set "QC_NODE_DIR=%s"
set "QC_COMMUNITY_ADDRESS=%s"
set "QC_DEV_FUND_ADDRESS=%s"
set "QC_PREMINE_ADDRESS=%s"
set "QC_MINED_PATH=%%CD%%\mined_balance.json"
set "ADDR=%s"

REM PowerShell var mı? (tee için)
where powershell >NUL 2>&1
if %%ERRORLEVEL%% EQU 0 (set "HAS_PS=1") else (set "HAS_PS=0")

REM Eski stop bayrağını temizle
del /q "miner_stop.flag" 2>NUL

echo ===============================
echo Mining to: %%ADDR%%
echo Folder    : %%CD%%
echo ===============================

:loop
if exist "miner_stop.flag" goto :done

if "%%HAS_PS%%"=="1" (
  powershell -NoLogo -ExecutionPolicy Bypass -Command ^
    "& { & '.\\quantumcoin.exe' %s 2>&1 | Tee-Object -File 'miner_out.log' -Append }"
) else (
  ".\\quantumcoin.exe" %s >> "miner_out.log" 2>&1
)

REM Kısa nefes
timeout /t 1 /nobreak >NUL
goto :loop

:done
echo Stopped by miner_stop.flag
`, dir, dir, addr, addr, addr, addr, argLine, argLine)

		if err := os.WriteFile(runBat, []byte(bat), 0644); err != nil {
			return fmt.Errorf("could not write run_miner.cmd: %w", err)
		}
		// CRLF zorlama
		batCRLF := strings.ReplaceAll(bat, "\n", "\r\n")
		if err := os.WriteFile(runBat, []byte(batCRLF), 0644); err != nil {
			return fmt.Errorf("could not rewrite run_miner.cmd (CRLF): %w", err)
		}

		cmd := exec.Command("cmd", "/c", "start", "QuantumCoin Miner", runBat)
		cmd.Dir = dir
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("could not start CMD: %w", err)
		}

		// GUI’nin tail’leyebilmesi için
		minerLogPath = filepath.Join(dir, "miner_out.log")
		minerRunningState = true
		if onMinerStateUpdate != nil {
			onMinerStateUpdate(true)
		}
		// İlk kısa gecikmeli yenile
		if walletRefreshHook != nil {
			go func() { time.Sleep(3 * time.Second); walletRefreshHook() }()
			// Periyodik (5 sn) ~10 dk boyunca — miner kapanırsa döngü doğal olarak biter
			go func() {
				for i := 0; i < 120 && minerRunningState; i++ {
					time.Sleep(5 * time.Second)
					walletRefreshHook()
				}
			}()
		}
		return nil
	}

	// --- macOS / LINUX ---
	args := []string{minerSubcommand, addr} // pozisyonel
	minerLogPath = filepath.Join(dir, "miner_out.log")
	lf, _ := os.OpenFile(minerLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	cmd := exec.Command(exe, args...)
	cmd.Dir = dir
	// Çevre değişkenleri (mined_balance.json yolu dahil)
	cmd.Env = append(os.Environ(),
		"QC_NODE_DIR="+dir,
		"QC_COMMUNITY_ADDRESS="+addr,
		"QC_DEV_FUND_ADDRESS="+addr,
		"QC_PREMINE_ADDRESS="+addr,
		"QC_MINED_PATH="+filepath.Join(dir, "mined_balance.json"),
	)
	if lf != nil {
		cmd.Stdout = lf
		cmd.Stderr = lf
	}
	if err := cmd.Start(); err != nil {
		if lf != nil {
			_ = lf.Close()
		}
		return err
	}
	writeMinerPID(cmd.Process.Pid)
	go func() {
		_ = cmd.Wait()
		if lf != nil {
			_ = lf.Close()
		}
	}()

	minerRunningState = true
	if onMinerStateUpdate != nil {
		onMinerStateUpdate(true)
	}
	// İlk kısa gecikmeli yenile + periyodik 5 sn (Linux/macOS)
	if walletRefreshHook != nil {
		go func() { time.Sleep(3 * time.Second); walletRefreshHook() }()
		go func() {
			for i := 0; i < 120 && minerRunningState; i++ {
				time.Sleep(5 * time.Second)
				walletRefreshHook()
			}
		}()
	}
	return nil
}

func ensureRewardAddress(cur string) (string, error) {
	// 1) Kullanıcı girdisi
	a := cleanBase58(strings.TrimSpace(cur))
	if isLikelyBase58Address(a) {
		return a, nil
	}

	if v := detectExistingAddress(); isLikelyBase58Address(v) {
		_ = saveText(walletAddressPath(), v)
		_ = saveText(minerAddressPath(), v)
		ensureBonusStore(v)
		return v, nil
	}

	addr, priv, err := genAddressPrivViaNode()
	if err != nil {
		return "", err
	}
	if !isLikelyBase58Address(addr) {
		return "", fmt.Errorf("üretilen adres geçersiz görünüyor")
	}

	_ = saveText(walletAddressPath(), addr)
	_ = saveText(minerAddressPath(), addr)
	_ = saveText(walletPrivPath(), priv)
	ensureBonusStore(addr)

	return addr, nil
}

func makeMinerTab(w fyne.Window, defaultAddr string) fyne.CanvasObject {
	addrEntry := widget.NewEntry()
	addrEntry.SetPlaceHolder(T("reward_addr"))
	if v := detectExistingAddress(); v != "" {
		addrEntry.SetText(v)
	} else if defaultAddr != "" {
		addrEntry.SetText(defaultAddr)
	}

	statusLab := widget.NewLabel(T("stopped"))
	logView := widget.NewRichText()
	logView.Wrapping = fyne.TextWrapWord
	logView.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: T("logs") + ":\n"}}

	// Sekme açılır açılmaz tail bağla
	minerLogPath = filepath.Join(nodeDir(), "miner_out.log")
	startMinerTailInto(logView)

	startBtn := widget.NewButton(T("miner_start"), func() {
		// Adresi garanti altına al (gerekirse otomatik üretir ve dosyaya yazar)
		addr, err := ensureRewardAddress(addrEntry.Text)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Adres bulunamadı/üretilemedi: %v", err), w)
			return
		}
		addrEntry.SetText(addr)

		// Miner'ı nodeDir() içindeki quantumcoin.exe ile başlat
		if err := startMinerInCmd(addr); err != nil {
			dialog.ShowError(fmt.Errorf("Miner could not start: %v", err), w)
			return
		}
		_ = startAPIBackground() // API’yi kaldır, balance için lazım
		dialog.ShowInformation(T("ok"), "Miner started.", w)

		// Yeni süreç için tail'i tazele (idempotent)
		minerLogPath = filepath.Join(nodeDir(), "miner_out.log")
		startMinerTailInto(logView)

		// Kısa süre sık yenile (bakiye/status)
		if walletRefreshHook != nil {
			go func() {
				deadline := time.Now().Add(30 * time.Second)
				for time.Now().Before(deadline) {
					time.Sleep(2 * time.Second)
					walletRefreshHook()
					if onMinerStateUpdate != nil {
						onMinerStateUpdate(true)
					}
				}
			}()
		}
	})
	// --- Stop + yardımcı butonlar (tam blok) ---
	stopBtn := widget.NewButton(T("miner_stop"), func() {
		// 0) Zarif durdurma bayrağı
		_ = os.WriteFile(filepath.Join(nodeDir(), "miner_stop.flag"), []byte("stop"), 0644)

		// 1) Kısa bekleme (döngüden çıkabilsin)
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if !minerActiveHeuristic() {
				break
			}
			time.Sleep(150 * time.Millisecond)
		}

		// 2) API üstünden durdurmayı dene (sessiz)
		_ = apiPost("/api/miner/stop", map[string]string{}, 800*time.Millisecond)

		// 3) PID varsa PID’den indir; yoksa imajdan indir
		if pid, err := readMinerPID(); err == nil && pid > 0 {
			if runtime.GOOS == "windows" {
				// Çalışan proses
				_ = exec.Command("taskkill", "/PID", fmt.Sprint(pid), "/F").Run()
				// Ayrı açılmış CMD penceresini başlığa göre kapat (wildcard ile)
				_ = exec.Command("taskkill", "/F", "/T", "/FI", `WINDOWTITLE eq QuantumCoin Miner*`).Run()
			} else {
				_ = exec.Command("kill", "-9", fmt.Sprint(pid)).Run()
			}
			clearMinerPID()
		} else {
			if runtime.GOOS == "windows" {
				_ = exec.Command("taskkill", "/IM", "quantumcoin.exe", "/F").Run()
				_ = exec.Command("taskkill", "/F", "/T", "/FI", `WINDOWTITLE eq QuantumCoin Miner*`).Run()
			} else {
				_ = exec.Command("pkill", "-f", "quantumcoin").Run()
			}
		}

		minerRunningState = false
		if onMinerStateUpdate != nil {
			onMinerStateUpdate(false)
		}
		stopMinerTail()
		dialog.ShowInformation(T("ok"), T("miner_stopped"), w)
	})

	openLogBtn := widget.NewButton(T("open_log_folder"), func() {
		dir := nodeDir()
		if minerLogPath == "" {
			minerLogPath = filepath.Join(dir, "miner_out.log")
		}
		if runtime.GOOS == "windows" {
			_ = exec.Command("explorer.exe", dir).Start()
		} else {
			_ = exec.Command("xdg-open", dir).Start()
		}
	})

	openManualCmdBtn := widget.NewButton(T("open_cmd_manual"), func() {
		dir := nodeDir()
		runBat := filepath.Join(dir, "run_miner.cmd")
		if _, err := os.Stat(runBat); err == nil {
			_ = exec.Command("cmd", "/c", "start", "QuantumCoin Miner", runBat).Start()
		}
	})

	// adres toparla
	addr := strings.TrimSpace(addrEntry.Text)
	if addr == "" {
		addr = loadText(minerAddressPath())
	}
	if addr == "" {
		addr = loadText(walletAddressPath())
	}
	if addr == "" {
		if v, err := genAddressViaNode(); err == nil && v != "" {
			addr = v
			_ = saveText(walletAddressPath(), addr)
			_ = saveText(minerAddressPath(), addr)
		}
	}
	addr = cleanBase58(addr)
	addrEntry.SetText(addr)

	// Adres geçersizse UI'yi kurmayı kesmeyelim; sadece uyarı verelim.
	// (Start butonu zaten ensureRewardAddress ile üretip kaydedecek.)
	if addr == "" || !isLikelyBase58Address(addr) {
		dialog.ShowError(fmt.Errorf("Geçersiz ödül adresi (Base58)"), w)
	} else {
		_ = saveText(minerAddressPath(), addr)
	}

	// --- Miner durum etiketini güncelleyen callback ---
	onMinerStateUpdate = func(running bool) {
		ui(func() {
			if running {
				statusLab.SetText(T("running"))
			} else {
				statusLab.SetText(T("stopped"))
			}
			statusLab.Refresh()
		})
	}
	onMinerStateUpdate(minerActiveHeuristic())

	// Üst form ve buton bar
	top := widget.NewForm(
		widget.NewFormItem(T("reward_addr"), addrEntry),
		widget.NewFormItem(T("miner_status"), statusLab),
	)

	btnBar := container.NewHBox(
		startBtn,
		stopBtn,
		openLogBtn,
		openManualCmdBtn,
	)

	// >>> DÖNÜŞ <<<  — makeMinerTab mutlaka bir CanvasObject döndürmeli
	return container.NewVBox(
		top,
		btnBar,
		widget.NewSeparator(),
		container.NewVScroll(logView),
	)
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
			dialog.ShowError(fmt.Errorf("Web wallet could not open: %v", err), w)
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
	langSelect = widget.NewSelect([]string{
		"Türkçe (tr)", "English (en)", "Español (es)", "中文 (zh)",
	}, func(s string) {
		switch {
		case strings.Contains(s, "(tr)"):
			curLang = "tr"
		case strings.Contains(s, "(en)"):
			curLang = "en"
		case strings.Contains(s, "(es)"):
			curLang = "es"
		case strings.Contains(s, "(zh)"):
			curLang = "zh"
		}
		// Otomatik yenilemeyi sıfırla ve UI'ı yeniden kur
		if walletAutoRefreshStop != nil {
			close(walletAutoRefreshStop)
			walletAutoRefreshStop = nil
		}
		w.SetTitle(T("title") + " — " + appVersion)
		buildUI(w)
	})
	langSelect.SetSelected("Türkçe (tr)")
	return langSelect
}

func buildUI(w fyne.Window) {
	myAddrEntry := widget.NewEntry()
	fromEntry := widget.NewEntry()
	myAddrEntry.SetPlaceHolder(T("my_address"))
	fromEntry.SetPlaceHolder(T("from"))
	// ---- Tek-adres politikası & env sabitleme (async) ----
	go func() {
		addr := detectExistingAddress()
		if strings.TrimSpace(addr) == "" {
			if a, err := genAddressViaNode(); err == nil && strings.TrimSpace(a) != "" {
				_ = saveText(walletAddressPath(), a)
				_ = saveText(minerAddressPath(), a)
				addr = a
			}
		}
		addr = cleanBase58(addr)
		if addr != "" {
			_ = os.Setenv("QC_COMMUNITY_ADDRESS", addr)
			_ = os.Setenv("QC_DEV_FUND_ADDRESS", addr)
			_ = os.Setenv("QC_PREMINE_ADDRESS", addr)
			_ = os.Setenv("QC_NODE_DIR", nodeDir())

			// >>> EKLE: GUI hangi mined_balance.json'ı okuyacağını sabitle
			_ = os.Setenv("QC_MINED_PATH", filepath.Join(nodeDir(), "mined_balance.json"))

			ensureBonusStore(addr)
		}
	}()

	// Var olan adresleri otomatik doldur
	if addr := detectExistingAddress(); addr != "" {
		myAddrEntry.SetText(addr)
		fromEntry.SetText(addr)
	}

	// Adres boşsa node üzerinden üret ve dosyaya yaz
	if strings.TrimSpace(myAddrEntry.Text) == "" {
		if addr, err := genAddressViaNode(); err == nil && strings.TrimSpace(addr) != "" {
			_ = saveText(walletAddressPath(), addr)
			_ = saveText(minerAddressPath(), addr)
			myAddrEntry.SetText(addr)
			fromEntry.SetText(addr)
		}
	}

	themeSel := widget.NewSelect(
		[]string{T("theme_auto"), T("theme_light"), T("theme_dark")},
		func(s string) {
			switch s {
			case T("theme_light"):
				fyne.CurrentApp().Settings().SetTheme(theme.LightTheme())
			case T("theme_dark"):
				fyne.CurrentApp().Settings().SetTheme(theme.DarkTheme())
			default: // T("theme_auto")
				fyne.CurrentApp().Settings().SetTheme(nil) // sistem/varsayılan
			}
		},
	)
	themeSel.SetSelected(T("theme_auto"))

	tabs := container.NewAppTabs(
		container.NewTabItem(T("tab_wallet"), makeWalletTab(w, myAddrEntry, fromEntry)),
		container.NewTabItem(T("tab_mine"), makeMinerTab(w, myAddrEntry.Text)),
		container.NewTabItem(T("tab_web"), makeWebWalletTab(w)),
	)

	header := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle(T("title"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(themeSel, makeLangSelect(w)),
	)

	w.SetContent(container.NewBorder(header, nil, nil, nil, tabs))
}

func main() {
	runtime.LockOSThread()
a := app.NewWithID("quantumcoin.gui")

// son dil (opsiyonel)
if lang := a.Preferences().String("lang"); lang != "" {
    curLang = lang
}

// son tema
applyTheme(a.Preferences().StringWithFallback("theme_value", "auto"))

w := a.NewWindow(T("title") + " — " + appVersion)

	w.Resize(fyne.NewSize(1000, 700))
	makeLangSelect(w) // default TR selected
	buildUI(w)
	w.Show()
	a.Run()
}

/*** --- BEGIN: universal balance fetch (legacy-compatible) --- ***/

// Eski kullanım için tek sayı döndüren sarmalayıcı.
// Mevcut tolerant fonksiyonu kullanır ve (confirmed+pending) toplamını verir.
func getBalanceUniversalInt(addr string) (int64, error) {
	confirmed, pending, ok := getBalanceUniversal(addr)
	if !ok {
		return 0, fmt.Errorf("balance not found")
	}
	total := confirmed + pending
	return int64(total), nil
}

// *Legacy* geri-dönüş uyumluluğu: Bazı bilinen uçları tek tek yoklar,
// JSON içinden "balance" anahtarını toleranslı biçimde çıkarır.
func getBalanceUniversalLegacy(addr string) (int64, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return 0, fmt.Errorf("address empty")
	}

	ports := []int{8082, 8081, 8080, 3001, 9090}
	patterns := []string{
		"/api/address/%s/balance",
		"/api/wallet/balance/%s",
		"/api/balance?addr=%s",
		"/api/wallet/balance?addr=%s",
		"/api/address/%s",
		"/address/%s/balance",
		"/balance/%s",
	}

	client := &http.Client{Timeout: 4 * time.Second}

	for _, p := range ports {
		base := fmt.Sprintf("http://127.0.0.1:%d", p)
		for _, pat := range patterns {
			var esc string
			if strings.Contains(pat, "?addr=%s") {
				esc = url.QueryEscape(addr)
			} else {
				esc = url.PathEscape(addr)
			}
			u := fmt.Sprintf(base+pat, esc)

			resp, err := client.Get(u)
			if err != nil {
				continue
			}
			b, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				continue
			}
			if val, ok := parseBalanceJSONLegacy(b); ok {
				return val, nil
			}
		}
	}
	return 0, fmt.Errorf("balance not found")
}

func parseBalanceJSONLegacy(b []byte) (int64, bool) {
	// Düz sayı (örn: 2872)
	var n float64
	if err := json.Unmarshal(b, &n); err == nil {
		return int64(n), true
	}

	// Düz string sayı (örn: "2872")
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return int64(f), true
		}
	}

	// Genel JSON, iç içe "balance" ara
	var any map[string]interface{}
	if err := json.Unmarshal(b, &any); err == nil {
		if v, ok := walkBalanceLegacy(any); ok {
			return v, true
		}
	}
	return 0, false
}

func walkBalanceLegacy(m map[string]interface{}) (int64, bool) {
	if v, ok := m["balance"]; ok {
		switch t := v.(type) {
		case float64:
			return int64(t), true
		case json.Number:
			if f, err := t.Float64(); err == nil {
				return int64(f), true
			}
		case string:
			if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
				return int64(f), true
			}
		}
	}
	for _, v := range m {
		if mm, ok := v.(map[string]interface{}); ok {
			if out, ok := walkBalanceLegacy(mm); ok {
				return out, true
			}
		}
	}
	return 0, false
}

/*** --- END: universal balance fetch (legacy-compatible) --- ***/

// --- keep legacy wrappers referenced to silence linters ---
var (
	_ = getBalanceUniversalInt
	_ = getBalanceUniversalLegacy
)

// --- HTTP no-cache helper: her çağrıda benzersiz query ekle
func noCacheURL(u string) string {
	sep := "?"
	if strings.Contains(u, "?") {
		sep = "&"
	}
	return u + sep + "_ts=" + strconv.FormatInt(time.Now().UnixNano(), 10)
}
