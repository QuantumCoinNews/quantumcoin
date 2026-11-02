package token

import (
	"errors"
	"sync"
)

var ErrQC20FixedSupplyMintDisabled = errors.New(
	"token: QC20 fixed supply cannot be minted after creation",
)

// QC20Token documents the QuantumCoin component used by this module.
// QC20Token: bağımsız QC20 token modeli.
// TOKEN_SAFE_BOUNDARY_V1:
// - Arz owner tanımlı kurucuda yalnız bir kez verilir.
// - Mint ve MintByOwner uyumluluk için kalır; sonradan basım yapmaz.
// - Mint reddi mevcut salt-okunur AI/security olay hattına bildirilir.
// - Ana QC total supply ile bağlantılı değildir.
type QC20Token struct {
	name        string
	symbol      string
	owner       string
	maxSupply   uint64
	totalSupply uint64
	decimals    uint8
	balances    map[string]uint64
	allowances  map[string]map[string]uint64

	// authorizationNonces prevents signed QC20 operation replay.
	// Custom JSON persistence keeps consumed nonces across reloads.
	authorizationNonces map[string]uint64
	mu                  sync.RWMutex
}

func NewQC20Token(
	name,
	symbol string,
	supply uint64,
	decimals uint8,
) *QC20Token {
	// Legacy ownerless constructor keeps the requested cap,
	// but cannot issue supply without an owner balance.
	return newQC20TokenV1(
		name,
		symbol,
		supply,
		decimals,
		"",
	)
}

func NewQC20TokenWithOwner(
	name,
	symbol string,
	supply uint64,
	decimals uint8,
	owner string,
) *QC20Token {
	return newQC20TokenV1(
		name,
		symbol,
		supply,
		decimals,
		owner,
	)
}

// NewValidatedQC20TokenV1 is the fail-closed constructor for untrusted owner
// input. It issues the fixed supply only after the owner address is proven
// canonical for the current QuantumCoin token-address network.
func NewValidatedQC20TokenV1(
	name,
	symbol string,
	supply uint64,
	decimals uint8,
	owner string,
) (*QC20Token, error) {
	canonicalOwner, err :=
		requireCanonicalTokenAddressV1(owner)
	if err != nil {
		return nil, err
	}

	return newQC20TokenV1(
		name,
		symbol,
		supply,
		decimals,
		canonicalOwner,
	), nil
}

func newQC20TokenV1(
	name,
	symbol string,
	supply uint64,
	decimals uint8,
	owner string,
) *QC20Token {
	initialSupply := supply

	// Supply cannot exist without a corresponding owner balance.
	// This removes the legacy ownerless phantom-supply state.
	if owner == "" {
		initialSupply = 0
	}

	token := &QC20Token{
		name:                name,
		symbol:              symbol,
		owner:               owner,
		maxSupply:           supply,
		totalSupply:         initialSupply,
		decimals:            decimals,
		balances:            make(map[string]uint64),
		allowances:          make(map[string]map[string]uint64),
		authorizationNonces: make(map[string]uint64),
	}

	if initialSupply > 0 {
		token.balances[owner] = initialSupply
	}

	return token
}

// Mint documents the QuantumCoin component used by this module.
// Mint: Eski uyumluluk wrapper'ı.
// Hatalar çağırana döndürülür ve mevcut salt-okunur
// AI/security olay hattına yalnız bir kez bildirilir.
func (t *QC20Token) Mint(
	to string,
	amount uint64,
) error {
	owner := ""

	if t != nil {
		owner = t.Owner()
	}

	return t.MintByOwner(
		owner,
		to,
		amount,
	)
}

// MintByOwner retains the historical API surface.
//
// QC20 supply is issued once by NewQC20TokenWithOwner.
// Burning permanently reduces circulating and total supply; it does not
// create new mint capacity.
//
// Observation is advisory-only. It cannot authorize minting or mutate
// token state.
func (t *QC20Token) MintByOwner(
	caller string,
	to string,
	amount uint64,
) (resultErr error) {
	const (
		model     = TokenAuthorizationModelQC20V1
		operation = TokenOperationMintV1
	)

	domain := ""

	if t == nil {
		resultErr = errors.New("QC20 token is nil")

		observeTokenAuthorizationV1(
			model,
			operation,
			caller,
			domain,
			0,
			tokenAuthorizationStageOperationV1,
			resultErr,
		)

		return resultErr
	}

	t.mu.RLock()

	domain =
		tokenAuthorizationDomainForQC20LockedV1(t)

	defer func() {
		t.mu.RUnlock()

		observeTokenAuthorizationV1(
			model,
			operation,
			caller,
			domain,
			0,
			tokenAuthorizationStageOperationV1,
			resultErr,
		)
	}()

	if t.owner == "" {
		resultErr = ErrTokenOwnerNotConfigured
		return resultErr
	}

	if !validTokenAddress(t.owner) ||
		!validTokenAddress(caller) {
		resultErr = ErrInvalidTokenAddress
		return resultErr
	}

	if caller != t.owner {
		resultErr = errors.New("not token owner")
		return resultErr
	}

	if !validTokenAddress(to) {
		resultErr = ErrInvalidTokenAddress
		return resultErr
	}

	if amount == 0 {
		resultErr = ErrTokenAmountZero
		return resultErr
	}

	resultErr = ErrQC20FixedSupplyMintDisabled

	return resultErr
}
func (t *QC20Token) Burn(
	from string,
	amount uint64,
) error {
	domain := ""

	if t != nil {
		t.mu.RLock()
		domain =
			tokenAuthorizationDomainForQC20LockedV1(t)
		t.mu.RUnlock()
	}

	err := ErrTokenAuthorizationRequired

	observeTokenAuthorizationV1(
		TokenAuthorizationModelQC20V1,
		TokenOperationBurnV1,
		from,
		domain,
		0,
		tokenAuthorizationStageAuthorizationV1,
		err,
	)

	return err
}
func (t *QC20Token) BalanceOf(addr string) uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.balances[addr]
}

func (t *QC20Token) Transfer(
	from string,
	to string,
	amount uint64,
) error {
	domain := ""

	if t != nil {
		t.mu.RLock()
		domain =
			tokenAuthorizationDomainForQC20LockedV1(t)
		t.mu.RUnlock()
	}

	err := ErrTokenAuthorizationRequired

	observeTokenAuthorizationV1(
		TokenAuthorizationModelQC20V1,
		TokenOperationTransferV1,
		from,
		domain,
		0,
		tokenAuthorizationStageAuthorizationV1,
		err,
	)

	return err
}
func (t *QC20Token) Approve(
	owner string,
	spender string,
	amount uint64,
) error {
	domain := ""

	if t != nil {
		t.mu.RLock()
		domain =
			tokenAuthorizationDomainForQC20LockedV1(t)
		t.mu.RUnlock()
	}

	err := ErrTokenAuthorizationRequired

	observeTokenAuthorizationV1(
		TokenAuthorizationModelQC20V1,
		TokenOperationApproveV1,
		owner,
		domain,
		0,
		tokenAuthorizationStageAuthorizationV1,
		err,
	)

	return err
}
func (t *QC20Token) TransferFrom(
	owner string,
	spender string,
	recipient string,
	amount uint64,
) error {
	domain := ""

	if t != nil {
		t.mu.RLock()
		domain =
			tokenAuthorizationDomainForQC20LockedV1(t)
		t.mu.RUnlock()
	}

	err := ErrTokenAuthorizationRequired

	observeTokenAuthorizationV1(
		TokenAuthorizationModelQC20V1,
		TokenOperationTransferFromV1,
		spender,
		domain,
		0,
		tokenAuthorizationStageAuthorizationV1,
		err,
	)

	return err
}
func (t *QC20Token) Allowance(owner, spender string) uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.allowances[owner][spender]
}
