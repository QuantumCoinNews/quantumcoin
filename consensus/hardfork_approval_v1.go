package consensus

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

const HardforkApprovalVersionV1 = "QC_HARDFORK_APPROVAL_V1"

type HardforkDecisionV1 struct {
	NetworkID           string
	LegacyGenesisHex    string
	ActivationHeight    int
	TargetProtocol      ProtocolVersion
	Reasons             []string
	SnapshotSHA256      string
	ReleaseBinarySHA256 string
}

type HardforkApprovalV1 struct {
	PublicKey []byte
	Signature []byte
}

type HardforkAuthoritySetV1 struct {
	FounderPublicKeys [][]byte
	AdminPublicKeys   [][]byte
	FounderQuorum     int
	AdminQuorum       int
}

type HardforkApprovalSetV1 struct {
	Founder []HardforkApprovalV1
	Admin   []HardforkApprovalV1
}

func CanonicalHardforkDecisionPayloadV1(
	decision HardforkDecisionV1,
) ([]byte, error) {
	if err := validateHardforkDecisionV1(decision); err != nil {
		return nil, err
	}

	reasons := append([]string(nil), decision.Reasons...)
	sort.Strings(reasons)

	var buffer bytes.Buffer

	writeHardforkStringV1(&buffer, HardforkApprovalVersionV1)
	writeHardforkStringV1(&buffer, strings.TrimSpace(decision.NetworkID))
	writeHardforkStringV1(
		&buffer,
		strings.ToLower(strings.TrimSpace(decision.LegacyGenesisHex)),
	)

	if err := binary.Write(
		&buffer,
		binary.BigEndian,
		uint64(decision.ActivationHeight),
	); err != nil {
		return nil, fmt.Errorf(
			"%s: encode activation height: %w",
			HardforkApprovalVersionV1,
			err,
		)
	}

	writeHardforkStringV1(
		&buffer,
		string(decision.TargetProtocol),
	)

	if err := binary.Write(
		&buffer,
		binary.BigEndian,
		uint32(len(reasons)),
	); err != nil {
		return nil, fmt.Errorf(
			"%s: encode reason count: %w",
			HardforkApprovalVersionV1,
			err,
		)
	}

	for _, reason := range reasons {
		writeHardforkStringV1(&buffer, reason)
	}

	writeHardforkStringV1(
		&buffer,
		strings.ToLower(strings.TrimSpace(decision.SnapshotSHA256)),
	)

	writeHardforkStringV1(
		&buffer,
		strings.ToLower(strings.TrimSpace(decision.ReleaseBinarySHA256)),
	)

	return buffer.Bytes(), nil
}

func ValidateHardforkApprovalsV1(
	decision HardforkDecisionV1,
	authorities HardforkAuthoritySetV1,
	approvals HardforkApprovalSetV1,
) error {
	payload, err := CanonicalHardforkDecisionPayloadV1(decision)
	if err != nil {
		return err
	}

	if err := validateHardforkAuthoritySetV1(authorities); err != nil {
		return err
	}

	digest := sha256.Sum256(payload)

	if err := validateHardforkApprovalRoleV1(
		"founder",
		authorities.FounderPublicKeys,
		authorities.FounderQuorum,
		approvals.Founder,
		digest,
	); err != nil {
		return err
	}

	if err := validateHardforkApprovalRoleV1(
		"admin",
		authorities.AdminPublicKeys,
		authorities.AdminQuorum,
		approvals.Admin,
		digest,
	); err != nil {
		return err
	}

	return nil
}

func validateHardforkDecisionV1(
	decision HardforkDecisionV1,
) error {
	if strings.TrimSpace(decision.NetworkID) == "" {
		return fmt.Errorf(
			"%s: network id is required",
			HardforkApprovalVersionV1,
		)
	}

	if err := validateHardforkSHA256HexV1(
		"legacy genesis",
		decision.LegacyGenesisHex,
	); err != nil {
		return err
	}

	if decision.ActivationHeight <= 0 {
		return fmt.Errorf(
			"%s: activation height must be greater than zero",
			HardforkApprovalVersionV1,
		)
	}

	switch decision.TargetProtocol {
	case ProtocolContinuityV2, ProtocolAtomV3, ProtocolNFTV4:
	default:
		return fmt.Errorf(
			"%s: unsupported target protocol %q",
			HardforkApprovalVersionV1,
			decision.TargetProtocol,
		)
	}

	if len(decision.Reasons) == 0 {
		return fmt.Errorf(
			"%s: at least one hardfork reason is required",
			HardforkApprovalVersionV1,
		)
	}

	seenReasons := make(map[string]struct{}, len(decision.Reasons))

	for _, reason := range decision.Reasons {
		if reason == "" || reason != strings.TrimSpace(reason) {
			return fmt.Errorf(
				"%s: invalid hardfork reason %q",
				HardforkApprovalVersionV1,
				reason,
			)
		}

		if _, exists := seenReasons[reason]; exists {
			return fmt.Errorf(
				"%s: duplicate hardfork reason %q",
				HardforkApprovalVersionV1,
				reason,
			)
		}

		seenReasons[reason] = struct{}{}
	}

	if err := validateHardforkSHA256HexV1(
		"snapshot",
		decision.SnapshotSHA256,
	); err != nil {
		return err
	}

	return validateHardforkSHA256HexV1(
		"release binary",
		decision.ReleaseBinarySHA256,
	)
}

func validateHardforkAuthoritySetV1(
	authorities HardforkAuthoritySetV1,
) error {
	founders, err := validateHardforkAuthorityRoleV1(
		"founder",
		authorities.FounderPublicKeys,
		authorities.FounderQuorum,
	)
	if err != nil {
		return err
	}

	admins, err := validateHardforkAuthorityRoleV1(
		"admin",
		authorities.AdminPublicKeys,
		authorities.AdminQuorum,
	)
	if err != nil {
		return err
	}

	for signer := range founders {
		if _, exists := admins[signer]; exists {
			return fmt.Errorf(
				"%s: signer cannot hold both founder and admin authority",
				HardforkApprovalVersionV1,
			)
		}
	}

	return nil
}

func validateHardforkAuthorityRoleV1(
	role string,
	publicKeys [][]byte,
	quorum int,
) (map[string]struct{}, error) {
	if len(publicKeys) == 0 {
		return nil, fmt.Errorf(
			"%s: %s authority set is empty",
			HardforkApprovalVersionV1,
			role,
		)
	}

	if quorum <= 0 || quorum > len(publicKeys) {
		return nil, fmt.Errorf(
			"%s: invalid %s quorum %d for %d authorities",
			HardforkApprovalVersionV1,
			role,
			quorum,
			len(publicKeys),
		)
	}

	seen := make(map[string]struct{}, len(publicKeys))

	for _, publicKey := range publicKeys {
		if _, err := ParseUncompressedPublicKeyV1(publicKey); err != nil {
			return nil, fmt.Errorf(
				"%s: invalid %s authority public key: %w",
				HardforkApprovalVersionV1,
				role,
				err,
			)
		}

		key := string(publicKey)

		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf(
				"%s: duplicate %s authority",
				HardforkApprovalVersionV1,
				role,
			)
		}

		seen[key] = struct{}{}
	}

	return seen, nil
}

func validateHardforkApprovalRoleV1(
	role string,
	authorizedKeys [][]byte,
	quorum int,
	approvals []HardforkApprovalV1,
	digest [32]byte,
) error {
	authorized := make(map[string]struct{}, len(authorizedKeys))

	for _, publicKey := range authorizedKeys {
		authorized[string(publicKey)] = struct{}{}
	}

	seen := make(map[string]struct{}, len(approvals))
	validCount := 0

	for _, approval := range approvals {
		key := string(approval.PublicKey)

		if _, allowed := authorized[key]; !allowed {
			return fmt.Errorf(
				"%s: unauthorized %s signer",
				HardforkApprovalVersionV1,
				role,
			)
		}

		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf(
				"%s: duplicate %s signer",
				HardforkApprovalVersionV1,
				role,
			)
		}

		seen[key] = struct{}{}

		if err := VerifyTransactionDigestSignatureV1(
			approval.PublicKey,
			digest,
			approval.Signature,
		); err != nil {
			return fmt.Errorf(
				"%s: invalid %s signature: %w",
				HardforkApprovalVersionV1,
				role,
				err,
			)
		}

		validCount++
	}

	if validCount < quorum {
		return fmt.Errorf(
			"%s: %s quorum not met: got %d, need %d",
			HardforkApprovalVersionV1,
			role,
			validCount,
			quorum,
		)
	}

	return nil
}

func validateHardforkSHA256HexV1(
	name string,
	value string,
) error {
	value = strings.TrimSpace(value)

	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size {
		return fmt.Errorf(
			"%s: %s hash must be a 32-byte SHA-256 hex value",
			HardforkApprovalVersionV1,
			name,
		)
	}

	return nil
}

func writeHardforkStringV1(
	buffer *bytes.Buffer,
	value string,
) {
	_ = binary.Write(
		buffer,
		binary.BigEndian,
		uint32(len(value)),
	)

	_, _ = buffer.WriteString(value)
}
