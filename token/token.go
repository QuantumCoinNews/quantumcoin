package token

import (
	"errors"
	"sync"
)

// Token documents the QuantumCoin component used by this module.
// Token: basit QC-20 çekirdeği.
// TOKEN_SAFE_BOUNDARY_V1:
// - Owner dışı mint yasak.
// - MaxSupply aşımı yasak.
// - Empty address ve overflow kontrolleri aktif.
type Token struct {
	symbol      string
	name        string
	decimals    uint8
	maxSupply   uint64
	totalSupply uint64
	owner       string
	balances    map[string]uint64

	minerTokenBlocks map[string]minerTokenBlockRecordV1

	// authorizationNonces is private runtime state. Custom JSON
	// persistence will store it without exposing mutable map access.
	authorizationNonces map[string]uint64
	mu                  sync.Mutex `json:"-"`
}

func New(symbol, name string, decimals uint8, maxSupply uint64, owner string) *Token {
	return &Token{
		symbol:              symbol,
		name:                name,
		decimals:            decimals,
		maxSupply:           maxSupply,
		owner:               owner,
		balances:            make(map[string]uint64),
		minerTokenBlocks:    make(map[string]minerTokenBlockRecordV1),
		authorizationNonces: make(map[string]uint64),
	}
}

// NewValidatedTokenV1 is the fail-closed constructor for untrusted owner
// input. Legacy New remains available for historical callers, but mutation
// methods reject a non-canonical owner before changing state.
func NewValidatedTokenV1(
	symbol,
	name string,
	decimals uint8,
	maxSupply uint64,
	owner string,
) (*Token, error) {
	canonicalOwner, err :=
		requireCanonicalTokenAddressV1(owner)
	if err != nil {
		return nil, err
	}

	return New(
		symbol,
		name,
		decimals,
		maxSupply,
		canonicalOwner,
	), nil
}

func (t *Token) Mint(caller, to string, amount uint64) error {
	if t == nil {
		return errors.New("token is nil")
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.owner == "" {
		return ErrTokenOwnerNotConfigured
	}
	if !validTokenAddress(t.owner) ||
		!validTokenAddress(caller) {
		return ErrInvalidTokenAddress
	}
	if caller != t.owner {
		return errors.New("not token owner")
	}
	if err := validateMintCommon(to, amount, t.totalSupply, t.maxSupply); err != nil {
		return err
	}

	newBalance, err := checkedAddUint64(t.balances[to], amount)
	if err != nil {
		return ErrTokenBalanceOverflow
	}
	newSupply, err := checkedAddUint64(t.totalSupply, amount)
	if err != nil {
		return err
	}

	t.balances[to] = newBalance
	t.totalSupply = newSupply
	return nil
}

func (t *Token) Transfer(
	from string,
	to string,
	amount uint64,
) error {
	domain := ""

	if t != nil {
		t.mu.Lock()
		domain =
			tokenAuthorizationDomainForTokenLockedV1(t)
		t.mu.Unlock()
	}

	err := ErrTokenAuthorizationRequired

	observeTokenAuthorizationV1(
		TokenAuthorizationModelLegacyV1,
		TokenOperationTransferV1,
		from,
		domain,
		0,
		tokenAuthorizationStageAuthorizationV1,
		err,
	)

	return err
}
func (t *Token) BalanceOf(addr string) uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.balances[addr]
}

func (t *Token) GetTotalSupply() uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.totalSupply
}
