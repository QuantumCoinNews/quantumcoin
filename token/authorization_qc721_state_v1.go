package token

import (
	"encoding/json"
	"errors"
	"fmt"
)

type qc721PersistenceV1 struct {
	Name                string
	Symbol              string
	Owner               string
	Owners              map[uint64]string
	TokenURIs           map[uint64]string
	AuthorizationNonces map[string]uint64 `json:",omitempty"`
}

func validateQC721PersistenceAddressesV1(
	persisted qc721PersistenceV1,
) error {
	if persisted.Owner != "" &&
		!validTokenAddress(persisted.Owner) {
		return fmt.Errorf(
			"token: persisted QC721 owner: %w",
			ErrInvalidTokenAddress,
		)
	}

	for _, owner := range persisted.Owners {
		if !validTokenAddress(owner) {
			return fmt.Errorf(
				"token: persisted QC721 token owner: %w",
				ErrInvalidTokenAddress,
			)
		}
	}

	for caller := range persisted.AuthorizationNonces {
		if !validTokenAddress(caller) {
			return fmt.Errorf(
				"token: persisted QC721 authorization caller: %w",
				ErrInvalidTokenAddress,
			)
		}
	}

	return nil
}

func cloneQC721StringMapV1(
	source map[uint64]string,
) map[uint64]string {
	clone := make(
		map[uint64]string,
		len(source),
	)

	for tokenID, value := range source {
		clone[tokenID] = value
	}

	return clone
}

// MarshalJSON preserves historical QC721 JSON field names and persists
// consumed authorization nonces for replay protection.
func (n *QC721Token) MarshalJSON() ([]byte, error) {
	if n == nil {
		return []byte("null"), nil
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	return json.Marshal(qc721PersistenceV1{
		Name:   n.name,
		Symbol: n.symbol,
		Owner:  n.owner,
		Owners: cloneQC721StringMapV1(
			n.owners,
		),
		TokenURIs: cloneQC721StringMapV1(
			n.tokenURIs,
		),
		AuthorizationNonces: cloneTokenUint64MapV1(
			n.authorizationNonces,
		),
	})
}

// UnmarshalJSON accepts previous QC721 JSON without authorization nonces.
func (n *QC721Token) UnmarshalJSON(data []byte) error {
	if n == nil {
		return errors.New(
			"token: unmarshal into nil QC721 token",
		)
	}

	var persisted qc721PersistenceV1

	if err := json.Unmarshal(data, &persisted); err != nil {
		return err
	}

	if persisted.Owners == nil {
		persisted.Owners =
			make(map[uint64]string)
	}

	if persisted.TokenURIs == nil {
		persisted.TokenURIs =
			make(map[uint64]string)
	}

	if persisted.AuthorizationNonces == nil {
		persisted.AuthorizationNonces =
			make(map[string]uint64)
	}

	if err := validateQC721PersistenceAddressesV1(
		persisted,
	); err != nil {
		return err
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.name = persisted.Name
	n.symbol = persisted.Symbol
	n.owner = persisted.Owner
	n.owners = persisted.Owners
	n.tokenURIs = persisted.TokenURIs
	n.authorizationNonces =
		persisted.AuthorizationNonces

	return nil
}

func tokenAuthorizationDomainForQC721LockedV1(
	n *QC721Token,
) string {
	return TokenAuthorizationDomainV1(
		TokenAuthorizationModelQC721V1,
		n.name,
		n.symbol,
		n.owner,
		0,
		0,
	)
}

func (n *QC721Token) AuthorizationDomainV1() (
	string,
	error,
) {
	if n == nil {
		return "", errors.New("QC721 token is nil")
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	return tokenAuthorizationDomainForQC721LockedV1(n), nil
}

func (n *QC721Token) NextAuthorizationNonceV1(
	caller string,
) (uint64, error) {
	if n == nil {
		return 0, errors.New("QC721 token is nil")
	}

	if !validTokenAddress(caller) {
		return 0, ErrInvalidTokenAddress
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.authorizationNonces == nil {
		n.authorizationNonces =
			make(map[string]uint64)
	}

	last := n.authorizationNonces[caller]

	if last == ^uint64(0) {
		return 0, fmt.Errorf(
			"%w: caller nonce is exhausted",
			ErrTokenAuthorizationNonce,
		)
	}

	return last + 1, nil
}
