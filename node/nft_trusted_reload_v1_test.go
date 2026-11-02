package node

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"quantumcoin/blockchain"
	"quantumcoin/consensus"
	"quantumcoin/wallet"
)

func nftTrustedReloadFixtureV1(t *testing.T) (*Runtime, *blockchain.Blockchain, blockchain.NFTMintRecordV1, blockchain.NFTAwardProofV1) {
	t.Helper()
	cfg := defaultRuntimePolicyConfigV1(t)
	trusted := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	trusted.SetCoinbaseMaturity(cfg.CoinbaseMaturity)
	network := "test-only-trusted-reload"
	genesis := hex.EncodeToString(trusted.Blocks[0].Hash)
	activation := verifiedActivationFixtureV1(t, consensus.ActivationPolicyV1{Enabled: true, Plan: consensus.ActivationPlan{
		NetworkID: network, LegacyGenesisHex: genesis, ActivationHeight: 1, TargetProtocol: consensus.ProtocolNFTV4, PreserveHistory: true, PreserveBalances: true}})
	trust := consensus.NFTAwardTrustV1{NetworkID: network, LegacyGenesisHex: genesis, AdminQuorum: 2}
	var keys []*ecdsa.PrivateKey
	for i := 0; i < 2; i++ {
		key, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		pub, err := consensus.UncompressedPublicKeyV1(key)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, key)
		trust.AdminPublicKeys = append(trust.AdminPublicKeys, pub)
	}
	program := consensus.NFTMonthlyProgramPolicyV1{NetworkID: network, CampaignID: "telegram-monthly", FirstAwardPeriod: time.Now().UTC().Format("2006-01")}
	bind := func(bc *blockchain.Blockchain) {
		if err := bc.BindActivationPolicySynchronizedV1(activation); err != nil {
			t.Fatal(err)
		}
		if err := bc.BindLocalNFTAwardTrustSynchronizedV1(network, trust); err != nil {
			t.Fatal(err)
		}
		if err := bc.BindLocalNFTMonthlyProgramSynchronizedV1(program); err != nil {
			t.Fatal(err)
		}
		bc.SetCoinbaseMaturity(cfg.CoinbaseMaturity)
	}
	bind(trusted)
	producer := blockchain.DeserializeBlockchain(blockchain.SerializeBlockchain(trusted))
	bind(producer)
	record := blockchain.NFTMintRecordV1{NetworkID: network, CampaignID: program.CampaignID, AwardPeriod: program.FirstAwardPeriod, Serial: 1, RecipientPubKeyHash: bytes.Repeat([]byte{0x11}, 20), MetadataHash: [32]byte{0x44}}
	commitment, err := blockchain.NFTMintRecordDigestV1(record)
	if err != nil {
		t.Fatal(err)
	}
	proof := blockchain.NFTAwardProofV1{Decision: consensus.NFTAwardDecisionV1{NetworkID: network, LegacyGenesisHex: genesis, RecordDigest: commitment}}
	digest, err := consensus.NFTAwardDecisionDigestV1(proof.Decision)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
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
	difficulty, err := producer.ExpectedNextDifficultyV1()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := producer.MineBlockWithNFTAwardSynchronizedV1(wallet.NewWallet().GetAddress(), difficulty, record, proof); err != nil {
		t.Fatal(err)
	}
	if err := producer.SaveToFile(cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	runtime := NewRuntime(trusted, cfg)
	runtime.SetAPIReadOnly(true)
	return runtime, producer, record, proof
}

func TestNFTTrustedReloadAcceptsApprovedPersistedChainV1(t *testing.T) {
	runtime, producer, _, _ := nftTrustedReloadFixtureV1(t)
	reloaded, err := runtime.ReloadChainFromDiskIfChanged()
	if err != nil || !reloaded {
		t.Fatalf("QMA0068_RELOAD_RED: actual runtime rejected locally approved persisted NFT chain: reloaded=%v error=%v", reloaded, err)
	}
	loaded, _ := runtime.Snapshot()
	block := loaded.AllBlocksSnapshot()[1]
	want := producer.AllBlocksSnapshot()[1]
	if !bytes.Equal(block.Hash, want.Hash) || !bytes.Equal(block.NFTMintRecordsV1[0].RecipientPubKeyHash, want.NFTMintRecordsV1[0].RecipientPubKeyHash) || !loaded.IsValidChainSynchronized() {
		t.Fatal("actual runtime reload lost approved block/recipient/policy")
	}
	if again, err := runtime.ReloadChainFromDiskIfChanged(); err != nil || again {
		t.Fatalf("unchanged file reloaded again: %v %v", again, err)
	}
}

func TestNFTTrustedReloadSerializesAndPreservesRuntimeOnFailureV1(t *testing.T) {
	runtime, _, _, _ := nftTrustedReloadFixtureV1(t)
	type outcome struct {
		changed bool
		err     error
	}
	start := make(chan struct{})
	results := make(chan outcome, 2)
	var workers sync.WaitGroup
	for i := 0; i < 2; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			changed, err := runtime.ReloadChainFromDiskIfChanged()
			results <- outcome{changed, err}
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	changed := 0
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.changed {
			changed++
		}
	}
	if changed != 1 {
		t.Fatalf("same file runtime installs=%d want=1", changed)
	}
	before, _ := runtime.Snapshot()
	stamp, size := runtime.ChainLastModifiedAt, runtime.ChainLastSize
	if err := os.WriteFile(runtime.Config.ChainFile, []byte("corrupt replacement chain file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if replaced, err := runtime.ReloadChainFromDiskIfChanged(); err == nil || replaced {
		t.Fatal("corrupt file replaced the active runtime")
	}
	after, _ := runtime.Snapshot()
	if before != after || !runtime.ChainLastModifiedAt.Equal(stamp) || runtime.ChainLastSize != size || !after.IsValidChainSynchronized() {
		t.Fatal("failed reload changed the chain/cache or lost local NFT policies")
	}
}
