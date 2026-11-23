package apiclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var errStartupLeaseBusyV1 = errors.New("API startup lease is held")
var ErrBackendPortBusyV1 = errors.New("API port is occupied by an unready or incompatible service")

// ConfiguredAPIPortV1 keeps GUI startup consistent with the node's canonical
// QC_HTTP_PORT setting. HTTP_PORT is a compatibility assertion, not an override.
func ConfiguredAPIPortV1() (string, error) {
	raw := strings.TrimSpace(os.Getenv("QC_HTTP_PORT"))
	if raw == "" {
		raw = "8082"
	}
	port, err := NormalizeAPIPortV1(raw)
	if err != nil {
		return "", err
	}
	if legacy := strings.TrimSpace(os.Getenv("HTTP_PORT")); legacy != "" {
		compatible, err := NormalizeAPIPortV1(legacy)
		if err != nil || compatible != port {
			return "", errors.New("HTTP_PORT conflicts with canonical QC_HTTP_PORT")
		}
	}
	return port, nil
}

// NormalizeAPIPortV1 gives every GUI the same port and startup-lease key.
func NormalizeAPIPortV1(value string) (string, error) {
	value = strings.TrimPrefix(strings.TrimSpace(value), ":")
	for _, r := range value {
		if r < '0' || r > '9' {
			return "", errors.New("invalid API port")
		}
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 || n > 65535 {
		return "", errors.New("API port must be between 1 and 65535")
	}
	return strconv.Itoa(n), nil
}

// StartupLeaseV1 is an OS-owned lock. Its file is never removed by age; the OS
// releases ownership on process exit, preventing live-lock stealing and ABA.
type StartupLeaseV1 struct {
	once   sync.Once
	file   *os.File
	unlock func() error
	err    error
}

func (l *StartupLeaseV1) Release() error {
	if l == nil {
		return nil
	}
	l.once.Do(func() {
		l.err = l.unlock()
		if err := l.file.Close(); l.err == nil {
			l.err = err
		}
	})
	return l.err
}

func AcquireStartupLeaseV1(ctx context.Context, port string) (*StartupLeaseV1, error) {
	return acquireStartupLeaseAtV1(ctx, os.TempDir(), port)
}

func acquireStartupLeaseAtV1(ctx context.Context, dir, port string) (*StartupLeaseV1, error) {
	if ctx == nil {
		return nil, errors.New("startup context required")
	}
	port, err := NormalizeAPIPortV1(port)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "quantumcoin-api-"+port+".start.guard"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	for {
		if err := ctx.Err(); err != nil {
			_ = f.Close()
			return nil, err
		}
		unlock, err := tryStartupLeaseV1(f)
		if err == nil {
			return &StartupLeaseV1{file: f, unlock: unlock}, nil
		}
		if !errors.Is(err, errStartupLeaseBusyV1) {
			_ = f.Close()
			return nil, err
		}
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// BackendHandleV1 contains operations on only the process started by this call.
type BackendHandleV1 struct {
	Stop    func()
	Release func() error
}

// EnsureLocalBackendV1 shares locked availability checks and owned-process cleanup
// across GUI entry points. Existing listeners are never killed or replaced.
func EnsureLocalBackendV1(ctx context.Context, port, token string, start func() (BackendHandleV1, error)) error {
	if ctx == nil {
		return errors.New("startup context required")
	}
	port, err := NormalizeAPIPortV1(port)
	if err != nil {
		return err
	}
	endpoint := "http://127.0.0.1:" + port + CanonicalHealthPathV1
	probe := func() (Health, error) {
		probeCtx, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
		defer cancel()
		return ProbeHealth(probeCtx, endpoint, token)
	}
	check := func() (bool, error) {
		status, err := probe()
		if err == nil && status.Reachable {
			if !status.Authenticated {
				return false, ErrHealthUnauthorizedV1
			}
			return true, nil
		}
		if errors.Is(err, ErrHealthContractV1) {
			return false, err
		}
		if err := ctx.Err(); err != nil {
			return false, err
		}
		connection, dialErr := (&net.Dialer{Timeout: 200 * time.Millisecond}).DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", port))
		if dialErr == nil {
			_ = connection.Close()
			return false, ErrBackendPortBusyV1
		}
		if !isConnectionRefusedV1(dialErr) {
			return false, fmt.Errorf("cannot establish API port availability: %w", dialErr)
		}
		return false, nil
	}
	lease, err := AcquireStartupLeaseV1(ctx, port)
	if err != nil {
		return err
	}
	defer lease.Release()
	if ready, err := check(); ready || err != nil {
		return err
	}
	if start == nil {
		return errors.New("backend starter required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	handle, err := start()
	if err != nil {
		return err
	}
	if handle.Stop == nil || handle.Release == nil {
		if handle.Stop != nil {
			handle.Stop()
		}
		return errors.New("backend process ownership callbacks required")
	}
	healthy := false
	defer func() {
		if !healthy {
			handle.Stop()
		}
	}()
	for {
		status, err := probe()
		if err == nil && status.Reachable {
			if !status.Authenticated {
				return ErrHealthUnauthorizedV1
			}
			if err := handle.Release(); err != nil {
				return fmt.Errorf("release healthy backend: %w", err)
			}
			healthy = true
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("API health startup timeout: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}
