package token

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var ErrTokenNetworkIdentityRequired = errors.New("token: verified network identity required for authorization")

// TokenAuthorizationIdentityV1 binds signed operations to one network and
// one persisted token instance. The caller must derive network fields from
// the verified chain, never from an untrusted request.
type TokenAuthorizationIdentityV1 struct {
	NetworkMagic uint32 `json:"network_magic"`
	GenesisHash  string `json:"genesis_hash"`
	InstanceID   string `json:"instance_id"`
}

// NewTokenAuthorizationIdentityV1 creates a persistent instance identifier
// for a token attached to the supplied verified chain identity.
func NewTokenAuthorizationIdentityV1(networkMagic uint32, genesisHash string) (TokenAuthorizationIdentityV1, error) {
	identity := TokenAuthorizationIdentityV1{
		NetworkMagic: networkMagic,
		GenesisHash:  strings.ToLower(strings.TrimSpace(genesisHash)),
	}
	instance := make([]byte, 32)
	if _, err := rand.Read(instance); err != nil {
		return TokenAuthorizationIdentityV1{}, fmt.Errorf("token: generate instance identity: %w", err)
	}
	identity.InstanceID = hex.EncodeToString(instance)
	if err := validateTokenAuthorizationIdentityV1(identity); err != nil {
		return TokenAuthorizationIdentityV1{}, err
	}
	return identity, nil
}

func validateTokenAuthorizationIdentityV1(identity TokenAuthorizationIdentityV1) error {
	genesis, genesisErr := hex.DecodeString(identity.GenesisHash)
	instance, instanceErr := hex.DecodeString(identity.InstanceID)
	if identity.NetworkMagic == 0 || genesisErr != nil || len(genesis) != 32 || instanceErr != nil || len(instance) != 32 {
		return ErrTokenNetworkIdentityRequired
	}
	var genesisNonZero, instanceNonZero byte
	for _, value := range genesis {
		genesisNonZero |= value
	}
	for _, value := range instance {
		instanceNonZero |= value
	}
	if genesisNonZero == 0 || instanceNonZero == 0 {
		return ErrTokenNetworkIdentityRequired
	}
	return nil
}

func validatePersistedTokenAuthorizationIdentityV1(identity TokenAuthorizationIdentityV1) error {
	if identity.NetworkMagic == 0 && strings.TrimSpace(identity.GenesisHash) == "" {
		if identity.InstanceID == "" {
			return nil // historical token state; readable, never signable
		}
		instance, err := hex.DecodeString(identity.InstanceID)
		if err != nil || len(instance) != 32 {
			return ErrTokenNetworkIdentityRequired
		}
		var nonZero byte
		for _, value := range instance {
			nonZero |= value
		}
		if nonZero == 0 {
			return ErrTokenNetworkIdentityRequired
		}
		return nil
	}
	return validateTokenAuthorizationIdentityV1(identity)
}
