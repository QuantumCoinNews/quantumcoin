package miner

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

	mu     sync.Mutex
	issued map[string]*blockchain.Block
}

// resolveQCLocalChainPathV1 chooses an explicit command-line override when
// supplied; otherwise the validated config chain file remains authoritative.
func resolveQCLocalChainPathV1(
	cfg *config.Config,
	requestedPath string,
) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf(
			"qc local chain path: nil config",
		)
	}

	chainPath := strings.TrimSpace(
		requestedPath,
	)
	if chainPath == "" {
		chainPath = strings.TrimSpace(
			cfg.ChainFile,
		)
	}

	if chainPath == "" {
		return "", fmt.Errorf(
			"qc local chain path: path is empty",
		)
	}

	chainPath = filepath.Clean(
		chainPath,
	)

	if err := config.ValidateChainFilePathForModeV1(
		chainPath,
	); err != nil {
		return "", fmt.Errorf(
			"qc local chain path mode guard: %w",
			err,
		)
	}

	return chainPath, nil
}

// NewQCLocalAdapter documents the QuantumCoin component used by this module.
// Adapter kurulum: config + chain yükle
func NewQCLocalAdapter(
	op QCLocalOpts,
) (Backend, error) {
	if executable, err := os.Executable(); err == nil {
		_ = os.Chdir(
			filepath.Dir(executable),
		)
	}

	cfg, err := config.Load(
		op.ConfigPath,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"config load: %w",
			err,
		)
	}

	chainPath, err :=
		resolveQCLocalChainPathV1(
			cfg,
			op.ChainPath,
		)
	if err != nil {
		return nil, err
	}

	cfg.ChainFile = chainPath
	op.ChainPath = chainPath

	if err := config.ValidateRuntimeNetworkProfileV1(
		cfg,
	); err != nil {
		return nil, fmt.Errorf(
			"adapter runtime profile: %w",
			err,
		)
	}

	var chain *blockchain.Blockchain

	_, statErr := os.Stat(
		chainPath,
	)

	switch {
	case statErr == nil:
		chain, err =
			blockchain.LoadBlockchainFromFile(
				chainPath,
			)
		if err != nil {
			return nil, fmt.Errorf(
				"chain load %q: %w",
				chainPath,
				err,
			)
		}

	case os.IsNotExist(statErr):
		chain = blockchain.NewBlockchain(
			cfg.InitialReward,
			cfg.TotalSupply,
		)

	default:
		return nil, fmt.Errorf(
			"chain stat %q: %w",
			chainPath,
			statErr,
		)
	}

	if err := chain.BindCanonicalActivationPolicySynchronizedV1(); err != nil {
		return nil, fmt.Errorf(
			"bind canonical activation policy v1: %w",
			err,
		)
	}

	chain.SetCoinbaseMaturity(
		cfg.CoinbaseMaturity,
	)

	return &qcLocal{
		cfg: cfg,
		bc:  chain,
		op:  op,
		issued: make(
			map[string]*blockchain.Block,
		),
	}, nil
}

const maxIssuedQCLocalTemplatesV1 = 128

func targetFromBits(bits int) *big.Int {
	if bits <= 0 {
		bits = 16
	}

	if bits > 255 {
		bits = 255
	}

	target := big.NewInt(1)

	return target.Lsh(
		target,
		uint(256-bits),
	)
}

func cloneQCLocalTemplateV1(
	source *blockchain.Block,
) *blockchain.Block {
	if source == nil {
		return nil
	}

	metadata := make(
		map[string]string,
		len(source.Metadata),
	)

	for key, value := range source.Metadata {
		metadata[key] = value
	}

	return &blockchain.Block{
		Index:     source.Index,
		Timestamp: source.Timestamp,
		Transactions: append(
			[]*blockchain.Transaction(nil),
			source.Transactions...,
		),
		PrevHash: append(
			[]byte(nil),
			source.PrevHash...,
		),
		Hash: append(
			[]byte(nil),
			source.Hash...,
		),
		Nonce:      source.Nonce,
		Miner:      source.Miner,
		Difficulty: source.Difficulty,
		Metadata:   metadata,
	}
}

func qCLocalTemplateTokenV1(
	template *blockchain.Block,
) (string, error) {
	if template == nil {
		return "", fmt.Errorf(
			"qc local template token: nil template",
		)
	}

	baseHash, err := blockchain.ComputePoWHashV1(
		template,
		0,
	)
	if err != nil {
		return "", fmt.Errorf(
			"qc local template token: %w",
			err,
		)
	}

	return fmt.Sprintf(
		"%x:%d:%s",
		baseHash,
		template.Index,
		template.Miner,
	), nil
}

func (q *qcLocal) GetWork(
	_ context.Context,
	minerAddr string,
) (*Work, error) {
	if q == nil || q.cfg == nil || q.bc == nil {
		return nil, fmt.Errorf(
			"qc local get work: adapter is not initialized",
		)
	}

	minerAddr = strings.TrimSpace(minerAddr)

	if minerAddr == "" {
		return nil, fmt.Errorf(
			"qc local get work: miner address is empty",
		)
	}

	pubKeyHash := wallet.Base58DecodeAddress(
		minerAddr,
	)
	if len(pubKeyHash) == 0 {
		return nil, fmt.Errorf(
			"qc local get work: invalid miner address",
		)
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	last := q.bc.TipSnapshot()
	if last == nil {
		return nil, fmt.Errorf(
			"qc local get work: chain is empty",
		)
	}

	difficulty, err := resolveRuntimeDifficultyV1(
		q.cfg,
		q.bc,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"qc local get work difficulty: %w",
			err,
		)
	}
	reward := q.bc.ExpectedSubsidyForHeightSnapshot(
		q.bc.BestHeightSnapshot() + 1,
	)
	if reward <= 0 {
		return nil, fmt.Errorf(
			"qc local get work: mining reward is zero",
		)
	}
	now := time.Now().UTC()

	coinbase := &blockchain.Transaction{
		ID:     nil,
		Inputs: []blockchain.TransactionInput{},
		Outputs: []blockchain.TransactionOutput{
			{
				Amount:     reward,
				PubKeyHash: pubKeyHash,
			},
		},
		Timestamp: now,
		Sender:    "COINBASE",
		Amount:    float64(reward),
	}

	coinbase.ID = coinbase.Hash()

	template := &blockchain.Block{
		Index:     last.Index + 1,
		Timestamp: now.Unix(),
		Transactions: []*blockchain.Transaction{
			coinbase,
		},
		PrevHash: append(
			[]byte(nil),
			last.Hash...,
		),
		Hash:       nil,
		Nonce:      0,
		Miner:      minerAddr,
		Difficulty: difficulty,
		Metadata: map[string]string{
			"ext_miner":        "cmd",
			"template_version": "QC_LOCAL_TEMPLATE_V1",
		},
	}

	token, err := qCLocalTemplateTokenV1(
		template,
	)
	if err != nil {
		return nil, err
	}

	if q.issued == nil ||
		len(q.issued) >= maxIssuedQCLocalTemplatesV1 {
		q.issued = make(
			map[string]*blockchain.Block,
		)
	}

	q.issued[token] = cloneQCLocalTemplateV1(
		template,
	)

	return &Work{
		Left:  []byte(token),
		Right: nil,
		Target: targetFromBits(
			difficulty,
		),
		Height: template.Index,
		Miner:  template.Miner,
		Template: cloneQCLocalTemplateV1(
			template,
		),
	}, nil
}

func (q *qcLocal) Submit(
	_ context.Context,
	work *Work,
	nonce uint64,
	hashHex string,
) (bool, error) {
	if q == nil || q.cfg == nil || q.bc == nil {
		return false, fmt.Errorf(
			"qc local submit: adapter is not initialized",
		)
	}

	if work == nil {
		return false, fmt.Errorf(
			"qc local submit: nil work",
		)
	}

	token := strings.TrimSpace(
		string(work.Left),
	)
	if token == "" {
		return false, fmt.Errorf(
			"qc local submit: missing template token",
		)
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.issued == nil {
		return false, fmt.Errorf(
			"qc local submit: no issued templates",
		)
	}

	issuedTemplate, exists := q.issued[token]
	if !exists || issuedTemplate == nil {
		return false, fmt.Errorf(
			"qc local submit: unknown or expired template",
		)
	}

	// Her issued work yalnızca bir kez submit edilebilir.
	defer delete(
		q.issued,
		token,
	)

	last := q.bc.TipSnapshot()
	if last == nil {
		return false, fmt.Errorf(
			"qc local submit: chain is empty",
		)
	}

	expectedDifficulty, err :=
		resolveRuntimeDifficultyV1(
			q.cfg,
			q.bc,
			0,
		)
	if err != nil {
		return false, fmt.Errorf(
			"qc local submit difficulty: %w",
			err,
		)
	}

	block, err := prepareQCLocalSubmissionV1(
		last,
		issuedTemplate,
		work,
		nonce,
		hashHex,
		expectedDifficulty,
	)
	if err != nil {
		return false, err
	}

	if err := q.bc.CommitValidatedBlockAndPersist(
		block,
		q.cfg.ChainFile,
	); err != nil {
		return false, fmt.Errorf(
			"qc local submit commit chain: %w",
			err,
		)
	}
	p2p.BroadcastMessage(
		p2p.BlockMessage(block),
	)

	return true, nil
}

func prepareQCLocalSubmissionV1(
	last *blockchain.Block,
	issuedTemplate *blockchain.Block,
	work *Work,
	nonce uint64,
	hashHex string,
	expectedDifficulty int,
) (*blockchain.Block, error) {
	if last == nil {
		return nil, fmt.Errorf(
			"qc local submit: nil chain tip",
		)
	}

	if issuedTemplate == nil {
		return nil, fmt.Errorf(
			"qc local submit: nil issued template",
		)
	}

	if work == nil || work.Template == nil {
		return nil, fmt.Errorf(
			"qc local submit: missing work template",
		)
	}

	if work.Target == nil {
		return nil, fmt.Errorf(
			"qc local submit: missing work target",
		)
	}

	if expectedDifficulty <= 0 {
		return nil, fmt.Errorf(
			"qc local submit: invalid expected difficulty %d",
			expectedDifficulty,
		)
	}

	if issuedTemplate.Index != last.Index+1 ||
		!bytes.Equal(
			issuedTemplate.PrevHash,
			last.Hash,
		) {
		return nil, fmt.Errorf(
			"qc local submit: stale work template",
		)
	}

	if work.Height != issuedTemplate.Index {
		return nil, fmt.Errorf(
			"qc local submit: work height mismatch",
		)
	}

	if work.Miner != issuedTemplate.Miner {
		return nil, fmt.Errorf(
			"qc local submit: work miner mismatch",
		)
	}

	if issuedTemplate.Difficulty != expectedDifficulty {
		return nil, fmt.Errorf(
			"qc local submit: difficulty changed from %d to %d",
			issuedTemplate.Difficulty,
			expectedDifficulty,
		)
	}

	expectedTarget := targetFromBits(
		issuedTemplate.Difficulty,
	)

	if work.Target.Cmp(expectedTarget) != 0 {
		return nil, fmt.Errorf(
			"qc local submit: work target mismatch",
		)
	}

	if issuedTemplate.Nonce != 0 ||
		len(issuedTemplate.Hash) != 0 {
		return nil, fmt.Errorf(
			"qc local submit: issued template was already finalized",
		)
	}

	issuedToken, err := qCLocalTemplateTokenV1(
		issuedTemplate,
	)
	if err != nil {
		return nil, err
	}

	workToken, err := qCLocalTemplateTokenV1(
		work.Template,
	)
	if err != nil {
		return nil, err
	}

	if issuedToken != workToken ||
		string(work.Left) != issuedToken {
		return nil, fmt.Errorf(
			"qc local submit: template identity mismatch",
		)
	}

	maxInt := uint64(^uint(0) >> 1)
	if nonce > maxInt {
		return nil, fmt.Errorf(
			"qc local submit: nonce exceeds platform int",
		)
	}

	providedHash, err := hex.DecodeString(
		strings.TrimSpace(hashHex),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"qc local submit: invalid hash encoding: %w",
			err,
		)
	}

	if len(providedHash) != 32 {
		return nil, fmt.Errorf(
			"qc local submit: invalid hash length %d",
			len(providedHash),
		)
	}

	meetsTarget, canonicalHash, err :=
		blockchain.PoWHashMeetsTargetUint64V1(
			issuedTemplate,
			nonce,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"qc local submit canonical pow: %w",
			err,
		)
	}

	if !meetsTarget {
		return nil, fmt.Errorf(
			"qc local submit: nonce does not meet target",
		)
	}

	if !bytes.Equal(
		providedHash,
		canonicalHash[:],
	) {
		return nil, fmt.Errorf(
			"qc local submit: provided hash mismatch",
		)
	}

	block := cloneQCLocalTemplateV1(
		issuedTemplate,
	)

	nonceInt, nonceErr :=
		blockchain.ConsensusNonceIntV1(nonce)
	if nonceErr != nil {
		return nil, fmt.Errorf(
			"qc local submit: invalid nonce: %w",
			nonceErr,
		)
	}

	block.Nonce = nonceInt
	block.Hash = append(
		[]byte(nil),
		canonicalHash[:]...,
	)

	if !block.ValidatePoW() {
		return nil, fmt.Errorf(
			"qc local submit: finalized block failed pow validation",
		)
	}

	return block, nil
}
