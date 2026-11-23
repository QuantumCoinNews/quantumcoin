package p2p

import (
	"bytes"
	"encoding/gob"
	"quantumcoin/blockchain"
	"quantumcoin/config"
	"testing"
)

func TestNetworkV2HandshakeCommitsGenesis(t *testing.T) {
	cfg := config.Default()
	cfg.GenesisProfile = config.NetworkGenesisProfileV2
	var magic []string
	for _, profile := range []config.NetworkProfileV1{config.QCNetworkProfileTestnetLocalV1, config.QCNetworkProfileMainnetV1} {
		chain, err := blockchain.NewNetworkBlockchainV2(cfg, profile)
		if err != nil {
			t.Fatal(err)
		}
		if err = chain.BindRuntimeNetworkIdentityV1(string(profile)); err != nil {
			t.Fatal(err)
		}
		var hello Handshake
		if err := gob.NewDecoder(bytes.NewReader(HandshakeMessage(chain).Data)).Decode(&hello); err != nil {
			t.Fatal(err)
		}
		identity := chain.NetworkIdentitySnapshotV1()
		if hello.Magic != identity.GenesisDomain+":"+identity.GenesisHash {
			t.Fatal("handshake did not commit the active genesis")
		}
		magic = append(magic, hello.Magic)
	}
	if magic[0] == magic[1] {
		t.Fatal("testnet and mainnet share a handshake domain")
	}
}

func TestNetworkV4HandshakeSeparatesSigningScopes(t *testing.T) {
	cfg := config.Default()
	cfg.GenesisProfile = config.NetworkGenesisProfileV4
	var magic []string
	for _, profile := range []config.NetworkProfileV1{config.QCNetworkProfileTestnetLocalV1, config.QCNetworkProfileTestnetPublicV1, config.QCNetworkProfileMainnetV1} {
		chain, err := blockchain.NewNetworkBlockchainV4(cfg, profile)
		if err != nil {
			t.Fatal(err)
		}
		var hello Handshake
		if err = gob.NewDecoder(bytes.NewReader(HandshakeMessage(chain).Data)).Decode(&hello); err != nil {
			t.Fatal(err)
		}
		id := chain.NetworkIdentitySnapshotV1()
		if hello.Magic != id.GenesisDomain+":"+id.GenesisHash {
			t.Fatal("handshake lost active signature scope")
		}
		magic = append(magic, hello.Magic)
	}
	if magic[0] != magic[1] || magic[0] == magic[2] {
		t.Fatal("local/public testnet disagree, or mainnet shares the testnet signing scope")
	}
}

func TestRuntimeHandshakeUsesPinnedNetworkDomain(t *testing.T) {
	for _, tc := range []struct{ profile, magic string }{{"local-release", "QC_LOCAL_RELEASE_V1"}, {"mainnet", "QC_MAINNET_V1"}, {"testnet-local", "QC_TESTNET_V1"}, {"testnet-public", "QC_TESTNET_V1"}} {
		chain := blockchain.NewBlockchain(50, 25_500_000)
		if err := chain.BindRuntimeNetworkIdentityV1(tc.profile); err != nil {
			t.Fatal(err)
		}
		t.Setenv("QC_NET_MAGIC", "spoofed-environment")
		msg := HandshakeMessage(chain)
		var hello Handshake
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&hello); err != nil {
			t.Fatal(err)
		}
		if hello.Magic != tc.magic {
			t.Fatal("runtime handshake followed mutable environment", hello, tc)
		}
	}
}
