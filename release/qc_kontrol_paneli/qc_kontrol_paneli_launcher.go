package main

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"quantumcoin/monitoring"
)

//go:embed live.html
var panelHTML string

var page = template.Must(template.New("panel").Parse(panelHTML))
var errAuthRequired = errors.New("kimlik doğrulaması gerekli")

type source struct {
	node, game, token string
	client            *http.Client
}

type telemetry struct {
	Height     *int `json:"height"`
	BlockCount *int `json:"block_count"`
	Peers      *int `json:"peers"`
}

type panel struct {
	Time, Node, Game, Network, Alerts string
	Telemetry                         *telemetry
	Warnings, Critical                []string
}

func loopbackOrigin(raw, fallback string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		raw = fallback
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" ||
		u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New("panel kaynağı yerel bir HTTP adresi olmalı")
	}
	if ip := net.ParseIP(u.Hostname()); ip == nil || !ip.IsLoopback() {
		return "", errors.New("panel kaynağı sayısal bir loopback adresi olmalı")
	}
	return "http://" + u.Host, nil
}

func (s source) get(ctx context.Context, base, path, token string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("X-QC-API-KEY", token)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return errAuthRequired
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	const maxBytes = 8 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil || len(body) > maxBytes {
		return errors.New("geçersiz yanıt boyutu")
	}
	return json.Unmarshal(body, out)
}

func (s source) snapshot(ctx context.Context) panel {
	view := panel{Time: time.Now().Format(time.RFC3339), Node: "Doğrulanamadı", Game: "Doğrulanamadı", Network: "Doğrulanmadı", Alerts: "Uyarı kaynağı doğrulanamadı"}
	var health struct {
		OK              bool `json:"ok"`
		NetworkIdentity *struct {
			Network string `json:"network"`
			Genesis string `json:"genesisHash"`
		} `json:"networkIdentity"`
	}
	if err := s.get(ctx, s.node, "/api/health", s.token, &health); errors.Is(err, errAuthRequired) {
		view.Node = "Kimlik doğrulaması gerekli"
	} else if err == nil && health.OK {
		view.Node = "Çevrim içi"
		if id := health.NetworkIdentity; id != nil {
			if decoded, err := hex.DecodeString(id.Genesis); err == nil && len(decoded) == 32 {
				switch id.Network {
				case "local-release", "testnet", "mainnet":
					view.Network = id.Network
				}
			}
		}
		var stats telemetry
		if s.get(ctx, s.node, "/api/telemetry", s.token, &stats) == nil &&
			stats.Height != nil && stats.BlockCount != nil && stats.Peers != nil {
			view.Telemetry = &stats
		}
		var state struct {
			Incidents json.RawMessage `json:"incidents"`
			Failures  uint64          `json:"failures"`
		}
		var incidents []monitoring.Incident
		if s.get(ctx, s.node, "/api/monitoring", s.token, &state) == nil && json.Unmarshal(state.Incidents, &incidents) == nil {
			view.Alerts = "Güncel"
			for _, incident := range incidents {
				if incident.Status != "open" && incident.Status != "acknowledged" {
					continue
				}
				switch incident.Severity {
				case monitoring.SeverityWarning:
					view.Warnings = append(view.Warnings, incident.Category)
				case monitoring.SeverityCritical:
					view.Critical = append(view.Critical, incident.Category)
				}
			}
			if state.Failures > 0 {
				view.Warnings = append(view.Warnings, "Monitoring kayıt hatası bildiriyor")
			}
		}
	}
	var game struct {
		OK bool `json:"ok"`
	}
	if s.get(ctx, s.game, "/health", "", &game) == nil && game.OK {
		view.Game = "Çevrim içi"
	}
	return view
}

func (s source) handler(host, path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != host || r.URL.Path != path || r.URL.RawQuery != "" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'")
		if err := page.Execute(w, s.snapshot(r.Context())); err != nil {
			log.Print("panel render failed")
		}
	})
}

func main() {
	node, err := loopbackOrigin(os.Getenv("QC_PANEL_NODE_BASE"), "http://127.0.0.1:8082")
	if err != nil {
		log.Fatal(err)
	}
	game, err := loopbackOrigin(os.Getenv("QC_PANEL_GAME_BASE"), "http://127.0.0.1:8080")
	if err != nil {
		log.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		log.Fatal(err)
	}
	path := "/" + hex.EncodeToString(nonce[:])
	host := listener.Addr().String()
	s := source{
		node: node, game: game, token: strings.TrimSpace(os.Getenv("QC_API_TOKEN")),
		client: &http.Client{
			Timeout: 2 * time.Second, Transport: &http.Transport{},
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
	panelURL := "http://" + host + path
	fmt.Println("Canlı panel", panelURL)
	fmt.Println("Kapatmak için Ctrl+C")
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", panelURL)
	if err := cmd.Start(); err == nil {
		_ = cmd.Process.Release()
	}
	server := &http.Server{Handler: s.handler(host, path), ReadHeaderTimeout: 3 * time.Second, WriteTimeout: 12 * time.Second, IdleTimeout: 15 * time.Second}
	if err := server.Serve(listener); err != nil {
		log.Fatal(err)
	}
}
