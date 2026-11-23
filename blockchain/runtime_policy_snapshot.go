package blockchain

import (
	"encoding/hex"
	"fmt"
	qconsensus "quantumcoin/consensus"
	"strconv"
)

// RuntimePolicySnapshot observes chain identity and the next-block rules under
// one read lock. A declaration is never evidence of hardfork activation.
func (bc *Blockchain) RuntimePolicySnapshot() (map[string]any, error) {
	if bc == nil {
		return nil, fmt.Errorf("blockchain unavailable")
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	if len(bc.Blocks) == 0 || bc.Blocks[0] == nil || bc.Blocks[len(bc.Blocks)-1] == nil {
		return nil, fmt.Errorf("chain identity unavailable")
	}
	tip := bc.Blocks[len(bc.Blocks)-1]
	next := tip.Index + 1
	protocol := bc.ProtocolAtHeightV1(next)
	unit, err := qconsensus.MonetaryUnitAtProtocolV1(protocol)
	if err != nil {
		return nil, err
	}
	reward, err := bc.ExpectedSubsidyUnitsForHeightV1(next)
	if err != nil {
		return nil, err
	}
	context := map[string]any{
		"height": tip.Index, "tipHash": hex.EncodeToString(tip.Hash),
		"genesisHash": hex.EncodeToString(bc.Blocks[0].Hash),
		"nextHeight":  next, "nextProtocol": protocol, "transactionIntegerUnit": unit,
		"atomsPerQc":               strconv.FormatInt(qconsensus.CanonicalAtomsPerQCV1, 10),
		"nextSubsidyProtocolUnits": strconv.Itoa(reward),
		"supplyCapQc":              strconv.Itoa(bc.TotalSupply),
		"activation":               bc.activationPolicy.policy,
	}
	if bc.TotalSupply == int(qconsensus.CanonicalTotalSupplyQCV1) {
		if genesisIssued, err := ValidateBlockSubsidyAgainstMintedV1(bc.Blocks[0], 0, bc.TotalSupply); err == nil {
			if projection, ok := qconsensus.ProjectCanonicalEmissionV1(int64(genesisIssued)); ok {
				context["emissionProjection"] = projection
			}
		}
	}
	return context, nil
}

// BalanceAtomsSnapshot returns exact presentation atoms for both old whole-QC
// protocols and atom protocols. Transaction wire amounts are not rewritten.
func (bc *Blockchain) BalanceAtomsSnapshot(address string) (int64, int64, int, error) {
	if bc == nil {
		return 0, 0, -1, fmt.Errorf("blockchain unavailable")
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	next := bc.nextSpendHeightV1()
	protocol := bc.ProtocolAtHeightV1(next)
	convert := func(n int) (int64, error) {
		if n < 0 {
			return 0, fmt.Errorf("negative balance")
		}
		if qconsensus.ProtocolUsesAtomDenominationV1(protocol) {
			return int64(n), nil
		}
		return qconsensus.WholeQCToAtomsV1(int64(n))
	}
	total, err := convert(bc.balanceInProtocolUnitsAtHeightLockedV1(address, next, false))
	if err != nil {
		return 0, 0, -1, err
	}
	spendable, err := convert(bc.balanceInProtocolUnitsAtHeightLockedV1(address, next, true))
	if err != nil {
		return 0, 0, -1, err
	}
	height := -1
	if len(bc.Blocks) > 0 && bc.Blocks[len(bc.Blocks)-1] != nil {
		height = bc.Blocks[len(bc.Blocks)-1].Index
	}
	return total, spendable, height, nil
}
