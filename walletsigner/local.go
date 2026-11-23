// Package walletsigner bridges the native wallet to the local trusted signer.
// Keys travel only through a local child-process stdin pipe, never HTTP or argv.
package walletsigner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"quantumcoin/apiclient"
)

const maxEnvelope = 1 << 20

type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > maxEnvelope {
		return 0, errors.New("signer output limit")
	}
	return b.Buffer.Write(p)
}

func SignLocal(ctx context.Context, executable, from, to, amount, key string, burn bool) (json.RawMessage, error) {
	if key == "" || !filepath.IsAbs(executable) {
		return nil, errors.New("local signer and key required")
	}
	args := []string{"sign-send", from, to, amount, "--key-stdin"}
	if burn {
		args = []string{"sign-burn", from, amount, "--key-stdin"}
	}
	cmd := exec.CommandContext(ctx, executable, args...)
	hideSignerProcess(cmd)
	cmd.Dir = filepath.Dir(executable)
	cmd.Stdin = strings.NewReader(key + "\n")
	var output limitedBuffer
	cmd.Stdout = &output
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, errors.New("local signing process failed")
	}
	for _, line := range strings.Split(output.String(), "\n") {
		if !strings.HasPrefix(line, "SIGNED_TRANSACTION_V1=") {
			continue
		}
		raw := []byte(strings.TrimSpace(strings.TrimPrefix(line, "SIGNED_TRANSACTION_V1=")))
		if bytes.Contains(raw, []byte(key)) || bytes.Contains(raw, []byte("priv_hex")) {
			return nil, errors.New("signer exposed secret material")
		}
		var envelope struct {
			Transaction json.RawMessage `json:"transaction"`
		}
		d := json.NewDecoder(bytes.NewReader(raw))
		d.DisallowUnknownFields()
		if d.Decode(&envelope) != nil || len(envelope.Transaction) == 0 || string(envelope.Transaction) == "null" {
			return nil, errors.New("invalid signer response")
		}
		if _, err := SignedTransactionIDV1(raw); err != nil {
			return nil, err
		}
		return json.RawMessage(raw), nil
	}
	return nil, errors.New("local signer did not produce a transaction")
}

func Submit(ctx context.Context, endpoint, token string, envelope json.RawMessage) ([]byte, int, error) {
	if _, err := apiclient.ValidateNodeURL(endpoint); err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(envelope))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-QC-API-KEY", token)
	client := apiclient.NewNodeClient(0)
	defer client.CloseIdleConnections()
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, errors.New("node request failed")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxEnvelope+1))
	if err != nil || len(raw) > maxEnvelope {
		return nil, resp.StatusCode, errors.New("node response limit")
	}
	return raw, resp.StatusCode, nil
}
