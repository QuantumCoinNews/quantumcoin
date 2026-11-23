package persistence

import (
	"errors"
	"quantumcoin/blockchain"
)

// ChainStorageVersionV1 identifies a historical format, not the canonical codec.
const ChainStorageVersionV1 = "QC_CHAIN_STORAGE_V1"

type ChainEnvelopeV1 struct {
	Version string
	Blocks  []*blockchain.Block
}

var ErrLegacyChainStoreRetiredV1 = errors.New("legacy chain store retired: use blockchain trusted loader and SaveToFile")

// SaveChainV1 fails without touching disk; it cannot replace the canonical codec.
func SaveChainV1(_ string, _ []*blockchain.Block) error { return ErrLegacyChainStoreRetiredV1 }

// LoadChainV1 cannot establish trusted network or consensus context.
func LoadChainV1(_ string) ([]*blockchain.Block, error) { return nil, ErrLegacyChainStoreRetiredV1 }
