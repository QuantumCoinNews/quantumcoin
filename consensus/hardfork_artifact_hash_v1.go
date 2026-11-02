package consensus

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const HardforkArtifactHashVersionV1 = "QC_HARDFORK_ARTIFACT_HASH_V1"

func ValidateHardforkArtifactsV1(
	decision HardforkDecisionV1,
	snapshot []byte,
	releaseBinary []byte,
) error {
	if len(snapshot) == 0 {
		return fmt.Errorf(
			"%s: snapshot artifact is required",
			HardforkArtifactHashVersionV1,
		)
	}

	if len(releaseBinary) == 0 {
		return fmt.Errorf(
			"%s: release binary artifact is required",
			HardforkArtifactHashVersionV1,
		)
	}

	if err := validateHardforkArtifactHashV1(
		"snapshot",
		decision.SnapshotSHA256,
		snapshot,
	); err != nil {
		return err
	}

	if err := validateHardforkArtifactHashV1(
		"release binary",
		decision.ReleaseBinarySHA256,
		releaseBinary,
	); err != nil {
		return err
	}

	return nil
}

func validateHardforkArtifactHashV1(
	name string,
	expectedHex string,
	artifact []byte,
) error {
	expectedHex = strings.TrimSpace(expectedHex)

	expected, err := hex.DecodeString(expectedHex)
	if err != nil || len(expected) != sha256.Size {
		return fmt.Errorf(
			"%s: %s expected hash must be a 32-byte SHA-256 hex value",
			HardforkArtifactHashVersionV1,
			name,
		)
	}

	actual := sha256.Sum256(artifact)

	if !bytes.Equal(expected, actual[:]) {
		return fmt.Errorf(
			"%s: %s hash mismatch",
			HardforkArtifactHashVersionV1,
			name,
		)
	}

	return nil
}
