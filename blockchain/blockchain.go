package blockchain

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"quantumcoin/config"
	qconsensus "quantumcoin/consensus"
	"quantumcoin/security"
	"quantumcoin/wallet"
)

type Blockchain struct {
	StoredNetworkIdentityV1 *NetworkIdentityV1
	networkIdentity         *NetworkIdentityV1
	stateMu                 sync.RWMutex
	Blocks                  []*Block
	UTXO                    map[string][]TransactionOutput
	TotalSupply             int
	coinbaseMaturity        int
	pendingTxs              []*Transaction
	confirmedTxIndex        map[string]int
	pendingTxIndex          map[string]struct{}
	blockHashIndex          map[string]*Block
	pendingObservedAt       map[string]time.Time
	activationPolicy        activationPolicyBindingStateV1
	nftAwardTrust           nftAwardTrustBindingStateV1
	nftMonthlyProgram       nftMonthlyProgramBindingStateV1
}

// Varsayılanlar (config yoksa devreye girer)
const (
	GenesisTimeDefault     = 1725158400             // 2024-09-01 UTC
	HalvingIntervalDefault = 2 * 365 * 24 * 60 * 60 // ~2 yıl
	MiningPeriodDefault    = 10 * 365 * 24 * 60 * 60
	InitialRewardDefault   = CanonicalInitialMiningRewardV1
)

// GetCurrentReward documents the QuantumCoin component used by this module.
// UYARISIZ ve doğru hal
func GetCurrentReward() int {
	// Legacy compatibility helper. Consensus paths use
	// ExpectedSubsidyForHeightV1 with an explicit chain height.
	return RewardAtHeightV1(1)
}

func NewBlockchain(initialReward, totalSupply int) *Blockchain {
	return newBlockchainWithGenesisRecipientV1(initialReward, totalSupply, "genesis-recipient", config.Current(), os.Getenv("QC_MAIN_ADDRESS"))
}

// NewIsolatedLocalBlockchainV1 commits the local domain to the genesis
// transaction digest and therefore to the genesis PoW header.
func NewIsolatedLocalBlockchainV1(initialReward, totalSupply int) *Blockchain {
	return newBlockchainWithGenesisRecipientV1(initialReward, totalSupply, IsolatedLocalGenesisDomainV1, config.Current(), os.Getenv("QC_MAIN_ADDRESS"))
}

func newBlockchainWithGenesisRecipientV1(initialReward, totalSupply int, genesisRecipient string, p *config.Config, legacyFallback string) *Blockchain {
	var txs []*Transaction

	genesisTime := CanonicalGenesisTimeV1()
	switch genesisRecipient {
	case IsolatedLocalGenesisDomainV1:
		// Keep prefund outpoints distinct even when the configured owners match.
		genesisTime = genesisTime.Add(-time.Second)
	case TestnetGenesisDomainV2:
		genesisTime = genesisTime.Add(-2 * time.Second)
	case MainnetGenesisDomainV2:
		genesisTime = genesisTime.Add(-3 * time.Second)
	case TestnetGenesisDomainV3:
		genesisTime = genesisTime.Add(-4 * time.Second)
	case MainnetGenesisDomainV3:
		genesisTime = genesisTime.Add(-5 * time.Second)
	case TestnetGenesisDomainV4:
		genesisTime = genesisTime.Add(-6 * time.Second)
	case MainnetGenesisDomainV4:
		genesisTime = genesisTime.Add(-7 * time.Second)
	}

	if initialReward <= 0 {
		initialReward = InitialRewardDefault
	}

	if totalSupply == int(QCTotalSupplyHardCap) {
		initialReward = CanonicalInitialMiningRewardV1
	}
	// The new opt-in domain marker commits identity with zero issuance.
	if isPublicNetworkGenesisDomainV1(genesisRecipient) {
		initialReward = 0
	}

	genesisCoinbase := &Transaction{
		ID:        nil,
		Inputs:    []TransactionInput{},
		Outputs:   []TransactionOutput{{Amount: initialReward, PubKeyHash: []byte(genesisRecipient)}},
		Timestamp: genesisTime,
		Sender:    "COINBASE",
		Amount:    float64(initialReward),
	}
	genesisCoinbase.ID = genesisCoinbase.Hash()
	if networkSignatureDomainV2(genesisRecipient) {
		id, err := CanonicalTransactionIDV2(genesisCoinbase)
		if err != nil {
			panic(err)
		}
		genesisCoinbase.ID = id
	}
	txs = append(txs, genesisCoinbase)

	premineFallbackAddr := strings.TrimSpace(
		p.PremineAddress,
	)
	if premineFallbackAddr == "" {
		premineFallbackAddr = strings.TrimSpace(
			legacyFallback,
		)
	}

	liquidAddr := strings.TrimSpace(
		p.PremineLiquidAddress,
	)
	if liquidAddr == "" {
		liquidAddr = premineFallbackAddr
	}

	lockedAddr := strings.TrimSpace(
		p.PremineLockedAddress,
	)
	if lockedAddr == "" {
		lockedAddr = premineFallbackAddr
	}

	liquidPercent := p.PremineLiquidPercent
	lockedPercent := p.PremineLockedPercent

	if totalSupply == int(QCTotalSupplyHardCap) {
		liquidPercent = 1
		lockedPercent = 4
	}

	if liquidPercent < 0 {
		liquidPercent = 0
	}
	if liquidPercent > 100 {
		liquidPercent = 100
	}

	if lockedPercent < 0 {
		lockedPercent = 0
	}
	if lockedPercent > 100 {
		lockedPercent = 100
	}

	var premineOutputs []TransactionOutput
	premineAmount := 0

	if totalSupply > 0 &&
		liquidAddr != "" &&
		liquidPercent > 0 {
		liquidAmount :=
			totalSupply * liquidPercent / 100

		if liquidAmount > 0 {
			premineOutputs = append(
				premineOutputs,
				TransactionOutput{
					Amount: liquidAmount,
					PubKeyHash: wallet.Base58DecodeAddress(
						liquidAddr,
					),
				},
			)

			premineAmount += liquidAmount
		}
	}

	if totalSupply > 0 &&
		lockedAddr != "" &&
		lockedPercent > 0 {
		lockedAmount :=
			totalSupply * lockedPercent / 100

		if lockedAmount > 0 {
			premineOutputs = append(
				premineOutputs,
				TransactionOutput{
					Amount: lockedAmount,
					PubKeyHash: wallet.Base58DecodeAddress(
						lockedAddr,
					),
				},
			)

			premineAmount += lockedAmount
		}
	}

	if len(premineOutputs) > 0 &&
		premineAmount > 0 {
		premineTransaction := &Transaction{
			ID:        nil,
			Inputs:    []TransactionInput{},
			Outputs:   premineOutputs,
			Timestamp: genesisTime,
			Sender:    "COINBASE",
			Amount:    float64(premineAmount),
		}

		premineTransaction.ID =
			premineTransaction.Hash()
		if networkSignatureDomainV2(genesisRecipient) {
			id, err := CanonicalTransactionIDV2(premineTransaction)
			if err != nil {
				panic(err)
			}
			premineTransaction.ID = id
		}

		txs = append(
			txs,
			premineTransaction,
		)
	}

	genesis, err :=
		NewCanonicalGenesisBlockV1(txs)
	if err != nil {
		log.Panicf(
			"create canonical genesis: %v",
			err,
		)
	}

	blockchain := &Blockchain{
		Blocks:      []*Block{genesis},
		UTXO:        map[string][]TransactionOutput{},
		TotalSupply: totalSupply,
		pendingTxs:  []*Transaction{},
	}
	blockchain.UpdateUTXOSet()
	blockchain.rebuildTransactionLocationIndexesLocked()

	return blockchain
}

func (bc *Blockchain) SetCoinbaseMaturity(n int) {
	if bc == nil {
		return
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()
	if n < 0 {
		n = 0
	}
	bc.coinbaseMaturity = n
}

func (bc *Blockchain) AddBlock(txs []*Transaction, miner string, difficulty int) *Block {
	if bc == nil {
		return nil
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()
	// Blok içindeki işlemleri doğrula (coinbase hariç imza zorunlu)
	if err := bc.validateBlockTxs(txs); err != nil {
		log.Printf("rejecting block: %v", err)
		return nil
	}

	prev := bc.Blocks[len(bc.Blocks)-1]
	if bc.usesMedianBlockTimeV2() {
		if err := bc.ValidateExpectedDifficultyV1(&Block{Index: prev.Index + 1, Difficulty: difficulty}); err != nil {
			return nil
		}
	}
	now, err := bc.nextMiningTimeV1(time.Now().UTC())
	if err != nil {
		return nil
	}
	nb, err := newBlockAtProtocolAndTimeV1(
		prev.Index+1,
		txs,
		prev.Hash,
		miner,
		difficulty,
		nil,
		bc.ProtocolAtHeightV1(prev.Index+1),
		now.Unix(),
	)
	if err != nil {
		log.Printf("rejecting block: proof of work: %v", err)
		return nil
	}
	if bc.usesMedianBlockTimeV2() {
		if err := bc.ValidatePeerBlockStrictV2(nb); err != nil {
			log.Printf("rejecting block: canonical validation: %v", err)
			return nil
		}
	} else if _, err := bc.ValidateBlockRewardWithFeesV1(nb); err != nil {
		log.Printf("rejecting block: reward validation: %v", err)
		return nil
	}
	bc.Blocks = append(bc.Blocks, cloneBlockV1(nb))
	bc.UpdateUTXOSet()
	bc.indexConfirmedBlockTransactionsLocked(nb)

	return nb
}

// BLOCK_REJECT_SECURITY_EVENTS_V1
func logPeerBlockReject(eventType string, blk *Block, err error) {
	msg := "peer block rejected"
	if err != nil {
		msg = err.Error()
	}

	height := int64(-1)
	hash := ""
	if blk != nil {
		height = int64(blk.Index)
		if len(blk.Hash) > 0 {
			hash = hex.EncodeToString(blk.Hash)
		}
	}

	_ = security.AppendEvent(".", security.NewEvent(
		eventType,
		security.SeverityHigh,
		"blockchain_peer_block",
		hash,
		msg,
		map[string]any{
			"height": height,
			"hash":   hash,
		},
	))
}

func (bc *Blockchain) addBlockFromPeerAfterStrictV2(blk *Block) error {
	if bc == nil {
		return fmt.Errorf("nil blockchain")
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()
	// Detach the submitted block and revalidate against current state at commit.
	blk = cloneBlockV1(blk)
	if err := bc.ValidatePeerBlockStrictV2(blk); err != nil {
		return err
	}
	if blk == nil {
		logPeerBlockReject("BLOCK_NIL_REJECTED", blk, ErrNilBlock)
		return ErrNilBlock
	}

	if err := ValidateBlockBasic(blk); err != nil {
		wrapped := fmt.Errorf("peer block basic validation failed: %w", err)
		logPeerBlockReject("BLOCK_BASIC_VALIDATION_FAILED", blk, wrapped)
		return wrapped
	}

	if err := bc.ValidateBlockPoWAtHeightV1(blk); err != nil {
		logPeerBlockReject("BLOCK_INVALID_POW", blk, ErrInvalidPoW)
		return ErrInvalidPoW
	}

	if len(bc.Blocks) > 0 {
		last := bc.Blocks[len(bc.Blocks)-1]
		if !bytes.Equal(blk.PrevHash, last.Hash) {
			logPeerBlockReject("BLOCK_PREVHASH_REJECTED", blk, ErrPrevHashMismatch)
			return ErrPrevHashMismatch
		}
	}
	if err := bc.ValidateBlockSubsidyV1(blk); err != nil {
		wrapped := fmt.Errorf(
			"peer block subsidy validation failed: %w",
			err,
		)
		logPeerBlockReject(
			"BLOCK_SUBSIDY_VALIDATION_FAILED",
			blk,
			wrapped,
		)
		return wrapped
	}

	if err := bc.ValidateFounderVestingBlockV1(blk); err != nil {
		wrapped := fmt.Errorf(
			"peer block founder vesting validation failed: %w",
			err,
		)
		logPeerBlockReject(
			"BLOCK_FOUNDER_VESTING_REJECTED",
			blk,
			wrapped,
		)
		return wrapped
	}

	if err := bc.validateBlockTxsAtHeightV1(
		blk.Transactions,
		blk.Index,
	); err != nil {
		wrapped := fmt.Errorf("peer block tx validation failed: %w", err)
		logPeerBlockReject("BLOCK_TX_VALIDATION_FAILED", blk, wrapped)
		return wrapped
	}
	bc.Blocks = append(bc.Blocks, blk)
	bc.UpdateUTXOSet()
	bc.indexConfirmedBlockTransactionsLocked(blk)

	return nil
}

func (bc *Blockchain) IsValidChain() bool {
	if bc == nil {
		return false
	}

	if err := validateNFTMintChainUniquenessV1(bc.Blocks); err != nil {
		return false
	}

	for i, block := range bc.Blocks {
		if block == nil {
			return false
		}

		if err := bc.validateNFTMintWithLocalContextV1(bc.Blocks[:i], block); err != nil {
			return false
		}

		if i == 0 {
			continue
		}

		if bc.ValidateBlockPoWAtHeightV1(block) != nil ||
			!bytes.Equal(block.PrevHash, bc.Blocks[i-1].Hash) {
			return false
		}
	}

	return true
}

func (bc *Blockchain) GetHeight() int { return len(bc.Blocks) - 1 }

// FORK_REORG_SECURITY_ALERTS_V1
func logChainReorgEvent(eventType string, severity security.EventSeverity, message string, localHeight int, incomingHeight int, meta map[string]any) {
	if meta == nil {
		meta = map[string]any{}
	}
	meta["localHeight"] = localHeight
	meta["incomingHeight"] = incomingHeight

	_ = security.AppendEvent(".", security.NewEvent(
		eventType,
		severity,
		"blockchain_reorg",
		"",
		message,
		meta,
	))
}

func (bc *Blockchain) replaceChainAfterStrictV1(blocks []*Block) error {
	if bc == nil {
		return fmt.Errorf("nil blockchain")
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()
	// Commit only a detached chain revalidated with current local NFT context.
	blocks = cloneBlocksV1(blocks)
	if err := bc.ValidateReplacementChainStrictV1(blocks); err != nil {
		return err
	}
	localHeight := len(bc.Blocks) - 1
	incomingHeight := len(blocks) - 1

	logChainReorgEvent(
		"FORK_REORG_ATTEMPT_DETECTED",
		security.SeverityWarning,
		"Incoming chain replacement attempt detected.",
		localHeight,
		incomingHeight,
		map[string]any{},
	)

	// The strict replay above checks PoW, linkage, authorization and spends in
	// each candidate prefix while holding stateMu. Revalidating historical
	// transactions against the old tip's UTXO incorrectly rejects spent common
	// history and valid competing spends. Install only the detached replayed
	// candidate, then rebuild its derived state under the same writer lock.
	bc.Blocks = cloneBlocksV1(blocks)
	bc.UpdateUTXOSet()
	bc.pendingTxs = []*Transaction{}
	bc.rebuildTransactionLocationIndexesLocked()
	logChainReorgEvent(
		"CHAIN_REPLACED_REORG_WARNING",
		security.SeverityHigh,
		"Local chain replaced by valid incoming chain with greater cumulative work.",
		localHeight,
		incomingHeight,
		map[string]any{},
	)

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

			rawTxID, err := hex.DecodeString(txID)
			if err != nil {
				continue
			}
			if !bc.isCoinbaseOutputMature(rawTxID, idx) {
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

// MEMPOOL_SECURITY_EVENTS_V1
func logMempoolReject(eventType string, tx *Transaction, err error) {
	msg := "mempool transaction rejected"
	if err != nil {
		msg = err.Error()
	}

	txid := ""
	if tx != nil && len(tx.ID) > 0 {
		txid = hex.EncodeToString(tx.ID)
	}
	subject := ""
	if tx != nil {
		subject = tx.Sender
	}
	security.LiveEvidence.ObserveSpend(txid, subject, msg)

	_ = security.AppendEvent(".", security.NewEvent(
		eventType,
		security.SeverityWarning,
		"blockchain_mempool",
		txid,
		msg,
		map[string]any{
			"txid": txid,
		},
	))
}

// AddTransaction documents the QuantumCoin component used by this module.
// --- İMZA ZORUNLULUĞU: mempool’a eklemeden önce doğrula ---
func (bc *Blockchain) AddTransaction(tx *Transaction) error {
	if bc == nil {
		err := fmt.Errorf("nil blockchain")
		logMempoolReject("MEMPOOL_NIL_BLOCKCHAIN", tx, err)
		return err
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()
	if tx == nil {
		logMempoolReject("MEMPOOL_NIL_TX", tx, ErrNilTransaction)
		return ErrNilTransaction
	}
	tx = cloneTransactionV1(tx)

	if err := ValidateTransactionBasic(tx); err != nil {
		logMempoolReject("MEMPOOL_INVALID_TX_BASIC", tx, err)
		return err
	}

	if tx.IsCoinbase() {
		err := fmt.Errorf("coinbase tx cannot be added to mempool")
		logMempoolReject("MEMPOOL_COINBASE_REJECTED", tx, err)
		return err
	}

	if len(tx.Inputs) == 0 {
		err := fmt.Errorf("empty inputs")
		logMempoolReject("MEMPOOL_EMPTY_INPUTS", tx, err)
		return err
	}
	if len(tx.Outputs) == 0 {
		err := fmt.Errorf("empty outputs")
		logMempoolReject("MEMPOOL_EMPTY_OUTPUTS", tx, err)
		return err
	}

	if !bc.verifyTransactionLockedV2(tx) {
		err := fmt.Errorf("invalid tx signature")
		logMempoolReject("MEMPOOL_INVALID_SIGNATURE", tx, err)
		return err
	}

	if bc.UTXO == nil {
		bc.UpdateUTXOSet()
	}

	if err := bc.validateTransactionSpend(tx); err != nil {
		eventType := "MEMPOOL_SPEND_VALIDATION_REJECTED"
		errText := err.Error()

		switch {
		case strings.Contains(errText, "input already pending in mempool"):
			eventType = "MEMPOOL_DOUBLE_SPEND_ATTEMPT"
		case strings.Contains(errText, "double spend inside transaction"):
			eventType = "TX_INTERNAL_DOUBLE_SPEND_ATTEMPT"
		case strings.Contains(errText, "missing utxo"):
			eventType = "TX_MISSING_UTXO_REJECTED"
		case strings.Contains(errText, "spent or empty utxo"):
			eventType = "TX_SPENT_OR_EMPTY_UTXO_REJECTED"
		case strings.Contains(errText, "coinbase output is not mature"):
			eventType = "TX_COINBASE_MATURITY_REJECTED"
		}

		logMempoolReject(eventType, tx, err)
		return err
	}
	bc.pendingTxs = append(bc.pendingTxs, tx)
	bc.indexPendingTransactionLocked(tx)
	return nil
}

// validateTransactionSpend normal transaction için UTXO, input/output toplamı
// ve mempool double-spend kontrollerini yapar.
func (bc *Blockchain) validateTransactionSpend(tx *Transaction) error {
	_, err := bc.ValidateTransactionUTXOV1(tx, true)
	return err
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
	spend := 0

	for _, block := range bc.Blocks {
		if block == nil {
			continue
		}
		for _, tx := range block.Transactions {
			if tx == nil {
				continue
			}
			for outIdx, out := range tx.Outputs {
				if !out.IsLockedWithKey(pubKeyHash) || bc.isOutputSpent(tx.ID, outIdx) {
					continue
				}
				if !bc.isCoinbaseOutputMature(tx.ID, outIdx) {
					continue
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
	if bc == nil || len(bc.Blocks) == 0 {
		return 0
	}

	total := 0
	genesis := bc.Blocks[0]
	if genesis != nil {
		for _, tx := range genesis.Transactions {
			if tx == nil || !tx.IsCoinbase() {
				continue
			}
			for _, output := range tx.Outputs {
				if output.Amount <= 0 {
					continue
				}
				next := total + output.Amount
				if next < total {
					return total
				}
				total = next
			}
		}
	}

	for blockPosition := 1; blockPosition < len(bc.Blocks); blockPosition++ {
		block := bc.Blocks[blockPosition]
		if block == nil {
			continue
		}

		subsidy := ExpectedSubsidyFromStateV1(
			block.Index,
			total,
			bc.TotalSupply,
		)
		if subsidy <= 0 {
			continue
		}

		next := total + subsidy
		if next < total {
			return total
		}
		total = next
	}

	return total
}

func (bc *Blockchain) MineBlock(miner string, difficulty int) (*Block, error) {
	return bc.MineBlockContextV1(context.Background(), miner, difficulty)
}

// MineBlockContextV1 cancels search/admission waiting, never an in-flight commit.
func (bc *Blockchain) MineBlockContextV1(ctx context.Context, miner string, difficulty int) (*Block, error) {
	if bc == nil {
		return nil, ErrChainNotInitialized
	}

	if err := lockMiningContextV1(ctx, &bc.stateMu); err != nil {
		return nil, err
	}
	defer bc.stateMu.Unlock()
	return bc.mineBlockWithNFTCommitContextLockedV1(ctx, miner, difficulty, nil, nil, "")
}

// MineBlockAndPersistV1 installs a normal mined block on disk before
// committing it to live state. The path comes from trusted node configuration.
func (bc *Blockchain) MineBlockAndPersistV1(miner string, difficulty int, filename string) (*Block, error) {
	return bc.MineBlockAndPersistContextV1(context.Background(), miner, difficulty, filename)
}

func (bc *Blockchain) MineBlockAndPersistContextV1(ctx context.Context, miner string, difficulty int, filename string) (*Block, error) {
	if bc == nil {
		return nil, ErrChainNotInitialized
	}
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return nil, fmt.Errorf("persistent mining: chain filename is empty")
	}
	if err := lockMiningContextV1(ctx, &bc.stateMu); err != nil {
		return nil, err
	}
	defer bc.stateMu.Unlock()
	return bc.mineBlockWithNFTCommitContextLockedV1(ctx, miner, difficulty, nil, nil, filename)
}

// The optional persistent path installs the validated candidate on disk before
// the existing authoritative RAM/UTXO/index/mempool commit and advisory hooks.
func (bc *Blockchain) mineBlockWithNFTCommitLockedV1(miner string, difficulty int, records []NFTMintRecordV1, proofs []NFTAwardProofV1, chainFile string) (*Block, error) {
	return bc.mineBlockWithNFTCommitContextLockedV1(context.Background(), miner, difficulty, records, proofs, chainFile)
}

func (bc *Blockchain) mineBlockWithNFTCommitContextLockedV1(ctx context.Context, miner string, difficulty int, records []NFTMintRecordV1, proofs []NFTAwardProofV1, chainFile string) (*Block, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(bc.Blocks) == 0 || bc.Blocks[len(bc.Blocks)-1] == nil {
		return nil, ErrChainNotInitialized
	}

	nextHeight := bc.Blocks[len(bc.Blocks)-1].Index + 1
	if err := bc.validateLocalMiningProtocolV1(
		nextHeight,
	); err != nil {
		return nil, fmt.Errorf(
			"local mining protocol gate: %w",
			err,
		)
	}
	if bc.usesMedianBlockTimeV2() || bc.ProtocolAtHeightV1(nextHeight) == qconsensus.ProtocolNFTV4 {
		if err := bc.ValidateExpectedDifficultyV1(&Block{Index: nextHeight, Difficulty: difficulty}); err != nil {
			return nil, fmt.Errorf("local mining difficulty: %w", err)
		}
	}
	now, err := bc.nextMiningTimeV1(time.Now().UTC())
	if err != nil {
		return nil, err
	}
	txs, err := bc.miningTransactionsLockedV1(miner, nextHeight, now)
	if err != nil {
		return nil, err
	}

	prev := bc.Blocks[len(bc.Blocks)-1]
	var nb *Block
	if len(records) != 0 || len(proofs) != 0 {
		nb, err = newNFTBlockWithAwardProofAndTimeContextV1(ctx, prev.Index+1, txs, prev.Hash, miner, difficulty, records, proofs, now.Unix())
	} else {
		nb, err = newBlockAtProtocolAndTimeContextV1(ctx, prev.Index+1, txs, prev.Hash, miner, difficulty, nil, bc.ProtocolAtHeightV1(prev.Index+1), now.Unix())
	}
	if err != nil {
		return nil, fmt.Errorf("mined block creation: %w", err)
	}
	committed := cloneBlockV1(nb)
	if bc.usesMedianBlockTimeV2() || bc.ProtocolAtHeightV1(nextHeight) == qconsensus.ProtocolNFTV4 {
		// Use the already implemented strict canonical gate before any mutation.
		if err := bc.ValidatePeerBlockStrictV2(nb); err != nil {
			return nil, fmt.Errorf("mined block canonical validation: %w", err)
		}
		// Returned evidence and pending/caller pointers cannot mutate live state.
	} else {
		if _, err := bc.ValidateBlockRewardWithFeesV1(nb); err != nil {
			return nil, fmt.Errorf("mined block reward: %w", err)
		}
		if err := bc.ValidateFounderVestingBlockV1(nb); err != nil {
			return nil, fmt.Errorf("mined block founder vesting: %w", err)
		}
	}
	// Cancellation wins until this commit barrier. Once installation starts,
	// complete disk and RAM together; do not return a false cancellation.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if chainFile != "" {
		if err := bc.persistNFTMiningCandidateLockedV1(committed, chainFile); err != nil {
			return nil, err
		}
	}
	bc.Blocks = append(bc.Blocks, committed)
	bc.UpdateUTXOSet()
	bc.indexConfirmedBlockTransactionsLocked(nb)
	bc.pendingTxs = []*Transaction{} // mempool’u boşalt
	bc.clearPendingTransactionIndexLocked()

	return nb, nil
}

func SerializeBlockchain(bc *Blockchain) []byte {
	if bc == nil {
		return nil
	}
	bc.stateMu.RLock()
	snapshot := &Blockchain{
		Blocks:      cloneBlocksV1(bc.Blocks),
		UTXO:        cloneUTXOForState(bc.UTXO),
		TotalSupply: bc.TotalSupply,
	}
	if bc.StoredNetworkIdentityV1 != nil {
		identity := *bc.StoredNetworkIdentityV1
		snapshot.StoredNetworkIdentityV1 = &identity
	}
	bc.stateMu.RUnlock()
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(snapshot); err != nil {
		log.Panicf("serialize error: %v", err)
	}
	return buf.Bytes()
}

func DeserializeBlockchain(data []byte) *Blockchain {
	var bc Blockchain
	if err := decodeChainGobV1(bytes.NewReader(data), &bc); err != nil {
		return nil
	}
	if bc.pendingTxs == nil {
		bc.pendingTxs = []*Transaction{}
	}
	bc.rebuildTransactionLocationIndexesLocked()
	return &bc
}

func (bc *Blockchain) SaveToFile(filename string) error {
	if bc == nil {
		return fmt.Errorf("nil blockchain")
	}

	// Lock order: chain state -> destination mutex -> OS lock. Hold state
	// through installation, so an older save cannot overtake a newer commit.
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	if err := bc.validateBoundNetworkIdentityLockedV1(); err != nil {
		return err
	}
	return persistChainFileV1(filename, func(writer io.Writer) error {
		return encodeDiskChainV1(writer, bc)
	}, bc.networkIdentity)
}

func LoadBlockchainFromFile(filename string) (*Blockchain, error) {
	loaded, err := loadDiskChainV1(filename, nil)
	if err != nil {
		return nil, fmt.Errorf("read blockchain file: %w", err)
	}
	loaded.rebuildTransactionLocationIndexesLocked()
	for index, block := range loaded.Blocks {
		if block == nil {
			return nil, fmt.Errorf("load blockchain NFT validation: nil block at index %d", index)
		}
		if len(block.NFTMintRecordsV1) == 0 && len(block.NFTAwardProofsV1) == 0 {
			continue
		}
		if err := validateNFTMintAuthorizationV1(
			block,
			loaded.ProtocolAtHeightV1(block.Index),
		); err != nil {
			return nil, fmt.Errorf("load blockchain NFT validation at block[%d]: %w", index, err)
		}
	}
	return loaded, nil
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

func (bc *Blockchain) isCoinbaseOutputMature(txid []byte, outIdx int) bool {
	if bc == nil || len(txid) == 0 || outIdx < 0 {
		return false
	}

	origin, ok := bc.utxoOriginsV1()[outpointKeyV1(txid, outIdx)]
	if !ok {
		return false
	}

	return bc.validateCoinbaseMaturityAtHeightV1(
		origin,
		outpointKeyV1(txid, outIdx),
		bc.nextSpendHeightV1(),
	) == nil
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
func (bc *Blockchain) validateBlockTxsAtHeightV1(
	txs []*Transaction,
	spendHeight int,
) error {
	if bc == nil {
		return fmt.Errorf("nil blockchain")
	}
	if len(txs) == 0 {
		return fmt.Errorf("empty block transactions")
	}

	coinbaseCount := 0
	for txIndex, tx := range txs {
		if tx == nil {
			return fmt.Errorf("nil tx at index %d", txIndex)
		}

		if err := ValidateTransactionBasic(tx); err != nil {
			return fmt.Errorf("tx[%d] basic validation: %w", txIndex, err)
		}

		if !tx.IsCoinbase() {
			continue
		}

		coinbaseCount++
		if coinbaseCount > 1 {
			return fmt.Errorf("multiple coinbase transactions in block")
		}
		if len(tx.Outputs) == 0 {
			return fmt.Errorf("invalid coinbase (no outputs)")
		}
	}

	if _, err := bc.ValidateTransactionsUTXOAtHeightV1(
		txs,
		spendHeight,
	); err != nil {
		return fmt.Errorf("canonical UTXO validation failed: %w", err)
	}

	return nil
}

func (bc *Blockchain) validateBlockTxs(txs []*Transaction) error {
	if bc == nil {
		return fmt.Errorf("nil blockchain")
	}

	return bc.validateBlockTxsAtHeightV1(
		txs,
		bc.nextSpendHeightV1(),
	)
}

// PendingTxs documents the QuantumCoin component used by this module.
// pendingTxs'in güvenli kopyası (API/mine kullanımı için)
func (bc *Blockchain) PendingTxs() []*Transaction {
	if bc == nil || bc.pendingTxs == nil {
		return []*Transaction{}
	}
	return cloneTransactionsForState(bc.pendingTxs)
}
