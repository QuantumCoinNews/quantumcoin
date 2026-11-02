package blockchain

import (
	"strconv"

	qconsensus "quantumcoin/consensus"
)

// OutputAmountQCAtHeightV1 formats an output for human-readable block display.
// It uses the output's containing block height, not the current chain tip.
func (bc *Blockchain) OutputAmountQCAtHeightV1(units int, height int) string {
	if bc != nil && qconsensus.ProtocolUsesAtomDenominationV1(bc.ProtocolAtHeightV1(height)) {
		return qconsensus.FormatAtomsAsQCV1(int64(units))
	}
	return strconv.Itoa(units)
}
