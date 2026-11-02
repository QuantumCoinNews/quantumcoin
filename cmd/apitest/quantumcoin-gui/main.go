//go:build windows

package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ---- ayarlar ----
const (
	// QC_GUI_RUNTIME_PORT_POLICY_V1
	defaultHTTPPort = "8082"
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

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}

func backendStartLockPath(httpPort string) string {
	safePort := strings.NewReplacer(
		":", "_",
		"/", "_",
		"\\", "_",
	).Replace(httpPort)

	return filepath.Join(
		os.TempDir(),
		"quantumcoin-api-"+safePort+".start.lock",
	)
}

func releaseBackendStartLock(
	lock *os.File,
	path string,
) {
	if lock != nil {
		_ = lock.Close()
	}

	if path != "" {
		_ = os.Remove(path)
	}
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

	if err := cmd.Process.Release(); err != nil {
		stopStartedBackend(cmd)

		return fmt.Errorf(
			"release healthy API backend: %w",
			err,
		)
	}

	return nil
}

func ensureBackend(
	backendPath,
	httpPort string,
) error {
	if err := waitHealth(
		httpPort,
		500*time.Millisecond,
	); err == nil {
		return nil
	}

	lockPath := backendStartLockPath(httpPort)
	deadline := time.Now().Add(20 * time.Second)

	for {
		lock, err := os.OpenFile(
			lockPath,
			os.O_CREATE|os.O_EXCL|os.O_WRONLY,
			0600,
		)

		if err == nil {
			_, _ = fmt.Fprintf(
				lock,
				"%d\n",
				os.Getpid(),
			)

			if err := waitHealth(
				httpPort,
				500*time.Millisecond,
			); err == nil {
				releaseBackendStartLock(
					lock,
					lockPath,
				)
				return nil
			}

			cmd, startErr :=
				startBackend(
					backendPath,
					httpPort,
				)

			if startErr != nil {
				releaseBackendStartLock(
					lock,
					lockPath,
				)
				return startErr
			}

			if err := waitHealth(httpPort, 5*time.Second); err != nil {
				stopStartedBackend(cmd)

				releaseBackendStartLock(
					lock,
					lockPath,
				)

				return fmt.Errorf(
					"started API backend did not become healthy: %w",
					err,
				)
			}

			if err := releaseStartedBackend(cmd); err != nil {
				releaseBackendStartLock(
					lock,
					lockPath,
				)
				return err
			}

			releaseBackendStartLock(
				lock,
				lockPath,
			)

			return nil
		}

		if !os.IsExist(err) {
			return fmt.Errorf(
				"acquire API startup lock: %w",
				err,
			)
		}

		if err := waitHealth(
			httpPort,
			500*time.Millisecond,
		); err == nil {
			return nil
		}

		if st, statErr := os.Stat(lockPath); statErr == nil &&
			time.Since(st.ModTime()) > 15*time.Second {
			_ = os.Remove(lockPath)
			continue
		}

		if time.Now().After(deadline) {
			return errors.New(
				"API startup lock timeout",
			)
		}

		time.Sleep(100 * time.Millisecond)
	}
}
func waitHealth(httpPort string, timeout time.Duration) error {
	url := fmt.Sprintf(
		"http://127.0.0.1:%s/api/health",
		httpPort,
	)

	deadline := time.Now().Add(timeout)
	client := &http.Client{
		Timeout: 500 * time.Millisecond,
	}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url) // #nosec G107 -- fixed loopback target
		if err == nil {
			status := resp.StatusCode
			_ = resp.Body.Close()

			if status == http.StatusOK {
				return nil
			}
		}

		time.Sleep(300 * time.Millisecond)
	}

	return errors.New("API /api/health zaman aşımı")
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
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)

	httpPort := strings.TrimPrefix(
		strings.TrimSpace(os.Getenv("QC_HTTP_PORT")),
		":",
	)
	if httpPort == "" {
		httpPort = defaultHTTPPort
	}

	backend, err := findBackend(exeDir)
	if err != nil {
		showGUIError(err)
		return
	}

	if err := ensureBackend(
		backend,
		httpPort,
	); err != nil {
		showGUIError(err)
		return
	}

	openBrowser(httpPort)
	time.Sleep(500 * time.Millisecond)
}
