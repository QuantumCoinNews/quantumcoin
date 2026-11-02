package token

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// TOKEN_SAFE_BOUNDARY_V1_STORAGE_GUARD
// Registry persistence for QC20/QC token modules.
// Keeps token_registry.json format stable while adding nil-map,
// empty-symbol, and safer-write protections.
const registryFile = "token_registry.json"

type Registry struct {
	Tokens map[string]*Token `json:"tokens"` // key: symbol
	mu     sync.Mutex        `json:"-"`
}

func NewRegistry() *Registry {
	return &Registry{Tokens: make(map[string]*Token)}
}

func (r *Registry) ensure() {
	if r.Tokens == nil {
		r.Tokens = make(map[string]*Token)
	}
}

func (r *Registry) Register(t *Token) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ensure()

	if t == nil {
		return errors.New("token registry: nil token")
	}
	if t.Symbol() == "" {
		return errors.New("token registry: empty symbol")
	}
	if !validTokenAddress(t.Owner()) {
		return fmt.Errorf(
			"token registry: %w",
			ErrInvalidTokenAddress,
		)
	}
	if _, exists := r.Tokens[t.Symbol()]; exists {
		return errors.New("token registry: token already registered")
	}

	r.Tokens[t.Symbol()] = t
	return nil
}

func (r *Registry) Get(symbol string) (*Token, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ensure()

	t, ok := r.Tokens[symbol]
	return t, ok
}

func (r *Registry) Save(dir string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ensure()

	path := registryFile
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		path = filepath.Join(dir, registryFile)
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

var ErrRegistryLoadIntegrityV1 = errors.New(
	"token registry: loaded state failed integrity validation",
)

func registryLoadIntegrityErrorV1(
	format string,
	arguments ...any,
) error {
	return fmt.Errorf(
		"%w: %s",
		ErrRegistryLoadIntegrityV1,
		fmt.Sprintf(format, arguments...),
	)
}

type registryTokenMapV1 map[string]*Token

// UnmarshalJSON rejects duplicate registry keys before encoding/json
// can silently replace an earlier token entry with a later entry.
func (tokens *registryTokenMapV1) UnmarshalJSON(
	data []byte,
) error {
	if bytes.Equal(
		bytes.TrimSpace(data),
		[]byte("null"),
	) {
		*tokens = make(registryTokenMapV1)
		return nil
	}

	decoder := json.NewDecoder(
		bytes.NewReader(data),
	)

	opening, err := decoder.Token()
	if err != nil {
		return err
	}
	if opening != json.Delim('{') {
		return registryLoadIntegrityErrorV1(
			"tokens must be a JSON object",
		)
	}

	loaded := make(registryTokenMapV1)
	seen := make(map[string]struct{})

	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return err
		}

		key, ok := keyToken.(string)
		if !ok {
			return registryLoadIntegrityErrorV1(
				"registry key is not a string",
			)
		}

		if _, exists := seen[key]; exists {
			return registryLoadIntegrityErrorV1(
				"duplicate registry key %q",
				key,
			)
		}
		seen[key] = struct{}{}

		var tok *Token
		if err := decoder.Decode(&tok); err != nil {
			return err
		}

		loaded[key] = tok
	}

	closing, err := decoder.Token()
	if err != nil {
		return err
	}
	if closing != json.Delim('}') {
		return registryLoadIntegrityErrorV1(
			"tokens JSON object was not closed",
		)
	}

	*tokens = loaded
	return nil
}

func validateLoadedRegistryTokenV1(
	registryKey string,
	tok *Token,
	seenSymbols map[string]struct{},
) error {
	trimmedKey := strings.TrimSpace(registryKey)

	if trimmedKey == "" ||
		trimmedKey != registryKey {
		return registryLoadIntegrityErrorV1(
			"invalid registry key %q",
			registryKey,
		)
	}

	if tok == nil {
		return registryLoadIntegrityErrorV1(
			"nil token for registry key %q",
			registryKey,
		)
	}

	tok.mu.Lock()
	defer tok.mu.Unlock()

	trimmedSymbol := strings.TrimSpace(
		tok.symbol,
	)

	if trimmedSymbol == "" ||
		trimmedSymbol != tok.symbol {
		return registryLoadIntegrityErrorV1(
			"invalid token symbol for registry key %q",
			registryKey,
		)
	}

	if registryKey != tok.symbol {
		return registryLoadIntegrityErrorV1(
			"registry key %q does not match token symbol %q",
			registryKey,
			tok.symbol,
		)
	}

	if _, exists := seenSymbols[tok.symbol]; exists {
		return registryLoadIntegrityErrorV1(
			"duplicate token symbol %q",
			tok.symbol,
		)
	}
	seenSymbols[tok.symbol] = struct{}{}

	if !validTokenAddress(tok.owner) {
		return registryLoadIntegrityErrorV1(
			"invalid owner for token %q",
			tok.symbol,
		)
	}

	if tok.maxSupply > 0 &&
		tok.totalSupply > tok.maxSupply {
		return registryLoadIntegrityErrorV1(
			"total supply exceeds max supply for token %q",
			tok.symbol,
		)
	}

	if tok.balances == nil {
		tok.balances =
			make(map[string]uint64)
	}

	if tok.authorizationNonces == nil {
		tok.authorizationNonces =
			make(map[string]uint64)
	}

	if tok.minerTokenBlocks == nil {
		tok.minerTokenBlocks =
			make(map[string]minerTokenBlockRecordV1)
	}

	var balanceTotal uint64

	for address, balance := range tok.balances {
		if !validTokenAddress(address) {
			return registryLoadIntegrityErrorV1(
				"invalid balance address for token %q",
				tok.symbol,
			)
		}

		nextTotal, err := checkedAddUint64(
			balanceTotal,
			balance,
		)
		if err != nil {
			return registryLoadIntegrityErrorV1(
				"balance total overflow for token %q",
				tok.symbol,
			)
		}

		balanceTotal = nextTotal
	}

	if balanceTotal != tok.totalSupply {
		return registryLoadIntegrityErrorV1(
			"balance total %d does not match total supply %d for token %q",
			balanceTotal,
			tok.totalSupply,
			tok.symbol,
		)
	}

	for caller := range tok.authorizationNonces {
		if !validTokenAddress(caller) {
			return registryLoadIntegrityErrorV1(
				"invalid authorization nonce caller for token %q",
				tok.symbol,
			)
		}
	}

	seenMinerTokenHashes := make(map[string]struct{})

	for key, record := range tok.minerTokenBlocks {
		if record.Version != MinerTokenBridgeStateVersionV1 {
			return registryLoadIntegrityErrorV1(
				"invalid miner-token record version for token %q",
				tok.symbol,
			)
		}

		if record.Height <= 0 ||
			key != strconv.Itoa(record.Height) {
			return registryLoadIntegrityErrorV1(
				"invalid miner-token height key %q for token %q",
				key,
				tok.symbol,
			)
		}

		decodedHash, err := hex.DecodeString(record.BlockHash)
		if err != nil ||
			len(decodedHash) != sha256.Size ||
			hex.EncodeToString(decodedHash) != record.BlockHash {
			return registryLoadIntegrityErrorV1(
				"invalid miner-token block hash at height %d for token %q",
				record.Height,
				tok.symbol,
			)
		}

		if _, exists := seenMinerTokenHashes[record.BlockHash]; exists {
			return registryLoadIntegrityErrorV1(
				"duplicate miner-token block hash for token %q",
				tok.symbol,
			)
		}
		seenMinerTokenHashes[record.BlockHash] = struct{}{}

		if !validTokenAddress(record.Miner) || record.Reward == 0 {
			return registryLoadIntegrityErrorV1(
				"invalid miner-token reward record at height %d for token %q",
				record.Height,
				tok.symbol,
			)
		}

		if record.Status != minerTokenBlockPendingV1 &&
			record.Status != minerTokenBlockCommittedV1 {
			return registryLoadIntegrityErrorV1(
				"invalid miner-token status at height %d for token %q",
				record.Height,
				tok.symbol,
			)
		}
	}

	return nil
}

func validateLoadedRegistryV1(
	tokens map[string]*Token,
) (map[string]*Token, error) {
	if tokens == nil {
		return make(map[string]*Token), nil
	}

	validated := make(
		map[string]*Token,
		len(tokens),
	)
	seenSymbols := make(
		map[string]struct{},
		len(tokens),
	)

	for registryKey, tok := range tokens {
		if err := validateLoadedRegistryTokenV1(
			registryKey,
			tok,
			seenSymbols,
		); err != nil {
			return nil, err
		}

		validated[registryKey] = tok
	}

	return validated, nil
}

func (r *Registry) Load(dir string) (resultErr error) {
	outcome := "LOADED"
	tokenCount := 0

	r.mu.Lock()
	defer func() {
		r.mu.Unlock()

		observeTokenRegistryLoadV1(
			dir,
			outcome,
			tokenCount,
			resultErr,
		)
	}()

	path := registryFile
	if dir != "" {
		path = filepath.Join(
			dir,
			registryFile,
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		tokenCount = len(r.Tokens)

		if os.IsNotExist(err) {
			r.ensure()
			tokenCount = len(r.Tokens)
			outcome = "NOT_FOUND"
			return nil
		}

		outcome = "READ_REJECTED"
		return err
	}

	var loaded struct {
		Tokens registryTokenMapV1 `json:"tokens"`
	}

	if err := json.Unmarshal(
		data,
		&loaded,
	); err != nil {
		tokenCount = len(r.Tokens)

		if errors.Is(
			err,
			ErrRegistryLoadIntegrityV1,
		) {
			outcome = "INTEGRITY_REJECTED"
			return err
		}

		if errors.Is(
			err,
			ErrInvalidTokenAddress,
		) {
			outcome = "INTEGRITY_REJECTED"

			return fmt.Errorf(
				"%w: decode persisted token state: %w",
				ErrRegistryLoadIntegrityV1,
				err,
			)
		}

		outcome = "DECODE_REJECTED"
		return err
	}

	validated, err := validateLoadedRegistryV1(
		loaded.Tokens,
	)
	if err != nil {
		tokenCount = len(r.Tokens)
		outcome = "INTEGRITY_REJECTED"
		return err
	}

	r.Tokens = validated
	tokenCount = len(validated)
	outcome = "LOADED"

	return nil
}

type Error string

func (e Error) Error() string { return string(e) }
