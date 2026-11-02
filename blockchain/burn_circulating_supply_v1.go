package blockchain

import (
	"fmt"

	qconsensus "quantumcoin/consensus"
)

const BurnCirculatingSupplyVersionV1 = "QC_BURN_CIRCULATING_SUPPLY_V1"

// CirculatingSupplyAtomsCheckedV1 returns issued supply minus permanently
// burned supply, expressed in canonical atoms.
//
// TotalMinted is whole-QC issuance accounting. It is converted to atoms before
// canonical burned atoms are subtracted.
//
// The configured TotalSupply hard cap is never modified by this calculation.
func (bc *Blockchain) CirculatingSupplyAtomsCheckedV1() (int64, error) {
	if bc == nil {
		return 0, fmt.Errorf(
			"%s: nil blockchain",
			BurnCirculatingSupplyVersionV1,
		)
	}

	issuedQC := bc.TotalMinted()
	if issuedQC < 0 {
		return 0, fmt.Errorf(
			"%s: negative issued supply: %d",
			BurnCirculatingSupplyVersionV1,
			issuedQC,
		)
	}

	issuedAtoms, err :=
		qconsensus.WholeQCToAtomsV1(int64(issuedQC))
	if err != nil {
		return 0, fmt.Errorf(
			"%s: issued supply conversion: %w",
			BurnCirculatingSupplyVersionV1,
			err,
		)
	}

	burnedAtoms, err := bc.TotalBurnedAtomsCheckedV1()
	if err != nil {
		return 0, fmt.Errorf(
			"%s: burned supply accounting: %w",
			BurnCirculatingSupplyVersionV1,
			err,
		)
	}

	if burnedAtoms > issuedAtoms {
		return 0, fmt.Errorf(
			"%s: burned supply exceeds issued supply: burned=%d issued=%d",
			BurnCirculatingSupplyVersionV1,
			burnedAtoms,
			issuedAtoms,
		)
	}

	return issuedAtoms - burnedAtoms, nil
}
