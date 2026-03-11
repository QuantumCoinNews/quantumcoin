
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/csv"
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

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRegexp.ReplaceAllString(s, "")
}

// ----------------------------------------------------------
// Windows'ta açılışta kendi konsolumuzu bırak (GUI siyah pencere göstermesin)
// --- Windows helper'ları ---
func hideConsoleOnStartup() {
	// sadece Windows
	if runtime.GOOS != "windows" {
		return
	}
	// mevcut konsolu bırak
	mod := syscall.NewLazyDLL("kernel32.dll")
	proc := mod.NewProc("FreeConsole")
	_, _, _ = proc.Call()
}

// İçeriden ortak "STOP'a indir" helper'ı (tek yerden yönetelim)
func markMinerStoppedFromWatcher() {
	// state
	minerRunningState = false

	// tail kapat
	stopMinerTail()

	// pid temizle
	clearMinerPID()

	// UI state indir
	// ÖNEMLİ: onMinerStateUpdate zaten ui() içinde çalışıyor.
	// Burada tekrar ui() sarmalama yapmayalım (double-ui bazı durumlarda güncellemeyi kaçırıyor).
	if onMinerStateUpdate != nil {
		onMinerStateUpdate(false)
	}

	// balance refresh
	if walletRefreshHook != nil {
		walletRefreshHook()
	}
}

// GUI'den miner'ı TEK bir görünen CMD penceresinde başlatır.
// CMD kapatılırsa (X) cmd.Wait() döner ve poller/watchdog STOP'a indirir.
func startMinerVisible() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	releaseDir := filepath.Dir(exePath)
	bat := filepath.Join(releaseDir, "run_miner.cmd")

	if _, err := os.Stat(bat); err != nil {
		return fmt.Errorf("run_miner.cmd not found: %w", err)
	}

	// Tek cmd.exe PID: detach yok
	cmdLine := fmt.Sprintf(`pushd "%s" & title QuantumCoin Miner & call "%s"`, releaseDir, bat)
	cmd := exec.Command("cmd.exe", "/K", cmdLine)
	cmd.Dir = releaseDir

	// Kritik: miner'ın data klasörü + pid dosyası + chain_data.dat hep releaseDir olsun.
	// (Aksi halde farklı QC_NODE_DIR ile aynı yükseklikte döngü görürsün.)
	api := strings.TrimSpace(os.Getenv("QC_API_BASE"))
	if api == "" {
		api = apiBase() // sende var; port tespit eder
	}
	cmd.Env = append(os.Environ(),
		"QC_NODE_DIR="+releaseDir,
		"QC_MINED_PATH="+filepath.Join(releaseDir, "mined_balance.json"),
		"QC_API_BASE="+api,
		"QC_LANG=en", // loglar İngilizce kalsın
	)

	// Görünür yeni console
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000010, // CREATE_NEW_CONSOLE
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// cmd.exe PID kaydet (GUI watchdog bununla çalışacak)
	if cmd.Process != nil {
		writeMinerPID(cmd.Process.Pid)
	}

	// Miner çalışıyor kabul et (poller'ın "hemen çıkmasını" engeller)
	minerRunningState = true

	// X kapanışını garanti yakalamak için poller'ı başlat
	startMinerStatePoller()

	// X ile kapanırsa -> Wait döner -> STOP
	go func() {
		_ = cmd.Wait()
		markMinerStoppedFromWatcher()
	}()

	return nil
}

// GUI'den miner'ı durdur (görünür CMD + miner süreçleri)
// NOT: quantumcoin.exe'yi (API dahil) öldürmeyiz. Sadece miner CMD ağacını kapatırız.
func stopMinerVisible() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	releaseDir := filepath.Dir(exePath)

	// run_miner.cmd döngüsüne "stop" de
	_ = os.WriteFile(filepath.Join(releaseDir, "miner_stop.flag"), []byte("stop"), 0644)

	if runtime.GOOS == "windows" {
		// 1) En temiz: bizim başlattığımız cmd.exe PID ağacını kapat
		if pid, err := readMinerPID(); err == nil && pid > 0 {
			// önce yumuşak dene
			c1 := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T")
			c1.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			_ = c1.Run()

			// çok kısa bekle, hâlâ yaşıyorsa force
			time.Sleep(350 * time.Millisecond)
			if isProcessAlive(pid) {
				c2 := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F")
				c2.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				_ = c2.Run()
			}
		} else {
			// 2) PID yoksa fallback: başlığa göre (wildcard)
			c := exec.Command("taskkill", "/F", "/FI", `WINDOWTITLE eq QuantumCoin Miner*`)
			c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			_ = c.Run()
		}
	} else {
		_ = exec.Command("pkill", "-f", "quantumcoin").Run()
	}

	// UI'ı da indir (butona basınca anında)
	markMinerStoppedFromWatcher()
	return nil
}

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

var appVersion = "wallet-gui v2.0 (AI)"

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

	// Miner durum poller'ı sadece 1 kere başlat
	minerStatePollerMu  sync.Mutex
	minerStatePollerRun bool
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

func startMinerStatePoller() {
	minerStatePollerMu.Lock()
	if minerStatePollerRun {
		minerStatePollerMu.Unlock()
		return
	}
	minerStatePollerRun = true
	minerStatePollerMu.Unlock()

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		misses := 0

		for range ticker.C {
			// miner çalışmıyorsa: izle ama aksiyon alma
			if !minerRunningState {
				misses = 0
				continue
			}

			// Önce PID ile kontrol et (en sağlam)
			pid, _ := readMinerPID()
			alive := false
			if pid > 0 {
				alive = isProcessAlive(pid)
			} else {
				// PID yoksa fallback: pencere başlığı
				alive = minerWindowAlive()
			}

			if alive {
				misses = 0
				continue
			}

			misses++
			if misses >= 2 {
				// STOP’a çek
				markMinerStoppedFromWatcher()
				misses = 0
			}
		}
	}()
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
		fyne.CurrentApp().Settings().SetTheme(theme.DarkTheme())
	case s == "light" || s == "aydınlık" || s == "claro" || s == "浅色" || s == "亮色" || s == "浅色模式":
		themeValue = "light"
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

func minedJSONPath() string {
	// küçük yardımcı: yineleneni ekleme
	appendUniq := func(list *[]string, p string) {
		if p == "" {
			return
		}
		cp := filepath.Clean(p)
		for _, s := range *list {
			if strings.EqualFold(filepath.Clean(s), cp) {
				return
			}
		}
		*list = append(*list, cp)
	}

	candidates := make([]string, 0, 32)

	// 0) Manuel override en üstte
	if v := strings.TrimSpace(os.Getenv("QC_MINED_PATH")); v != "" {
		appendUniq(&candidates, v)
	}

	// 1) QC_NODE_DIR verildiyse orası
	if v := strings.TrimSpace(os.Getenv("QC_NODE_DIR")); v != "" {
		appendUniq(&candidates, filepath.Join(v, "mined_balance.json"))
		appendUniq(&candidates, filepath.Join(v, "miner", "mined_balance.json"))
		appendUniq(&candidates, filepath.Join(v, "data", "mined_balance.json"))
	}

	// 2) nodeDir() (çoğu zaman gerçek node klasörü)
	if nd := nodeDir(); nd != "" {
		appendUniq(&candidates, filepath.Join(nd, "mined_balance.json"))
		appendUniq(&candidates, filepath.Join(nd, "miner", "mined_balance.json"))
		appendUniq(&candidates, filepath.Join(nd, "data", "mined_balance.json"))
	}

	// 3) Windows sabitleri (sizin listedeki gibi)
	if runtime.GOOS == "windows" {
		appendUniq(&candidates, `C:\QuantumMiner\mined_balance.json`)
		appendUniq(&candidates, `C:\QuantumMiner\miner\mined_balance.json`)
		appendUniq(&candidates, `C:\QuantumMiner\data\mined_balance.json`)
	}

	// 4) exeDir() (wallet-gui’nin durduğu yer)
	ed := exeDir()
	appendUniq(&candidates, filepath.Join(ed, "mined_balance.json"))
	appendUniq(&candidates, filepath.Join(ed, "miner", "mined_balance.json"))

	// 5) Çalışma dizini
	if wd, err := os.Getwd(); err == nil {
		appendUniq(&candidates, filepath.Join(wd, "mined_balance.json"))
		appendUniq(&candidates, filepath.Join(wd, "miner", "mined_balance.json"))
	}

	// 6) Alternatif isimler – aynı baz dizinlerde dene
	altNames := []string{"mined.json", "mining_balance.json", "rewards.json", "miner_balance.json"}
	baseDirs := []string{ed, nodeDir()}
	if v := strings.TrimSpace(os.Getenv("QC_NODE_DIR")); v != "" {
		baseDirs = append(baseDirs, v)
	}
	if runtime.GOOS == "windows" {
		baseDirs = append(baseDirs, `C:\QuantumMiner`, `C:\QuantumMiner\data`, `C:\QuantumMiner\miner`)
	}
	for _, bd := range baseDirs {
		if bd == "" {
			continue
		}
		for _, name := range altNames {
			appendUniq(&candidates, filepath.Join(bd, name))
		}
	}

	// 7) İlk mevcut dosyayı seç
	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}

	// 8) Fallback: nodeDir() varsa orası, yoksa exeDir()
	if nd := nodeDir(); nd != "" {
		return filepath.Join(nd, "mined_balance.json")
	}
	return filepath.Join(ed, "mined_balance.json")
}

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

// Adres + private key (hex) birlikte üretir (quantumcoin.exe newaddr-priv)
func genAddressPrivViaNode() (addr, privHex string, err error) {
	exe := findNodeExe()
	if _, e := os.Stat(exe); e != nil {
		return "", "", fmt.Errorf("quantumcoin executable not found: %w", e)
	}

	cmd := exec.Command(exe, "newaddr-priv")
	cmd.Dir = filepath.Dir(exe)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	if os.Getenv("QC_NODE_DIR") == "" {
		cmd.Env = append(os.Environ(), "QC_NODE_DIR="+filepath.Dir(exe))
	}

	out, e := cmd.CombinedOutput()
	if e != nil {
		return "", "", fmt.Errorf("newaddr-priv error: %w (out=%s)", e, string(out))
	}
	s := strings.TrimSpace(string(out))

	// ---- 1) Etiketli satırlar (Address / PrivateKey) ----
	for _, marker := range []string{"New Wallet Address:", "Wallet Address:", "New Address:", "Address:"} {
		if i := strings.Index(s, marker); i >= 0 {
			part := strings.TrimSpace(s[i+len(marker):])
			if j := strings.IndexByte(part, '\n'); j >= 0 {
				part = part[:j]
			}
			part = cleanBase58(part)
			if isLikelyBase58Address(part) {
				addr = part
				break
			}
		}
	}
	for _, marker := range []string{
		"PrivateKey (hex):", "Private Key (hex):", "PrivateKey:", "Private Key:", "Priv:", "Secret:",
	} {
		if i := strings.Index(s, marker); i >= 0 {
			part := strings.TrimSpace(s[i+len(marker):])
			if j := strings.IndexByte(part, '\n'); j >= 0 {
				part = part[:j]
			}
			part = strings.Trim(part, "\"' ")
			part = strings.TrimPrefix(part, "0x")
			privHex = part
			break
		}
	}

	// ---- 2) JSON olasılıkları ----
	if addr == "" || privHex == "" {
		var obj struct {
			Address       string `json:"address"`
			WalletAddress string `json:"wallet_address"`
			Addr          string `json:"addr"`

			Private       string `json:"private"`
			Priv          string `json:"priv"`
			PrivKey       string `json:"privkey"`
			PrivateKey    string `json:"private_key"`
			PrivateKeyHex string `json:"privateKeyHex"`
		}
		if json.Unmarshal([]byte(s), &obj) == nil {
			if addr == "" {
				for _, c := range []string{obj.Address, obj.WalletAddress, obj.Addr} {
					a := cleanBase58(strings.TrimSpace(c))
					if a != "" && isLikelyBase58Address(a) {
						addr = a
						break
					}
				}
			}
			if privHex == "" {
				for _, c := range []string{obj.Private, obj.Priv, obj.PrivKey, obj.PrivateKey, obj.PrivateKeyHex} {
					h := strings.TrimSpace(c)
					h = strings.Trim(h, "\"' ")
					h = strings.TrimPrefix(h, "0x")
					if h != "" {
						privHex = h
						break
					}
				}
			}
		}
	}

	// Generic map (anahtar adları farklıysa)
	if (addr == "" || privHex == "") && json.Valid([]byte(s)) {
		var any map[string]interface{}
		if json.Unmarshal([]byte(s), &any) == nil {
			// address
			if addr == "" {
				for _, k := range []string{"address", "wallet_address", "addr"} {
					if v, ok := any[k].(string); ok {
						a := cleanBase58(strings.TrimSpace(v))
						if a != "" && isLikelyBase58Address(a) {
							addr = a
							break
						}
					}
				}
			}
			// priv
			if privHex == "" {
				for _, k := range []string{"private", "priv", "privkey", "private_key", "privatekeyhex"} {
					if v, ok := any[k].(string); ok {
						h := strings.TrimSpace(v)
						h = strings.Trim(h, "\"' ")
						h = strings.TrimPrefix(h, "0x")
						if h != "" {
							privHex = h
							break
						}
					}
				}
			}
		}
	}

	// ---- 3) Regex fallbacks ----
	if addr == "" {
		reB58 := regexp.MustCompile(`[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]{26,50}`)
		if m := reB58.FindString(s); m != "" {
			m = cleanBase58(m)
			if isLikelyBase58Address(m) {
				addr = m
			}
		}
	}
	if privHex == "" {
		reHex := regexp.MustCompile(`(?i)(?:0x)?[0-9a-f]{32,128}`)
		if m := reHex.FindString(s); m != "" {
			privHex = strings.TrimPrefix(m, "0x")
		}
	}

	// ---- doğrula & dön ----
	if addr == "" || !isLikelyBase58Address(addr) || strings.TrimSpace(privHex) == "" {
		return "", "", fmt.Errorf("could not parse address/private key from newaddr-priv output")
	}
	return addr, privHex, nil
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
	backupBtn := widget.NewButton(T("backup_wallet"), func() {
		fs := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
			if err != nil || uc == nil {
				return
			}
			path := uc.URI().Path()
			uc.Close()

			pass := widget.NewEntry()
			pass.Password = true
			dialog.NewCustomConfirm(
				T("backup_password_title"),
				T("ok"),
				T("cancel"),
				container.NewVBox(widget.NewLabel(T("backup_password_prompt")), pass),
				func(ok bool) {
					if !ok {
						return
					}
					if strings.TrimSpace(pass.Text) == "" {
						dialog.ShowError(fmt.Errorf(T("password_empty_error")), w)
						return
					}
					if err := createWalletBackup(path, pass.Text); err != nil {
						dialog.ShowError(err, w)
						return
					}
					dialog.ShowInformation(T("ok"), T("backup_created"), w)
				},
				w,
			).Show()
		}, w)
		fs.SetFileName("qc_wallet_backup.qcbak")
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
	// önce ANSI kaçışlarını temizle
	cleaned := stripANSI(line)

	// panic/invalid base58 => kırmızı
	if rePanic.MatchString(cleaned) {
		rt.Segments = append(rt.Segments, &widget.TextSegment{
			Text: cleaned + "\n",
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

	// Varsayılan node dizini (legacy davranış)
	dir := nodeDir()

	// WINDOWS için: ÖNCE release klasöründeki quantumcoin.exe'yi tercih et
	// (wallet-gui.exe ile aynı klasör)
	if runtime.GOOS == "windows" {
		if exePath, err := os.Executable(); err == nil && exePath != "" {
			releaseDir := filepath.Dir(exePath)
			localNode := filepath.Join(releaseDir, "quantumcoin.exe")
			if st, err := os.Stat(localNode); err == nil && !st.IsDir() {
				dir = releaseDir
			}
		}
	}

	winExe := filepath.Join(dir, "quantumcoin.exe")
	if _, err := os.Stat(winExe); err != nil && runtime.GOOS == "windows" {
		return fmt.Errorf("quantumcoin.exe not found: %w", err)
	}

	// Tek-adres env’leri ve yapı (Windows dışı için de anlamlı)
	_ = os.Setenv("QC_NODE_DIR", dir)
	_ = os.Setenv("QC_COMMUNITY_ADDRESS", addr)
	_ = os.Setenv("QC_DEV_FUND_ADDRESS", addr)
	_ = os.Setenv("QC_PREMINE_ADDRESS", addr)
	ensureBonusStore(addr)

	// --- WINDOWS ---
	if runtime.GOOS == "windows" {
		// POZİSYONEL argüman: mine "ADDR"
		argLine := fmt.Sprintf(`%s "%s"`, minerSubcommand, addr)

		runBat := filepath.Join(dir, "run_miner.cmd")

		// Eğer klasörde zaten run_miner.cmd varsa onu EZME.
		// (Senin release klasörüne koyduğun sade dosya korunur.)
		if _, err := os.Stat(runBat); os.IsNotExist(err) {

			// >>> hazır değilse fallback script üret (eski davranış korunuyor)
			logPath := filepath.Join(dir, "miner_out.log")
			if _, err := os.Stat(logPath); os.IsNotExist(err) {
				// UTF-8 BOM
				_ = os.WriteFile(logPath, []byte{0xEF, 0xBB, 0xBF}, 0644)
			}

			bat := fmt.Sprintf(`@echo off
setlocal enableextensions enabledelayedexpansion
chcp 65001 >NUL
cd /d "%s"

REM --- Tek-adres env'leri ---
set "QC_NODE_DIR=%s"
set "QC_COMMUNITY_ADDRESS=%s"
set "QC_DEV_FUND_ADDRESS=%s"
set "QC_PREMINE_ADDRESS=%s"
set "QC_MINED_PATH=%%CD%%\mined_balance.json"
set "ADDR=%s"
set "LOG=miner_out.log"
set "QC_LANG=en"
set "LANG=en_US.UTF-8"
set "LANGUAGE=en"
set "LC_ALL=C"

REM --- Eski stop bayrağını temizle ---
del /q "miner_stop.flag" 2>NUL

REM --- Basit log rotasyonu (~10MB) ---
if exist "%%LOG%%" (
  for %%%%A in ("%%LOG%%") do if %%%%~zA GTR 10485760 (
    if exist "%%LOG%%.1" del /q "%%LOG%%.1" 2>NUL
    ren "%%LOG%%" "%%LOG%%.1"
  )
)

REM --- Bilgi başlığı ---
title QuantumCoin Miner
echo ===============================>>"%%LOG%%"
echo Mining to: %%ADDR%%>>"%%LOG%%"
echo Folder    : %%CD%%>>"%%LOG%%"
echo Started   : %%date%% %%time%%>>"%%LOG%%"
echo ===============================>>"%%LOG%%"

REM --- PowerShell var mı? (tee için) ---
where powershell >NUL 2>&1 && set "HAS_PS=1"
if not defined HAS_PS set "HAS_PS=0"

:loop
if exist "miner_stop.flag" goto :done

if "%%HAS_PS%%"=="1" (
  powershell -NoLogo -ExecutionPolicy Bypass -Command ^
    "& { & '.\\quantumcoin.exe' %s 2>&1 | Tee-Object -File '%%LOG%%' -Append }"
) else (
  ".\\quantumcoin.exe" %s >> "%%LOG%%" 2>&1
)

REM Kısa nefes; crash sonrası döngü devam eder
timeout /t 1 /nobreak >NUL
goto :loop

:done
echo Stopped by miner_stop.flag>>"%%LOG%%"
endlocal
`, dir, dir, addr, addr, addr, addr, argLine, argLine)

			// CRLF ile yaz
			batCRLF := strings.ReplaceAll(bat, "\n", "\r\n")
			if err := os.WriteFile(runBat, []byte(batCRLF), 0644); err != nil {
				return fmt.Errorf("could not write run_miner.cmd: %w", err)
			}
		}

		// API arka planda hazır değilse kaldır (mevcut davranışı bozmaz)
		_ = startAPIBackground()

		// ÖNEMLİ:
		// "cmd /c start ..." DETACH ettiği için GUI X kapanışını göremiyor.
		// Bunun yerine görünür cmd.exe'yi direkt başlatıyoruz ve PID/Wait ile takip ediyoruz.
		cmdLine := fmt.Sprintf(`title QuantumCoin Miner & cd /d "%s" & call "%s"`, dir, runBat)
		cmd := exec.Command("cmd.exe", "/K", cmdLine)
		cmd.Dir = dir

		// run_miner.cmd için ENV'leri garanti et
		env := append(os.Environ(),
			"ADDR="+addr,
			"QC_NODE_DIR="+dir,
			"QC_COMMUNITY_ADDRESS="+addr,
			"QC_DEV_FUND_ADDRESS="+addr,
			"QC_PREMINE_ADDRESS="+addr,
			"QC_MINED_PATH="+filepath.Join(dir, "mined_balance.json"),
		)
		// apiBase() varsa kullan; yoksa fallback bırak
		api := strings.TrimSpace(apiBase())
		if api == "" {
			api = "http://127.0.0.1:8081"
		}
		env = append(env, "QC_API_BASE="+api)
		cmd.Env = env

		// GUI console-less olsa bile CMD görünür açılsın
		// (0x00000010 = CREATE_NEW_CONSOLE)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: 0x00000010,
		}

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("could not start visible CMD: %w", err)
		}

		// PID kaydet (watchdog/poller için)
		if cmd.Process != nil {
			writeMinerPID(cmd.Process.Pid)
		}

		// GUI tarafına "çalışıyor" bilgisini yansıt
		minerLogPath = filepath.Join(dir, "miner_out.log")
		minerRunningState = true
		if onMinerStateUpdate != nil {
			onMinerStateUpdate(true)
		}

		// Poller/watchdog'ı BİR kez başlat
		startMinerStatePoller()
		go windowsMinerWatchdog()

		// CMD kullanıcı X ile kapatırsa -> GUI otomatik STOPPED
		go func() {
			_ = cmd.Wait()

			minerRunningState = false
			stopMinerTail()
			clearMinerPID()

			if onMinerStateUpdate != nil {
				onMinerStateUpdate(false)
			}

			if walletRefreshHook != nil {
				walletRefreshHook()
			}
		}()

		// Kısa gecikmeli refresh (orijinal davranış)
		if walletRefreshHook != nil {
			go func() {
				time.Sleep(3 * time.Second)
				walletRefreshHook()
			}()
		}

		return nil
	}

	// --- macOS / LINUX ---
	exe := findNodeExe()
	if exe == "" {
		// Emniyetli geri dönüş
		exe = filepath.Join(dir, "quantumcoin")
	}
	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("quantumcoin executable not found: %w", err)
	}
	dir = filepath.Dir(exe)

	args := []string{minerSubcommand, addr} // pozisyonel
	minerLogPath = filepath.Join(dir, "miner_out.log")
	lf, _ := os.OpenFile(minerLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	cmd := exec.Command(exe, args...)

	// env ve çalışma dizini
	env := append(os.Environ(),
		"QC_LANG=en", // loglar İngilizce
		"QC_NODE_DIR="+dir,
		"QC_COMMUNITY_ADDRESS="+addr,
		"QC_DEV_FUND_ADDRESS="+addr,
		"QC_PREMINE_ADDRESS="+addr,
		"QC_MINED_PATH="+filepath.Join(dir, "mined_balance.json"),
	)
	cmd.Env = env
	cmd.Dir = dir

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
	// Poller + watchdog (CMD X kapanışını garanti yakalar)
	startMinerStatePoller()
	go windowsMinerWatchdog()
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
	minerLogPath = filepath.Join(nodeDir(), "miner_out.log")

	// Butonlar (onMinerStateUpdate içinde kullanacağız)
	var startBtn, stopBtn *widget.Button

	// UI state callback (TEK kez)
	onMinerStateUpdate = func(running bool) {
		ui(func() {
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
	}

	// === START ===
	startBtn = widget.NewButton(T("miner_start"), func() {
		addr, err := ensureRewardAddress(addrEntry.Text)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		addrEntry.SetText(addr)

		// run_miner.cmd ENV önceliği: ADDR
		_ = os.Setenv("ADDR", addr)

		// Miner başlat (görünür CMD + PID yazılır)
		if err := startMinerVisible(); err != nil {
			dialog.ShowError(err, w)
			return
		}

		// API (varsa/kapalıysa) arka planda
		_ = startAPIBackground()

		// log tail
		minerLogPath = filepath.Join(nodeDir(), "miner_out.log")
		startMinerTailInto(logView)

		// UI running
		minerRunningState = true
		if onMinerStateUpdate != nil {
			onMinerStateUpdate(true)
		}

		// Watchdog: CMD X kapanınca otomatik stop
		go windowsMinerWatchdog()

		// Ek garanti: poller varsa (idempotent)
		startMinerStatePoller()
	})

	// === STOP ===
	stopBtn = widget.NewButton(T("miner_stop"), func() {
		writeMinerStopFlag()
		_ = stopMinerVisible() // bu zaten UI’ı indirir
		dialog.ShowInformation(T("ok"), T("miner_stopped"), w)
	})

	// İlk açılışta miner kapalı
	stopBtn.Disable()

	// log klasörü
	openLogBtn := widget.NewButton(T("open_log_folder"), func() {
		dir := nodeDir()
		if runtime.GOOS == "windows" {
			_ = exec.Command("explorer.exe", dir).Start()
		} else {
			_ = exec.Command("xdg-open", dir).Start()
		}
	})

	// manuel cmd
	openManualCmdBtn := widget.NewButton(T("open_cmd_manual"), func() {
		dir := nodeDir()
		runBat := filepath.Join(dir, "run_miner.cmd")
		if _, err := os.Stat(runBat); err == nil && runtime.GOOS == "windows" {
			_ = exec.Command("cmd", "/c", "start", "QuantumCoin Miner", runBat).Start()
		}
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
		ensureBonusStore(v)
		return v, nil
	}

	// 3) Node'dan yeni adres + priv üret
	addr, privHex, err := genAddressPrivViaNode()
	if err != nil {
		return "", err
	}
	if !isLikelyBase58Address(addr) {
		return "", fmt.Errorf("generated address looks invalid")
	}

	// 4) Kalıcı kaydet
	_ = saveText(walletAddressPath(), addr)
	_ = saveText(minerAddressPath(), addr)

	// priv varsa kaydet
	if strings.TrimSpace(privHex) != "" {
		_ = saveText(walletPrivPath(), privHex)
	}

	ensureBonusStore(addr)
	return addr, nil
}
func writeMinerStopFlag() {
	f := filepath.Join(nodeDir(), "miner_stop.flag")
	_ = os.WriteFile(f, []byte("stop"), 0644)
}


// --- Windows: PID'nin hangi process olduğunu hızlıca anlamak için ---
func firstCSVField(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	// "cmd.exe","1234",...
	if strings.HasPrefix(line, `"`) {
		if j := strings.Index(line[1:], `"`); j >= 0 {
			return line[1 : 1+j]
		}
	}
	if j := strings.Index(line, ","); j >= 0 {
		return strings.Trim(line[:j], `"`)
	}
	return strings.Trim(line, `"`)
}

func processImageName(pid int) string {
	if pid <= 0 || runtime.GOOS != "windows" {
		return ""
	}
	out, err := exec.Command(
		"tasklist", "/fo", "csv", "/nh",
		"/fi", fmt.Sprintf("PID eq %d", pid),
	).Output()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return ""
	}
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	return strings.ToLower(firstCSVField(line))
}

func isCmdPID(pid int) bool {
	return processImageName(pid) == "cmd.exe"
}

// PowerShell ile sadece "quantumcoin.exe mine" süreçlerini bulur ve kapatır.
// API'yi öldürmez (api komutu farklı).
func killQuantumcoinMineOnly() {
	if runtime.GOOS != "windows" {
		return
	}

	psCmd := `Get-CimInstance Win32_Process -Filter "Name='quantumcoin.exe'" | ` +
		`Where-Object { $_.CommandLine -match '(\s|^)(mine)(\s|$)' } | ` +
		`Select-Object -ExpandProperty ProcessId`

	out, err := exec.Command("powershell", "-NoProfile", "-Command", psCmd).Output()
	if err != nil {
		out2, err2 := exec.Command("pwsh", "-NoProfile", "-Command", psCmd).Output()
		if err2 != nil {
			return
		}
		out = out2
	}

	for _, ln := range strings.Split(string(out), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		pid, e := strconv.Atoi(ln)
		if e != nil || pid <= 0 {
			continue
		}
		c := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F")
		c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = c.Run()
	}
}

// tasklist /v /fo csv ile PID'ye ait Window Title'ı okumaya çalışır.
func cmdWindowTitleByPID(pid int) (title string, ok bool) {
	if runtime.GOOS != "windows" || pid <= 0 {
		return "", false
	}

	out, err := exec.Command(
		"tasklist", "/v", "/fo", "csv", "/nh",
		"/fi", fmt.Sprintf("PID eq %d", pid),
	).Output()
	if err != nil {
		return "", false
	}

	s := strings.TrimSpace(string(out))
	if s == "" {
		return "", false
	}

	line := strings.Split(s, "\n")[0]
	low := strings.ToLower(strings.TrimSpace(line))

	if strings.HasPrefix(low, "info:") || strings.Contains(low, "no tasks are running") {
		return "", false
	}

	r := csv.NewReader(strings.NewReader(line))
	r.FieldsPerRecord = -1
	rec, err := r.Read()
	if err != nil || len(rec) < 2 {
		return "", true
	}

	if len(rec) >= 9 {
		return rec[8], true
	}
	return rec[len(rec)-1], true
}

// tasklist'ten "QuantumCoin Miner" title'lı cmd.exe PID'sini bulur.
func findMinerCmdPIDByTitle() int {
	if runtime.GOOS != "windows" {
		return 0
	}

	out, err := exec.Command(
		"tasklist", "/v", "/fo", "csv", "/nh",
		"/fi", "IMAGENAME eq cmd.exe",
	).Output()
	if err != nil {
		return 0
	}

	r := csv.NewReader(strings.NewReader(string(out)))
	r.FieldsPerRecord = -1

	for {
		rec, err := r.Read()
		if err != nil {
			break
		}
		// Image Name, PID, ..., Window Title
		if len(rec) < 2 {
			continue
		}
		img := strings.ToLower(strings.TrimSpace(rec[0]))
		if img != "cmd.exe" {
			continue
		}
		pidStr := strings.TrimSpace(rec[1])
		title := ""
		if len(rec) >= 9 {
			title = rec[8]
		} else {
			title = rec[len(rec)-1]
		}
		if strings.Contains(strings.ToLower(title), "quantumcoin miner") {
			if pid, e := strconv.Atoi(pidStr); e == nil && pid > 0 {
				return pid
			}
		}
	}
	return 0
}

// Miner penceresi (bizim CMD) yaşıyor mu?
func minerWindowAlive() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	pid, _ := readMinerPID()

	// PID yanlış yazıldıysa (cmd.exe değilse) title'dan doğru PID'yi bul ve düzelt
	if pid <= 0 || !isCmdPID(pid) {
		if p2 := findMinerCmdPIDByTitle(); p2 > 0 {
			pid = p2
			writeMinerPID(pid)
		}
	}

	if pid > 0 {
		if !isProcessAlive(pid) {
			return false
		}
		if title, ok := cmdWindowTitleByPID(pid); ok {
			return strings.Contains(strings.ToLower(title), "quantumcoin miner")
		}
		// title okunamadı ama cmd yaşıyor -> alive say
		return true
	}

	// PID yoksa en son fallback: tarama
	if p2 := findMinerCmdPIDByTitle(); p2 > 0 {
		writeMinerPID(p2)
		return true
	}
	return false
}

// ===== Windows miner watchdog =====
// CMD kapanırsa (X) UI'ı "Stopped" yapar + orphan "mine" süreçlerini temizler.
func windowsMinerWatchdog() {
	if runtime.GOOS != "windows" {
		return
	}

	misses := 0
	for {
		if !minerRunningState {
			return
		}

		time.Sleep(900 * time.Millisecond)

		if minerWindowAlive() {
			misses = 0
			continue
		}

		misses++
		if misses < 2 {
			continue
		}

		// Kullanıcı X ile kapatmışsa: miner_stop.flag yaz + mine süreçlerini temizle
		writeMinerStopFlag()
		killQuantumcoinMineOnly()

		markMinerStoppedFromWatcher()
		return
	}
}

		// 3) Süreç yaşıyor ama pencere kapandıysa (X):
		//    bazı durumlarda process orphan kalabiliyor. Bu durumda kesin stop uygula.
		title, ok := cmdWindowTitleByPID(pid)
		if !ok {
			// PID görünmüyor => pratikte öldü kabul
			misses++
			if misses >= 2 {
				markMinerStoppedFromWatcher()
				return
			}
			continue
		}

		t := strings.ToLower(strings.TrimSpace(title))
		// title boş / N/A => pencere yok => kullanıcı X ile kapattı (veya window gitti)
		if t == "" || t == "n/a" || t == "yok" {
			// Bu noktada: Stop button gibi davran -> stop flag yaz + PID tree kapat (gerekirse)
			_ = stopMinerVisible() // bu zaten markMinerStoppedFromWatcher() çağırıyor
			return
		}

		// normal: her şey canlı
		misses = 0
	}
}

func minerWindowAlive() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	// 1) En sağlam: bizim başlattığımız cmd PID
	if pid, err := readMinerPID(); err == nil && pid > 0 {
		if !isProcessAlive(pid) {
			return false
		}
		// PID var ama title okunamıyorsa yine de alive say (tasklist bazen title veremeyebilir)
		if title, ok := cmdWindowTitleByPID(pid); ok {
			return strings.Contains(strings.ToLower(title), "quantumcoin miner")
		}
		return true
	}

	// 2) PID yoksa fallback: tüm cmd.exe'lerde başlığa bak
	out, err := exec.Command(
		"tasklist", "/v", "/fo", "csv", "/nh",
		"/fi", "IMAGENAME eq cmd.exe",
	).Output()
	if err != nil {
		return false
	}
	s := strings.ToLower(string(out))
	return strings.Contains(s, "quantumcoin miner")
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
		_ = os.Setenv("QC_COMMUNITY_ADDRESS", addr)
		_ = os.Setenv("QC_DEV_FUND_ADDRESS", addr)
		_ = os.Setenv("QC_PREMINE_ADDRESS", addr)
		_ = os.Setenv("QC_NODE_DIR", nodeDir())
		_ = os.Setenv("QC_MINED_PATH", filepath.Join(nodeDir(), "mined_balance.json"))
		_ = os.Setenv("ADDR", addr) // ← EKLEDİĞİMİZ TEK SATIR

		// bonus/NFT store hazırlanması
		go ensureBonusStore(addr)
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
						b.WriteString("Height : " + height + "\n")
						hasMetric = true
					}
					if peers != "" {
						b.WriteString("Peers  : " + peers + "\n")
						hasMetric = true
					}
					if mempool != "" {
						b.WriteString("Mempool: " + mempool + " tx\n")
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
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	resp, err := client.Do(req)
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
