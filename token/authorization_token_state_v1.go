package token

import (
	"encoding/json"
	"errors"
	"fmt"
)

type tokenPersistenceV1 struct {
	Symbol              string                             `json:"symbol"`
	Name                string                             `json:"name"`
	Decimals            uint8                              `json:"decimals"`
	MaxSupply           uint64                             `json:"max_supply"`
	TotalSupply         uint64                             `json:"total_supply"`
	Owner               string                             `json:"owner"`
	Balances            map[string]uint64                  `json:"balances"`
	AuthorizationNonces map[string]uint64                  `json:"authorization_nonces,omitempty"`
	MinerTokenBlocks    map[string]minerTokenBlockRecordV1 `json:"miner_token_blocks,omitempty"`
}

func validateTokenPersistenceAddressesV1(
	persisted tokenPersistenceV1,
) error {
	if !validTokenAddress(persisted.Owner) {
		return fmt.Errorf(
			"token: persisted owner: %w",
			ErrInvalidTokenAddress,
		)
	}

	for address := range persisted.Balances {
		if !validTokenAddress(address) {
			return fmt.Errorf(
				"token: persisted balance address: %w",
				ErrInvalidTokenAddress,
			)
		}
	}

	for caller := range persisted.AuthorizationNonces {
		if !validTokenAddress(caller) {
			return fmt.Errorf(
				"token: persisted authorization caller: %w",
				ErrInvalidTokenAddress,
			)
		}
	}

	for _, record := range persisted.MinerTokenBlocks {
		if !validTokenAddress(record.Miner) {
			return fmt.Errorf(
				"token: persisted miner address: %w",
				ErrInvalidTokenAddress,
			)
		}
	}

	return nil
}

func cloneTokenUint64MapV1(
	source map[string]uint64,
) map[string]uint64 {
	clone := make(
		map[string]uint64,
		len(source),
	)

	for key, value := range source {
		clone[key] = value
	}

	return clone
}

func cloneMinerTokenBlockRecordsV1(
	source map[string]minerTokenBlockRecordV1,
) map[string]minerTokenBlockRecordV1 {
	clone := make(
		map[string]minerTokenBlockRecordV1,
		len(source),
	)

	for key, value := range source {
		clone[key] = value
	}

	return clone
}

func tokenPersistenceSnapshotLockedV1(
	t *Token,
) tokenPersistenceV1 {
	return tokenPersistenceV1{
		Symbol:      t.symbol,
		Name:        t.name,
		Decimals:    t.decimals,
		MaxSupply:   t.maxSupply,
		TotalSupply: t.totalSupply,
		Owner:       t.owner,
		Balances: cloneTokenUint64MapV1(
			t.balances,
		),
		AuthorizationNonces: cloneTokenUint64MapV1(
			t.authorizationNonces,
		),
		MinerTokenBlocks: cloneMinerTokenBlockRecordsV1(
			t.minerTokenBlocks,
		),
	}
}

func restoreTokenPersistenceLockedV1(
	t *Token,
	persisted tokenPersistenceV1,
) {
	t.symbol = persisted.Symbol
	t.name = persisted.Name
	t.decimals = persisted.Decimals
	t.maxSupply = persisted.MaxSupply
	t.totalSupply = persisted.TotalSupply
	t.owner = persisted.Owner
	t.balances = cloneTokenUint64MapV1(
		persisted.Balances,
	)
	t.authorizationNonces = cloneTokenUint64MapV1(
		persisted.AuthorizationNonces,
	)
	t.minerTokenBlocks = cloneMinerTokenBlockRecordsV1(
		persisted.MinerTokenBlocks,
	)
}

// MarshalJSON persists consumed authorization nonces. Restarting the
// process therefore cannot make a consumed signed operation valid again.
func (t *Token) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte("null"), nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	return json.Marshal(
		tokenPersistenceSnapshotLockedV1(t),
	)
}

// UnmarshalJSON accepts both the previous token_registry.json format
// and the authorization-nonce field introduced by authorization nonce migration.
func (t *Token) UnmarshalJSON(data []byte) error {
	if t == nil {
		return errors.New(
			"token: unmarshal into nil token",
		)
	}

	var persisted tokenPersistenceV1

	if err := json.Unmarshal(data, &persisted); err != nil {
		return err
	}

	if persisted.Balances == nil {
		persisted.Balances =
			make(map[string]uint64)
	}

	if persisted.AuthorizationNonces == nil {
		persisted.AuthorizationNonces =
			make(map[string]uint64)
	}

	if persisted.MinerTokenBlocks == nil {
		persisted.MinerTokenBlocks =
			make(map[string]minerTokenBlockRecordV1)
	}

	if err := validateTokenPersistenceAddressesV1(
		persisted,
	); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.symbol = persisted.Symbol
	t.name = persisted.Name
	t.decimals = persisted.Decimals
	t.maxSupply = persisted.MaxSupply
	t.totalSupply = persisted.TotalSupply
	t.owner = persisted.Owner
	t.balances = persisted.Balances
	t.authorizationNonces =
		persisted.AuthorizationNonces
	t.minerTokenBlocks = persisted.MinerTokenBlocks

	return nil
}

func tokenAuthorizationDomainForTokenLockedV1(
	t *Token,
) string {
	return TokenAuthorizationDomainV1(
		TokenAuthorizationModelLegacyV1,
		t.name,
		t.symbol,
		t.owner,
		t.decimals,
		t.maxSupply,
	)
}

// AuthorizationDomainV1 returns the deterministic domain that wallets
// must bind into signatures for this token instance.
func (t *Token) AuthorizationDomainV1() (
	string,
	error,
) {
	if t == nil {
		return "", errors.New("token is nil")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	return tokenAuthorizationDomainForTokenLockedV1(t), nil
}

// NextAuthorizationNonceV1 returns the next sequential caller nonce.
// Reading it does not reserve or consume it.
func (t *Token) NextAuthorizationNonceV1(
	caller string,
) (uint64, error) {
	if t == nil {
		return 0, errors.New("token is nil")
	}

	if !validTokenAddress(caller) {
		return 0, ErrInvalidTokenAddress
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.authorizationNonces == nil {
		t.authorizationNonces =
			make(map[string]uint64)
	}

	last := t.authorizationNonces[caller]

	if last == ^uint64(0) {
		return 0, fmt.Errorf(
			"%w: caller nonce is exhausted",
			ErrTokenAuthorizationNonce,
		)
	}

	return last + 1, nil
}
