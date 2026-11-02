package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	qconsensus "quantumcoin/consensus"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func nftMonthlyPolicyFixtureV1(t *testing.T) (*Blockchain, qconsensus.NFTMonthlyProgramPolicyV1, []*ecdsa.PrivateKey) {
	t.Helper()
	chain := NewBlockchain(1, 0)
	policy, err := qconsensus.NewNFTMonthlyProgramPolicyFromGameActivationV1("test-only-monthly", "telegram-monthly", time.Date(2031, time.November, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	trust := qconsensus.NFTAwardTrustV1{NetworkID: policy.NetworkID, LegacyGenesisHex: hex.EncodeToString(chain.Blocks[0].Hash), AdminQuorum: 2}
	var keys []*ecdsa.PrivateKey
	for i := 0; i < 2; i++ {
		key, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		pub, err := qconsensus.UncompressedPublicKeyV1(key)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, key)
		trust.AdminPublicKeys = append(trust.AdminPublicKeys, pub)
	}
	if err := chain.BindLocalNFTAwardTrustSynchronizedV1(policy.NetworkID, trust); err != nil {
		t.Fatal(err)
	}
	return chain, policy, keys
}

func nftMonthlyPolicyRecordV1(p qconsensus.NFTMonthlyProgramPolicyV1, offset int) NFTMintRecordV1 {
	first, _ := time.Parse("2006-01", p.FirstAwardPeriod)
	return NFTMintRecordV1{NetworkID: p.NetworkID, CampaignID: p.CampaignID, AwardPeriod: first.AddDate(0, offset, 0).Format("2006-01"), Serial: uint32(offset + 1), RecipientPubKeyHash: bytes.Repeat([]byte{0x11}, 20), MetadataHash: [32]byte{0x44}}
}

func nftMonthlyPolicyBlockV1(t *testing.T, chain *Blockchain, records []NFTMintRecordV1, keys []*ecdsa.PrivateKey) *Block {
	t.Helper()
	var proofs []NFTAwardProofV1
	for _, record := range records {
		commitment, err := NFTMintRecordDigestV1(record)
		if err != nil {
			t.Fatal(err)
		}
		d := qconsensus.NFTAwardDecisionV1{NetworkID: record.NetworkID, LegacyGenesisHex: hex.EncodeToString(chain.Blocks[0].Hash), RecordDigest: commitment}
		digest, err := qconsensus.NFTAwardDecisionDigestV1(d)
		if err != nil {
			t.Fatal(err)
		}
		proof := NFTAwardProofV1{Decision: d}
		for _, key := range keys {
			pub, err := qconsensus.UncompressedPublicKeyV1(key)
			if err != nil {
				t.Fatal(err)
			}
			sig, err := qconsensus.SignTransactionDigestV1(key, digest)
			if err != nil {
				t.Fatal(err)
			}
			proof.Approvals = append(proof.Approvals, qconsensus.NFTAwardApprovalV1{PublicKey: pub, Signature: sig})
		}
		proofs = append(proofs, proof)
	}
	last := chain.Blocks[len(chain.Blocks)-1]
	block, err := NewNFTBlockWithAwardProofV1(last.Index+1, nil, last.Hash, "test-only-monthly-miner", 1, records, proofs)
	if err != nil {
		t.Fatal(err)
	}
	return block
}

func TestNFTMonthlyProgramRequiresLocalBindingAndRejectsRebindV1(t *testing.T) {
	chain, p, keys := nftMonthlyPolicyFixtureV1(t)
	candidate := nftMonthlyPolicyBlockV1(t, chain, []NFTMintRecordV1{nftMonthlyPolicyRecordV1(p, 0)}, keys)
	if err := chain.ValidateNFTMonthlyAwardBlockSynchronizedV1(candidate); !errors.Is(err, ErrNFTMonthlyProgramUnboundV1) {
		t.Fatalf("unbound: %v", err)
	}
	invalid := p
	invalid.NetworkID = "different-network"
	if err := chain.BindLocalNFTMonthlyProgramSynchronizedV1(invalid); err == nil {
		t.Fatal("wrong-network program bound")
	}
	if err := chain.BindLocalNFTMonthlyProgramSynchronizedV1(p); err != nil {
		t.Fatal(err)
	}
	if err := chain.ValidateNFTMonthlyAwardBlockSynchronizedV1(candidate); err != nil {
		t.Fatal(err)
	}
	if err := chain.BindLocalNFTMonthlyProgramSynchronizedV1(p); err == nil {
		t.Fatal("program rebound")
	}
	var absent *Blockchain
	if err := absent.BindLocalNFTMonthlyProgramSynchronizedV1(p); !errors.Is(err, ErrNilBlockchain) {
		t.Fatal(err)
	}
	if err := absent.ValidateNFTMonthlyAwardBlockSynchronizedV1(candidate); !errors.Is(err, ErrNilBlockchain) {
		t.Fatal(err)
	}
	fresh := NewBlockchain(1, 0)
	if err := fresh.BindLocalNFTMonthlyProgramSynchronizedV1(p); !errors.Is(err, ErrNFTAwardTrustUnboundV1) {
		t.Fatalf("program trusted without local keys: %v", err)
	}
	if err := chain.ValidateNFTMonthlyAwardBlockSynchronizedV1(nil); err == nil {
		t.Fatal("nil candidate accepted")
	}
}

func TestNFTMonthlyProgramAcceptsTwelveAndRejectsThirteenthV1(t *testing.T) {
	chain, p, keys := nftMonthlyPolicyFixtureV1(t)
	if err := chain.BindLocalNFTMonthlyProgramSynchronizedV1(p); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		block := nftMonthlyPolicyBlockV1(t, chain, []NFTMintRecordV1{nftMonthlyPolicyRecordV1(p, i)}, keys)
		if err := chain.ValidateNFTMonthlyAwardBlockSynchronizedV1(block); err != nil {
			t.Fatalf("month %d rejected: %v", i+1, err)
		}
		chain.Blocks = append(chain.Blocks, block) // Test-only history; no live admission.
	}
	extra := nftMonthlyPolicyBlockV1(t, chain, []NFTMintRecordV1{nftMonthlyPolicyRecordV1(p, 12)}, keys)
	if err := chain.ValidateNFTMonthlyAwardBlockSynchronizedV1(extra); err == nil {
		t.Fatal("thirteenth monthly award accepted")
	}
}

func TestNFTMonthlyProgramRejectsDuplicatePeriodAcrossAndWithinBlocksV1(t *testing.T) {
	chain, p, keys := nftMonthlyPolicyFixtureV1(t)
	if err := chain.BindLocalNFTMonthlyProgramSynchronizedV1(p); err != nil {
		t.Fatal(err)
	}
	record := nftMonthlyPolicyRecordV1(p, 0)
	first := nftMonthlyPolicyBlockV1(t, chain, []NFTMintRecordV1{record}, keys)
	chain.Blocks = append(chain.Blocks, first) // Test-only history.
	record.RecipientPubKeyHash = bytes.Repeat([]byte{0x22}, 20)
	record.MetadataHash[0] = 0x55
	duplicate := nftMonthlyPolicyBlockV1(t, chain, []NFTMintRecordV1{record}, keys)
	if err := chain.ValidateNFTMonthlyAwardBlockSynchronizedV1(duplicate); err == nil {
		t.Fatal("second recipient in same month accepted")
	}
	next := nftMonthlyPolicyRecordV1(p, 1)
	twice := nftMonthlyPolicyBlockV1(t, chain, []NFTMintRecordV1{next, next}, keys)
	if err := chain.ValidateNFTMonthlyAwardBlockSynchronizedV1(twice); err == nil {
		t.Fatal("two awards for same month in one block accepted")
	}
}

func TestNFTMonthlyProgramRejectsValidlySignedWrongProgramAwardsV1(t *testing.T) {
	chain, p, keys := nftMonthlyPolicyFixtureV1(t)
	if err := chain.BindLocalNFTMonthlyProgramSynchronizedV1(p); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		change func(*NFTMintRecordV1)
	}{
		{"wrong_network", func(r *NFTMintRecordV1) { r.NetworkID = "other-network" }},
		{"wrong_campaign", func(r *NFTMintRecordV1) { r.CampaignID = "other-campaign" }},
		{"before_program", func(r *NFTMintRecordV1) { r.AwardPeriod = "2031-11" }},
		{"after_program", func(r *NFTMintRecordV1) { r.AwardPeriod = "2032-12" }},
		{"wrong_serial", func(r *NFTMintRecordV1) { r.Serial = 2 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := nftMonthlyPolicyRecordV1(p, 0)
			tc.change(&r)
			block := nftMonthlyPolicyBlockV1(t, chain, []NFTMintRecordV1{r}, keys)
			if err := ValidateBlockPoWV4(block); err != nil {
				t.Fatal(err)
			}
			if err := chain.ValidateNFTMonthlyAwardBlockSynchronizedV1(block); err == nil {
				t.Fatal("signed award bypassed program rules")
			}
		})
	}
	valid := nftMonthlyPolicyBlockV1(t, chain, []NFTMintRecordV1{nftMonthlyPolicyRecordV1(p, 0)}, keys)
	if err := chain.ValidateNFTMonthlyAwardBlockSynchronizedV1(valid); err != nil {
		t.Fatal(err)
	}
	if err := validateNFTMintAuthorizationV1(valid, qconsensus.ProtocolNFTV4); err == nil {
		t.Fatal("preflight enabled live admission")
	}
}

func TestNFTMonthlyProgramNotImportedThroughGobV1(t *testing.T) {
	chain, p, keys := nftMonthlyPolicyFixtureV1(t)
	if err := chain.BindLocalNFTMonthlyProgramSynchronizedV1(p); err != nil {
		t.Fatal(err)
	}
	candidate := nftMonthlyPolicyBlockV1(t, chain, []NFTMintRecordV1{nftMonthlyPolicyRecordV1(p, 0)}, keys)
	loaded := DeserializeBlockchain(SerializeBlockchain(chain))
	if err := loaded.ValidateNFTMonthlyAwardBlockSynchronizedV1(candidate); !errors.Is(err, ErrNFTMonthlyProgramUnboundV1) {
		t.Fatalf("serialized program imported: %v", err)
	}
	trust := cloneNFTAwardTrustV1(chain.nftAwardTrust.trust)
	if err := loaded.BindLocalNFTAwardTrustSynchronizedV1(p.NetworkID, trust); err != nil {
		t.Fatal(err)
	}
	if err := loaded.BindLocalNFTMonthlyProgramSynchronizedV1(p); err != nil {
		t.Fatal(err)
	}
	if err := loaded.ValidateNFTMonthlyAwardBlockSynchronizedV1(candidate); err != nil {
		t.Fatal(err)
	}
}

func TestNFTMonthlyProgramConcurrentBindingAndPreflightV1(t *testing.T) {
	chain, p, keys := nftMonthlyPolicyFixtureV1(t)
	block := nftMonthlyPolicyBlockV1(t, chain, []NFTMintRecordV1{nftMonthlyPolicyRecordV1(p, 0)}, keys)
	var wg sync.WaitGroup
	var bound, unexpected int32
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := chain.BindLocalNFTMonthlyProgramSynchronizedV1(p); err == nil {
				atomic.AddInt32(&bound, 1)
			}
		}()
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := chain.ValidateNFTMonthlyAwardBlockSynchronizedV1(block)
			if err != nil && !errors.Is(err, ErrNFTMonthlyProgramUnboundV1) {
				atomic.AddInt32(&unexpected, 1)
			}
		}()
	}
	wg.Wait()
	if bound != 1 || unexpected != 0 {
		t.Fatalf("bind=%d unexpected=%d", bound, unexpected)
	}
}
