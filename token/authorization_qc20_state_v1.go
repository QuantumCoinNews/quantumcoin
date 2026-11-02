package token

import (
	"encoding/json"
	"errors"
	"fmt"
)

type qc20PersistenceV1 struct {
	Name                string
	Symbol              string
	Owner               string
	MaxSupply           uint64
	TotalSupply         uint64
	Decimals            uint8
	Balances            map[string]uint64
	Allowances          map[string]map[string]uint64
	AuthorizationNonces map[string]uint64 `json:",omitempty"`
}

func cloneQC20AllowancesV1(
	source map[string]map[string]uint64,
) map[string]map[string]uint64 {
	clone := make(
		map[string]map[string]uint64,
		len(source),
	)

	for owner, allowances := range source {
		ownerClone := make(
			map[string]uint64,
			len(allowances),
		)

		for spender, amount := range allowances {
			ownerClone[spender] = amount
		}

		clone[owner] = ownerClone
	}

	return clone
}

// MarshalJSON preserves the historical exported QC20 JSON field names
// and adds consumed authorization nonces for replay protection.
func (t *QC20Token) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte("null"), nil
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	return json.Marshal(qc20PersistenceV1{
		Name:        t.name,
		Symbol:      t.symbol,
		Owner:       t.owner,
		MaxSupply:   t.maxSupply,
		TotalSupply: t.totalSupply,
		Decimals:    t.decimals,
		Balances: cloneTokenUint64MapV1(
			t.balances,
		),
		Allowances: cloneQC20AllowancesV1(
			t.allowances,
		),
		AuthorizationNonces: cloneTokenUint64MapV1(
			t.authorizationNonces,
		),
	})
}

// UnmarshalJSON accepts previous QC20 JSON without AuthorizationNonces.
func (t *QC20Token) UnmarshalJSON(data []byte) error {
	if t == nil {
		return errors.New(
			"token: unmarshal into nil QC20 token",
		)
	}

	var persisted qc20PersistenceV1

	if err := json.Unmarshal(data, &persisted); err != nil {
		return err
	}
	if err := normalizeAndValidateQC20PersistenceV1(
		&persisted,
	); err != nil {
		return err
	}

	if persisted.Balances == nil {
		persisted.Balances =
			make(map[string]uint64)
	}

	if persisted.Allowances == nil {
		persisted.Allowances =
			make(map[string]map[string]uint64)
	}

	if persisted.AuthorizationNonces == nil {
		persisted.AuthorizationNonces =
			make(map[string]uint64)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.name = persisted.Name
	t.symbol = persisted.Symbol
	t.owner = persisted.Owner
	t.maxSupply = persisted.MaxSupply
	t.totalSupply = persisted.TotalSupply
	t.decimals = persisted.Decimals
	t.balances = persisted.Balances
	t.allowances = persisted.Allowances
	t.authorizationNonces =
		persisted.AuthorizationNonces

	return nil
}

func tokenAuthorizationDomainForQC20LockedV1(
	t *QC20Token,
) string {
	return TokenAuthorizationDomainV1(
		TokenAuthorizationModelQC20V1,
		t.name,
		t.symbol,
		t.owner,
		t.decimals,
		t.maxSupply,
	)
}

func (t *QC20Token) AuthorizationDomainV1() (
	string,
	error,
) {
	if t == nil {
		return "", errors.New("QC20 token is nil")
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	return tokenAuthorizationDomainForQC20LockedV1(t), nil
}

func (t *QC20Token) NextAuthorizationNonceV1(
	caller string,
) (uint64, error) {
	if t == nil {
		return 0, errors.New("QC20 token is nil")
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
