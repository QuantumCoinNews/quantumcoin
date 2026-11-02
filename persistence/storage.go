package persistence

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Network string

const (
	NetworkMainnet Network = "mainnet"
	NetworkTestnet Network = "testnet"
)

type StorageConfig struct {
	RootDir string
	Network Network
}

func (c StorageConfig) Validate() error {
	if strings.TrimSpace(c.RootDir) == "" {
		return errors.New("storage root directory required")
	}

	switch c.Network {
	case NetworkMainnet, NetworkTestnet:
		return nil
	default:
		return fmt.Errorf("invalid storage network: %q", c.Network)
	}
}

func ResolveDomainPath(
	cfg StorageConfig,
	domain DataDomain,
) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}

	if strings.TrimSpace(domain.Name) == "" {
		return "", errors.New("persistence domain name required")
	}

	if strings.TrimSpace(domain.SchemaVersion) == "" {
		return "", errors.New("persistence schema version required")
	}

	if cfg.Network == NetworkMainnet && !domain.MainnetSeparated {
		return "", fmt.Errorf(
			"domain %s is not separated for mainnet",
			domain.Name,
		)
	}

	if cfg.Network == NetworkTestnet && !domain.TestnetSeparated {
		return "", fmt.Errorf(
			"domain %s is not separated for testnet",
			domain.Name,
		)
	}

	networkRoot := filepath.Join(
		cfg.RootDir,
		string(cfg.Network),
	)

	return filepath.Join(
		networkRoot,
		domain.Name,
		domain.SchemaVersion,
	), nil
}

func EnsureDomainPath(
	cfg StorageConfig,
	domain DataDomain,
) (string, error) {
	path, err := ResolveDomainPath(cfg, domain)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(path, 0o700); err != nil {
		return "", fmt.Errorf(
			"create persistence domain path: %w",
			err,
		)
	}

	return path, nil
}
