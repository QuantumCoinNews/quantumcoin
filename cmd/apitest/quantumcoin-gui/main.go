//go:build windows

package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ---- ayarlar ----
const (
	defaultHTTPPort = "8081"
	openURL         = "http://localhost:%s/"
)

// exe ile aynı klasörde quantumcoin.exe bekliyoruz
func findBackend(exeDir string) (string, error) {
	candidates := []string{
		filepath.Join(exeDir, "quantumcoin.exe"),
		filepath.Join(exeDir, "..", "..", "quantumcoin.exe"), // projeden doğrudan çalıştırma ihtimali
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return filepath.Clean(p), nil
		}
	}
	return "", errors.New("quantumcoin.exe bulunamadı (GUI ile aynı klasöre koy)")
}

func startBackend(backendPath, httpPort string) error {
	cmd := exec.Command(backendPath, "api")
	// HTTP_PORT ortam değişkeni
	env := os.Environ()
	env = append(env, "HTTP_PORT="+httpPort)
	cmd.Env = env

	// Çalışma dizini executable’ın olduğu yer olsun
	cmd.Dir = filepath.Dir(backendPath)

	// Arka planda başlat (konsolsuz)
	return cmd.Start()
}

func waitHealth(httpPort string, timeout time.Duration) error {
	url := fmt.Sprintf("http://localhost:%s/api/health", httpPort)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) // #nosec G107 – local loopback
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return nil
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(300 * time.Millisecond)
	}
	return errors.New("API /api/health zaman aşımı")
}

func openBrowser(httpPort string) {
	target := fmt.Sprintf(openURL, httpPort)
	_ = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", target).Start()
}

func main() {
	// exe klasörü
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = defaultHTTPPort
	}

	backend, err := findBackend(exeDir)
	if err != nil {
		// son çare: mesaj kutusu
		_ = exec.Command("mshta", "javascript:alert('"+err.Error()+"');close()").Run()
		return
	}

	// Backend ayakta mı? pingle; değilse başlat
	if err := waitHealth(httpPort, 500*time.Millisecond); err != nil {
		_ = startBackend(backend, httpPort)
	}

	// Sağlık kontrolü (max 5 sn)
	_ = waitHealth(httpPort, 5*time.Second)

	// Tarayıcıda aç
	openBrowser(httpPort)

	// küçük bir gecikme bırakıp çık (backend ayrı proseste çalışıyor)
	time.Sleep(500 * time.Millisecond)
}
