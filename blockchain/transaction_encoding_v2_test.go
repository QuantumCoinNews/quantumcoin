package blockchain

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"quantumcoin/config"
	"quantumcoin/consensus"
	"quantumcoin/wallet"
	"strings"
	"testing"
	"time"
)

type encodingProcessResult struct {
	Genesis     string       `json:"genesisHash"`
	GenesisTxID string       `json:"genesisTransactionID"`
	TxID        string       `json:"transactionID"`
	Digests     []string     `json:"signingDigests"`
	Minted      int          `json:"genesisMintedQC"`
	Timestamp   int64        `json:"genesisTimestamp"`
	Domain      string       `json:"genesisDomain"`
	Tx          *Transaction `json:"transaction"`
}

func TestCanonicalEncodingProcessFixture(t *testing.T) {
	mode := os.Getenv("QC_CANONICAL_ENCODING_TEST_CHILD")
	if mode == "" {
		return
	}
	if mode == "warm" {
		if err := gob.NewEncoder(io.Discard).Encode(struct {
			Noise  string
			Values []int
		}{"unrelated", []int{1, 2}}); err != nil {
			t.Fatal(err)
		}
	}
	chain := freshSignatureChain(t, config.QCNetworkProfileTestnetLocalV1)
	// Public test-only scalar, never a wallet used outside this test.
	private, err := wallet.ImportPrivateKeyHex(strings.Repeat("0", 63) + "1")
	if err != nil {
		t.Fatal(err)
	}
	pub, err := consensus.UncompressedPublicKeyV1(private)
	if err != nil {
		t.Fatal(err)
	}
	owner := &wallet.Wallet{PrivateKey: private, PublicKey: pub}
	tx := &Transaction{Inputs: []TransactionInput{{TxID: bytes.Repeat([]byte{4}, 32), OutIndex: 3}}, Outputs: []TransactionOutput{{Amount: 7, PubKeyHash: bytes.Repeat([]byte{5}, 20)}}, Sender: owner.GetAddress(), Amount: 7, Timestamp: time.Unix(1_700_000_000, 123456789).UTC()}
	if err = chain.SignTransaction(tx, private); err != nil {
		t.Fatal(err)
	}
	hashes, err := chain.TransactionSigningHashes(tx)
	if err != nil {
		t.Fatal(err)
	}
	genesis := chain.BlockByIndexSnapshot(0)
	result := encodingProcessResult{Genesis: hex.EncodeToString(genesis.Hash), GenesisTxID: hex.EncodeToString(genesis.Transactions[0].ID), TxID: hex.EncodeToString(tx.ID), Digests: hashes, Minted: chain.TotalMintedSnapshot(), Timestamp: genesis.Timestamp, Domain: chain.NetworkIdentitySnapshotV1().GenesisDomain, Tx: tx}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("CANONICAL_ENCODING_RESULT=%s\n", raw)
}

func TestNetworkV4IndependentProcessesIgnoreGobAllocationOrder(t *testing.T) {
	var results []encodingProcessResult
	for _, mode := range []string{"cold", "warm", "cold"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestCanonicalEncodingProcessFixture$")
		cmd.Env = append(os.Environ(), "QC_CANONICAL_ENCODING_TEST_CHILD="+mode)
		raw, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("isolated encoding fixture failed: %v %s", err, raw)
		}
		var result encodingProcessResult
		found := false
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(line, "CANONICAL_ENCODING_RESULT=") {
				if err = json.Unmarshal([]byte(strings.TrimPrefix(line, "CANONICAL_ENCODING_RESULT=")), &result); err != nil {
					t.Fatal(err)
				}
				found = true
			}
		}
		if !found {
			t.Fatal("missing independent-process fingerprint")
		}
		chain := freshSignatureChain(t, config.QCNetworkProfileTestnetLocalV1)
		if result.Minted != 0 || !chain.VerifyTransaction(result.Tx) {
			t.Fatal("cross-process transaction validation or zero-allocation genesis failed")
		}
		if len(results) > 0 {
			prior := results[0]
			if result.Genesis != prior.Genesis || result.GenesisTxID != prior.GenesisTxID || result.TxID != prior.TxID || strings.Join(result.Digests, ",") != strings.Join(prior.Digests, ",") {
				t.Fatal("process allocation order changed consensus bytes")
			}
		}
		results = append(results, result)
	}
}

func TestCanonicalV2ContentBindsFieldsAndNormalizesTimeRepresentation(t *testing.T) {
	tx := &Transaction{Inputs: []TransactionInput{{TxID: bytes.Repeat([]byte{4}, 32), OutIndex: 3}}, Outputs: []TransactionOutput{{Amount: 7, PubKeyHash: []byte{0xaa, 0xbb}}}, Timestamp: time.Unix(1_700_000_000, 123456789).UTC(), Sender: "codec-vector", Amount: 7.25}
	expected, err := CanonicalTransactionIDV2(tx)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := TransactionContentV2(tx)
	if err != nil {
		t.Fatal(err)
	}
	// Independent fixed-format vector, generated with Python struct.pack.
	if hex.EncodeToString(expected) != canonicalV2VectorHex {
		t.Fatalf("canonical V2 vector changed: %x", expected)
	}
	tx.ID = []byte("not part of content")
	tx.Inputs[0].Signature = []byte("public signature")
	tx.Inputs[0].PubKey = []byte("public key")
	tx.Timestamp = tx.Timestamp.In(time.FixedZone("test-offset", 3*3600))
	actual, _ := TransactionContentV2(tx)
	if !bytes.Equal(raw, actual) {
		t.Fatal("excluded fields or time zone altered content commitment")
	}
	tx.Outputs[0].Amount++
	changed, _ := CanonicalTransactionIDV2(tx)
	if bytes.Equal(changed, expected) {
		t.Fatal("output mutation did not change commitment")
	}
}

func TestCanonicalV2RejectsInvalidAndUnboundedContent(t *testing.T) {
	for _, amount := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := TransactionContentV2(&Transaction{Amount: amount}); err == nil {
			t.Fatal("non-finite amount accepted")
		}
	}
	for _, tx := range []*Transaction{nil, {Sender: strings.Repeat("x", (1<<20)+1)}, {Inputs: make([]TransactionInput, 5001)}, {Outputs: make([]TransactionOutput, 5001)}} {
		if _, err := TransactionContentV2(tx); err == nil {
			t.Fatal("nil or oversized content accepted")
		}
	}
	large := &Transaction{Outputs: make([]TransactionOutput, 17)}
	for i := range large.Outputs {
		large.Outputs[i].PubKeyHash = make([]byte, 1<<20)
	}
	if _, err := TransactionContentV2(large); err == nil {
		t.Fatal("aggregate content limit not enforced")
	}
}

func TestCanonicalV2ChainContextRejectsOtherCodecForCoinbase(t *testing.T) {
	v4 := freshSignatureChain(t, config.QCNetworkProfileTestnetLocalV1)
	legacy := NewBlockchain(50, int(QCTotalSupplyHardCap))
	tx, err := newCoinbaseTxWithRewardV1(wallet.NewWallet().GetAddress(), 50, time.Unix(1800, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if legacy.validateTransactionIDLockedV2(tx) != nil || v4.validateTransactionIDLockedV2(tx) == nil {
		t.Fatal("V4 accepted legacy coinbase commitment")
	}
	tx.ID, err = CanonicalTransactionIDV2(tx)
	if err != nil {
		t.Fatal(err)
	}
	if v4.validateTransactionIDLockedV2(tx) != nil || legacy.validateTransactionIDLockedV2(tx) == nil {
		t.Fatal("legacy chain accepted V4 coinbase commitment")
	}
}

const canonicalV2VectorHex = "a613bfb56b72a8678cdb33c8cc6750c5d8b4d37933c8d15926dd4f60656b69c8"
