package token

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
)

var ErrTokenMintDisabled = errors.New("token: mint is disabled after initial issuance")

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
	authorizationNonces   map[string]uint64
	authorizationIdentity TokenAuthorizationIdentityV1
	mu                    sync.Mutex `json:"-"`
}

func New(symbol, name string, decimals uint8, maxSupply uint64, owner string) *Token {
	instance := make([]byte, 32)
	_, _ = rand.Read(instance)
	return &Token{
		symbol:                symbol,
		name:                  name,
		decimals:              decimals,
		maxSupply:             maxSupply,
		owner:                 owner,
		balances:              make(map[string]uint64),
		minerTokenBlocks:      make(map[string]minerTokenBlockRecordV1),
		authorizationNonces:   make(map[string]uint64),
		authorizationIdentity: TokenAuthorizationIdentityV1{InstanceID: hex.EncodeToString(instance)},
	}
}

// NewValidatedTokenForNetworkV1 binds future signed operations to a verified
// network identity and a new persistent token instance ID.
func NewValidatedTokenForNetworkV1(symbol, name string, decimals uint8, maxSupply uint64, owner string, networkMagic uint32, genesisHash string) (*Token, error) {
	canonicalOwner, err := requireCanonicalTokenAddressV1(owner)
	if err != nil {
		return nil, err
	}
	identity, err := NewTokenAuthorizationIdentityV1(networkMagic, genesisHash)
	if err != nil {
		return nil, err
	}
	token := New(symbol, name, decimals, maxSupply, canonicalOwner)
	token.authorizationIdentity = identity
	return token, nil
}

// NewValidatedTokenWithInitialSupplyForNetworkV1 assigns the initial supply
// atomically at creation. Further public mint calls are disabled.
func NewValidatedTokenWithInitialSupplyForNetworkV1(symbol, name string, decimals uint8, maxSupply, initialSupply uint64, owner string, networkMagic uint32, genesisHash string) (*Token, error) {
	return NewValidatedTokenWithInitialAllocationsForNetworkV1(symbol, name, decimals, maxSupply, owner, networkMagic, genesisHash, map[string]uint64{owner: initialSupply})
}

// NewValidatedTokenWithInitialAllocationsForNetworkV1 assigns all initial
// balances atomically and rejects allocations that exceed the supply cap.
func NewValidatedTokenWithInitialAllocationsForNetworkV1(symbol, name string, decimals uint8, maxSupply uint64, owner string, networkMagic uint32, genesisHash string, allocations map[string]uint64) (*Token, error) {
	canonicalOwner, err := requireCanonicalTokenAddressV1(owner)
	if err != nil {
		return nil, err
	}
	total := uint64(0)
	for address, amount := range allocations {
		if !validTokenAddress(address) {
			return nil, ErrInvalidTokenAddress
		}
		var addErr error
		total, addErr = checkedAddUint64(total, amount)
		if addErr != nil {
			return nil, ErrTokenBalanceOverflow
		}
	}
	if maxSupply != 0 && total > maxSupply {
		return nil, ErrTokenMaxSupplyExceeded
	}
	token, err := NewValidatedTokenForNetworkV1(symbol, name, decimals, maxSupply, canonicalOwner, networkMagic, genesisHash)
	if err != nil {
		return nil, err
	}
	for address, amount := range allocations {
		if amount > 0 {
			token.balances[address] = amount
		}
	}
	token.totalSupply = total
	return token, nil
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
	_ = to
	_ = amount
	return ErrTokenMintDisabled
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
