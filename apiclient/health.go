// Package apiclient defines the protected node health contract for launchers.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

const CanonicalHealthPathV1 = "/api/health"
const HealthServiceV1 = "QuantumCoin API"
const HealthProtocolV1 = "QC_NODE_HEALTH_V1"
const HealthAuthRealmV1 = `Bearer realm="QuantumCoin API"`

var ErrHealthContractV1 = errors.New("incompatible node health response")
var ErrHealthUnauthorizedV1 = errors.New("existing node requires a valid QC_API_TOKEN")

type Health struct {
	Reachable     bool
	Authenticated bool
	ReadOnly      bool
	ModeKnown     bool
}

func ProbeHealth(ctx context.Context, endpoint, token string) (Health, error) {
	u, err := ValidateNodeURL(endpoint)
	if err != nil {
		return Health{}, err
	}
	if u.Path != CanonicalHealthPathV1 || u.RawQuery != "" {
		return Health{}, ErrHealthContractV1
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Health{}, err
	}
	req.Header.Set("X-QC-API-KEY", strings.TrimSpace(token))
	client := NewNodeClient(2 * time.Second)
	defer client.CloseIdleConnections()
	resp, err := client.Do(req)
	if err != nil {
		return Health{}, err
	}
	defer resp.Body.Close()
	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return Health{}, ErrHealthContractV1
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil || len(raw) > 65536 {
		return Health{}, ErrHealthContractV1
	}
	// Reject duplicate top-level protocol/status fields before typed decoding.
	decoder := json.NewDecoder(bytes.NewReader(raw))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return Health{}, ErrHealthContractV1
	}
	seen := make(map[string]bool)
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return Health{}, ErrHealthContractV1
		}
		name, ok := key.(string)
		if !ok || seen[name] {
			return Health{}, ErrHealthContractV1
		}
		seen[name] = true
		for _, reserved := range []string{"ok", "success", "service", "healthVersion", "apiReadOnly"} {
			if strings.EqualFold(name, reserved) && name != reserved {
				return Health{}, ErrHealthContractV1
			}
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return Health{}, ErrHealthContractV1
		}
	}
	if closing, err := decoder.Token(); err != nil || closing != json.Delim('}') {
		return Health{}, ErrHealthContractV1
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Health{}, ErrHealthContractV1
	}
	var out struct {
		OK            bool   `json:"ok"`
		Success       *bool  `json:"success"`
		Service       string `json:"service"`
		HealthVersion string `json:"healthVersion"`
		ReadOnly      *bool  `json:"apiReadOnly"`
	}
	if json.Unmarshal(raw, &out) != nil {
		return Health{}, ErrHealthContractV1
	}
	if resp.StatusCode == http.StatusOK && out.OK && (out.Success == nil || *out.Success) && out.Service == HealthServiceV1 && out.HealthVersion == HealthProtocolV1 {
		status := Health{Reachable: true, Authenticated: true, ModeKnown: out.ReadOnly != nil}
		if out.ReadOnly != nil {
			status.ReadOnly = *out.ReadOnly
		}
		return status, nil
	}
	// A protected QuantumCoin 401 means an existing node, not an offline one.
	if resp.StatusCode == http.StatusUnauthorized && strings.TrimSpace(resp.Header.Get("WWW-Authenticate")) == HealthAuthRealmV1 && !out.OK && out.Success != nil && !*out.Success {
		return Health{Reachable: true}, nil
	}
	return Health{}, ErrHealthContractV1
}
