package node

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/consensus"
)

func TestNFTGameBridgeHTTPReadsVerifiedStartupProgramV1(t *testing.T) {
	f := nftStartupFixtureV1(t)
	r, err := bootstrapWithNFTStartupV1(f.cfg, true, &f.release, f.snapshotDir, &f.trust, f.manifest)
	if err != nil {
		t.Fatal(err)
	}
	// Gob bytes are not a state fingerprint: map entry order is not fixed.
	// Compare detached state by contents and keep byte equality for disk only.
	chainBefore := r.Blockchain.StateSnapshot()
	ownershipBefore, err := r.Blockchain.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil {
		t.Fatal(err)
	}
	activationBefore := r.Blockchain.ActivationPolicySnapshotV1()
	programBefore := r.NFTGameProgramSnapshotV1()
	fileBefore, err := os.ReadFile(f.cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	r.ServeNFTGameProgramV1(rec, httptest.NewRequest(http.MethodGet, "/api/nft/monthly/program", nil))
	var p nftGameProgramHTTPResponseV1
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 || !p.Success || !p.Verified || !p.MintProtocolActive || p.Version != NFTGameProgramCapabilityVersionV1 || p.GameActivatedAt != f.program.Decision.GameActivatedAt || p.NetworkID != f.trust.NetworkID || p.LegacyGenesisHex != f.trust.LegacyGenesisHex || p.CampaignID != f.program.Decision.CampaignID {
		t.Fatalf("actual verified runtime response: %d %s", rec.Code, rec.Body)
	}
	if len(p.AwardPeriods) != 12 || p.AwardCount != 12 || p.MaxAwardsPerMonth != 1 || p.FirstAwardPeriod != f.record.AwardPeriod || p.AwardPeriods[0] != p.FirstAwardPeriod {
		t.Fatal("HTTP program changed monthly rule")
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("program may be cached")
	}
	for _, forbidden := range []string{"PublicKey", "AdminQuorum", "Approvals", "Signature", "PrivateKey"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatal("HTTP program exposed trust/signing material")
		}
	}
	fileAfter, err := os.ReadFile(f.cfg.ChainFile)
	if err != nil {
		t.Fatalf("read persisted chain after program read: %v", err)
	}
	if !bytes.Equal(fileBefore, fileAfter) {
		t.Fatal("program read changed persisted chain bytes")
	}
	chainAfter := r.Blockchain.StateSnapshot()
	if !reflect.DeepEqual(chainBefore.Blocks, chainAfter.Blocks) {
		t.Fatal("program read changed canonical blocks")
	}
	if !reflect.DeepEqual(chainBefore.UTXO, chainAfter.UTXO) {
		t.Fatal("program read changed canonical UTXO contents")
	}
	if !reflect.DeepEqual(chainBefore.PendingTransactions, chainAfter.PendingTransactions) {
		t.Fatal("program read changed pending transactions")
	}
	if chainBefore.TotalSupply != chainAfter.TotalSupply || chainBefore.CoinbaseMaturity != chainAfter.CoinbaseMaturity {
		t.Fatal("program read changed supply or maturity")
	}
	ownershipAfter, err := r.Blockchain.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil || !reflect.DeepEqual(ownershipBefore, ownershipAfter) {
		t.Fatalf("program read changed canonical NFT ownership: %v", err)
	}
	if r.Blockchain.ActivationPolicySnapshotV1() != activationBefore || r.NFTGameProgramSnapshotV1() != programBefore {
		t.Fatal("program read changed verified activation or game program")
	}
	// Windows bridge tests consume these exact bytes from the real Go runtime
	// handler, rather than inventing a second JSON contract in the Python suite.
	if path := os.Getenv("QMA0066_NFT_PROGRAM_CONTRACT_PATH"); path != "" {
		if err := os.WriteFile(path, rec.Body.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNFTGameBridgeHTTPRejectsMissingContextAndMutationV1(t *testing.T) {
	f := nftStartupFixtureV1(t)
	r, err := bootstrapWithNFTStartupV1(f.cfg, true, &f.release, f.snapshotDir, &f.trust, f.manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		rec := httptest.NewRecorder()
		r.ServeNFTGameProgramV1(rec, httptest.NewRequest(method, "/api/nft/monthly/program", nil))
		if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != http.MethodGet {
			t.Fatal("read endpoint allowed mutation")
		}
	}
	// A runtime with a formerly verified game program must not report that
	// program against a different/unbound chain installed underneath it.
	r.Blockchain = blockchain.NewBlockchain(f.cfg.InitialReward, f.cfg.TotalSupply)
	rec := httptest.NewRecorder()
	r.ServeNFTGameProgramV1(rec, httptest.NewRequest(http.MethodGet, "/api/nft/monthly/program", nil))
	if rec.Code != 503 || strings.Contains(rec.Body.String(), `"verified":true`) {
		t.Fatal("unbound chain advertised verified program")
	}
	legacy := NewRuntime(r.Blockchain, f.cfg)
	rec = httptest.NewRecorder()
	legacy.ServeNFTGameProgramV1(rec, httptest.NewRequest(http.MethodGet, "/api/nft/monthly/program", nil))
	var p nftGameProgramHTTPResponseV1
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 || p.Verified || p.GameActivatedAt != 0 || len(p.AwardPeriods) != 0 || p.AwardCount != 0 {
		t.Fatal("legacy default advertised active program")
	}
	if consensus.CanonicalNFTAwardTrustV1() != nil {
		t.Fatal("fixture installed real authorities")
	}
}
