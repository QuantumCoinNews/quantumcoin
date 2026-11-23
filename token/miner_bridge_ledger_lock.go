package token

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"quantumcoin/persistence"
)

func lockMinerTokenLedgerV1(directory string) (*persistence.FileLock, error) {
	lock, err := persistence.AcquireFileLock(filepath.Join(directory, registryFile), 0)
	if err != nil {
		return nil, fmt.Errorf("%w: lock ledger: %w", ErrMinerTokenPersistenceV1, err)
	}
	return lock, nil
}

func loadOrCreateMinerTokenLedgerV1(cfg MinerTokenConfig) (*Registry, *Token, error) {
	lock, err := lockMinerTokenLedgerV1(cfg.SaveDir)
	if err != nil {
		return nil, nil, err
	}
	defer lock.Release()
	registry := NewRegistry()
	if err := registry.Load(cfg.SaveDir); err != nil {
		return nil, nil, err
	}
	tok, exists := registry.Get(cfg.Symbol)
	if !exists {
		tok = New(cfg.Symbol, cfg.Name, cfg.Decimals, cfg.MaxSupply, cfg.Owner)
		if err := registry.Register(tok); err != nil {
			return nil, nil, err
		}
		data, err := json.MarshalIndent(registry, "", "  ")
		if err != nil {
			return nil, nil, err
		}
		if err := writeMinerTokenRegistryV1(cfg.SaveDir, data); err != nil {
			return nil, nil, fmt.Errorf("%w: create ledger: %w", ErrMinerTokenPersistenceV1, err)
		}
	}
	return registry, tok, nil
}

// Caller holds the ledger file lock, registry mutex and target token mutex.
// A second bridge/process must reload after the first writer changes the file;
// it must never finalize from an independent stale in-memory registry.
func verifyMinerTokenLedgerCurrentV1(directory string, registry *Registry, lockedToken *Token) error {
	expected, err := marshalRegistryWithLockedTokenV1(registry, lockedToken)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(directory, registryFile))
	if err != nil {
		return err
	}
	tokens, err := decodeRegistryStateV1(raw)
	if err != nil {
		return err
	}
	validated, err := validateLoadedRegistryV1(tokens)
	if err != nil {
		return err
	}
	actual, err := marshalRegistryWithLockedTokenV1(&Registry{tokens: validated}, nil)
	if err != nil {
		return err
	}
	if !bytes.Equal(expected, actual) {
		return fmt.Errorf("ledger changed by another writer; reload bridge before retrying")
	}
	return nil
}
