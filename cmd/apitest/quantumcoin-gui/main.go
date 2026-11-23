//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"quantumcoin/apiclient"
	"strings"
	"syscall"
	"time"
)

// ---- ayarlar ----
const (
	// QC_GUI_RUNTIME_PORT_POLICY_V1
	openURL = "http://localhost:%s/"
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

func startBackend(
	backendPath,
	httpPort string,
) (*exec.Cmd, error) {
	cmd := exec.Command(
		backendPath,
		"api",
	)

	cmd.Env = append(
		os.Environ(),
		"QC_HTTP_PORT=:"+httpPort,
		"QC_API_LISTEN=127.0.0.1:"+httpPort,
	)

	cmd.Dir = filepath.Dir(backendPath)

	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}

func stopStartedBackend(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}

func releaseStartedBackend(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Release()
}

func ensureBackend(bin, port string) error {
	canonical, err := apiclient.NormalizeAPIPortV1(port)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return apiclient.EnsureLocalBackendV1(ctx, canonical, os.Getenv("QC_API_TOKEN"), func() (apiclient.BackendHandleV1, error) {
		if bin == "" {
			executable, err := os.Executable()
			if err != nil {
				return apiclient.BackendHandleV1{}, err
			}
			bin, err = findBackend(filepath.Dir(executable))
			if err != nil {
				return apiclient.BackendHandleV1{}, err
			}
		}
		cmd, err := startBackend(bin, canonical)
		if err != nil {
			return apiclient.BackendHandleV1{}, err
		}
		return apiclient.BackendHandleV1{Stop: func() { stopStartedBackend(cmd) }, Release: func() error { return releaseStartedBackend(cmd) }}, nil
	})
}

func showGUIError(err error) {
	if err == nil {
		return
	}

	msg := strings.ReplaceAll(
		err.Error(),
		`"`,
		"'",
	)

	_ = exec.Command(
		"mshta",
		`javascript:alert("`+msg+`");close()`,
	).Run()
}
func openBrowser(httpPort string) {
	target := fmt.Sprintf(openURL, httpPort)
	_ = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", target).Start()
}

func main() {
	httpPort, err := apiclient.ConfiguredAPIPortV1()
	if err != nil {
		showGUIError(err)
		return
	}

	if err := ensureBackend(
		"",
		httpPort,
	); err != nil {
		showGUIError(err)
		return
	}

	openBrowser(httpPort)
	time.Sleep(500 * time.Millisecond)
}
