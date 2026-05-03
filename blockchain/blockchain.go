package blockchain

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"quantumcoin/ai"
	"quantumcoin/config"
	"quantumcoin/wallet"
)

type Blockchain struct {
	Blocks           []*Block
	UTXO             map[string][]TransactionOutput
	TotalSupply      int
	coinbaseMaturity int
	pendingTxs       []*Transaction
}

// Varsayılanlar (config yoksa devreye girer)
const (
	GenesisTimeDefault     = 1725158400             // 2024-09-01 UTC
	HalvingIntervalDefault = 2 * 365 * 24 * 60 * 60 // ~2 yıl
	MiningPeriodDefault    = 10 * 365 * 24 * 60 * 60
	InitialRewardDefault   = 50
)

// UYARISIZ ve doğru hal
func GetCurrentReward() int {
	p := config.Current()
	now := time.Now().Unix()

	genesis := p.GenesisUnix
	if genesis <= 0 {
		genesis = GenesisTimeDefault
	}
	elapsed := now - genesis
	if elapsed < 0 {
		elapsed = 0
	}

	halvingSecs := p.HalvingIntervalSecs
	if halvingSecs <= 0 {
		halvingSecs = HalvingIntervalDefault
	}
	halvings := int(elapsed / halvingSecs)

	r := p.InitialReward
	if r <= 0 {
		r = InitialRewardDefault
	}
	r >>= halvings
	if r < 1 {
		r = 1
	}

	miningSecs := p.MiningPeriodSecs
	if miningSecs <= 0 {
		miningSecs = MiningPeriodDefault
	}
	if elapsed > miningSecs {
		r = 0
	}
	return r
}

func NewBlockchain(initialReward, totalSupply int) *Blockchain {
	var txs []*Transaction

	genesisCoinbase := &Transaction{
		ID:        nil,
		Inputs:    []TransactionInput{},
		Outputs:   []TransactionOutput{{Amount: initialReward, PubKeyHash: []byte("genesis-recipient")}},
		Timestamp: time.Now(),
		Sender:    "COINBASE",
		Amount:    float64(initialReward),
	}
	genesisCoinbase.ID = genesisCoinbase.Hash()
	txs = append(txs, genesisCoinbase)

	if mainAddr := strings.TrimSpace(os.Getenv("QC_MAIN_ADDRESS")); mainAddr != "" && totalSupply > 0 {
		premine := int(float64(totalSupply) * 0.12)
		if premine > 0 {
			pk := wallet.Base58DecodeAddress(mainAddr)
			premTx := &Transaction{
				ID:        nil,
				Inputs:    []TransactionInput{},
				Outputs:   []TransactionOutput{{Amount: premine, PubKeyHash: pk}},
				Timestamp: time.Now(),
				Sender:    "COINBASE",
				Amount:    float64(premine),
			}
			premTx.ID = premTx.Hash()
			txs = append(txs, premTx)
		}
	}

	genesis := NewBlock(0, txs, []byte{}, "genesis", 1)

	bc := &Blockchain{
		Blocks:      []*Block{genesis},
		UTXO:        map[string][]TransactionOutput{},
		TotalSupply: totalSupply,
		pendingTxs:  []*Transaction{},
	}
	bc.UpdateUTXOSet()

	// AI'ye genesis’i bildir
	if ai.Enabled() {
		ai.OnBlockAdded(genesis)
	}

	return bc
}

func (bc *Blockchain) SetCoinbaseMaturity(n int) {
	if n < 0 {
		n = 0
	}
	bc.coinbaseMaturity = n
}

func (bc *Blockchain) AddBlock(txs []*Transaction, miner string, difficulty int) *Block {
	// Blok içindeki işlemleri doğrula (coinbase hariç imza zorunlu)
	if err := bc.validateBlockTxs(txs); err != nil {
		log.Printf("rejecting block: %v", err)
		return nil
	}

	prev := bc.Blocks[len(bc.Blocks)-1]
	nb := NewBlock(prev.Index+1, txs, prev.Hash, miner, difficulty)
	bc.Blocks = append(bc.Blocks, nb)
	bc.UpdateUTXOSet()

	// AI'ye haber ver
	if ai.Enabled() {
		ai.OnBlockAdded(nb)
	}

	return nb
}

func (bc *Blockchain) AddBlockFromPeer(blk *Block) error {
	if blk == nil {
		return ErrNilBlock
	}

	// 1) Temel blok güvenliği (timestamp, tx sayısı, outputs vb.)
	if err := ValidateBlockBasic(blk); err != nil {
		return fmt.Errorf("peer block basic validation failed: %w", err)
	}

	// 2) PoW doğrulaması
	if !blk.ValidatePoW() {
		return ErrInvalidPoW
	}

	// 3) Zincire bağlanırlık kontrolü
	if len(bc.Blocks) > 0 {
		last := bc.Blocks[len(bc.Blocks)-1]
		if !bytes.Equal(blk.PrevHash, last.Hash) {
			return ErrPrevHashMismatch
		}
	}

	// 4) İşlem imzaları / yapısı
	if err := bc.validateBlockTxs(blk.Transactions); err != nil {
		return fmt.Errorf("peer block tx validation failed: %w", err)
	}

	// 5) Bloku zincire ekle
	bc.Blocks = append(bc.Blocks, blk)
	bc.UpdateUTXOSet()

	// 6) AI'ye haber ver
	if ai.Enabled() {
		ai.OnBlockAdded(blk)
	}

	return nil
}

func (bc *Blockchain) IsValidChain() bool {
	for i := 1; i < len(bc.Blocks); i++ {
		if !bc.Blocks[i].ValidatePoW() || !bytes.Equal(bc.Blocks[i].PrevHash, bc.Blocks[i-1].Hash) {
			return false
		}
	}
	return true
}

func (bc *Blockchain) GetHeight() int { return len(bc.Blocks) - 1 }

func (bc *Blockchain) ReplaceChain(blocks []*Block) error {
	if len(blocks) <= len(bc.Blocks) {
		return ErrIncomingChainNotLonger
	}
	for i := 1; i < len(blocks); i++ {
		if !blocks[i].ValidatePoW() || !bytes.Equal(blocks[i].PrevHash, blocks[i-1].Hash) {
			return ErrIncomingChainInvalid
		}
		// Zincir değiştirmede her bloğun işlemlerini denetle
		if err := bc.validateBlockTxs(blocks[i].Transactions); err != nil {
			return fmt.Errorf("incoming chain invalid tx: %w", err)
		}
	}
	bc.Blocks = blocks
	bc.UpdateUTXOSet()

	// AI'ye bütün zincirin değiştiğini bildir
	if ai.Enabled() {
		ai.OnChainReplaced(blocks)
	}

	return nil
}

func (bc *Blockchain) GetAllBlocks() []*Block { return bc.Blocks }

func (bc *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (map[string][]int, int) {
	acc := 0
	unspent := make(map[string][]int)

	if bc == nil || amount <= 0 || len(pubKeyHash) == 0 {
		return unspent, acc
	}

	for txID, outs := range bc.UTXO {
		for idx, out := range outs {
			// UpdateUTXOSet gerçek OutIndex'i korumak için harcanmış slotları
			// zero-value bırakabilir. Bu slotları harcanabilir sayma.
			if out.Amount <= 0 {
				continue
			}

			if out.IsLockedWithKey(pubKeyHash) {
				acc += out.Amount
				unspent[txID] = append(unspent[txID], idx)

				if acc >= amount {
					return unspent, acc
				}
			}
		}
	}

	return unspent, acc
}

func (bc *Blockchain) UpdateUTXOSet() {
	utxo := make(map[string][]TransactionOutput)

	if bc == nil {
		return
	}

	// Önce zincirde harcanmış tüm OutPoint'leri topla.
	// key: txID hex, value: harcanmış output index'leri
	spent := make(map[string]map[int]bool)

	for _, block := range bc.Blocks {
		if block == nil {
			continue
		}

		for _, tx := range block.Transactions {
			if tx == nil || tx.IsCoinbase() {
				continue
			}

			for _, in := range tx.Inputs {
				if len(in.TxID) == 0 || in.OutIndex < 0 {
					continue
				}

				txID := hex.EncodeToString(in.TxID)
				if spent[txID] == nil {
					spent[txID] = make(map[int]bool)
				}
				spent[txID][in.OutIndex] = true
			}
		}
	}

	// Sonra tüm output'ları gerçek outIdx pozisyonunu koruyarak ekle.
	// Harcanmış output'lar zero-value kalır; FindSpendableOutputs bunları atlar.
	for _, block := range bc.Blocks {
		if block == nil {
			continue
		}

		for _, tx := range block.Transactions {
			if tx == nil || len(tx.Outputs) == 0 {
				continue
			}

			txID := hex.EncodeToString(tx.ID)
			outs := make([]TransactionOutput, len(tx.Outputs))
			hasUnspent := false

			for outIdx, out := range tx.Outputs {
				if spent[txID] != nil && spent[txID][outIdx] {
					continue
				}

				outs[outIdx] = out
				if out.Amount > 0 {
					hasUnspent = true
				}
			}

			if hasUnspent {
				utxo[txID] = outs
			}
		}
	}

	bc.UTXO = utxo
}

// --- İMZA ZORUNLULUĞU: mempool’a eklemeden önce doğrula ---
func (bc *Blockchain) AddTransaction(tx *Transaction) error {
	if bc == nil {
		return fmt.Errorf("nil blockchain")
	}
	if tx == nil {
		return ErrNilTransaction
	}

	// --- Temel güvenlik kontrolleri (security.go) ---
	if err := ValidateTransactionBasic(tx); err != nil {
		return err
	}

	// Coinbase sadece blok üretiminde oluşturulmalı; mempool'a alınmaz.
	if tx.IsCoinbase() {
		return fmt.Errorf("coinbase tx cannot be added to mempool")
	}

	if len(tx.Inputs) == 0 {
		return fmt.Errorf("empty inputs")
	}
	if len(tx.Outputs) == 0 {
		return fmt.Errorf("empty outputs")
	}

	// Normal işlemler imzalı ve doğrulanmış olmalı.
	if !tx.Verify() {
		return fmt.Errorf("invalid tx signature")
	}

	if bc.UTXO == nil {
		bc.UpdateUTXOSet()
	}

	if err := bc.validateTransactionSpend(tx); err != nil {
		return err
	}

	bc.pendingTxs = append(bc.pendingTxs, tx)
	return nil
}

// validateTransactionSpend normal transaction için UTXO, input/output toplamı
// ve mempool double-spend kontrollerini yapar.
func (bc *Blockchain) validateTransactionSpend(tx *Transaction) error {
	if bc == nil || tx == nil {
		return fmt.Errorf("nil blockchain or transaction")
	}
	if tx.IsCoinbase() {
		return nil
	}

	inputTotal := 0
	outputTotal := 0
	seenInputs := make(map[string]bool)

	for _, out := range tx.Outputs {
		if out.Amount <= 0 {
			return fmt.Errorf("invalid output amount")
		}
		outputTotal += out.Amount
	}

	for _, in := range tx.Inputs {
		if len(in.TxID) == 0 || in.OutIndex < 0 {
			return fmt.Errorf("invalid input outpoint")
		}

		outpoint := fmt.Sprintf("%x:%d", in.TxID, in.OutIndex)
		if seenInputs[outpoint] {
			return fmt.Errorf("double spend inside transaction: %s", outpoint)
		}
		seenInputs[outpoint] = true

		if bc.pendingSpendsOutpoint(in.TxID, in.OutIndex) {
			return fmt.Errorf("input already pending in mempool: %s", outpoint)
		}

		txID := hex.EncodeToString(in.TxID)
		outs, ok := bc.UTXO[txID]
		if !ok || in.OutIndex >= len(outs) {
			return fmt.Errorf("missing utxo: %s", outpoint)
		}

		prevOut := outs[in.OutIndex]
		if prevOut.Amount <= 0 {
			return fmt.Errorf("spent or empty utxo: %s", outpoint)
		}

		if len(in.PubKey) == 0 {
			return fmt.Errorf("missing input pubkey: %s", outpoint)
		}

		pubKeyHash := wallet.HashPubKey(in.PubKey)
		if !prevOut.IsLockedWithKey(pubKeyHash) {
			return fmt.Errorf("input pubkey does not unlock utxo: %s", outpoint)
		}

		inputTotal += prevOut.Amount
	}

	if inputTotal < outputTotal {
		return fmt.Errorf("insufficient input amount: input=%d output=%d", inputTotal, outputTotal)
	}

	return nil
}

func (bc *Blockchain) pendingSpendsOutpoint(txID []byte, outIdx int) bool {
	if bc == nil || len(txID) == 0 || outIdx < 0 {
		return false
	}

	for _, pending := range bc.pendingTxs {
		if pending == nil || pending.IsCoinbase() {
			continue
		}

		for _, in := range pending.Inputs {
			if in.OutIndex == outIdx && bytes.Equal(in.TxID, txID) {
				return true
			}
		}
	}

	return false
}
func (bc *Blockchain) GetSpendableBalance(address string) int {
	pubKeyHash := wallet.Base58DecodeAddress(address)
	best := bc.GetBestHeight()
	spend := 0

	for height, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			for outIdx, out := range tx.Outputs {
				if !out.IsLockedWithKey(pubKeyHash) || bc.isOutputSpent(tx.ID, outIdx) {
					continue
				}
				if tx.IsCoinbase() && bc.coinbaseMaturity > 0 {
					age := best - height
					if age < bc.coinbaseMaturity {
						continue
					}
				}
				spend += out.Amount
			}
		}
	}
	return spend
}

func (bc *Blockchain) GetBalance(address string) int {
	pubKeyHash := wallet.Base58DecodeAddress(address)
	total := 0
	for _, outs := range bc.UTXO {
		for _, out := range outs {
			if out.IsLockedWithKey(pubKeyHash) {
				total += out.Amount
			}
		}
	}
	return total
}

func (bc *Blockchain) TotalMinted() int {
	total := 0
	for _, b := range bc.Blocks {
		for _, tx := range b.Transactions {
			if tx.IsCoinbase() {
				for _, out := range tx.Outputs {
					total += out.Amount
				}
			}
		}
	}
	return total
}

func (bc *Blockchain) MineBlock(miner string, difficulty int) (*Block, error) {
	if len(bc.Blocks) == 0 {
		return nil, ErrChainNotInitialized
	}

	// yalnız başarılı kazımdan sonra mined_balance.json yaz
	var _minedOK bool
	_minedReward := GetCurrentReward()

	if _minedReward <= 0 {

		return nil, fmt.Errorf("mining reward is zero")

	}

	if bc.TotalSupply > 0 && bc.TotalMinted()+_minedReward > bc.TotalSupply {

		return nil, fmt.Errorf("total supply cap exceeded: minted=%d reward=%d supply=%d", bc.TotalMinted(), _minedReward, bc.TotalSupply)

	}
	defer func() {
		if _minedOK && _minedReward > 0 {
			AddMinedBalance(miner, _minedReward)
		}
	}()

	cbTx, err := newCoinbaseTx(miner)
	if err != nil {
		return nil, fmt.Errorf("coinbase tx: %w", err)
	}

	// pending kopyasını al, coinbase ile birleştir ve doğrula
	txs := append([]*Transaction{cbTx}, bc.PendingTxs()...)
	if err := bc.validateBlockTxs(txs); err != nil {
		return nil, err
	}

	prev := bc.Blocks[len(bc.Blocks)-1]
	nb := NewBlock(prev.Index+1, txs, prev.Hash, miner, difficulty)

	bc.Blocks = append(bc.Blocks, nb)
	bc.UpdateUTXOSet()
	bc.pendingTxs = []*Transaction{} // mempool’u boşalt

	_minedOK = true

	// AI: bizim node kazdı, haber ver
	if ai.Enabled() {
		ai.OnBlockAdded(nb)

		// === BURASI YENİ: bloktan AI bonusu üret ve dosyaya yaz ===
		lites := make([]ai.TxLite, 0, len(nb.Transactions))
		for _, tx := range nb.Transactions {
			if tx == nil {
				continue
			}
			lites = append(lites, ai.TxLite{
				TxID:          hex.EncodeToString(tx.ID),
				WalletAddress: tx.Sender,
				Sender:        tx.Sender,
				Amount:        tx.Amount,
				Timestamp:     tx.Timestamp,
			})
		}
		bon := ai.BuildAIBonuses(lites)

		nodeDir := strings.TrimSpace(os.Getenv("QC_NODE_DIR"))
		if nodeDir == "" {
			nodeDir = "."
		}
		_ = ai.WriteAIBonusesTo(nodeDir, bon)
		// === /YENİ ===
	}

	return nb, nil
}

func SerializeBlockchain(bc *Blockchain) []byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(bc); err != nil {
		log.Panicf("serialize error: %v", err)
	}
	return buf.Bytes()
}

func DeserializeBlockchain(data []byte) *Blockchain {
	var bc Blockchain
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&bc); err != nil {
		log.Panicf("deserialize error: %v", err)
	}
	if bc.pendingTxs == nil {
		bc.pendingTxs = []*Transaction{}
	}
	return &bc
}

func (bc *Blockchain) SaveToFile(filename string) error {
	return os.WriteFile(filename, SerializeBlockchain(bc), 0o600)
}

func LoadBlockchainFromFile(filename string) (*Blockchain, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read blockchain file: %w", err)
	}
	return DeserializeBlockchain(data), nil
}

// Helpers

func (bc *Blockchain) GetBestHeight() int {
	if bc == nil || len(bc.Blocks) == 0 {
		return -1
	}
	return bc.Blocks[len(bc.Blocks)-1].Index
}

func (bc *Blockchain) GetLastBlock() *Block {
	if bc == nil || len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}

func (bc *Blockchain) GetBlockByIndex(idx int) *Block {
	for _, b := range bc.Blocks {
		if b.Index == idx {
			return b
		}
	}
	return nil
}

func (bc *Blockchain) GetBlockByHash(hash []byte) *Block {
	for _, b := range bc.Blocks {
		if bytes.Equal(b.Hash, hash) {
			return b
		}
	}
	return nil
}

func (bc *Blockchain) isOutputSpent(txid []byte, outIdx int) bool {
	for _, blk := range bc.Blocks {
		for _, tx := range blk.Transactions {
			for _, in := range tx.Inputs {
				if bytes.Equal(in.TxID, txid) && in.OutIndex == outIdx {
					return true
				}
			}
		}
	}
	return false
}

// ---- Eklenen yardımcılar ----

// Blok içindeki tüm işlemleri doğrula.
// Coinbase için: temel security + en az 1 output.
// Normal tx için: temel security + imza doğrulaması.
func (bc *Blockchain) validateBlockTxs(txs []*Transaction) error {
	if bc == nil {
		return fmt.Errorf("nil blockchain")
	}
	if len(txs) == 0 {
		return fmt.Errorf("empty block transactions")
	}

	blockSpent := make(map[string]bool)
	coinbaseCount := 0

	for _, tx := range txs {
		if tx == nil {
			return fmt.Errorf("nil tx")
		}

		if err := ValidateTransactionBasic(tx); err != nil {
			return err
		}

		if tx.IsCoinbase() {
			coinbaseCount++
			if coinbaseCount > 1 {
				return fmt.Errorf("multiple coinbase transactions in block")
			}
			if len(tx.Outputs) == 0 {
				return fmt.Errorf("invalid coinbase (no outputs)")
			}
			continue
		}

		if !tx.Verify() {
			return fmt.Errorf("invalid tx signature")
		}

		for _, in := range tx.Inputs {
			if len(in.TxID) == 0 || in.OutIndex < 0 {
				return fmt.Errorf("invalid input outpoint in block")
			}

			outpoint := fmt.Sprintf("%x:%d", in.TxID, in.OutIndex)
			if blockSpent[outpoint] {
				return fmt.Errorf("double spend inside block: %s", outpoint)
			}
			blockSpent[outpoint] = true
		}
	}

	return nil
}

// pendingTxs'in güvenli kopyası (API/mine kullanımı için)
func (bc *Blockchain) PendingTxs() []*Transaction {
	if bc == nil || bc.pendingTxs == nil {
		return []*Transaction{}
	}
	cp := make([]*Transaction, len(bc.pendingTxs))
	copy(cp, bc.pendingTxs)
	return cp
}
