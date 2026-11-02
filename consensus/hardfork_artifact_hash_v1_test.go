package consensus

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func hardforkArtifactDecisionV1(
	snapshot []byte,
	releaseBinary []byte,
) HardforkDecisionV1 {
	decision := hardforkApprovalTestDecisionV1()

	snapshotHash := sha256.Sum256(snapshot)
	binaryHash := sha256.Sum256(releaseBinary)

	decision.SnapshotSHA256 =
		hex.EncodeToString(snapshotHash[:])

	decision.ReleaseBinarySHA256 =
		hex.EncodeToString(binaryHash[:])

	return decision
}

func TestHardforkArtifactHashesAcceptExactBytesV1(
	t *testing.T,
) {
	snapshot := []byte(
		"quantumcoin-audited-snapshot-v1",
	)
	releaseBinary := []byte(
		"quantumcoin-release-binary-v1",
	)

	decision := hardforkArtifactDecisionV1(
		snapshot,
		releaseBinary,
	)

	if err := ValidateHardforkArtifactsV1(
		decision,
		snapshot,
		releaseBinary,
	); err != nil {
		t.Fatalf(
			"matching snapshot/binary artifacts rejected: %v",
			err,
		)
	}
}

func TestHardforkArtifactHashesRejectMutatedSnapshotV1(
	t *testing.T,
) {
	snapshot := []byte(
		"quantumcoin-audited-snapshot-v1",
	)
	releaseBinary := []byte(
		"quantumcoin-release-binary-v1",
	)

	decision := hardforkArtifactDecisionV1(
		snapshot,
		releaseBinary,
	)

	mutatedSnapshot := append(
		[]byte(nil),
		snapshot...,
	)
	mutatedSnapshot[0] ^= 0x01

	if err := ValidateHardforkArtifactsV1(
		decision,
		mutatedSnapshot,
		releaseBinary,
	); err == nil {
		t.Fatal(
			"mutated snapshot bytes matched signed snapshot hash",
		)
	}
}

func TestHardforkArtifactHashesRejectMutatedBinaryV1(
	t *testing.T,
) {
	snapshot := []byte(
		"quantumcoin-audited-snapshot-v1",
	)
	releaseBinary := []byte(
		"quantumcoin-release-binary-v1",
	)

	decision := hardforkArtifactDecisionV1(
		snapshot,
		releaseBinary,
	)

	mutatedBinary := append(
		[]byte(nil),
		releaseBinary...,
	)
	mutatedBinary[len(mutatedBinary)-1] ^= 0x01

	if err := ValidateHardforkArtifactsV1(
		decision,
		snapshot,
		mutatedBinary,
	); err == nil {
		t.Fatal(
			"mutated release binary bytes matched signed binary hash",
		)
	}
}

func TestHardforkArtifactHashesRejectMissingArtifactsV1(
	t *testing.T,
) {
	snapshot := []byte(
		"quantumcoin-audited-snapshot-v1",
	)
	releaseBinary := []byte(
		"quantumcoin-release-binary-v1",
	)

	decision := hardforkArtifactDecisionV1(
		snapshot,
		releaseBinary,
	)

	if err := ValidateHardforkArtifactsV1(
		decision,
		nil,
		releaseBinary,
	); err == nil {
		t.Fatal("missing snapshot artifact was accepted")
	}

	if err := ValidateHardforkArtifactsV1(
		decision,
		snapshot,
		nil,
	); err == nil {
		t.Fatal("missing release binary artifact was accepted")
	}
}
