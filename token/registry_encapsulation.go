package token

import "errors"

// ErrTokenStateAlreadyInitialized prevents JSON from bypassing authorized
// operations on a live object. Persistence must decode into a new zero value.
var ErrTokenStateAlreadyInitialized = errors.New("token: state already initialized; decode persistence into a new object")

// MarshalJSON preserves the historical registry format under the registry lock.
func (r *Registry) MarshalJSON() ([]byte, error) {
	if r == nil {
		return []byte("null"), nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return marshalRegistryWithLockedTokenV1(r, nil)
}

// UnmarshalJSON refuses direct registry replacement. Load is the explicit
// persistence boundary and validates every entry before replacing the registry.
func (r *Registry) UnmarshalJSON([]byte) error {
	return errors.New("token registry: use Load for validated persistence")
}
