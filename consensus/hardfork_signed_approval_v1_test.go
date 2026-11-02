package consensus

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

func hardforkApprovalTestKeyV1(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(
		secp256k1.S256(),
		rand.Reader,
	)
	if err != nil {
		t.Fatalf("generate approval key: %v", err)
	}

	return key
}

func hardforkApprovalTestDecisionV1() HardforkDecisionV1 {
	return HardforkDecisionV1{
		NetworkID:        "quantumcoin-mainnet",
		LegacyGenesisHex: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ActivationHeight: 1000,
		TargetProtocol:   ProtocolAtomV3,
		Reasons: []string{
			"NETWORK_WIDE_PROTOCOL_UPGRADE_WITH_PUBLIC_NOTICE",
		},
		SnapshotSHA256:      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ReleaseBinarySHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}
}

func hardforkApprovalTestSignatureV1(
	t *testing.T,
	privateKey *ecdsa.PrivateKey,
	decision HardforkDecisionV1,
) HardforkApprovalV1 {
	t.Helper()

	payload, err := CanonicalHardforkDecisionPayloadV1(decision)
	if err != nil {
		t.Fatalf("canonical hardfork payload: %v", err)
	}

	digest := sha256.Sum256(payload)

	signature, err := SignTransactionDigestV1(
		privateKey,
		digest,
	)
	if err != nil {
		t.Fatalf("sign hardfork decision: %v", err)
	}

	publicKey, err := UncompressedPublicKeyV1(privateKey)
	if err != nil {
		t.Fatalf("hardfork approval public key: %v", err)
	}

	return HardforkApprovalV1{
		PublicKey: publicKey,
		Signature: signature,
	}
}

func TestHardforkSignedApprovalRequiresFounderAndAdminQuorumV1(
	t *testing.T,
) {
	founder := hardforkApprovalTestKeyV1(t)
	admin1 := hardforkApprovalTestKeyV1(t)
	admin2 := hardforkApprovalTestKeyV1(t)

	founderPub, err := UncompressedPublicKeyV1(founder)
	if err != nil {
		t.Fatal(err)
	}

	admin1Pub, err := UncompressedPublicKeyV1(admin1)
	if err != nil {
		t.Fatal(err)
	}

	admin2Pub, err := UncompressedPublicKeyV1(admin2)
	if err != nil {
		t.Fatal(err)
	}

	authorities := HardforkAuthoritySetV1{
		FounderPublicKeys: [][]byte{
			founderPub,
		},
		AdminPublicKeys: [][]byte{
			admin1Pub,
			admin2Pub,
		},
		FounderQuorum: 1,
		AdminQuorum:   2,
	}

	decision := hardforkApprovalTestDecisionV1()

	approvals := HardforkApprovalSetV1{
		Founder: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(
				t,
				founder,
				decision,
			),
		},
		Admin: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(
				t,
				admin1,
				decision,
			),
			hardforkApprovalTestSignatureV1(
				t,
				admin2,
				decision,
			),
		},
	}

	if err := ValidateHardforkApprovalsV1(
		decision,
		authorities,
		approvals,
	); err != nil {
		t.Fatalf(
			"valid founder/admin quorum rejected: %v",
			err,
		)
	}
}

func TestHardforkSignedApprovalRejectsMissingAdminQuorumV1(
	t *testing.T,
) {
	founder := hardforkApprovalTestKeyV1(t)
	admin1 := hardforkApprovalTestKeyV1(t)
	admin2 := hardforkApprovalTestKeyV1(t)

	founderPub, _ := UncompressedPublicKeyV1(founder)
	admin1Pub, _ := UncompressedPublicKeyV1(admin1)
	admin2Pub, _ := UncompressedPublicKeyV1(admin2)

	authorities := HardforkAuthoritySetV1{
		FounderPublicKeys: [][]byte{founderPub},
		AdminPublicKeys: [][]byte{
			admin1Pub,
			admin2Pub,
		},
		FounderQuorum: 1,
		AdminQuorum:   2,
	}

	decision := hardforkApprovalTestDecisionV1()

	approvals := HardforkApprovalSetV1{
		Founder: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(
				t,
				founder,
				decision,
			),
		},
		Admin: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(
				t,
				admin1,
				decision,
			),
		},
	}

	if err := ValidateHardforkApprovalsV1(
		decision,
		authorities,
		approvals,
	); err == nil {
		t.Fatal("hardfork accepted without required admin quorum")
	}
}

func TestHardforkSignedApprovalRejectsDuplicateSignerV1(
	t *testing.T,
) {
	founder := hardforkApprovalTestKeyV1(t)
	admin1 := hardforkApprovalTestKeyV1(t)
	admin2 := hardforkApprovalTestKeyV1(t)

	founderPub, _ := UncompressedPublicKeyV1(founder)
	admin1Pub, _ := UncompressedPublicKeyV1(admin1)
	admin2Pub, _ := UncompressedPublicKeyV1(admin2)

	authorities := HardforkAuthoritySetV1{
		FounderPublicKeys: [][]byte{founderPub},
		AdminPublicKeys: [][]byte{
			admin1Pub,
			admin2Pub,
		},
		FounderQuorum: 1,
		AdminQuorum:   2,
	}

	decision := hardforkApprovalTestDecisionV1()

	adminApproval := hardforkApprovalTestSignatureV1(
		t,
		admin1,
		decision,
	)

	approvals := HardforkApprovalSetV1{
		Founder: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(
				t,
				founder,
				decision,
			),
		},
		Admin: []HardforkApprovalV1{
			adminApproval,
			adminApproval,
		},
	}

	if err := ValidateHardforkApprovalsV1(
		decision,
		authorities,
		approvals,
	); err == nil {
		t.Fatal("duplicate admin signer counted twice toward quorum")
	}
}
func TestHardforkSignedApprovalBindsConsensusCriticalFieldsV1(
	t *testing.T,
) {
	founder := hardforkApprovalTestKeyV1(t)
	admin := hardforkApprovalTestKeyV1(t)

	founderPub, _ := UncompressedPublicKeyV1(founder)
	adminPub, _ := UncompressedPublicKeyV1(admin)

	authorities := HardforkAuthoritySetV1{
		FounderPublicKeys: [][]byte{founderPub},
		AdminPublicKeys:   [][]byte{adminPub},
		FounderQuorum:     1,
		AdminQuorum:       1,
	}

	decision := hardforkApprovalTestDecisionV1()

	approvals := HardforkApprovalSetV1{
		Founder: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(
				t,
				founder,
				decision,
			),
		},
		Admin: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(
				t,
				admin,
				decision,
			),
		},
	}

	mutated := decision
	mutated.ActivationHeight++

	if err := ValidateHardforkApprovalsV1(
		mutated,
		authorities,
		approvals,
	); err == nil {
		t.Fatal(
			"signature remained valid after activation height mutation",
		)
	}

	mutated = decision
	mutated.SnapshotSHA256 =
		"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"

	if err := ValidateHardforkApprovalsV1(
		mutated,
		authorities,
		approvals,
	); err == nil {
		t.Fatal(
			"signature remained valid after snapshot hash mutation",
		)
	}

	mutated = decision
	mutated.ReleaseBinarySHA256 =
		"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"

	if err := ValidateHardforkApprovalsV1(
		mutated,
		authorities,
		approvals,
	); err == nil {
		t.Fatal(
			"signature remained valid after binary hash mutation",
		)
	}
}
