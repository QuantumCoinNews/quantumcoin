package blockchain

import (
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"quantumcoin/config"
	"quantumcoin/wallet"
	"strings"
	"testing"
	"time"
)

func freshSignatureChain(t *testing.T, profile config.NetworkProfileV1) *Blockchain {
	t.Helper()
	cfg := config.Default()
	cfg.GenesisProfile = config.NetworkGenesisProfileV4
	bc, err := NewNetworkBlockchainV4(cfg, profile)
	if err != nil {
		t.Fatal(err)
	}
	bc.SetCoinbaseMaturity(0)
	return bc
}

func TestNetworkV4SignatureBindsTrustedGenesisWithoutChangingLegacy(t *testing.T) {
	a := freshSignatureChain(t, config.QCNetworkProfileTestnetLocalV1)
	same := freshSignatureChain(t, config.QCNetworkProfileTestnetPublicV1)
	other := freshSignatureChain(t, config.QCNetworkProfileMainnetV1)
	owner := wallet.NewWallet()
	tx := &Transaction{Inputs: []TransactionInput{{TxID: bytes.Repeat([]byte{4}, 32), OutIndex: 0}}, Outputs: []TransactionOutput{{Amount: 1, PubKeyHash: wallet.HashPubKey(wallet.NewWallet().PublicKey)}}, Sender: owner.GetAddress(), Amount: 1, Timestamp: time.Unix(1800, 0).UTC()}
	tx.ID, _ = CanonicalTransactionIDV2(tx)
	before := append([]byte(nil), tx.ID...)
	if err := a.SignTransaction(tx, owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if !a.VerifyTransaction(tx) || !same.VerifyTransaction(tx) || other.VerifyTransaction(tx) || tx.Verify() {
		t.Fatal("network domain isolation failed")
	}
	if !bytes.Equal(before, tx.ID) {
		t.Fatal("signing changed canonical transaction ID format")
	}
	left, _ := a.TransactionSigningHashes(tx)
	right, _ := other.TransactionSigningHashes(tx)
	if left[0] == right[0] || left[0] == SigningHashes(tx)[0] {
		t.Fatal("digest missing genesis binding")
	}
	// Removing optional persistence identity must not downgrade an intrinsic V4 rule.
	a.networkIdentity = nil
	if !a.VerifyTransaction(tx) {
		t.Fatal("genesis alone did not preserve the activated rule")
	}
	if err := tx.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if a.VerifyTransaction(tx) || same.VerifyTransaction(tx) || !tx.Verify() {
		t.Fatal("legacy signature accepted on V4")
	}
	legacy := NewBlockchain(50, int(QCTotalSupplyHardCap))
	if !legacy.VerifyTransaction(tx) {
		t.Fatal("legacy verification was changed retroactively")
	}
	if !same.usesMedianBlockTimeV2() {
		t.Fatal("new profile dropped median-time validation")
	}
}

func addSignatureFundingBlock(t *testing.T, bc *Blockchain, owner *wallet.Wallet) *Block {
	t.Helper()
	timestamp := bc.Blocks[0].Timestamp + 30
	coinbase, err := newCoinbaseTxWithRewardV1(owner.GetAddress(), 50, time.Unix(timestamp, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err = bc.assignTransactionIDLockedV2(coinbase); err != nil {
		t.Fatal(err)
	}
	block, err := newBlockAtTimeV1(1, []*Transaction{coinbase}, bc.Blocks[0].Hash, owner.GetAddress(), config.QCTestnetDefaultDifficultyBitsV1, timestamp)
	if err != nil {
		t.Fatal(err)
	}
	if err = bc.AddBlockFromPeerStrictV2(block); err != nil {
		t.Fatal(err)
	}
	return block
}

func TestNetworkV4MempoolPeerMiningAndDiskShareSigningRule(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "")
	producer := freshSignatureChain(t, config.QCNetworkProfileTestnetLocalV1)
	peer := freshSignatureChain(t, config.QCNetworkProfileTestnetPublicV1)
	owner, recipient := wallet.NewWallet(), wallet.NewWallet()
	funded := addSignatureFundingBlock(t, producer, owner)
	if err := peer.AddBlockFromPeerStrictV2(cloneBlockV1(funded)); err != nil {
		t.Fatal(err)
	}
	tx, err := NewTransaction(owner.GetAddress(), recipient.GetAddress(), 2, producer)
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if err = producer.AddTransaction(tx); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatal("mempool accepted legacy signature", err)
	}
	if err = peer.AddTransactionFromPeerStrictV1(tx); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatal("P2P accepted legacy signature", err)
	}
	if err = producer.SignTransaction(tx, owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if err = producer.AddTransaction(tx); err != nil {
		t.Fatal(err)
	}
	if err = peer.AddTransactionFromPeerStrictV1(cloneTransactionV1(tx)); err != nil {
		t.Fatal("P2P rejected correctly scoped spend", err)
	}
	block, err := producer.MineBlock(owner.GetAddress(), config.QCTestnetDefaultDifficultyBitsV1)
	if err != nil {
		t.Fatal(err)
	}
	if len(block.Transactions) != 2 {
		t.Fatal("mining dropped a valid network-signed transaction")
	}
	if err = peer.AddBlockFromPeerStrictV2(cloneBlockV1(block)); err != nil {
		t.Fatal("peer block rejected network signature", err)
	}
	fresh := freshSignatureChain(t, config.QCNetworkProfileTestnetLocalV1)
	if err = fresh.ValidateReplacementChainStrictV1(producer.BlocksSnapshot()); err != nil {
		t.Fatal("replacement replay lost signing context", err)
	}
	path := filepath.Join(t.TempDir(), "chain.dat")
	if err = producer.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(path, fresh)
	if err != nil || !loaded.VerifyTransaction(tx) {
		t.Fatal("disk replay lost signing context", err)
	}
	original, _ := os.ReadFile(path)
	if _, err = LoadBlockchainFromFileWithLocalContextV1(path, freshSignatureChain(t, config.QCNetworkProfileMainnetV1)); err == nil {
		t.Fatal("mainnet loaded testnet chain")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(original, after) {
		t.Fatal("foreign load modified disk")
	}
}

func TestNetworkV4RejectsLegacySignaturesInsideValidPoWAndGob(t *testing.T) {
	chain := freshSignatureChain(t, config.QCNetworkProfileTestnetLocalV1)
	owner, to := wallet.NewWallet(), wallet.NewWallet()
	funding := addSignatureFundingBlock(t, chain, owner)
	tx, err := NewTransaction(owner.GetAddress(), to.GetAddress(), 1, chain)
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	timestamp := funding.Timestamp + 30
	// Keep a valid V2 content ID with the deliberately wrong legacy signature.
	if err = chain.assignTransactionIDLockedV2(tx); err != nil {
		t.Fatal(err)
	}
	coinbase, err := newCoinbaseTxWithRewardV1(owner.GetAddress(), 50, time.Unix(timestamp, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err = chain.assignTransactionIDLockedV2(coinbase); err != nil {
		t.Fatal(err)
	}
	block, err := newBlockAtTimeV1(2, []*Transaction{coinbase, tx}, funding.Hash, owner.GetAddress(), config.QCTestnetDefaultDifficultyBitsV1, timestamp)
	if err != nil {
		t.Fatal(err)
	}
	if err = chain.ValidatePeerBlockStrictV2(block); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatal("valid PoW bypassed network signature validation", err)
	}
	incoming := append(chain.BlocksSnapshot(), block)
	if err = chain.ValidateReplacementChainStrictV1(incoming); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatal("replacement accepted legacy signature", err)
	}
	// A checksum-free old Gob container is still subject to trusted consensus replay.
	chain.Blocks = incoming
	var raw bytes.Buffer
	if err = gob.NewEncoder(&raw).Encode(chain); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "untrusted.gob")
	if err = os.WriteFile(path, raw.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadBlockchainFromFileWithLocalContextV1(path, freshSignatureChain(t, config.QCNetworkProfileTestnetLocalV1)); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatal("disk accepted legacy signature", err)
	}
}

func TestNetworkV4MalformedTrustedContextNeverFallsBackToLegacy(t *testing.T) {
	owner := wallet.NewWallet()
	tx := &Transaction{Inputs: []TransactionInput{{TxID: bytes.Repeat([]byte{3}, 32)}}, Outputs: []TransactionOutput{{Amount: 1, PubKeyHash: wallet.HashPubKey(owner.PublicKey)}}, Sender: owner.GetAddress(), Amount: 1, Timestamp: time.Unix(100, 0).UTC()}
	tx.ID = tx.Hash()
	if err := tx.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	for _, alter := range []func(*Blockchain){
		func(bc *Blockchain) { bc.Blocks[0].Hash = make([]byte, 32) },
		func(bc *Blockchain) { bc.Blocks[0].Hash = nil },
		func(bc *Blockchain) {
			bc.Blocks[0].Transactions[0].Outputs[0].PubKeyHash = []byte(TestnetGenesisDomainV3)
		},
		func(bc *Blockchain) { bc.Blocks = nil },
	} {
		bc := freshSignatureChain(t, config.QCNetworkProfileTestnetLocalV1)
		alter(bc)
		if bc.VerifyTransaction(tx) {
			t.Fatal("invalid trusted genesis downgraded to legacy signature verification")
		}
		if err := bc.SignTransaction(tx, owner.PrivateKey); err == nil {
			t.Fatal("invalid trusted genesis allowed signing")
		}
	}
}
