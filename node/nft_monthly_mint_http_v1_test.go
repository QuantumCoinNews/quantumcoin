package node

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/consensus"
	"quantumcoin/wallet"
)

func nftMintTransportPayloadV1(t *testing.T, record blockchain.NFTMintRecordV1, proof blockchain.NFTAwardProofV1) []byte {
	t.Helper()
	var p nftMonthlyMintHTTPRequestV1
	p.Version = NFTMonthlyMintRequestVersionV1
	p.Record.NetworkID, p.Record.CampaignID, p.Record.AwardPeriod, p.Record.Serial = record.NetworkID, record.CampaignID, record.AwardPeriod, record.Serial
	p.Record.RecipientPubKeyHash, p.Record.MetadataHash = hex.EncodeToString(record.RecipientPubKeyHash), hex.EncodeToString(record.MetadataHash[:])
	p.Decision.NetworkID, p.Decision.LegacyGenesisHex, p.Decision.RecordDigest = proof.Decision.NetworkID, proof.Decision.LegacyGenesisHex, hex.EncodeToString(proof.Decision.RecordDigest[:])
	for _, a := range proof.Approvals {
		p.Approvals = append(p.Approvals, struct {
			PublicKey string `json:"publicKey"`
			Signature string `json:"signature"`
		}{hex.EncodeToString(a.PublicKey), hex.EncodeToString(a.Signature)})
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func nftMintTransportCallV1(r *Runtime, body []byte, mint bool) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/nft/monthly/receipt", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	if mint {
		r.ServeNFTMonthlyMintV1(response, request)
	} else {
		r.ServeNFTMonthlyReceiptV1(response, request)
	}
	return response
}

func nftMintTransportReceiptV1(t *testing.T, response *httptest.ResponseRecorder) nftMonthlyMintHTTPReceiptV1 {
	t.Helper()
	var receipt nftMonthlyMintHTTPReceiptV1
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &receipt) != nil || !receipt.Success || !receipt.Persisted || receipt.Version != NFTMonthlyMintReceiptVersionV1 {
		t.Fatalf("no confirmed receipt: %d %s", response.Code, response.Body)
	}
	return receipt
}

// The underlying mint completes; the client receives a failed response write.
// A retry must reconcile the persisted canonical award instead of mining again.
type nftMintDisconnectedWriterV1 struct {
	header http.Header
	body   []byte
	status int
}

func (w *nftMintDisconnectedWriterV1) Header() http.Header    { return w.header }
func (w *nftMintDisconnectedWriterV1) WriteHeader(status int) { w.status = status }
func (w *nftMintDisconnectedWriterV1) Write(data []byte) (int, error) {
	w.body = append(w.body, data...)
	return 0, io.ErrClosedPipe
}

func TestNFTMintTransportHTTPPersistsReceiptAndReconcilesLostResponseV1(t *testing.T) {
	r, f, proof := nftDurableRuntimeFixtureV1(t)
	r.Config.RewardAddrMiner = wallet.NewWallet().GetAddress()
	body := nftMintTransportPayloadV1(t, f.record, proof)
	missing := nftMintTransportCallV1(r, body, false)
	if missing.Code != http.StatusNotFound || r.Blockchain.BestHeightSnapshot() != 0 {
		t.Fatalf("absent receipt: %d %s", missing.Code, missing.Body)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/nft/monthly/mint", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	lost := &nftMintDisconnectedWriterV1{header: make(http.Header)}
	r.ServeNFTMonthlyMintV1(lost, request)
	var first nftMonthlyMintHTTPReceiptV1
	if lost.status != http.StatusOK || json.Unmarshal(lost.body, &first) != nil || !first.Success || !first.Persisted || first.AlreadyMinted {
		t.Fatalf("failed response write prevented durable mint: %d %s", lost.status, lost.body)
	}
	state, file := r.Blockchain.StateSnapshot(), nftDurableRuntimeReadV1(t, f.cfg.ChainFile)
	id, err := blockchain.NFTTokenIDV1(f.record)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := blockchain.NFTMintRecordDigestV1(f.record)
	if err != nil {
		t.Fatal(err)
	}
	if first.TokenID != hex.EncodeToString(id[:]) || first.RecordDigest != hex.EncodeToString(digest[:]) || first.RecipientPubKeyHash != hex.EncodeToString(f.record.RecipientPubKeyHash) || first.MintHeight != 1 || first.ChainHeight != 1 || first.PersistedHeight != 1 || first.MintBlockHash != first.ChainTipHash || first.MintBlockHash != first.PersistedTipHash {
		t.Fatalf("receipt differs from actual canonical token/owner/block: %+v", first)
	}
	for _, mint := range []bool{false, true, true} {
		response := nftMintTransportCallV1(r, body, mint)
		receipt := nftMintTransportReceiptV1(t, response)
		if !receipt.AlreadyMinted || receipt.TokenID != first.TokenID || receipt.MintBlockHash != first.MintBlockHash || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("retry changed canonical receipt: %+v", receipt)
		}
		if strings.Contains(response.Body.String(), "publicKey") || strings.Contains(response.Body.String(), "signature") || strings.Contains(response.Body.String(), "private") {
			t.Fatal("receipt exposes signing material")
		}
	}
	if !reflect.DeepEqual(state, r.Blockchain.StateSnapshot()) || !bytes.Equal(file, nftDurableRuntimeReadV1(t, f.cfg.ChainFile)) {
		t.Fatal("HTTP lookup/retry changed chain or disk")
	}
	restarted, err := bootstrapWithNFTStartupV1(f.cfg, true, &f.release, f.snapshotDir, &f.trust, f.manifest)
	if err != nil {
		t.Fatal(err)
	}
	// Already minted confirmation needs no new coinbase destination.
	restarted.Config.RewardAddrMiner = ""
	if receipt := nftMintTransportReceiptV1(t, nftMintTransportCallV1(restarted, body, true)); !receipt.AlreadyMinted || receipt.MintBlockHash != first.MintBlockHash {
		t.Fatal("restart retry did not reconcile persisted ownership")
	}
	if filename := os.Getenv("QMA0066_NFT_MINT_CONTRACT_PATH"); filename != "" {
		contract, err := json.MarshalIndent(map[string]any{"request": json.RawMessage(body), "receipt": first}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, append(contract, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNFTMintTransportHTTPRejectsMalformedUnsignedAndChangedReplayV1(t *testing.T) {
	r, f, proof := nftDurableRuntimeFixtureV1(t)
	r.Config.RewardAddrMiner = wallet.NewWallet().GetAddress()
	body := nftMintTransportPayloadV1(t, f.record, proof)
	state, file := r.Blockchain.StateSnapshot(), nftDurableRuntimeReadV1(t, f.cfg.ChainFile)
	var original map[string]any
	if err := json.Unmarshal(body, &original); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"unknown_authorities", "private_key", "duplicate_key", "trailing_json", "oversized", "invalid_signature", "unknown_signer", "quorum_short", "missing_approvals", "changed_recipient", "changed_digest", "bad_media", "wrong_method", "missing_miner", "read_only", "missing_program"} {
		t.Run(name, func(t *testing.T) {
			var p map[string]any
			if err := json.Unmarshal(body, &p); err != nil {
				t.Fatal(err)
			}
			code := http.StatusUnprocessableEntity
			var input []byte
			switch name {
			case "unknown_authorities":
				p["authorities"] = []string{"untrusted"}
				code = http.StatusBadRequest
			case "private_key":
				p["privateKey"] = "not-accepted"
				code = http.StatusBadRequest
			case "duplicate_key":
				input = append([]byte(`{"version":"other",`), body[1:]...)
				code = http.StatusBadRequest
			case "trailing_json":
				input = append(append([]byte(nil), body...), []byte(" {}")...)
				code = http.StatusBadRequest
			case "oversized":
				input = bytes.Repeat([]byte(" "), nftMonthlyMintMaxBodyV1+1)
				code = http.StatusRequestEntityTooLarge
			case "invalid_signature":
				p["approvals"].([]any)[0].(map[string]any)["signature"] = "01010101"
			case "unknown_signer":
				p["approvals"].([]any)[0].(map[string]any)["publicKey"] = hex.EncodeToString(wallet.NewWallet().PublicKey)
			case "quorum_short":
				p["approvals"] = p["approvals"].([]any)[:1]
			case "missing_approvals":
				p["approvals"] = []any{}
			case "changed_recipient":
				p["record"].(map[string]any)["recipientPubKeyHash"] = strings.Repeat("77", 20)
			case "changed_digest":
				p["decision"].(map[string]any)["recordDigest"] = strings.Repeat("88", 32)
			case "bad_media":
				code = http.StatusUnsupportedMediaType
			case "wrong_method":
				code = http.StatusMethodNotAllowed
			case "missing_miner":
				code = http.StatusServiceUnavailable
			case "read_only":
				code = http.StatusForbidden
			case "missing_program":
				code = http.StatusServiceUnavailable
			}
			if input == nil {
				var err error
				input, err = json.Marshal(p)
				if err != nil {
					t.Fatal(err)
				}
			}
			target := r
			if name == "missing_program" {
				target = NewRuntime(r.Blockchain, r.Config)
			}
			if name == "missing_miner" {
				old := r.Config.RewardAddrMiner
				r.Config.RewardAddrMiner = ""
				defer func() { r.Config.RewardAddrMiner = old }()
			}
			if name == "read_only" {
				r.SetAPIReadOnly(true)
				defer r.SetAPIReadOnly(false)
			}
			request := httptest.NewRequest(http.MethodPost, "/api/nft/monthly/mint", bytes.NewReader(input))
			request.Header.Set("Content-Type", "application/json")
			if name == "bad_media" {
				request.Header.Set("Content-Type", "text/plain")
			}
			if name == "wrong_method" {
				request.Method = http.MethodGet
			}
			response := httptest.NewRecorder()
			target.ServeNFTMonthlyMintV1(response, request)
			if response.Code != code || !reflect.DeepEqual(state, r.Blockchain.StateSnapshot()) || !bytes.Equal(file, nftDurableRuntimeReadV1(t, f.cfg.ChainFile)) {
				t.Fatalf("invalid request changed state or returned unexpected status: %d want=%d %s", response.Code, code, response.Body)
			}
		})
	}
	nftMintTransportReceiptV1(t, nftMintTransportCallV1(r, body, true))
	state, file = r.Blockchain.StateSnapshot(), nftDurableRuntimeReadV1(t, f.cfg.ChainFile)
	changed := f.record
	changed.RecipientPubKeyHash = bytes.Repeat([]byte{0x77}, 20)
	// Token identity does not include owner; approval commitment does. A
	// freshly signed changed-owner replay must conflict, not return success.
	changedDigest, err := blockchain.NFTMintRecordDigestV1(changed)
	if err != nil {
		t.Fatal(err)
	}
	changedProof := blockchain.NFTAwardProofV1{Decision: proof.Decision}
	changedProof.Decision.RecordDigest = changedDigest
	decisionDigest, err := consensus.NFTAwardDecisionDigestV1(changedProof.Decision)
	if err != nil {
		t.Fatal(err)
	}
	for i, key := range f.keys {
		signature, err := consensus.SignTransactionDigestV1(key, decisionDigest)
		if err != nil {
			t.Fatal(err)
		}
		changedProof.Approvals = append(changedProof.Approvals, consensus.NFTAwardApprovalV1{PublicKey: proof.Approvals[i].PublicKey, Signature: signature})
	}
	for _, mint := range []bool{false, true} {
		response := nftMintTransportCallV1(r, nftMintTransportPayloadV1(t, changed, changedProof), mint)
		if response.Code != http.StatusConflict || !reflect.DeepEqual(state, r.Blockchain.StateSnapshot()) || !bytes.Equal(file, nftDurableRuntimeReadV1(t, f.cfg.ChainFile)) {
			t.Fatalf("changed-owner replay accepted: %d %s", response.Code, response.Body)
		}
		bad := proof
		bad.Approvals = nil
		if response := nftMintTransportCallV1(r, nftMintTransportPayloadV1(t, f.record, bad), mint); response.Code != http.StatusUnprocessableEntity {
			t.Fatalf("unsigned retry accepted: %d %s", response.Code, response.Body)
		}
	}
}

func TestNFTMintTransportHTTPRefusesRAMOnlyCorruptDiskAndWriteFailureV1(t *testing.T) {
	r, f, proof := nftDurableRuntimeFixtureV1(t)
	r.Config.RewardAddrMiner = wallet.NewWallet().GetAddress()
	body := nftMintTransportPayloadV1(t, f.record, proof)
	genesisFile := nftDurableRuntimeReadV1(t, f.cfg.ChainFile)
	difficulty, err := r.Blockchain.ExpectedNextDifficultyV1()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Blockchain.MineBlockWithNFTAwardSynchronizedV1(r.Config.RewardAddrMiner, difficulty, f.record, proof); err != nil {
		t.Fatal(err)
	}
	state := r.Blockchain.StateSnapshot()
	for _, file := range [][]byte{genesisFile, []byte("not a canonical gob chain")} {
		if err := os.WriteFile(f.cfg.ChainFile, file, 0600); err != nil {
			t.Fatal(err)
		}
		for _, mint := range []bool{false, true} {
			response := nftMintTransportCallV1(r, body, mint)
			if response.Code != http.StatusServiceUnavailable || !reflect.DeepEqual(state, r.Blockchain.StateSnapshot()) || !bytes.Equal(file, nftDurableRuntimeReadV1(t, f.cfg.ChainFile)) {
				t.Fatalf("RAM-only/corrupt disk claimed persistence: %d %s", response.Code, response.Body)
			}
		}
	}
	if err := r.Blockchain.SaveToFile(f.cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	if receipt := nftMintTransportReceiptV1(t, nftMintTransportCallV1(r, body, false)); !receipt.AlreadyMinted {
		t.Fatal("repaired canonical file receipt missing")
	}
	r2, f2, proof2 := nftDurableRuntimeFixtureV1(t)
	r2.Config.RewardAddrMiner = wallet.NewWallet().GetAddress()
	originalFilename := f2.cfg.ChainFile
	before, file := r2.Blockchain.StateSnapshot(), nftDurableRuntimeReadV1(t, originalFilename)
	r2.Config.ChainFile = filepath.Join(t.TempDir(), "missing-parent", "chain.dat")
	response := nftMintTransportCallV1(r2, nftMintTransportPayloadV1(t, f2.record, proof2), true)
	if response.Code != http.StatusServiceUnavailable || !reflect.DeepEqual(before, r2.Blockchain.StateSnapshot()) || !bytes.Equal(file, nftDurableRuntimeReadV1(t, originalFilename)) {
		t.Fatalf("HTTP write failure claimed success or changed chain: %d %s", response.Code, response.Body)
	}
}

func TestNFTMintTransportHTTPSerializesDuplicateRequestsV1(t *testing.T) {
	r, f, proof := nftDurableRuntimeFixtureV1(t)
	r.Config.RewardAddrMiner = wallet.NewWallet().GetAddress()
	body := nftMintTransportPayloadV1(t, f.record, proof)
	responses := make([]*httptest.ResponseRecorder, 2)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for i := range responses {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			<-start
			responses[index] = nftMintTransportCallV1(r, body, true)
		}(i)
	}
	close(start)
	workers.Wait()
	a, b := nftMintTransportReceiptV1(t, responses[0]), nftMintTransportReceiptV1(t, responses[1])
	if a.AlreadyMinted == b.AlreadyMinted || a.TokenID != b.TokenID || a.MintBlockHash != b.MintBlockHash || r.Blockchain.BestHeightSnapshot() != 1 {
		t.Fatalf("duplicate HTTP requests did not reconcile one award: %+v %+v", a, b)
	}
	loaded, err := blockchain.LoadBlockchainFromFileWithLocalContextV1(f.cfg.ChainFile, r.Blockchain)
	if err != nil {
		t.Fatal(err)
	}
	ownership, err := loaded.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil || ownership.Height != 1 || len(ownership.NFTs) != 1 {
		t.Fatalf("duplicate HTTP award persistence: %+v %v", ownership, err)
	}
}
