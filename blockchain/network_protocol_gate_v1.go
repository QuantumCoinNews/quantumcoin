package blockchain

import (
	"fmt"

	qconsensus "quantumcoin/consensus"
)

const NetworkProtocolGateVersionV1 = "QC_NETWORK_PROTOCOL_GATE_V1"

// validateNetworkBlockProtocolV1 controls which protocol versions are allowed
// to enter peer-block and replacement-chain validation.
//
// Allowing AtomV3 here does not activate it. ProtocolAtHeightV1 can select
// AtomV3 only when an explicit valid activation policy has been bound.
//
// ContinuityV2 remains fail-closed until its rules are implemented.
func validateNetworkBlockProtocolV1(
	protocol qconsensus.ProtocolVersion,
) error {
	switch protocol {
	case qconsensus.ProtocolLegacyV1,
		qconsensus.ProtocolAtomV3,
		qconsensus.ProtocolNFTV4:
		return qconsensus.ValidateProtocolAmountRuntimeV1(
			protocol,
		)

	case qconsensus.ProtocolContinuityV2:
		return fmt.Errorf(
			"%s: protocol %q is not implemented",
			NetworkProtocolGateVersionV1,
			protocol,
		)

	default:
		return fmt.Errorf(
			"%s: unsupported protocol %q",
			NetworkProtocolGateVersionV1,
			protocol,
		)
	}
}
