package blockchain

import qconsensus "quantumcoin/consensus"

// balanceProtocolUnitsToQCV1 is presentation-only conversion.
// Consensus arithmetic remains integer protocol units.
func balanceProtocolUnitsToQCV1(
	units int,
	protocol qconsensus.ProtocolVersion,
) float64 {
	if qconsensus.ProtocolUsesAtomDenominationV1(protocol) {
		return float64(units) /
			float64(qconsensus.CanonicalAtomsPerQCV1)
	}

	return float64(units)
}

// BalanceQCSnapshotV1 returns total/spendable balances in user-facing QC.
//
// Candidate height, denomination, UTXO state and response height are observed
// under one state read boundary so activation-height changes cannot split the
// API view across two protocol states.
func (bc *Blockchain) BalanceQCSnapshotV1(
	address string,
) (
	totalQC float64,
	spendableQC float64,
	height int,
) {
	if bc == nil {
		return 0, 0, -1
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	spendHeight := bc.nextSpendHeightV1()

	totalUnits :=
		bc.balanceInProtocolUnitsAtHeightLockedV1(
			address,
			spendHeight,
			false,
		)

	spendableUnits :=
		bc.balanceInProtocolUnitsAtHeightLockedV1(
			address,
			spendHeight,
			true,
		)

	protocol := bc.ProtocolAtHeightV1(spendHeight)

	height = -1
	if len(bc.Blocks) > 0 {
		if tip := bc.Blocks[len(bc.Blocks)-1]; tip != nil {
			height = tip.Index
		}
	}

	return balanceProtocolUnitsToQCV1(
			totalUnits,
			protocol,
		),
		balanceProtocolUnitsToQCV1(
			spendableUnits,
			protocol,
		),
		height
}
