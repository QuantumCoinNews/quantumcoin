package miner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/p2p"
	"quantumcoin/wallet"
)

type QCLocalOpts struct {
	ConfigPath string
	ChainPath  string
	P2PPort    string // (şimdilik bilgi amaçlı)
}

type qcLocal struct {
	cfg *config.Config
	bc  *blockchain.Blockchain
	op  QCLocalOpts
}

// Adapter kurulum: config + chain yükle
func NewQCLocalAdapter(op QCLocalOpts) (Backend, error) {
	// Çift tıkla çalıştırma uyumu
	if exe, err := os.Executable(); err == nil {
		_ = os.Chdir(filepath.Dir(exe))
	}
	cfg, err := config.Load(op.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("config load: %w", err)
	}
	var bc *blockchain.Blockchain
	if _, err := os.Stat(cfg.ChainFile); err == nil {
		bc, err = blockchain.LoadBlockchainFromFile(cfg.ChainFile)
		if err != nil {
			return nil, fmt.Errorf("chain load: %w", err)
		}
	} else {
		bc = blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	}
	bc.SetCoinbaseMaturity(cfg.CoinbaseMaturity)
	return &qcLocal{cfg: cfg, bc: bc, op: op}, nil
}

// Helpers — zincirin prepareData düzenini üretelim (nonce hariç)
func powLeftRight(prevHash []byte, txHash []byte, index int, ts int64, diff int, miner string) (left, right []byte) {
	left = bytes.Join([][]byte{
		prevHash,
		txHash,
		[]byte(strconv.Itoa(index)),
		[]byte(strconv.FormatInt(ts, 10)),
	}, []byte{})
	right = bytes.Join([][]byte{
		[]byte(strconv.Itoa(diff)),
		[]byte(miner),
	}, []byte{})
	return
}
func targetFromBits(bits int) *big.Int {
	if bits <= 0 {
		bits = 16
	}
	if bits > 255 {
		bits = 255
	}
	t := big.NewInt(1)
	return t.Lsh(t, uint(256-bits))
}

// ————— Backend interface —————

func (q *qcLocal) GetWork(_ context.Context, minerAddr string) (*Work, error) {
	last := q.bc.GetLastBlock()
	if last == nil {
		return nil, fmt.Errorf("chain empty")
	}
	// Basit coinbase (+mempool ileride eklenebilir)
	reward := blockchain.GetCurrentReward()
	cb := &blockchain.Transaction{
		ID:        nil,
		Inputs:    []blockchain.TransactionInput{},
		Outputs:   []blockchain.TransactionOutput{{Amount: reward, PubKeyHash: wallet.Base58DecodeAddress(minerAddr)}},
		Timestamp: time.Now(),
		Sender:    "COINBASE",
		Amount:    float64(reward),
	}
	cb.ID = cb.Hash()

	txs := []*blockchain.Transaction{cb}
	// Block.HashTransactions() eşleniği
	var txHashes [][]byte
	for _, t := range txs {
		txHashes = append(txHashes, t.Hash())
	}
	txHash := sha256.Sum256(bytes.Join(txHashes, []byte{}))

	height := last.Index + 1
	ts := time.Now().Unix()
	diff := q.cfg.DefaultDifficultyBits

	left, right := powLeftRight(last.Hash, txHash[:], height, ts, diff, minerAddr)
	return &Work{
		Left:   left,
		Right:  right,
		Target: targetFromBits(diff),
		Height: height,
		Miner:  minerAddr,
	}, nil
}

func (q *qcLocal) Submit(_ context.Context, w *Work, nonce uint64, hashHex string) (bool, error) {
	last := q.bc.GetLastBlock()
	if last == nil {
		return false, fmt.Errorf("empty chain")
	}

	// Hex hash doğrula ve hedef altında mı kontrol et
	hashBytes, _ := hex.DecodeString(strings.TrimSpace(hashHex))
	if new(big.Int).SetBytes(hashBytes).Cmp(w.Target) > 0 {
		return false, nil
	}

	// Coinbaseli tx set (GetWork ile aynı)
	reward := blockchain.GetCurrentReward()
	cb := &blockchain.Transaction{
		ID:        nil,
		Inputs:    []blockchain.TransactionInput{},
		Outputs:   []blockchain.TransactionOutput{{Amount: reward, PubKeyHash: wallet.Base58DecodeAddress(w.Miner)}},
		Timestamp: time.Now(),
		Sender:    "COINBASE",
		Amount:    float64(reward),
	}
	cb.ID = cb.Hash()
	txs := []*blockchain.Transaction{cb}

	blk := &blockchain.Block{
		Index:        w.Height,
		Timestamp:    time.Now().Unix(),
		Transactions: txs,
		PrevHash:     last.Hash,
		Hash:         nil, // dolduracağız
		Nonce:        int(nonce),
		Miner:        w.Miner,
		Difficulty:   q.cfg.DefaultDifficultyBits,
		Metadata:     map[string]string{"ext_miner": "cmd"},
	}

	// Zincirin prepareData kuralına göre Hash'i doldur (görsel/telemetri amaçlı)
	var txHashes [][]byte
	for _, t := range blk.Transactions {
		txHashes = append(txHashes, t.Hash())
	}
	sum := sha256.Sum256(bytes.Join(txHashes, []byte{}))
	h := sha256.Sum256(bytes.Join([][]byte{
		blk.PrevHash,
		sum[:],
		[]byte(strconv.Itoa(blk.Index)),
		[]byte(strconv.FormatInt(blk.Timestamp, 10)),
		[]byte(strconv.Itoa(int(nonce))),
		[]byte(strconv.Itoa(blk.Difficulty)),
		[]byte(blk.Miner),
	}, []byte{}))
	blk.Hash = h[:]

	// PoW & bağlanırlık kontrolü
	if !blk.ValidatePoW() {
		return false, fmt.Errorf("pow validate failed")
	}
	if !bytes.Equal(blk.PrevHash, last.Hash) {
		return false, fmt.Errorf("prev hash mismatch")
	}

	// Zincire ekle, kaydet, duyur
	q.bc.Blocks = append(q.bc.Blocks, blk)
	q.bc.UpdateUTXOSet()
	if err := q.bc.SaveToFile(q.cfg.ChainFile); err != nil {
		return false, fmt.Errorf("save chain: %w", err)
	}
	p2p.BroadcastMessage(p2p.BlockMessage(blk))
	return true, nil
}
