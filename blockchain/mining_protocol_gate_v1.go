package blockchain

import (
	"fmt"

	qconsensus "quantumcoin/consensus"
)

const MiningProtocolGateVersionV1 = "QC_LOCAL_MINING_PROTOCOL_GATE_V1"

// validateLocalMiningProtocolV1 prevents the local miner from producing a
// block under a protocol that the canonical/peer validation paths do not yet
// accept.
//
// Legacy mining remains unchanged. Future protocols fail closed until their
// complete block-validation contract is implemented.
func (bc *Blockchain) validateLocalMiningProtocolV1(
	height int,
) error {
	if bc == nil {
		return fmt.Errorf(
			"%s: nil blockchain",
			MiningProtocolGateVersionV1,
		)
	}

	protocol := bc.ProtocolAtHeightV1(height)

	switch protocol {
	case qconsensus.ProtocolLegacyV1:
		return nil

	case qconsensus.ProtocolAtomV3, qconsensus.ProtocolNFTV4:
		if err := qconsensus.ValidateProtocolAmountRuntimeV1(
			protocol,
		); err != nil {
			return fmt.Errorf(
				"%s: protocol runtime validation failed: %w",
				MiningProtocolGateVersionV1,
				err,
			)
		}

		return nil

	case qconsensus.ProtocolContinuityV2:
		return fmt.Errorf(
			"%s: protocol %q is not implemented for local mining",
			MiningProtocolGateVersionV1,
			protocol,
		)

	default:
		return fmt.Errorf(
			"%s: unsupported protocol %q for local mining",
			MiningProtocolGateVersionV1,
			protocol,
		)
	}
}
