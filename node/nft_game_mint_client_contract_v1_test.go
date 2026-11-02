package node

import (
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/consensus"
	"quantumcoin/wallet"
)

// Emit public evidence and a real durable receipt with known metadata bytes
// for the Python transport/service tests. No private fixture keys are emitted.
func TestNFTGameMintClientContractV1(t *testing.T) {
	testNFTGameMintClientContractV1(t, nil, "QMA0066_TG_MINT_CLIENT_CONTRACT_PATH")
}

// A known TEST-ONLY recipient lets the game prove wallet control before the
// real node contract mints to that same owner. Private keys are never emitted.
func TestNFTVerifiedLifecycleContractV1(t *testing.T) {
	key, err := wallet.ImportPrivateKeyHex(strings.Repeat("0", 63) + "1")
	if err != nil {
		t.Fatal(err)
	}
	public := elliptic.Marshal(key.Curve, key.X, key.Y)
	testNFTGameMintClientContractV1(t, wallet.HashPubKey(public), "QMA_NFT_LIFECYCLE_CONTRACT_PATH")
}

func testNFTGameMintClientContractV1(t *testing.T, recipient []byte, outputEnv string) {
	r, f, _ := nftDurableRuntimeFixtureV1(t)
	r.Config.RewardAddrMiner = wallet.NewWallet().GetAddress()
	metadata := []byte("{\"name\":\"client-contract-test-only\"}\n")
	record := f.record
	if recipient != nil {
		record.RecipientPubKeyHash = append([]byte(nil), recipient...)
	}
	record.MetadataHash = sha256.Sum256(metadata)
	digest, err := blockchain.NFTMintRecordDigestV1(record)
	if err != nil {
		t.Fatal(err)
	}
	proof := blockchain.NFTAwardProofV1{Decision: consensus.NFTAwardDecisionV1{
		NetworkID: record.NetworkID, LegacyGenesisHex: f.trust.LegacyGenesisHex, RecordDigest: digest,
	}}
	signingDigest, err := consensus.NFTAwardDecisionDigestV1(proof.Decision)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range f.keys {
		public, err := consensus.UncompressedPublicKeyV1(key)
		if err != nil {
			t.Fatal(err)
		}
		signature, err := consensus.SignTransactionDigestV1(key, signingDigest)
		if err != nil {
			t.Fatal(err)
		}
		proof.Approvals = append(proof.Approvals, consensus.NFTAwardApprovalV1{PublicKey: public, Signature: signature})
	}
	payload := nftMintTransportPayloadV1(t, record, proof)
	if outputEnv == "QMA_NFT_LIFECYCLE_CONTRACT_PATH" && os.Getenv("QMA_NFT_LIFECYCLE_SERVER") == "1" {
		filename := os.Getenv(outputEnv)
		if filename == "" {
			t.Fatal("live test contract output path required")
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/api/nft/monthly/program", r.ServeNFTGameProgramV1)
		mux.HandleFunc("/api/nft/monthly/receipt", r.ServeNFTMonthlyReceiptV1)
		mux.HandleFunc("/api/nft/monthly/mint", r.ServeNFTMonthlyMintV1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Header.Get("X-QC-API-KEY") != "test-only-lifecycle-node" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			mux.ServeHTTP(w, req)
		}))
		defer server.Close()
		programResponse := httptest.NewRecorder()
		r.ServeNFTGameProgramV1(programResponse, httptest.NewRequest(http.MethodGet, "/api/nft/monthly/program", nil))
		live, err := json.Marshal(map[string]any{"request": json.RawMessage(payload),
			"program": json.RawMessage(programResponse.Body.Bytes()), "metadata": string(metadata), "baseURL": server.URL})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, live, 0600); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(180 * time.Second)
		for {
			if _, err := os.Stat(filename + ".done"); err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("Python lifecycle client did not finish")
			}
			time.Sleep(50 * time.Millisecond)
		}
		// The external game client must have caused a real persisted mint.
		receipt := nftMintTransportReceiptV1(t, nftMintTransportCallV1(r, payload, false))
		if !receipt.Persisted || !receipt.AlreadyMinted {
			t.Fatal("game client did not produce the durable canonical mint")
		}
		result, err := json.Marshal(receipt)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename+".receipt.json", result, 0600); err != nil {
			t.Fatal(err)
		}
		return
	}
	receipt := nftMintTransportReceiptV1(t, nftMintTransportCallV1(r, payload, true))
	if receipt.AlreadyMinted {
		t.Fatal("first request was not a fresh durable award")
	}
	replay := nftMintTransportReceiptV1(t, nftMintTransportCallV1(r, payload, false))
	if !replay.AlreadyMinted || replay.TokenID != receipt.TokenID {
		t.Fatal("receipt lookup changed award")
	}
	response := httptest.NewRecorder()
	r.ServeNFTGameProgramV1(response, httptest.NewRequest(http.MethodGet, "/api/nft/monthly/program", nil))
	if response.Code != http.StatusOK {
		t.Fatal("verified program unavailable")
	}
	contract, err := json.MarshalIndent(map[string]any{
		"request": json.RawMessage(payload), "receipt": receipt,
		"program": json.RawMessage(response.Body.Bytes()), "metadata": string(metadata),
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if filename := os.Getenv(outputEnv); filename != "" {
		if err := os.WriteFile(filename, append(contract, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
