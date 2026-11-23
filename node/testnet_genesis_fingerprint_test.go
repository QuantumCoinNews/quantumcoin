package node

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"quantumcoin/config"
	"testing"
)

func TestOfficialTestnetGenesisMatchesPublishedFingerprint(t *testing.T) {
	previous := config.Current()
	t.Cleanup(func() { config.Set(previous) })
	clearRuntimePolicyNetworkEnvironmentV1(t)
	t.Setenv("QC_NETWORK", "testnet-local")
	cfg, err := config.LoadWithDataDir(filepath.Join("..", "release", "config.testnet.json"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join("..", "testnet", "genesis.json"))
	if err != nil {
		t.Fatal(err)
	}
	var published struct {
		Version   string `json:"version"`
		Hash      string `json:"genesisHash"`
		TxID      string `json:"genesisTransactionID"`
		Minted    int    `json:"genesisMintedQC"`
		Timestamp int64  `json:"genesisTimestamp"`
	}
	if err = json.Unmarshal(raw, &published); err != nil {
		t.Fatal(err)
	}
	if published.Version != "QC_TESTNET_GENESIS_FINGERPRINT_V4" || published.Minted != 0 {
		t.Fatal("missing zero-allocation testnet fingerprint")
	}
	for i := 0; i < 2; i++ {
		chain, err := newRuntimeChainContextV1(cfg)
		if err != nil {
			t.Fatal(err)
		}
		g := chain.BlockByIndexSnapshot(0)
		if hex.EncodeToString(g.Hash) != published.Hash || hex.EncodeToString(g.Transactions[0].ID) != published.TxID || g.Timestamp != published.Timestamp || chain.TotalMintedSnapshot() != published.Minted {
			t.Fatal("fresh node genesis differs from published testnet fingerprint")
		}
	}
}
