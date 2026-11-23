package token

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
)

var ErrQC721MintDisabled = errors.New("token: QC721 mint is closed after collection creation")

type QC721InitialItemV1 struct {
	TokenID  uint64
	Owner    string
	TokenURI string
}

// QC721Token documents the QuantumCoin component used by this module.
// QC721Token: bağımsız NFT koleksiyonu.
// TOKEN_SAFE_BOUNDARY_V1:
// - Mint için owner/policy sınırı eklendi.
// - Eski Mint(to, tokenID, uri) uyumluluk için kalır ama owner yoksa çalışmaz.
type QC721Token struct {
	name      string
	symbol    string
	owner     string
	owners    map[uint64]string
	tokenURIs map[uint64]string

	// authorizationNonces prevents signed QC721 operation replay.
	// Custom JSON persistence keeps consumed nonces across reloads.
	authorizationNonces   map[string]uint64
	authorizationIdentity TokenAuthorizationIdentityV1
	mu                    sync.RWMutex
}

func NewQC721Token(name, symbol string) *QC721Token {
	return NewQC721TokenWithOwner(name, symbol, "")
}

func NewQC721TokenWithOwner(name, symbol, owner string) *QC721Token {
	instance := make([]byte, 32)
	_, _ = rand.Read(instance)
	return &QC721Token{
		name:                  name,
		symbol:                symbol,
		owner:                 owner,
		owners:                make(map[uint64]string),
		tokenURIs:             make(map[uint64]string),
		authorizationNonces:   make(map[string]uint64),
		authorizationIdentity: TokenAuthorizationIdentityV1{InstanceID: hex.EncodeToString(instance)},
	}
}

// NewValidatedQC721TokenForNetworkV1 scopes collection signatures to one
// verified network and persists a unique collection instance identifier.
func NewValidatedQC721TokenForNetworkV1(name, symbol, owner string, networkMagic uint32, genesisHash string) (*QC721Token, error) {
	canonicalOwner, err := requireCanonicalTokenAddressV1(owner)
	if err != nil {
		return nil, err
	}
	identity, err := NewTokenAuthorizationIdentityV1(networkMagic, genesisHash)
	if err != nil {
		return nil, err
	}
	collection := NewQC721TokenWithOwner(name, symbol, canonicalOwner)
	collection.authorizationIdentity = identity
	return collection, nil
}

// NewValidatedQC721TokenWithInitialItemsForNetworkV1 creates a collection
// with all initial items in one validated operation; later mint APIs stay shut.
func NewValidatedQC721TokenWithInitialItemsForNetworkV1(name, symbol, owner string, networkMagic uint32, genesisHash string, items []QC721InitialItemV1) (*QC721Token, error) {
	collection, err := NewValidatedQC721TokenForNetworkV1(name, symbol, owner, networkMagic, genesisHash)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if !validTokenAddress(item.Owner) {
			return nil, ErrInvalidTokenAddress
		}
		if strings.TrimSpace(item.TokenURI) == "" {
			return nil, errors.New("tokenURI boş olamaz")
		}
		if _, exists := collection.owners[item.TokenID]; exists {
			return nil, errors.New("duplicate initial QC721 token ID")
		}
		collection.owners[item.TokenID] = item.Owner
		collection.tokenURIs[item.TokenID] = item.TokenURI
	}
	return collection, nil
}

// NewValidatedQC721TokenV1 is the fail-closed constructor for untrusted
// collection-owner input.
func NewValidatedQC721TokenV1(
	name,
	symbol,
	owner string,
) (*QC721Token, error) {
	canonicalOwner, err :=
		requireCanonicalTokenAddressV1(owner)
	if err != nil {
		return nil, err
	}

	return NewQC721TokenWithOwner(
		name,
		symbol,
		canonicalOwner,
	), nil
}

func (n *QC721Token) Mint(to string, tokenID uint64, tokenURI string) error {
	if n == nil {
		return ErrTokenOwnerNotConfigured
	}
	if !validTokenAddress(to) {
		return ErrInvalidTokenAddress
	}
	if strings.TrimSpace(tokenURI) == "" {
		return errors.New("tokenURI boş olamaz")
	}
	_ = tokenID
	return ErrQC721MintDisabled
}

func (n *QC721Token) MintByOwner(caller, to string, tokenID uint64, tokenURI string) error {
	if n == nil {
		return ErrTokenOwnerNotConfigured
	}
	if !validTokenAddress(n.owner) || !validTokenAddress(caller) || !validTokenAddress(to) {
		return ErrInvalidTokenAddress
	}
	if strings.TrimSpace(tokenURI) == "" {
		return errors.New("tokenURI boş olamaz")
	}
	_ = caller
	_ = tokenID
	return ErrQC721MintDisabled
}

func (n *QC721Token) OwnerOf(tokenID uint64) (string, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	owner, ok := n.owners[tokenID]
	if !ok {
		return "", errors.New("token bulunamadı")
	}
	return owner, nil
}

func (n *QC721Token) Transfer(
	from string,
	to string,
	tokenID uint64,
) error {
	domain := ""

	if n != nil {
		n.mu.RLock()
		domain =
			tokenAuthorizationDomainForQC721LockedV1(n)
		n.mu.RUnlock()
	}

	err := ErrTokenAuthorizationRequired

	observeTokenAuthorizationV1(
		TokenAuthorizationModelQC721V1,
		TokenOperationTransferV1,
		from,
		domain,
		0,
		tokenAuthorizationStageAuthorizationV1,
		err,
	)

	return err
}
func (n *QC721Token) TokenURI(tokenID uint64) (string, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	uri, ok := n.tokenURIs[tokenID]
	if !ok {
		return "", errors.New("token bulunamadı")
	}
	return uri, nil
}

func (n *QC721Token) Burn(
	tokenID uint64,
	owner string,
) error {
	domain := ""

	if n != nil {
		n.mu.RLock()
		domain =
			tokenAuthorizationDomainForQC721LockedV1(n)
		n.mu.RUnlock()
	}

	err := ErrTokenAuthorizationRequired

	observeTokenAuthorizationV1(
		TokenAuthorizationModelQC721V1,
		TokenOperationBurnV1,
		owner,
		domain,
		0,
		tokenAuthorizationStageAuthorizationV1,
		err,
	)

	return err
}
