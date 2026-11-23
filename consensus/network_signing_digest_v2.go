package consensus

import (
	"crypto/sha256"
	"errors"
)

// NetworkSigningDigestV2 binds a transaction to the trusted genesis. Network-v4
// genesis activates this digest; older chains retain SigningDigestV1.
func NetworkSigningDigestV2(genesis, legacyDigest [32]byte) ([32]byte, error) {
	if genesis == ([32]byte{}) {
		return [32]byte{}, errors.New("network signing digest requires a trusted genesis")
	}
	preimage := append([]byte("QC_TX_SIG_NETWORK_V2\x00"), genesis[:]...)
	preimage = append(preimage, legacyDigest[:]...)
	return sha256.Sum256(preimage), nil
}
