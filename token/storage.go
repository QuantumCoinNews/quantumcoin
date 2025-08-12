package token

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const registryFile = "token_registry.json"

type Registry struct {
	Tokens map[string]*Token `json:"tokens"` // key: symbol
	mu     sync.Mutex        `json:"-"`
}

func NewRegistry() *Registry {
	return &Registry{Tokens: make(map[string]*Token)}
}

func (r *Registry) Register(t *Token) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t == nil || t.Symbol == "" {
		return ErrInvalidToken
	}
	if _, ok := r.Tokens[t.Symbol]; ok {
		return ErrExists
	}
	r.Tokens[t.Symbol] = t
	return nil
}

func (r *Registry) Get(symbol string) (*Token, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.Tokens[symbol]
	return t, ok
}

func (r *Registry) Save(dir string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	path := registryFile
	if dir != "" {
		path = filepath.Join(dir, registryFile)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (r *Registry) Load(dir string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	path := registryFile
	if dir != "" {
		path = filepath.Join(dir, registryFile)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		// yoksa sorun değil
		return nil
	}
	type regAlias Registry
	var tmp regAlias
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	// pointers rebuild
	r.Tokens = make(map[string]*Token, len(tmp.Tokens))
	for sym, tok := range tmp.Tokens {
		if tok.Balances == nil {
			tok.Balances = make(map[string]uint64)
		}
		r.Tokens[sym] = tok
	}
	return nil
}

// errors
var (
	ErrInvalidToken = Error("invalid token")
	ErrExists       = Error("token exists")
)

type Error string

func (e Error) Error() string { return string(e) }
