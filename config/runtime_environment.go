package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

//go:embed runtime-environment.schema.json
var runtimeEnvironmentSchemaJSONV1 []byte

type environmentPropertyV1 struct {
	Kind    string   `json:"x-kind"`
	Owners  []string `json:"x-owners"`
	Status  string   `json:"x-status"`
	Case    string   `json:"x-case"`
	Enum    []string `json:"enum"`
	Minimum int64    `json:"minimum"`
	Maximum int64    `json:"maximum"`
}
type environmentAliasV1 struct {
	Keys      []string `json:"keys"`
	Normalize string   `json:"normalize"`
	Owners    []string `json:"owners"`
}
type environmentSchemaV1 struct {
	Version     int                              `json:"x-version"`
	Services    []string                         `json:"x-services"`
	Properties  map[string]environmentPropertyV1 `json:"properties"`
	Aliases     []environmentAliasV1             `json:"x-aliasGroups"`
	NetworkKeys []string                         `json:"x-networkKeys"`
}

var envIntegerPatternV1 = regexp.MustCompile(`^-?[0-9]+$`)
var envTimezonePatternV1 = regexp.MustCompile(`^[A-Za-z_]+(?:/[A-Za-z_+0-9-]+){1,2}$`)
var envWindowsAbsolutePatternV1 = regexp.MustCompile(`^[A-Za-z]:[\\/]`)
var envWindowsUNCAbsolutePatternV1 = regexp.MustCompile(`^\\\\[^\\/]+[\\/][^\\/]+`)

func environmentContainsV1(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
func environmentTrueV1(value string) bool {
	return environmentContainsV1([]string{"1", "true", "yes", "on"}, strings.ToLower(value))
}
func environmentPortV1(value string) (int, error) {
	raw := strings.TrimPrefix(value, ":")
	if raw == "" || strings.IndexFunc(raw, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return 0, fmt.Errorf("invalid port")
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 65535 {
		return 0, fmt.Errorf("invalid port")
	}
	return n, nil
}
func environmentOriginV1(value string, monitor bool) bool {
	u, err := url.Parse(value)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return false
	}
	if u.Port() != "" {
		if _, err := environmentPortV1(u.Port()); err != nil {
			return false
		}
	}
	host := strings.ToLower(u.Hostname())
	ip := net.ParseIP(host)
	local := host == "localhost" || (ip != nil && ip.IsLoopback())
	if ip != nil && ip.To4() != nil && strings.Contains(host, ":") {
		local = false
	}
	if monitor {
		return strings.EqualFold(u.Scheme, "http") && (host == "127.0.0.1" || host == "::1")
	}
	return strings.EqualFold(u.Scheme, "https") || (strings.EqualFold(u.Scheme, "http") && local)
}
func environmentNetworkV1(value string, game bool) string {
	switch strings.ToLower(value) {
	case "local", "local-release":
		return "local"
	case "testnet", "testnet-local", "testnet-public":
		return "testnet"
	case "mainnet":
		return "mainnet"
	case "main", "production", "prod":
		if game {
			return "mainnet"
		}
	}
	return ""
}
func environmentGameModeV1(value string) string {
	v := strings.ToUpper(value)
	if environmentContainsV1([]string{"TESTNET", "SIMULATED", "TESTNET_SIM", "TESTNET_SIMULATED"}, v) {
		return "simulated"
	}
	if environmentContainsV1([]string{"OFF", "DISABLE", "DISABLED", "NONE"}, v) {
		return "disabled"
	}
	return v
}

// ValidateRuntimeEnvironmentV1 is read-only. Its diagnostics never contain values.
// Unknown host/system variables are not interpreted as application settings.
func ValidateRuntimeEnvironmentV1(env map[string]string, service string) map[string]string {
	errors := map[string]string{}
	var schema environmentSchemaV1
	if json.Unmarshal(runtimeEnvironmentSchemaJSONV1, &schema) != nil || schema.Version != 1 || !environmentContainsV1(schema.Services, service) {
		errors["CONFIG_SCHEMA"] = "unsupported-schema-or-service"
		return errors
	}
	get := func(key string) string { return strings.TrimSpace(env[key]) }
	for key, entry := range schema.Properties {
		value := get(key)
		if value == "" || (service != "tooling" && !environmentContainsV1(entry.Owners, service)) {
			continue
		}
		if entry.Status == "retired" {
			errors[key] = "retired-setting"
			continue
		}
		valid := true
		switch entry.Kind {
		case "boolean":
			valid = environmentContainsV1([]string{"1", "true", "yes", "on", "0", "false", "no", "off"}, strings.ToLower(value))
		case "boolean-01":
			valid = env[key] == "0" || env[key] == "1"
		case "integer":
			n, err := strconv.ParseInt(value, 10, 64)
			valid = envIntegerPatternV1.MatchString(value) && err == nil && n >= entry.Minimum && n <= entry.Maximum
		case "port":
			_, err := environmentPortV1(value)
			valid = err == nil
		case "bind":
			host, port, err := net.SplitHostPort(value)
			_, perr := environmentPortV1(port)
			valid = err == nil && perr == nil && (host == "" || strings.EqualFold(host, "localhost") || net.ParseIP(host) != nil)
			if strings.HasPrefix(value, "[") {
				ip := net.ParseIP(host)
				valid = valid && ip != nil && ip.To4() == nil
			}
		case "origin", "monitor-origin":
			valid = environmentOriginV1(value, entry.Kind == "monitor-origin")
		case "absolute-path":
			valid = envWindowsAbsolutePatternV1.MatchString(value) || strings.HasPrefix(value, "/") || envWindowsUNCAbsolutePatternV1.MatchString(value)
		case "network", "game-network":
			valid = environmentNetworkV1(value, entry.Kind == "game-network") != ""
		case "enum":
			candidate := value
			if entry.Case == "upper" {
				candidate = strings.ToUpper(candidate)
			}
			valid = environmentContainsV1(entry.Enum, candidate)
		case "platforms":
			seen := map[string]bool{}
			for _, part := range strings.Split(value, ",") {
				v := strings.TrimSpace(part)
				if seen[v] || !environmentContainsV1([]string{"youtube", "x", "telegram", "instagram", "tiktok"}, v) {
					valid = false
				}
				seen[v] = true
			}
		case "timezone":
			valid = value == "UTC" || envTimezonePatternV1.MatchString(value)
		case "text":
		default:
			valid = false
		}
		if !valid {
			errors[key] = "invalid-" + entry.Kind
		}
	}
	for key := range env {
		if strings.HasPrefix(strings.ToUpper(key), "QC_") && key != strings.ToUpper(key) {
			errors[key] = "noncanonical-key-case"
			continue
		}
		if _, known := schema.Properties[key]; !known && strings.HasPrefix(key, "QC_") && !strings.HasPrefix(key, "QC_TEST_") {
			errors[key] = "unknown-setting"
		}
	}
	for _, group := range schema.Aliases {
		if service != "tooling" && !environmentContainsV1(group.Owners, service) {
			continue
		}
		values := map[string]bool{}
		keys := []string{}
		for _, key := range group.Keys {
			value := get(key)
			if value == "" {
				continue
			}
			keys = append(keys, key)
			switch group.Normalize {
			case "origin":
				value = strings.TrimRight(value, "/")
			case "game-mode":
				value = environmentGameModeV1(value)
			case "port":
				if n, err := environmentPortV1(value); err == nil {
					value = strconv.Itoa(n)
				}
			}
			values[value] = true
		}
		if len(values) > 1 {
			for _, key := range keys {
				errors[key] = "conflicting-alias"
			}
		}
	}
	if service == "node" || service == "game" || service == "tooling" {
		selected := map[string]string{}
		modes := map[string]bool{}
		for _, key := range schema.NetworkKeys {
			value := get(key)
			if value == "" {
				continue
			}
			mode := environmentNetworkV1(value, strings.HasPrefix(key, "TELEGRAM_GAME_"))
			if mode == "" {
				errors[key] = "invalid-network"
			} else {
				selected[key] = mode
				modes[mode] = true
			}
		}
		for _, key := range []string{"QC_TESTNET", "QC_PUBLIC_TESTNET", "QC_API_PUBLIC_TESTNET", "QC_TESTNET_PUBLIC_API", "QC_MAINNET"} {
			if environmentTrueV1(get(key)) {
				mode := "testnet"
				if key == "QC_MAINNET" {
					mode = "mainnet"
				}
				selected[key] = mode
				modes[mode] = true
			}
		}
		if magic := strings.ToUpper(get("QC_NET_MAGIC")); magic != "" {
			mode := map[string]string{"QC_LOCAL_RELEASE_V1": "local", "QC_TESTNET_V1": "testnet", "QC_MAINNET_V1": "mainnet"}[magic]
			if mode == "" {
				errors["QC_NET_MAGIC"] = "invalid-enum"
			} else {
				selected["QC_NET_MAGIC"] = mode
				modes[mode] = true
			}
		}
		if len(modes) > 1 {
			for key := range selected {
				errors[key] = "conflicting-network"
			}
		}
		if len(modes) == 1 {
			for _, key := range []string{"QC_API_BASE", "QUANTUMCOIN_API_BASE", "QC_NODE_API"} {
				value := get(key)
				if value == "" || errors[key] != "" {
					continue
				}
				u, err := url.Parse(value)
				if err != nil {
					continue
				}
				port := u.Port()
				if n, err := environmentPortV1(port); err == nil {
					port = strconv.Itoa(n)
				}
				if (modes["testnet"] && (port == "8081" || port == "8082") && !environmentTrueV1(get("TELEGRAM_GAME_ALLOW_TESTNET_MAIN_PORT"))) || (!modes["testnet"] && port == "18082") {
					errors[key] = "network-port-conflict"
				}
			}
		}
	}
	if service == "node" || service == "tooling" {
		if bind := get("QC_API_LISTEN"); bind != "" && errors["QC_API_LISTEN"] == "" {
			host, port, _ := net.SplitHostPort(bind)
			ip := net.ParseIP(host)
			local := strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback())
			if !local && get("QC_API_TOKEN") == "" {
				errors["QC_API_TOKEN"] = "required-for-non-loopback-bind"
			}
			if token := strings.ToUpper(get("QC_API_TOKEN")); !local && (strings.HasPrefix(token, "__SET_") || strings.HasPrefix(token, "YOUR_") || strings.HasPrefix(token, "CHANGE_ME")) {
				errors["QC_API_TOKEN"] = "placeholder-credential"
			}
			if get("QC_HTTP_PORT") != "" && errors["QC_HTTP_PORT"] == "" {
				listen, _ := environmentPortV1(port)
				expected, _ := environmentPortV1(get("QC_HTTP_PORT"))
				if listen != expected {
					errors["QC_API_LISTEN"] = "http-port-conflict"
				}
			}
		}
	}
	if (service == "publisher" || service == "tooling") && get("QC_PUBLISHER_MONITOR_ORIGIN") != "" && get("QC_PUBLISHER_MONITOR_TOKEN") == "" {
		errors["QC_PUBLISHER_MONITOR_TOKEN"] = "required-with-monitor-origin"
	}
	return errors
}

func ValidateProcessRuntimeEnvironmentV1(service string) error {
	env := map[string]string{}
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			env[key] = value
		}
	}
	issues := ValidateRuntimeEnvironmentV1(env, service)
	if len(issues) == 0 {
		return nil
	}
	keys := make([]string, 0, len(issues))
	for key := range issues {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+":"+issues[key])
	}
	return fmt.Errorf("runtime-config: %s", strings.Join(parts, ","))
}

// ValidateRuntimeHTTPBindV1 checks the effective file/environment port before
// publication/startup, rather than silently falling back inside presentation.
func ValidateRuntimeHTTPBindV1(c *Config) error {
	if c == nil {
		return fmt.Errorf("runtime-config: CONFIG_SCHEMA:missing-config")
	}
	expected, err := environmentPortV1(strings.TrimPrefix(c.HTTPPort, ":"))
	if err != nil {
		return fmt.Errorf("runtime-config: http_port:invalid-port")
	}
	if legacy := strings.TrimSpace(os.Getenv("HTTP_PORT")); legacy != "" {
		n, err := environmentPortV1(legacy)
		if err != nil || n != expected {
			return fmt.Errorf("runtime-config: HTTP_PORT:http-port-conflict")
		}
	}
	if bind := strings.TrimSpace(os.Getenv("QC_API_LISTEN")); bind != "" {
		_, port, err := net.SplitHostPort(bind)
		if err != nil {
			return fmt.Errorf("runtime-config: QC_API_LISTEN:invalid-bind")
		}
		n, err := environmentPortV1(port)
		if err != nil || n != expected {
			return fmt.Errorf("runtime-config: QC_API_LISTEN:http-port-conflict")
		}
	}
	return nil
}
