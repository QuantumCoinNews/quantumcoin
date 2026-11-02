package persistence

import (
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"quantumcoin/blockchain"
)

const ChainStorageVersionV1 = "QC_CHAIN_STORAGE_V1"

type ChainEnvelopeV1 struct {
	Version string
	Blocks  []*blockchain.Block
}

func SaveChainV1(
	path string,
	blocks []*blockchain.Block,
) error {
	if path == "" {
		return errors.New("chain storage path required")
	}

	if len(blocks) == 0 {
		return errors.New("chain blocks required")
	}

	for i, block := range blocks {
		if block == nil {
			return fmt.Errorf("nil block at index %d", i)
		}
	}

	parent := filepath.Dir(path)
	if parent != "." && parent != "" {
		if err := os.MkdirAll(parent, 0o700); err != nil {
			return fmt.Errorf("create chain storage directory: %w", err)
		}
	}

	tempPath := path + ".tmp"

	file, err := os.OpenFile(
		tempPath,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		0o600,
	)
	if err != nil {
		return fmt.Errorf("open chain temp file: %w", err)
	}

	envelope := ChainEnvelopeV1{
		Version: ChainStorageVersionV1,
		Blocks:  blocks,
	}

	encodeErr := gob.NewEncoder(file).Encode(envelope)
	syncErr := file.Sync()
	closeErr := file.Close()

	if encodeErr != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("encode chain: %w", encodeErr)
	}

	if syncErr != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("sync chain: %w", syncErr)
	}

	if closeErr != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close chain: %w", closeErr)
	}

	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("commit chain storage: %w", err)
	}

	return nil
}

func LoadChainV1(path string) ([]*blockchain.Block, error) {
	if path == "" {
		return nil, errors.New("chain storage path required")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open chain storage: %w", err)
	}
	defer file.Close()

	var envelope ChainEnvelopeV1
	if err := gob.NewDecoder(file).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode chain storage: %w", err)
	}

	if envelope.Version != ChainStorageVersionV1 {
		return nil, fmt.Errorf(
			"unsupported chain storage version: %q",
			envelope.Version,
		)
	}

	if len(envelope.Blocks) == 0 {
		return nil, errors.New("chain storage contains no blocks")
	}

	for i, block := range envelope.Blocks {
		if block == nil {
			return nil, fmt.Errorf("chain storage has nil block at index %d", i)
		}
	}

	return envelope.Blocks, nil
}
