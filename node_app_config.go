package main

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"quantumcoin/config"
	"quantumcoin/wallet"
	"strings"
)

func getHTTPPort() string {
	if cfg == nil {
		return ""
	}

	port, err := config.ResolveEffectiveHTTPPortV1(cfg)
	if err != nil {
		log.Printf(
			"[config] HTTP port policy rejected legacy override: %v; using canonical=%s",
			err,
			cfg.HTTPPort,
		)

		return cfg.HTTPPort
	}

	return port
}

func getHTTPAddr() string {
	addr := getHTTPPort()
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	return addr
}

// getAPIListenAddr preserves the localhost default while permitting
// an explicit host/interface only when it uses the canonical API port.
func getAPIListenAddr() string {
	canonicalAddr := getHTTPAddr()
	if canonicalAddr == "" {
		return ""
	}

	canonicalPort := strings.TrimPrefix(
		canonicalAddr,
		":",
	)

	localDefault := "127.0.0.1:" + canonicalPort

	override := strings.TrimSpace(
		os.Getenv("QC_API_LISTEN"),
	)

	if override == "" {
		return localDefault
	}

	host, port, err := net.SplitHostPort(override)
	if err != nil {
		log.Printf(
			"[config] invalid QC_API_LISTEN=%q: %v; using %s",
			override,
			err,
			localDefault,
		)

		return localDefault
	}

	if port != canonicalPort {
		log.Printf(
			"[config] QC_API_LISTEN=%q conflicts with canonical port %s; using %s",
			override,
			canonicalPort,
			localDefault,
		)

		return localDefault
	}

	if strings.TrimSpace(host) == "" {
		return localDefault
	}

	return override
}

// All compatibility helpers delegate to the same read-only selection policy.
func resolveDefaultMinerAddressV1() (wallet.MinerAddressSelection, error) {
	c := cfg
	if c == nil {
		c = config.Current()
	}
	dataDir := activeCLIPathsV1.DataDir
	if dataDir == "" && c != nil {
		dataDir = filepath.Dir(c.ChainFile)
	}
	return wallet.ResolveMinerAddress(wallet.MinerAddressOptions{
		Config: c, DataDir: dataDir, AppDataDir: os.Getenv("APPDATA"),
	})
}

func getDefaultAddress() string { return lookupDefaultAddressV1() }

func lookupDefaultAddressV1() string {
	selection, err := resolveDefaultMinerAddressV1()
	if err != nil {
		return ""
	}
	return selection.Address
}

func ensureMinerAddress() (string, error) {
	selection, err := resolveDefaultMinerAddressV1()
	return selection.Address, err
}
