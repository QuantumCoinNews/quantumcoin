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

const defaultHTTPPort = "8081"

func findBackend(exeDir string) (string, error) {
	cands := []string{
		filepath.Join(exeDir, "quantumcoin.exe"),
		filepath.Join(exeDir, "..", "..", "quantumcoin.exe"),
	}
	for _, p := range cands {
		if _, err := os.Stat(p); err == nil {
			return filepath.Clean(p), nil
		}
	}
	return "", errors.New("quantumcoin.exe bulunamadı (GUI ile aynı klasöre koy)")
}

func startBackend(bin, port string) error {
	cmd := exec.Command(bin, "api")
	cmd.Env = append(os.Environ(), "HTTP_PORT="+port)
	cmd.Dir = filepath.Dir(bin)
	return cmd.Start()
}

func waitOK(port string, d time.Duration) error {
	url := fmt.Sprintf("http://localhost:%s/api/health", port)
	dead := time.Now().Add(d)
	for time.Now().Before(dead) {
		if resp, err := http.Get(url); err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return errors.New("API /api/health timeout")
}

func openBrowser(port string) {
	_ = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", "http://localhost:"+port+"/").Start()
}

func main() {
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = defaultHTTPPort
	}
	bin, err := findBackend(exeDir)
	if err != nil {
		_ = exec.Command("mshta", "javascript:alert(\""+err.Error()+"\");close()").Run()
		return
	}
	if waitOK(port, 500*time.Millisecond) != nil {
		_ = startBackend(bin, port)
	}
	_ = waitOK(port, 5*time.Second)
	openBrowser(port)
	time.Sleep(500 * time.Millisecond)
}
