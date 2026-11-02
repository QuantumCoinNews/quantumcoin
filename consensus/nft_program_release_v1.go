package consensus

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

const NFTMonthlyProgramManifestNameV1 = "nft-monthly-program.json"
const NFTMonthlyProgramReleaseVersionV1 = "QC_NFT_MONTHLY_PROGRAM_RELEASE_V1"

// The detached file carries a signed decision, never authorities or quorum.
type NFTMonthlyProgramDecisionV1 struct {
	NetworkID        string
	LegacyGenesisHex string
	CampaignID       string
	GameActivatedAt  int64
}

type NFTMonthlyProgramReleaseV1 struct {
	Decision  NFTMonthlyProgramDecisionV1
	Approvals []NFTAwardApprovalV1
}

// CanonicalNFTAwardTrustV1 is independent of hardfork governance. Production
// authorities have not been finalized; nil leaves the game program disabled.
// Environment, config, peer payloads and the program file cannot supply keys.
func CanonicalNFTAwardTrustV1() *NFTAwardTrustV1 { return nil }

// VerifiedNFTMonthlyProgramV1 can only be constructed by signature verification.
// Its private fields cannot be manufactured by gob/JSON or mutable public fields.
type VerifiedNFTMonthlyProgramV1 struct {
	verified        bool
	trust           NFTAwardTrustV1
	policy          NFTMonthlyProgramPolicyV1
	gameActivatedAt int64
}

func (p VerifiedNFTMonthlyProgramV1) VerifiedV1() bool { return p.verified }
func (p VerifiedNFTMonthlyProgramV1) PolicySnapshotV1() NFTMonthlyProgramPolicyV1 {
	return p.policy
}
func (p VerifiedNFTMonthlyProgramV1) GameActivatedAtV1() int64 { return p.gameActivatedAt }
func (p VerifiedNFTMonthlyProgramV1) TrustSnapshotV1() NFTAwardTrustV1 {
	result := p.trust
	result.AdminPublicKeys = make([][]byte, len(p.trust.AdminPublicKeys))
	for i, key := range p.trust.AdminPublicKeys {
		result.AdminPublicKeys[i] = append([]byte(nil), key...)
	}
	return result
}

// The separate domain prevents NFT award or hardfork signatures from being
// replayed as a program approval. All fields affecting the twelve periods bind.
func NFTMonthlyProgramDecisionDigestV1(d NFTMonthlyProgramDecisionV1) ([32]byte, error) {
	scope, err := nftAwardScopeV1(d.NetworkID, d.LegacyGenesisHex)
	if err != nil {
		return [32]byte{}, err
	}
	if d.GameActivatedAt <= 0 {
		return [32]byte{}, ErrMonthlyNFTGameActivationRequiredV1
	}
	if _, err := NewNFTMonthlyProgramPolicyFromGameActivationV1(d.NetworkID, d.CampaignID, time.Unix(d.GameActivatedAt, 0)); err != nil {
		return [32]byte{}, err
	}
	payload := append([]byte(NFTMonthlyProgramReleaseVersionV1+"\x00"), scope...)
	payload = binary.BigEndian.AppendUint16(payload, uint16(len(d.CampaignID)))
	payload = append(payload, []byte(d.CampaignID)...)
	payload = binary.BigEndian.AppendUint64(payload, uint64(d.GameActivatedAt))
	return sha256.Sum256(payload), nil
}

func VerifyNFTMonthlyProgramReleaseV1(release NFTMonthlyProgramReleaseV1, trust NFTAwardTrustV1) (VerifiedNFTMonthlyProgramV1, error) {
	var zero VerifiedNFTMonthlyProgramV1
	if err := ValidateNFTAwardTrustV1(trust); err != nil {
		return zero, err
	}
	digest, err := NFTMonthlyProgramDecisionDigestV1(release.Decision)
	if err != nil {
		return zero, err
	}
	want, _ := nftAwardScopeV1(trust.NetworkID, trust.LegacyGenesisHex)
	got, _ := nftAwardScopeV1(release.Decision.NetworkID, release.Decision.LegacyGenesisHex)
	if !bytes.Equal(want, got) {
		return zero, fmt.Errorf("NFT program: decision differs from local network/genesis")
	}
	if len(release.Approvals) > len(trust.AdminPublicKeys) {
		return zero, fmt.Errorf("NFT program: too many approvals")
	}
	approvals := make([]HardforkApprovalV1, len(release.Approvals))
	for i, a := range release.Approvals {
		approvals[i] = HardforkApprovalV1{PublicKey: a.PublicKey, Signature: a.Signature}
	}
	if err := validateHardforkApprovalRoleV1("NFT program admin", trust.AdminPublicKeys, trust.AdminQuorum, approvals, digest); err != nil {
		return zero, fmt.Errorf("NFT program: approvals: %w", err)
	}
	policy, err := NewNFTMonthlyProgramPolicyFromGameActivationV1(release.Decision.NetworkID, release.Decision.CampaignID, time.Unix(release.Decision.GameActivatedAt, 0))
	if err != nil {
		return zero, err
	}
	result := VerifiedNFTMonthlyProgramV1{verified: true, trust: trust, policy: policy, gameActivatedAt: release.Decision.GameActivatedAt}
	result.trust = result.TrustSnapshotV1()
	return result, nil
}

// path is selected by trusted bootstrap next to the detached hardfork release.
// The fixed small file contains public decision/signatures, no private keys.
func LoadNFTMonthlyProgramManifestV1(path string, trust NFTAwardTrustV1) (VerifiedNFTMonthlyProgramV1, error) {
	var zero VerifiedNFTMonthlyProgramV1
	file, err := os.Open(path)
	if err != nil {
		return zero, fmt.Errorf("NFT program manifest: %w", err)
	}
	defer file.Close()
	const maximum = 64 * 1024
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || len(data) > maximum {
		return zero, fmt.Errorf("NFT program manifest: read failed or size exceeded")
	}
	var release NFTMonthlyProgramReleaseV1
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&release); err != nil {
		return zero, fmt.Errorf("NFT program manifest: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return zero, fmt.Errorf("NFT program manifest: trailing content")
	}
	return VerifyNFTMonthlyProgramReleaseV1(release, trust)
}
