package consensus

import "fmt"

// ValidateNFTAwardTrustV1 validates the shape of a local authority policy.
// It does not establish who supplied that policy. Node bootstrap must supply
// trusted local configuration, never data from a peer or an award payload.
func ValidateNFTAwardTrustV1(trust NFTAwardTrustV1) error {
	if _, err := nftAwardScopeV1(trust.NetworkID, trust.LegacyGenesisHex); err != nil {
		return fmt.Errorf("NFT local trust: %w", err)
	}
	if _, err := validateHardforkAuthorityRoleV1("NFT admin", trust.AdminPublicKeys, trust.AdminQuorum); err != nil {
		return fmt.Errorf("NFT local trust: %w", err)
	}
	return nil
}
