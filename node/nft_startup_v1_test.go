package node

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/consensus"
	"quantumcoin/wallet"
)

type nftStartupFixtureDataV1 struct {
	cfg                   *config.Config
	release               consensus.HardforkReleaseV1
	snapshotDir, manifest string
	trust                 consensus.NFTAwardTrustV1
	program               consensus.NFTMonthlyProgramReleaseV1
	keys                  []*ecdsa.PrivateKey
	producer              *blockchain.Blockchain
	record                blockchain.NFTMintRecordV1
}

func nftStartupWriteProgramV1(t *testing.T, f *nftStartupFixtureDataV1, resign bool) {
	t.Helper()
	if resign {
		digest, err := consensus.NFTMonthlyProgramDecisionDigestV1(f.program.Decision)
		if err != nil {
			t.Fatal(err)
		}
		f.program.Approvals = nil
		for _, key := range f.keys {
			pub, err := consensus.UncompressedPublicKeyV1(key)
			if err != nil {
				t.Fatal(err)
			}
			sig, err := consensus.SignTransactionDigestV1(key, digest)
			if err != nil {
				t.Fatal(err)
			}
			f.program.Approvals = append(f.program.Approvals, consensus.NFTAwardApprovalV1{PublicKey: pub, Signature: sig})
		}
	}
	data, err := json.Marshal(f.program)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.manifest, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func nftStartupFixtureV1(t *testing.T) *nftStartupFixtureDataV1 {
	t.Helper()
	cfg := defaultRuntimePolicyConfigV1(t)
	cfg.BonusFile = filepath.Join(t.TempDir(), "bonus.json")
	chain := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	snapshot := blockchain.SerializeBlockchain(chain)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	sh, bh := sha256.Sum256(snapshot), sha256.Sum256(binary)
	network, genesis := "test-only-nft-startup", hex.EncodeToString(chain.Blocks[0].Hash)
	d := consensus.HardforkDecisionV1{NetworkID: network, LegacyGenesisHex: genesis, ActivationHeight: 1, TargetProtocol: consensus.ProtocolNFTV4,
		Reasons: []string{"NETWORK_WIDE_PROTOCOL_UPGRADE_WITH_PUBLIC_NOTICE"}, SnapshotSHA256: hex.EncodeToString(sh[:]), ReleaseBinarySHA256: hex.EncodeToString(bh[:])}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, d.SnapshotSHA256+".snapshot"), snapshot, 0600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	activated := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC).Unix()
	f := &nftStartupFixtureDataV1{cfg: cfg, release: signActivationReleaseFixtureV1(t, d), snapshotDir: dir, manifest: filepath.Join(dir, consensus.NFTMonthlyProgramManifestNameV1),
		trust:   consensus.NFTAwardTrustV1{NetworkID: network, LegacyGenesisHex: genesis, AdminQuorum: 2},
		program: consensus.NFTMonthlyProgramReleaseV1{Decision: consensus.NFTMonthlyProgramDecisionV1{NetworkID: network, LegacyGenesisHex: genesis, CampaignID: "telegram-monthly", GameActivatedAt: activated}}, producer: chain}
	for i := 0; i < 2; i++ {
		key, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		pub, err := consensus.UncompressedPublicKeyV1(key)
		if err != nil {
			t.Fatal(err)
		}
		f.keys = append(f.keys, key)
		f.trust.AdminPublicKeys = append(f.trust.AdminPublicKeys, pub)
	}
	nftStartupWriteProgramV1(t, f, true)
	program, err := consensus.VerifyNFTMonthlyProgramReleaseV1(f.program, f.trust)
	if err != nil {
		t.Fatal(err)
	}
	activation, err := consensus.LoadHardforkActivationV1(&f.release, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := chain.BindActivationPolicySynchronizedV1(activation); err != nil {
		t.Fatal(err)
	}
	if err := chain.BindLocalNFTAwardTrustSynchronizedV1(network, f.trust); err != nil {
		t.Fatal(err)
	}
	if err := chain.BindLocalNFTMonthlyProgramSynchronizedV1(program.PolicySnapshotV1()); err != nil {
		t.Fatal(err)
	}
	chain.SetCoinbaseMaturity(cfg.CoinbaseMaturity)
	f.record = blockchain.NFTMintRecordV1{NetworkID: network, CampaignID: f.program.Decision.CampaignID, AwardPeriod: program.PolicySnapshotV1().FirstAwardPeriod,
		Serial: 1, RecipientPubKeyHash: bytes.Repeat([]byte{0x11}, 20), MetadataHash: [32]byte{0x44}}
	recordDigest, err := blockchain.NFTMintRecordDigestV1(f.record)
	if err != nil {
		t.Fatal(err)
	}
	proof := blockchain.NFTAwardProofV1{Decision: consensus.NFTAwardDecisionV1{NetworkID: network, LegacyGenesisHex: genesis, RecordDigest: recordDigest}}
	digest, err := consensus.NFTAwardDecisionDigestV1(proof.Decision)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range f.keys {
		pub, err := consensus.UncompressedPublicKeyV1(key)
		if err != nil {
			t.Fatal(err)
		}
		sig, err := consensus.SignTransactionDigestV1(key, digest)
		if err != nil {
			t.Fatal(err)
		}
		proof.Approvals = append(proof.Approvals, consensus.NFTAwardApprovalV1{PublicKey: pub, Signature: sig})
	}
	difficulty, err := chain.ExpectedNextDifficultyV1()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chain.MineBlockWithNFTAwardSynchronizedV1(wallet.NewWallet().GetAddress(), difficulty, f.record, proof); err != nil {
		t.Fatal(err)
	}
	if err := chain.SaveToFile(cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestNFTStartupRestoresApprovedPersistedAwardV1(t *testing.T) {
	f := nftStartupFixtureV1(t)
	before, err := os.ReadFile(f.cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	r, err := bootstrapWithNFTStartupV1(f.cfg, true, &f.release, f.snapshotDir, &f.trust, f.manifest)
	if err != nil || r == nil {
		t.Fatalf("approved persisted NFT startup: %v", err)
	}
	chain, _ := r.Snapshot()
	got, err := chain.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil || got.Height != 1 || len(got.NFTs) != 1 {
		t.Fatalf("restored ownership: %+v %v", got, err)
	}
	if !bytes.Equal(got.TipHash, f.producer.AllBlocksSnapshot()[1].Hash) || !bytes.Equal(got.NFTs[0].Record.RecipientPubKeyHash, f.record.RecipientPubKeyHash) {
		t.Fatal("startup changed canonical NFT")
	}
	program := r.NFTGameProgramSnapshotV1()
	if !program.Verified || program.GameActivatedAt != f.program.Decision.GameActivatedAt || program.Policy.FirstAwardPeriod != f.record.AwardPeriod {
		t.Fatal("verified game program unavailable")
	}
	program.Policy.FirstAwardPeriod = "2099-01"
	f.trust.AdminPublicKeys[0][0] ^= 1
	if !r.NFTGameProgramSnapshotV1().Verified || r.NFTGameProgramSnapshotV1().Policy.FirstAwardPeriod != f.record.AwardPeriod {
		t.Fatal("runtime program mutated")
	}
	if _, err := chain.NFTOwnershipSnapshotSynchronizedV1(); err != nil {
		t.Fatalf("startup retained caller authority slices: %v", err)
	}
	after, err := os.ReadFile(f.cfg.ChainFile)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("startup wrote chain file")
	}
	if _, err := blockchain.LoadBlockchainFromFile(f.cfg.ChainFile); err == nil {
		t.Fatal("context-free loader gained NFT trust")
	}
}

func TestNFTStartupRejectsIncompleteOrUntrustedProgramV1(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*testing.T, *nftStartupFixtureDataV1)
	}{
		{"missing_manifest", func(t *testing.T, f *nftStartupFixtureDataV1) {
			if err := os.Remove(f.manifest); err != nil {
				t.Fatal(err)
			}
		}},
		{"changed_date", func(t *testing.T, f *nftStartupFixtureDataV1) {
			f.program.Decision.GameActivatedAt++
			nftStartupWriteProgramV1(t, f, false)
		}},
		{"quorum_short", func(t *testing.T, f *nftStartupFixtureDataV1) {
			f.program.Approvals = f.program.Approvals[:1]
			nftStartupWriteProgramV1(t, f, false)
		}},
		{"signed_other_network", func(t *testing.T, f *nftStartupFixtureDataV1) {
			f.trust.NetworkID = "other-network"
			f.program.Decision.NetworkID = f.trust.NetworkID
			nftStartupWriteProgramV1(t, f, true)
		}},
		{"signed_other_genesis", func(t *testing.T, f *nftStartupFixtureDataV1) {
			f.trust.LegacyGenesisHex = strings.Repeat("ef", 32)
			f.program.Decision.LegacyGenesisHex = f.trust.LegacyGenesisHex
			nftStartupWriteProgramV1(t, f, true)
		}},
		{"different_campaign", func(t *testing.T, f *nftStartupFixtureDataV1) {
			f.program.Decision.CampaignID = "other-campaign"
			nftStartupWriteProgramV1(t, f, true)
		}},
		{"broken_chain", func(t *testing.T, f *nftStartupFixtureDataV1) {
			if err := os.WriteFile(f.cfg.ChainFile, []byte("not gob"), 0600); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := nftStartupFixtureV1(t)
			tc.mutate(t, f)
			before, err := os.ReadFile(f.cfg.ChainFile)
			if err != nil {
				t.Fatal(err)
			}
			r, err := bootstrapWithNFTStartupV1(f.cfg, true, &f.release, f.snapshotDir, &f.trust, f.manifest)
			if err == nil || r != nil {
				t.Fatal("invalid startup published runtime")
			}
			after, err := os.ReadFile(f.cfg.ChainFile)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("failed startup changed file")
			}
		})
	}
	f := nftStartupFixtureV1(t)
	if r, err := bootstrapWithNFTStartupV1(f.cfg, true, &f.release, f.snapshotDir, nil, f.manifest); err == nil || r != nil {
		t.Fatal("unfinalized NFT trust started NFT runtime")
	}
	if r, err := bootstrapWithNFTStartupV1(f.cfg, true, nil, f.snapshotDir, &f.trust, f.manifest); err == nil || r != nil {
		t.Fatal("program manifest activated unsigned hardfork")
	}
}

func TestNFTStartupFreshContextAndDisabledLegacyDefaultsV1(t *testing.T) {
	f := nftStartupFixtureV1(t)
	if err := os.Remove(f.cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	r, err := bootstrapWithNFTStartupV1(f.cfg, true, &f.release, f.snapshotDir, &f.trust, f.manifest)
	if err != nil || r == nil || r.Blockchain.BestHeightSnapshot() != 0 || !r.NFTGameProgramSnapshotV1().Verified {
		t.Fatalf("fresh trusted genesis: %v", err)
	}
	if _, err := os.Stat(f.cfg.ChainFile); !os.IsNotExist(err) {
		t.Fatal("fresh bootstrap wrote chain")
	}
	cfg := defaultRuntimePolicyConfigV1(t)
	cfg.BonusFile = filepath.Join(t.TempDir(), "bonus.json")
	t.Setenv("QC_ENABLE_CHAIN_NFT_MINT", "1")
	t.Setenv("QC_NFT_GAME_ACTIVATED_AT", "1")
	t.Setenv("QC_NFT_PROGRAM_MANIFEST", f.manifest)
	legacy, err := BootstrapWithConfig(cfg, true)
	if err != nil || legacy == nil {
		t.Fatalf("disabled legacy default: %v", err)
	}
	if legacy.NFTGameProgramSnapshotV1().Verified || legacy.Blockchain.NFTAwardTrustBoundSnapshotV1() || legacy.Blockchain.ProtocolAtHeightV1(1) != consensus.ProtocolLegacyV1 {
		t.Fatal("environment/manifest enabled NFT program")
	}
}
