package token

import (
	"errors"
	"sync"
)

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
	authorizationNonces map[string]uint64
	mu                  sync.RWMutex
}

func NewQC721Token(name, symbol string) *QC721Token {
	return NewQC721TokenWithOwner(name, symbol, "")
}

func NewQC721TokenWithOwner(name, symbol, owner string) *QC721Token {
	return &QC721Token{
		name:                name,
		symbol:              symbol,
		owner:               owner,
		owners:              make(map[uint64]string),
		tokenURIs:           make(map[uint64]string),
		authorizationNonces: make(map[string]uint64),
	}
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
	owner := n.Owner()
	return n.MintByOwner(owner, to, tokenID, tokenURI)
}

func (n *QC721Token) MintByOwner(caller, to string, tokenID uint64, tokenURI string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.owner == "" {
		return ErrTokenOwnerNotConfigured
	}
	if !validTokenAddress(n.owner) ||
		!validTokenAddress(caller) {
		return ErrInvalidTokenAddress
	}
	if caller != n.owner {
		return errors.New("not collection owner")
	}
	if !validTokenAddress(to) {
		return ErrInvalidTokenAddress
	}
	if tokenURI == "" {
		return errors.New("tokenURI boş olamaz")
	}
	if _, exists := n.owners[tokenID]; exists {
		return errors.New("token zaten var")
	}

	n.owners[tokenID] = to
	n.tokenURIs[tokenID] = tokenURI
	return nil
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
