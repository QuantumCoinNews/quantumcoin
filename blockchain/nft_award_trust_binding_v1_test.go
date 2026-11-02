package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	qconsensus "quantumcoin/consensus"
)

// Ephemeral keys and test-only periods are not production program settings.
func nftLocalTrustFixtureV1(t *testing.T) (*Blockchain, NFTMintRecordV1, qconsensus.NFTAwardDecisionV1, qconsensus.NFTAwardTrustV1, []qconsensus.NFTAwardApprovalV1) {
	t.Helper()
	chain := NewBlockchain(CanonicalInitialMiningRewardV1, int(QCTotalSupplyHardCap))
	record := NFTMintRecordV1{NetworkID: "test-only-bound-nft", CampaignID: "telegram-monthly", AwardPeriod: "2026-10", Serial: 1, RecipientPubKeyHash: bytes.Repeat([]byte{0x11}, 20), MetadataHash: [32]byte{0x44}}
	commitment, err := NFTMintRecordDigestV1(record)
	if err != nil {
		t.Fatal(err)
	}
	decision := qconsensus.NFTAwardDecisionV1{NetworkID: record.NetworkID, LegacyGenesisHex: hex.EncodeToString(chain.Blocks[0].Hash), RecordDigest: commitment}
	digest, err := qconsensus.NFTAwardDecisionDigestV1(decision)
	if err != nil {
		t.Fatal(err)
	}
	trust := qconsensus.NFTAwardTrustV1{NetworkID: record.NetworkID, LegacyGenesisHex: decision.LegacyGenesisHex, AdminQuorum: 2}
	var approvals []qconsensus.NFTAwardApprovalV1
	for i := 0; i < 2; i++ {
		key, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		pub, err := qconsensus.UncompressedPublicKeyV1(key)
		if err != nil {
			t.Fatal(err)
		}
		sig, err := qconsensus.SignTransactionDigestV1(key, digest)
		if err != nil {
			t.Fatal(err)
		}
		trust.AdminPublicKeys = append(trust.AdminPublicKeys, pub)
		approvals = append(approvals, qconsensus.NFTAwardApprovalV1{PublicKey: append([]byte(nil), pub...), Signature: sig})
	}
	return chain, record, decision, trust, approvals
}

func TestNFTLocalTrustRequiresBindingAndAcceptsAuthorizedApprovalV1(t *testing.T) {
	chain, record, decision, trust, approvals := nftLocalTrustFixtureV1(t)
	if err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(record, decision, approvals); !errors.Is(err, ErrNFTAwardTrustUnboundV1) {
		t.Fatalf("unbound policy: %v", err)
	}
	trust.LegacyGenesisHex = strings.ToUpper(trust.LegacyGenesisHex)
	if err := chain.BindLocalNFTAwardTrustSynchronizedV1(record.NetworkID, trust); err != nil {
		t.Fatal(err)
	}
	if !chain.NFTAwardTrustBoundSnapshotV1() {
		t.Fatal("local binding was not published")
	}
	if err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(record, decision, approvals); err != nil {
		t.Fatalf("authorized approval rejected: %v", err)
	}
	if err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(record, decision, approvals[:1]); err == nil {
		t.Fatal("missing quorum accepted")
	}
	changed := record
	changed.RecipientPubKeyHash = bytes.Repeat([]byte{0x22}, 20)
	if err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(changed, decision, approvals); err == nil {
		t.Fatal("changed recipient accepted")
	}
	if err := chain.BindLocalNFTAwardTrustSynchronizedV1(record.NetworkID, trust); err == nil {
		t.Fatal("local binding replaced")
	}
}

func TestNFTLocalTrustRejectsInvalidPolicyWithoutConsumingBindingV1(t *testing.T) {
	cases := []struct {
		name    string
		change  func(*qconsensus.NFTAwardTrustV1)
		network string
	}{
		{"empty_authorities", func(p *qconsensus.NFTAwardTrustV1) { p.AdminPublicKeys = nil }, ""},
		{"zero_quorum", func(p *qconsensus.NFTAwardTrustV1) { p.AdminQuorum = 0 }, ""},
		{"impossible_quorum", func(p *qconsensus.NFTAwardTrustV1) { p.AdminQuorum = 3 }, ""},
		{"duplicate_authority", func(p *qconsensus.NFTAwardTrustV1) { p.AdminPublicKeys[1] = p.AdminPublicKeys[0] }, ""},
		{"invalid_authority", func(p *qconsensus.NFTAwardTrustV1) { p.AdminPublicKeys[0] = []byte{1, 2} }, ""},
		{"wrong_genesis", func(p *qconsensus.NFTAwardTrustV1) { p.LegacyGenesisHex = strings.Repeat("00", 32) }, ""},
		{"invalid_genesis", func(p *qconsensus.NFTAwardTrustV1) { p.LegacyGenesisHex = "bad" }, ""},
		{"invalid_network", func(p *qconsensus.NFTAwardTrustV1) { p.NetworkID = "bad network" }, ""},
		{"wrong_local_network", func(p *qconsensus.NFTAwardTrustV1) {}, "different-local-network"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			chain, record, decision, trust, approvals := nftLocalTrustFixtureV1(t)
			invalid := cloneNFTAwardTrustV1(trust)
			tc.change(&invalid)
			network := record.NetworkID
			if tc.network != "" {
				network = tc.network
			}
			if err := chain.BindLocalNFTAwardTrustSynchronizedV1(network, invalid); err == nil {
				t.Fatal("invalid local trust accepted")
			}
			if chain.NFTAwardTrustBoundSnapshotV1() {
				t.Fatal("failed binding changed local state")
			}
			if err := chain.BindLocalNFTAwardTrustSynchronizedV1(record.NetworkID, trust); err != nil {
				t.Fatalf("valid retry rejected: %v", err)
			}
			if err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(record, decision, approvals); err != nil {
				t.Fatal(err)
			}
		})
	}
	var absent *Blockchain
	if err := absent.BindLocalNFTAwardTrustSynchronizedV1("test", qconsensus.NFTAwardTrustV1{}); !errors.Is(err, ErrNilBlockchain) {
		t.Fatalf("nil chain: %v", err)
	}
	if absent.NFTAwardTrustBoundSnapshotV1() {
		t.Fatal("nil chain has trust")
	}
	if err := absent.ValidateNFTAwardWithLocalTrustSynchronizedV1(NFTMintRecordV1{}, qconsensus.NFTAwardDecisionV1{}, nil); !errors.Is(err, ErrNilBlockchain) {
		t.Fatalf("nil validation: %v", err)
	}
	for _, blocks := range [][]*Block{nil, {nil}} {
		chain, _, _, trust, _ := nftLocalTrustFixtureV1(t)
		chain.Blocks = blocks
		if err := chain.BindLocalNFTAwardTrustSynchronizedV1(trust.NetworkID, trust); err == nil {
			t.Fatal("missing genesis accepted")
		}
	}
}

func TestNFTLocalTrustCopiesCallerKeysV1(t *testing.T) {
	chain, record, decision, trust, approvals := nftLocalTrustFixtureV1(t)
	if err := chain.BindLocalNFTAwardTrustSynchronizedV1(record.NetworkID, trust); err != nil {
		t.Fatal(err)
	}
	trust.AdminPublicKeys[0][0] ^= 1
	trust.AdminPublicKeys[1] = nil
	trust.AdminQuorum = 0
	trust.NetworkID = "changed-by-caller"
	if err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(record, decision, approvals); err != nil {
		t.Fatalf("caller mutation changed bound policy: %v", err)
	}
}

func TestNFTLocalTrustNotImportedFromPeerOrDiskV1(t *testing.T) {
	chain, record, decision, trust, approvals := nftLocalTrustFixtureV1(t)
	if err := chain.BindLocalNFTAwardTrustSynchronizedV1(record.NetworkID, trust); err != nil {
		t.Fatal(err)
	}
	assertUnbound := func(t *testing.T, loaded *Blockchain) {
		t.Helper()
		if loaded.NFTAwardTrustBoundSnapshotV1() {
			t.Fatal("serialized policy became trusted")
		}
		if err := loaded.ValidateNFTAwardWithLocalTrustSynchronizedV1(record, decision, approvals); !errors.Is(err, ErrNFTAwardTrustUnboundV1) {
			t.Fatalf("import adopted trust: %v", err)
		}
		if err := loaded.BindLocalNFTAwardTrustSynchronizedV1(record.NetworkID, trust); err != nil {
			t.Fatalf("explicit local rebind: %v", err)
		}
		if err := loaded.ValidateNFTAwardWithLocalTrustSynchronizedV1(record, decision, approvals); err != nil {
			t.Fatalf("local trust after import: %v", err)
		}
	}
	t.Run("real_gob_roundtrip", func(t *testing.T) { assertUnbound(t, DeserializeBlockchain(SerializeBlockchain(chain))) })
	t.Run("file_load", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "nft-local-trust.gob")
		if err := chain.SaveToFile(name); err != nil {
			t.Fatal(err)
		}
		loaded, err := LoadBlockchainFromFile(name)
		if err != nil {
			t.Fatal(err)
		}
		assertUnbound(t, loaded)
	})
	t.Run("forged_exported_peer_policy", func(t *testing.T) {
		// Unknown exported fields in a peer's gob schema must not initialize
		// the real Blockchain's private runtime authority state.
		payload := struct {
			Blocks        []*Block
			UTXO          map[string][]TransactionOutput
			TotalSupply   int
			NFTAwardTrust qconsensus.NFTAwardTrustV1
		}{chain.AllBlocksSnapshot(), chain.UTXO, chain.TotalSupply, trust}
		var data bytes.Buffer
		if err := gob.NewEncoder(&data).Encode(payload); err != nil {
			t.Fatal(err)
		}
		assertUnbound(t, DeserializeBlockchain(data.Bytes()))
	})
}

func TestNFTLocalTrustRejectsSelfDeclaredSignersV1(t *testing.T) {
	chain, record, decision, trust, approvals := nftLocalTrustFixtureV1(t)
	if err := chain.BindLocalNFTAwardTrustSynchronizedV1(record.NetworkID, trust); err != nil {
		t.Fatal(err)
	}
	_, _, _, otherTrust, otherApprovals := nftLocalTrustFixtureV1(t)
	// Fresh fixture has the same canonical genesis/record, but different keys.
	if err := ValidateNFTAwardRecordApprovalV1(record, decision, otherTrust, otherApprovals); err != nil {
		t.Fatalf("self-declared valid fixture: %v", err)
	}
	if err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(record, decision, otherApprovals); err == nil {
		t.Fatal("self-declared peer authorities accepted")
	}
	if err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(record, decision, approvals); err != nil {
		t.Fatalf("attack changed local trust: %v", err)
	}
}

func TestNFTLocalTrustConcurrentBindAndReadV1(t *testing.T) {
	chain, record, decision, trust, approvals := nftLocalTrustFixtureV1(t)
	var bound int32
	var unexpected int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := chain.BindLocalNFTAwardTrustSynchronizedV1(record.NetworkID, trust); err == nil {
				atomic.AddInt32(&bound, 1)
			}
		}()
	}
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = chain.NFTAwardTrustBoundSnapshotV1()
				err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(record, decision, approvals)
				if err != nil && !errors.Is(err, ErrNFTAwardTrustUnboundV1) {
					atomic.AddInt32(&unexpected, 1)
				}
			}
		}()
	}
	wg.Wait()
	if bound != 1 || unexpected != 0 {
		t.Fatalf("bind count=%d unexpected reads=%d", bound, unexpected)
	}
	if err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(record, decision, approvals); err != nil {
		t.Fatal(err)
	}
}

func TestNFTLocalTrustDoesNotEnableMintOrBlockAdmissionV1(t *testing.T) {
	chain, record, decision, trust, approvals := nftLocalTrustFixtureV1(t)
	before := chain.AllBlocksSnapshot()
	supply := chain.TotalSupply
	if err := chain.BindLocalNFTAwardTrustSynchronizedV1(record.NetworkID, trust); err != nil {
		t.Fatal(err)
	}
	if err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(record, decision, approvals); err != nil {
		t.Fatal(err)
	}
	if id, err := chain.MintNFT("QC_TEST", "rare", nil); id != "" || !errors.Is(err, ErrNFTMintDisabled) {
		t.Fatalf("mint unexpectedly enabled: %q %v", id, err)
	}
	if err := validateNFTMintAuthorizationV1(&Block{NFTMintRecordsV1: []NFTMintRecordV1{record}}, qconsensus.ProtocolNFTV4); err == nil {
		t.Fatal("block admitted without integrated authorization proof")
	}
	// Both sides use the same snapshot path: no RAM-vs-gob nil/empty or
	// serialized map ordering differences enter this comparison.
	if !reflect.DeepEqual(before, chain.AllBlocksSnapshot()) || chain.TotalSupply != supply {
		t.Fatal("binding mutated canonical chain")
	}
}

func TestNFTLocalTrustRejectsChangedChainGenesisV1(t *testing.T) {
	chain, record, decision, trust, approvals := nftLocalTrustFixtureV1(t)
	if err := chain.BindLocalNFTAwardTrustSynchronizedV1(record.NetworkID, trust); err != nil {
		t.Fatal(err)
	}
	chain.Blocks[0].Hash[0] ^= 1
	if err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(record, decision, approvals); err == nil {
		t.Fatal("approval accepted after local genesis changed")
	}
}
