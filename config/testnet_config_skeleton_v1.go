package config

import qconsensus "quantumcoin/consensus"

import (
	"errors"
	"strings"
)

// QC_TESTNET_CONFIG_SKELETON_V1 defines the pre-testnet configuration skeleton.
//
// This file does not start testnet, does not modify consensus, does not mine,
// and does not change balances. It only defines the expected config boundary
// that later activation code must satisfy.
const (
	QCTestnetConfigSkeletonVersionV1 = "QC_TESTNET_CONFIG_SKELETON_V1"

	QCNetworkMainnetV1 = "mainnet"
	QCNetworkTestnetV1 = "testnet"

	QCTestnetMagicV1 = "QC_TESTNET_V1"
	QCMainnetMagicV1 = "QC_MAINNET_V1"

	QCMainnetChainFileV1 = "chain_data.dat"
	QCTestnetChainFileV1 = "chain_data_testnet.dat"
)

type TestnetConfigSkeletonV1 struct {
	Version                     string `json:"version"`
	Network                     string `json:"network"`
	NetworkMagic                string `json:"network_magic"`
	ChainFile                   string `json:"chain_file"`
	HTTPPort                    string `json:"http_port"`
	P2PPort                     string `json:"p2p_port"`
	TotalSupply                 int    `json:"total_supply"`
	DefaultDifficultyBits       int    `json:"default_difficulty_bits"`
	GenesisUnix                 int64  `json:"genesis_unix"`
	PublicAPIRequiresToken      bool   `json:"public_api_requires_token"`
	PublicCORSRequiresAllowlist bool   `json:"public_cors_requires_allowlist"`
	DevMinerEndpointsDisabled   bool   `json:"dev_miner_endpoints_disabled"`
	SeparateChainFileRequired   bool   `json:"separate_chain_file_required"`
	FaucetEnabled               bool   `json:"faucet_enabled"`
	PremineEnabled              bool   `json:"premine_enabled"`
	Notes                       string `json:"notes"`
}

func DefaultTestnetConfigSkeletonV1() TestnetConfigSkeletonV1 {
	return TestnetConfigSkeletonV1{
		Version:                     QCTestnetConfigSkeletonVersionV1,
		Network:                     QCNetworkTestnetV1,
		NetworkMagic:                QCTestnetMagicV1,
		ChainFile:                   QCTestnetChainFileV1,
		HTTPPort:                    ":18082",
		P2PPort:                     ":13001",
		TotalSupply:                 int(qconsensus.CanonicalTotalSupplyQCV1),
		DefaultDifficultyBits:       8,
		GenesisUnix:                 1725158400,
		PublicAPIRequiresToken:      true,
		PublicCORSRequiresAllowlist: true,
		DevMinerEndpointsDisabled:   true,
		SeparateChainFileRequired:   true,
		FaucetEnabled:               false,
		PremineEnabled:              false,
		Notes:                       "Pre-testnet skeleton only; activation requires later checklist items.",
	}
}

func ValidateTestnetConfigSkeletonV1(c TestnetConfigSkeletonV1) error {
	if strings.TrimSpace(c.Version) != QCTestnetConfigSkeletonVersionV1 {
		return errors.New("invalid testnet config skeleton version")
	}
	if strings.ToLower(strings.TrimSpace(c.Network)) != QCNetworkTestnetV1 {
		return errors.New("testnet skeleton network must be testnet")
	}
	if strings.TrimSpace(c.NetworkMagic) != QCTestnetMagicV1 {
		return errors.New("testnet skeleton network magic must be QC_TESTNET_V1")
	}
	if strings.TrimSpace(c.ChainFile) == "" {
		return errors.New("testnet skeleton chain file is required")
	}
	if strings.TrimSpace(c.ChainFile) == QCMainnetChainFileV1 {
		return errors.New("testnet skeleton must not use mainnet chain file")
	}
	if !strings.Contains(strings.ToLower(c.ChainFile), "testnet") {
		return errors.New("testnet skeleton chain file must be clearly labeled testnet")
	}
	if strings.TrimSpace(c.HTTPPort) == "" || strings.TrimSpace(c.P2PPort) == "" {
		return errors.New("testnet skeleton ports are required")
	}
	if c.HTTPPort == ":8082" {
		return errors.New("testnet skeleton must not use current local main API port :8082")
	}
	if c.P2PPort == ":3001" {
		return errors.New("testnet skeleton must not use current local main P2P port :3001")
	}
	if c.TotalSupply != int(qconsensus.CanonicalTotalSupplyQCV1) {
		return errors.New("testnet skeleton must preserve QC total supply cap")
	}
	if c.DefaultDifficultyBits <= 0 || c.DefaultDifficultyBits > 255 {
		return errors.New("testnet skeleton difficulty bits invalid")
	}
	if c.GenesisUnix <= 0 {
		return errors.New("testnet skeleton genesis unix required")
	}
	if !c.PublicAPIRequiresToken {
		return errors.New("testnet skeleton must require API token for public exposure")
	}
	if !c.PublicCORSRequiresAllowlist {
		return errors.New("testnet skeleton must require CORS allowlist for public exposure")
	}
	if !c.DevMinerEndpointsDisabled {
		return errors.New("testnet skeleton must disable dev/miner endpoints in public mode")
	}
	if !c.SeparateChainFileRequired {
		return errors.New("testnet skeleton must require separate chain file")
	}
	if c.FaucetEnabled {
		return errors.New("testnet faucet must remain disabled in skeleton")
	}
	if c.PremineEnabled {
		return errors.New("testnet premine must remain disabled in skeleton")
	}
	return nil
}

func TestnetEnvSkeletonV1() map[string]string {
	c := DefaultTestnetConfigSkeletonV1()
	return map[string]string{
		"QC_TESTNET":                 "1",
		"QC_PUBLIC_TESTNET":          "0",
		"QC_NET_MAGIC":               c.NetworkMagic,
		"QC_NETWORK":                 c.Network,
		"QC_CHAIN_MODE":              c.Network,
		"QC_CHAIN_FILE":              c.ChainFile,
		"QC_HTTP_PORT":               c.HTTPPort,
		"QC_P2P_PORT":                c.P2PPort,
		"QC_DEFAULT_DIFFICULTY_BITS": "8",
		"QC_API_TOKEN":               "__SET_STRONG_TOKEN_BEFORE_PUBLIC_TESTNET__",
		"QC_CORS_ALLOWED_ORIGINS":    "https://wallet.quantumcoin.test,https://game.quantumcoin.test",
	}
}
