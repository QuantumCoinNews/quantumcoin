package blockchain

import (
	"encoding/hex"
	"fmt"

	"quantumcoin/config"
)

const IsolatedLocalGenesisDomainV1 = "QC_LOCAL_RELEASE_GENESIS_V2"
const TestnetGenesisDomainV2 = "QC_TESTNET_GENESIS_V2"
const MainnetGenesisDomainV2 = "QC_MAINNET_GENESIS_V2"
const TestnetGenesisDomainV3 = "QC_TESTNET_GENESIS_V3"
const MainnetGenesisDomainV3 = "QC_MAINNET_GENESIS_V3"
const TestnetGenesisDomainV4 = "QC_TESTNET_GENESIS_V4"
const MainnetGenesisDomainV4 = "QC_MAINNET_GENESIS_V4"

// NewNetworkBlockchainV4 activates genesis-bound signatures from the first
// spend, retaining V3 median-time rules. Existing profiles are never relabeled.
func NewNetworkBlockchainV4(cfg *config.Config, profile config.NetworkProfileV1) (*Blockchain, error) {
	if cfg == nil || cfg.GenesisProfile != config.NetworkGenesisProfileV4 {
		return nil, fmt.Errorf("network genesis requires an explicit network-v4 configuration")
	}
	chain, err := newDomainBoundChain(cfg, profile, TestnetGenesisDomainV4, MainnetGenesisDomainV4)
	if err != nil {
		return nil, err
	}
	if err = chain.BindRuntimeNetworkIdentityV1(string(profile)); err != nil {
		return nil, err
	}
	return chain, nil
}

// NewNetworkBlockchainV2 creates a fresh chain from explicit configuration.
// It commits a zero-value domain marker and uses the configured founder rules.
// Legacy disk chains never switch profiles by relabeling their identity.
func NewNetworkBlockchainV2(cfg *config.Config, profile config.NetworkProfileV1) (*Blockchain, error) {
	if cfg == nil || cfg.GenesisProfile != config.NetworkGenesisProfileV2 {
		return nil, fmt.Errorf("network genesis requires an explicit network-v2 configuration")
	}
	return newDomainBoundChain(cfg, profile, TestnetGenesisDomainV2, MainnetGenesisDomainV2)
}

// NewNetworkBlockchainV3 selects the genesis domain that activates median time.
func NewNetworkBlockchainV3(cfg *config.Config, profile config.NetworkProfileV1) (*Blockchain, error) {
	if cfg == nil || cfg.GenesisProfile != config.NetworkGenesisProfileV3 {
		return nil, fmt.Errorf("network genesis requires an explicit network-v3 configuration")
	}
	chain, err := newDomainBoundChain(cfg, profile, TestnetGenesisDomainV3, MainnetGenesisDomainV3)
	if err != nil {
		return nil, err
	}
	if err := chain.BindRuntimeNetworkIdentityV1(string(profile)); err != nil {
		return nil, err
	}
	return chain, nil
}

func newDomainBoundChain(cfg *config.Config, profile config.NetworkProfileV1, testnetDomain, mainnetDomain string) (*Blockchain, error) {
	domain := ""
	switch profile {
	case config.QCNetworkProfileTestnetLocalV1, config.QCNetworkProfileTestnetPublicV1:
		domain = testnetDomain
	case config.QCNetworkProfileMainnetV1:
		domain = mainnetDomain
	default:
		return nil, fmt.Errorf("network genesis requires testnet or mainnet")
	}
	return newBlockchainWithGenesisRecipientV1(cfg.InitialReward, cfg.TotalSupply, domain, cfg, ""), nil
}

func isPublicNetworkGenesisDomainV1(domain string) bool {
	switch domain {
	case TestnetGenesisDomainV2, MainnetGenesisDomainV2, TestnetGenesisDomainV3, MainnetGenesisDomainV3, TestnetGenesisDomainV4, MainnetGenesisDomainV4:
		return true
	default:
		return false
	}
}

func isNetworkGenesisMarkerV1(tx *Transaction) bool {
	if tx == nil || !tx.IsCoinbase() || tx.Sender != "COINBASE" || tx.Amount != 0 ||
		len(tx.Outputs) != 1 || tx.Outputs[0].Amount != 0 {
		return false
	}
	return isPublicNetworkGenesisDomainV1(string(tx.Outputs[0].PubKeyHash))
}

func (id NetworkIdentityV1) DomainMagicV2() (string, bool) {
	if !isPublicNetworkGenesisDomainV1(id.GenesisDomain) {
		return "", false
	}
	return id.GenesisDomain + ":" + id.GenesisHash, true
}

// NetworkIdentityV1 binds the persisted observation domain to the local trusted
// genesis. It is disk evidence, never an authority chosen by an incoming file.
type NetworkIdentityV1 struct {
	Version       string `json:"version"`
	Network       string `json:"network"`
	GenesisHash   string `json:"genesisHash"`
	GenesisDomain string `json:"genesisDomain,omitempty"`
}

func cloneNetworkIdentityV1(id *NetworkIdentityV1) *NetworkIdentityV1 {
	if id == nil {
		return nil
	}
	copy := *id
	return &copy
}

func (bc *Blockchain) BindRuntimeNetworkIdentityV1(profile string) error {
	if bc == nil {
		return ErrNilBlockchain
	}
	network := profile
	switch profile {
	case "testnet-local", "testnet-public":
		network = "testnet"
	case "local-release", "mainnet":
	default:
		return fmt.Errorf("unsupported runtime network identity")
	}
	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()
	if len(bc.Blocks) == 0 || bc.Blocks[0] == nil {
		return fmt.Errorf("network identity: genesis missing")
	}
	genesis := bc.Blocks[0]
	id := &NetworkIdentityV1{Version: "QC_CHAIN_NETWORK_IDENTITY_V1", Network: network, GenesisHash: hex.EncodeToString(genesis.Hash)}
	if len(genesis.Transactions) > 0 && genesis.Transactions[0] != nil && len(genesis.Transactions[0].Outputs) > 0 {
		marker := string(genesis.Transactions[0].Outputs[0].PubKeyHash)
		expected := ""
		switch marker {
		case IsolatedLocalGenesisDomainV1:
			expected = "local-release"
		case TestnetGenesisDomainV2, TestnetGenesisDomainV3, TestnetGenesisDomainV4:
			expected = "testnet"
		case MainnetGenesisDomainV2, MainnetGenesisDomainV3, MainnetGenesisDomainV4:
			expected = "mainnet"
		}
		if expected != "" {
			if network != expected {
				return fmt.Errorf("domain-bound genesis requires the %s network", expected)
			}
			id.GenesisDomain = marker
		}
	}
	if bc.networkIdentity != nil && *bc.networkIdentity != *id {
		return fmt.Errorf("runtime network identity cannot be rebound")
	}
	if bc.StoredNetworkIdentityV1 != nil && *bc.StoredNetworkIdentityV1 != *id {
		return fmt.Errorf("persisted network identity differs from runtime")
	}
	bc.networkIdentity = cloneNetworkIdentityV1(id)
	bc.StoredNetworkIdentityV1 = cloneNetworkIdentityV1(id)
	return nil
}

func (bc *Blockchain) NetworkIdentitySnapshotV1() *NetworkIdentityV1 {
	if bc == nil {
		return nil
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	return cloneNetworkIdentityV1(bc.networkIdentity)
}

func (bc *Blockchain) validateBoundNetworkIdentityLockedV1() error {
	if bc.networkIdentity == nil {
		return nil
	}
	if bc.StoredNetworkIdentityV1 == nil || *bc.StoredNetworkIdentityV1 != *bc.networkIdentity {
		return fmt.Errorf("network identity changed before persistence")
	}
	if len(bc.Blocks) == 0 || bc.Blocks[0] == nil || hex.EncodeToString(bc.Blocks[0].Hash) != bc.networkIdentity.GenesisHash {
		return fmt.Errorf("network identity genesis changed")
	}
	return nil
}
