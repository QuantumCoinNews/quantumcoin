package consensus

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// Version bağımsız QuantumCoin consensus sözleşmesinin sürümüdür.
const Version = "QC_CONSENSUS_V1"

// ProtocolVersion, belirli bir blok yüksekliğinde kullanılacak
// consensus kurallarını belirtir.
type ProtocolVersion string

const (
	// ProtocolLegacyV1 mevcut zincirdeki eski blokların kurallarıdır.
	ProtocolLegacyV1 ProtocolVersion = "QC_LEGACY_PROTOCOL_V1"

	// ProtocolContinuityV2 hardfork aktivasyon yüksekliğinden
	// itibaren kullanılacak düzeltilmiş kurallardır.
	ProtocolContinuityV2 ProtocolVersion = "QC_CONTINUITY_PROTOCOL_V2"
	// ProtocolNFTV4 reserves the activation target for consensus NFT records.
	ProtocolNFTV4 ProtocolVersion = "QC_NFT_PROTOCOL_V4"

	// ProtocolAtomV3 uses integer atom denomination for monetary consensus.
	//
	// 1 QC = 1,000,000 atom.
	//
	// Declaring this protocol does not activate it. Canonical activation
	// remains disabled until an explicit validated activation plan is bound.
	ProtocolAtomV3 ProtocolVersion = "QC_ATOM_PROTOCOL_V3"
)

// ActivationPlan mevcut zincirin geçmişini ve bakiyelerini koruyarak
// yeni consensus kurallarının hangi yükseklikte başlayacağını tanımlar.
type ActivationPlan struct {
	NetworkID        string
	LegacyGenesisHex string
	ActivationHeight int
	PreserveHistory  bool
	PreserveBalances bool

	// TargetProtocol selects the protocol that starts at ActivationHeight.
	//
	// Empty preserves historical behavior and resolves to ProtocolContinuityV2.
	TargetProtocol ProtocolVersion
}

// Validate aktivasyon planının güvenli ve eksiksiz olduğunu doğrular.
func (p ActivationPlan) Validate() error {
	if strings.TrimSpace(p.NetworkID) == "" {
		return fmt.Errorf("%s: network id is required", Version)
	}

	genesisHex := strings.TrimSpace(p.LegacyGenesisHex)

	decoded, err := hex.DecodeString(genesisHex)
	if err != nil || len(decoded) != 32 {
		return fmt.Errorf(
			"%s: legacy genesis must be a 32-byte SHA-256 hex value",
			Version,
		)
	}

	if p.ActivationHeight <= 0 {
		return fmt.Errorf(
			"%s: activation height must be greater than zero",
			Version,
		)
	}

	if !p.PreserveHistory {
		return fmt.Errorf(
			"%s: existing block history must be preserved",
			Version,
		)
	}

	if !p.PreserveBalances {
		return fmt.Errorf(
			"%s: existing balances must be preserved",
			Version,
		)
	}

	switch p.EffectiveTargetProtocol() {
	case ProtocolContinuityV2, ProtocolAtomV3, ProtocolNFTV4:
	default:
		return fmt.Errorf(
			"%s: unsupported activation target protocol %q",
			Version,
			p.TargetProtocol,
		)
	}

	return nil
}

// EffectiveTargetProtocol preserves historical activation behavior.
// Empty TargetProtocol means ProtocolContinuityV2.
func (p ActivationPlan) EffectiveTargetProtocol() ProtocolVersion {
	if p.TargetProtocol == "" {
		return ProtocolContinuityV2
	}

	return p.TargetProtocol
}

// ProtocolAtHeight eski bloklarda legacy kurallarını,
// aktivasyon yüksekliğinden itibaren yeni kuralları seçer.
func ProtocolAtHeight(
	height int,
	activationHeight int,
) ProtocolVersion {
	return ProtocolAtHeightTargetV1(
		height,
		activationHeight,
		ProtocolContinuityV2,
	)
}

// ProtocolAtHeightTargetV1 selects an explicit activation target while
// preserving ProtocolAtHeight's historical ContinuityV2 behavior.
func ProtocolAtHeightTargetV1(
	height int,
	activationHeight int,
	target ProtocolVersion,
) ProtocolVersion {
	if activationHeight > 0 && height >= activationHeight {
		return target
	}

	return ProtocolLegacyV1
}
