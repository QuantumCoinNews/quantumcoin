package consensus

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

const NFTAwardApprovalVersionV1 = "QC_NFT_AWARD_APPROVAL_V1"

// RecordDigest must be computed from the actual canonical NFT mint record.
// That commitment includes the recipient, period, asset and metadata.
type NFTAwardDecisionV1 struct {
	NetworkID        string
	LegacyGenesisHex string
	RecordDigest     [32]byte
}

// NFTAwardTrustV1 is supplied by local authoritative configuration. It must
// never be accepted from a peer payload or an award's self-declared signers.
// This primitive does not load a trust policy or enable runtime NFT minting.
type NFTAwardTrustV1 struct {
	NetworkID        string
	LegacyGenesisHex string
	AdminPublicKeys  [][]byte
	AdminQuorum      int
}

type NFTAwardApprovalV1 struct {
	PublicKey []byte
	Signature []byte
}

func nftAwardScopeV1(networkID, genesisHex string) ([]byte, error) {
	if networkID == "" || len(networkID) > 128 || strings.TrimSpace(networkID) != networkID {
		return nil, fmt.Errorf("%s: invalid network ID", NFTAwardApprovalVersionV1)
	}
	for i := 0; i < len(networkID); i++ {
		c := networkID[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == ':' {
			continue
		}
		return nil, fmt.Errorf("%s: invalid network ID", NFTAwardApprovalVersionV1)
	}
	genesis, err := hex.DecodeString(genesisHex)
	if err != nil || len(genesis) != sha256.Size {
		return nil, fmt.Errorf("%s: genesis must be a SHA256 hex value", NFTAwardApprovalVersionV1)
	}
	payload := binary.BigEndian.AppendUint16(nil, uint16(len(networkID)))
	payload = append(payload, []byte(networkID)...)
	payload = append(payload, genesis...)
	return payload, nil
}

// NFTAwardDecisionDigestV1 uses its own approval domain. Genesis is encoded
// as bytes, so uppercase/lowercase hex representations have one commitment.
func NFTAwardDecisionDigestV1(decision NFTAwardDecisionV1) ([32]byte, error) {
	scope, err := nftAwardScopeV1(decision.NetworkID, decision.LegacyGenesisHex)
	if err != nil {
		return [32]byte{}, err
	}
	if decision.RecordDigest == ([32]byte{}) {
		return [32]byte{}, fmt.Errorf("%s: NFT record digest is required", NFTAwardApprovalVersionV1)
	}
	payload := append([]byte(NFTAwardApprovalVersionV1+"\x00"), scope...)
	payload = append(payload, decision.RecordDigest[:]...)
	return sha256.Sum256(payload), nil
}

// ValidateNFTAwardApprovalsV1 validates signatures against independent local
// trust and the digest recomputed from the actual record. Inputs are read-only.
// It neither writes ownership nor accepts a block; integration is separate.
func ValidateNFTAwardApprovalsV1(
	decision NFTAwardDecisionV1,
	trust NFTAwardTrustV1,
	approvals []NFTAwardApprovalV1,
	expectedRecordDigest [32]byte,
) error {
	digest, err := NFTAwardDecisionDigestV1(decision)
	if err != nil {
		return err
	}
	if expectedRecordDigest == ([32]byte{}) || decision.RecordDigest != expectedRecordDigest {
		return fmt.Errorf("%s: approval does not match actual NFT record", NFTAwardApprovalVersionV1)
	}
	trustedScope, err := nftAwardScopeV1(trust.NetworkID, trust.LegacyGenesisHex)
	if err != nil {
		return fmt.Errorf("%s: invalid local trust: %w", NFTAwardApprovalVersionV1, err)
	}
	decisionScope, err := nftAwardScopeV1(decision.NetworkID, decision.LegacyGenesisHex)
	if err != nil {
		return err
	}
	if !bytes.Equal(trustedScope, decisionScope) {
		return fmt.Errorf("%s: approval network/genesis differs from local trust", NFTAwardApprovalVersionV1)
	}
	// Reuse canonical key/quorum and signature validation. The NFT signer set
	// is independent of hardfork governance; its keys are not inherited.
	if _, err := validateHardforkAuthorityRoleV1("NFT admin", trust.AdminPublicKeys, trust.AdminQuorum); err != nil {
		return fmt.Errorf("%s: invalid local NFT authority set: %w", NFTAwardApprovalVersionV1, err)
	}
	if len(approvals) > len(trust.AdminPublicKeys) {
		return fmt.Errorf("%s: too many NFT approvals", NFTAwardApprovalVersionV1)
	}
	canonicalApprovals := make([]HardforkApprovalV1, len(approvals))
	for i, approval := range approvals {
		canonicalApprovals[i] = HardforkApprovalV1{PublicKey: approval.PublicKey, Signature: approval.Signature}
	}
	if err := validateHardforkApprovalRoleV1("NFT admin", trust.AdminPublicKeys, trust.AdminQuorum, canonicalApprovals, digest); err != nil {
		return fmt.Errorf("%s: NFT approval rejected: %w", NFTAwardApprovalVersionV1, err)
	}
	return nil
}
