package blockchain

import (
	"bytes"
	"fmt"
	"os"
	"quantumcoin/config"
	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
	"strings"
	"time"
)

const (
	FounderVestingConsensusVersionV1 = "QC_FOUNDER_VESTING_CONSENSUS_V1"
	FounderCliffMonthsV1             = qconsensus.CanonicalFounderCliffMonthsV1
	FounderVestingMonthsV1           = qconsensus.CanonicalFounderVestingMonthsV1
	FounderMonthlyUnlockQCV1         = int(qconsensus.CanonicalFounderMonthlyUnlockQCV1)
	FounderLockedAllocationV1        = int(qconsensus.CanonicalFounderLockedQCV1)
)

func founderAddressStringsV1() (
	string,
	string,
	bool,
) {
	p := config.Current()
	if p == nil {
		return "", "", false
	}

	fallback := strings.TrimSpace(p.PremineAddress)
	if fallback == "" {
		fallback = strings.TrimSpace(
			os.Getenv("QC_MAIN_ADDRESS"),
		)
	}

	liquid := strings.TrimSpace(
		p.PremineLiquidAddress,
	)
	if liquid == "" {
		liquid = fallback
	}

	locked := strings.TrimSpace(
		p.PremineLockedAddress,
	)
	if locked == "" {
		locked = fallback
	}

	if liquid == "" ||
		locked == "" ||
		liquid == locked {
		return liquid, locked, false
	}

	return liquid, locked, true
}

func FounderVestingReadyV1() bool {
	_, _, ready := founderAddressStringsV1()

	return ready
}

func founderLockedPubKeyHashV1() (
	[]byte,
	bool,
) {
	_, lockedAddress, ready :=
		founderAddressStringsV1()

	if !ready {
		return nil, false
	}

	pubKeyHash := wallet.Base58DecodeAddress(
		lockedAddress,
	)
	if len(pubKeyHash) == 0 {
		return nil, false
	}

	return pubKeyHash, true
}

func founderLockedPubKeyHashFromGenesisV1(
	bc *Blockchain,
) ([]byte, bool) {
	if bc == nil ||
		len(bc.Blocks) == 0 ||
		bc.Blocks[0] == nil {
		return nil, false
	}

	var lockedPubKeyHash []byte

	for _, tx := range bc.Blocks[0].Transactions {
		if tx == nil {
			continue
		}

		for _, output := range tx.Outputs {
			if output.Amount != FounderLockedAllocationV1 ||
				len(output.PubKeyHash) == 0 {
				continue
			}

			if len(lockedPubKeyHash) != 0 {
				// Consensus must never guess between multiple
				// candidate founder locked genesis outputs.
				return nil, false
			}

			lockedPubKeyHash = append(
				[]byte(nil),
				output.PubKeyHash...,
			)
		}
	}

	if len(lockedPubKeyHash) == 0 {
		return nil, false
	}

	return lockedPubKeyHash, true
}

func completedCalendarMonthsV1(
	start time.Time,
	end time.Time,
) int {
	start = start.UTC()
	end = end.UTC()

	if end.Before(start) {
		return 0
	}

	months := (end.Year()-start.Year())*12 +
		int(end.Month()-start.Month())

	if end.Before(start.AddDate(0, months, 0)) {
		months--
	}

	if months < 0 {
		return 0
	}

	return months
}

// FounderUnlockedAtUnixV1 starts the first monthly tranche at month 13 and
// finishes all 48 tranches at month 60.
func FounderUnlockedAtUnixV1(
	genesisUnix int64,
	atUnix int64,
) int {
	if atUnix <= genesisUnix {
		return 0
	}

	genesisTime := time.Unix(
		genesisUnix,
		0,
	).UTC()

	atTime := time.Unix(
		atUnix,
		0,
	).UTC()

	elapsedMonths := completedCalendarMonthsV1(
		genesisTime,
		atTime,
	)

	if elapsedMonths < FounderCliffMonthsV1+1 {
		return 0
	}

	tranches := elapsedMonths - FounderCliffMonthsV1
	if tranches > FounderVestingMonthsV1 {
		tranches = FounderVestingMonthsV1
	}

	unlocked := tranches * FounderMonthlyUnlockQCV1
	if unlocked > FounderLockedAllocationV1 {
		return FounderLockedAllocationV1
	}

	return unlocked
}

func (bc *Blockchain) findOutputByOutpointWithHeightV1(
	txID []byte,
	outputIndex int,
) (TransactionOutput, int, bool) {
	if bc == nil ||
		len(txID) == 0 ||
		outputIndex < 0 {
		return TransactionOutput{}, -1, false
	}

	for _, block := range bc.Blocks {
		if block == nil {
			continue
		}

		for _, tx := range block.Transactions {
			if tx == nil ||
				!bytes.Equal(tx.ID, txID) {
				continue
			}

			if outputIndex >= len(tx.Outputs) {
				return TransactionOutput{}, -1, false
			}

			return tx.Outputs[outputIndex],
				block.Index,
				true
		}
	}

	return TransactionOutput{}, -1, false
}

func (bc *Blockchain) findOutputByOutpointV1(
	txID []byte,
	outputIndex int,
) (TransactionOutput, bool) {
	output, _, found :=
		bc.findOutputByOutpointWithHeightV1(
			txID,
			outputIndex,
		)

	return output, found
}

func (bc *Blockchain) founderNetReleaseForTxAtHeightV1(
	tx *Transaction,
	lockedPubKeyHash []byte,
	txHeight int,
	accountingHeight int,
) (int, error) {
	if tx == nil || tx.IsCoinbase() {
		return 0, nil
	}

	if bc == nil {
		return 0, fmt.Errorf(
			"founder vesting denomination: nil blockchain",
		)
	}

	if txHeight < 0 ||
		accountingHeight < 0 ||
		txHeight > accountingHeight {
		return 0, fmt.Errorf(
			"founder vesting denomination: invalid heights tx=%d accounting=%d",
			txHeight,
			accountingHeight,
		)
	}

	lockedInputTotal := 0
	lockedOutputTotal := 0

	for _, input := range tx.Inputs {
		previousOutput, originHeight, found :=
			bc.findOutputByOutpointWithHeightV1(
				input.TxID,
				input.OutIndex,
			)

		if !found {
			continue
		}

		if previousOutput.IsLockedWithKey(
			lockedPubKeyHash,
		) {
			normalizedAmount, err :=
				bc.normalizeUTXOAmountForSpendV1(
					previousOutput.Amount,
					originHeight,
					accountingHeight,
				)
			if err != nil {
				return 0, fmt.Errorf(
					"founder locked input denomination: %w",
					err,
				)
			}

			next := lockedInputTotal +
				normalizedAmount

			if next < lockedInputTotal {
				return 0, fmt.Errorf(
					"founder locked input total overflow",
				)
			}

			lockedInputTotal = next
		}
	}

	if lockedInputTotal == 0 {
		return 0, nil
	}

	for _, output := range tx.Outputs {
		if output.IsLockedWithKey(
			lockedPubKeyHash,
		) {
			normalizedAmount, err :=
				bc.normalizeUTXOAmountForSpendV1(
					output.Amount,
					txHeight,
					accountingHeight,
				)
			if err != nil {
				return 0, fmt.Errorf(
					"founder locked change denomination: %w",
					err,
				)
			}

			next := lockedOutputTotal +
				normalizedAmount

			if next < lockedOutputTotal {
				return 0, fmt.Errorf(
					"founder locked change total overflow",
				)
			}

			lockedOutputTotal = next
		}
	}

	if lockedOutputTotal >= lockedInputTotal {
		return 0, nil
	}

	return lockedInputTotal -
			lockedOutputTotal,
		nil
}

func (bc *Blockchain) founderNetReleaseForTxV1(
	tx *Transaction,
	lockedPubKeyHash []byte,
) (int, error) {
	spendHeight := bc.nextSpendHeightV1()

	return bc.founderNetReleaseForTxAtHeightV1(
		tx,
		lockedPubKeyHash,
		spendHeight,
		spendHeight,
	)
}

func (bc *Blockchain) founderReleasedConfirmedAtHeightV1(
	lockedPubKeyHash []byte,
	accountingHeight int,
) (int, error) {
	if bc == nil {
		return 0, fmt.Errorf(
			"nil blockchain",
		)
	}

	if accountingHeight < 0 {
		return 0, fmt.Errorf(
			"founder vesting invalid accounting height %d",
			accountingHeight,
		)
	}

	total := 0

	for _, block := range bc.Blocks {
		if block == nil {
			continue
		}

		for _, tx := range block.Transactions {
			netRelease, err :=
				bc.founderNetReleaseForTxAtHeightV1(
					tx,
					lockedPubKeyHash,
					block.Index,
					accountingHeight,
				)
			if err != nil {
				return 0, err
			}

			next := total + netRelease
			if next < total {
				return 0, fmt.Errorf(
					"founder released total overflow",
				)
			}

			total = next
		}
	}

	return total, nil
}

func (bc *Blockchain) founderReleasedConfirmedV1(
	lockedPubKeyHash []byte,
) (int, error) {
	return bc.founderReleasedConfirmedAtHeightV1(
		lockedPubKeyHash,
		bc.nextSpendHeightV1(),
	)
}

func (bc *Blockchain) founderUnlockedUnitsAtHeightV1(
	unlockedQC int,
	height int,
) (int, error) {
	if bc == nil {
		return 0, fmt.Errorf(
			"founder vesting denomination: nil blockchain",
		)
	}

	if unlockedQC < 0 || height < 0 {
		return 0, fmt.Errorf(
			"founder vesting denomination: invalid unlocked=%d height=%d",
			unlockedQC,
			height,
		)
	}

	protocol := bc.ProtocolAtHeightV1(height)

	units, err :=
		qconsensus.WholeQCAmountToProtocolUnitsV1(
			protocol,
			int64(unlockedQC),
		)
	if err != nil {
		return 0, fmt.Errorf(
			"founder unlocked denomination: %w",
			err,
		)
	}

	maxInt := int64(^uint(0) >> 1)
	if units > maxInt {
		return 0, fmt.Errorf(
			"founder unlocked denomination exceeds runtime int capacity",
		)
	}

	return int(units), nil
}

func (bc *Blockchain) ValidateFounderVestingSpendV1(
	tx *Transaction,
	atUnix int64,
) error {
	lockedPubKeyHash, active :=
		founderLockedPubKeyHashFromGenesisV1(bc)

	if !active {
		return nil
	}

	if bc == nil ||
		len(bc.Blocks) == 0 ||
		bc.Blocks[0] == nil {
		return fmt.Errorf(
			"founder vesting requires a valid genesis block",
		)
	}

	spendHeight := bc.nextSpendHeightV1()

	netRelease, err :=
		bc.founderNetReleaseForTxAtHeightV1(
			tx,
			lockedPubKeyHash,
			spendHeight,
			spendHeight,
		)
	if err != nil {
		return err
	}

	if netRelease == 0 {
		return nil
	}

	confirmedReleased, err :=
		bc.founderReleasedConfirmedAtHeightV1(
			lockedPubKeyHash,
			spendHeight,
		)
	if err != nil {
		return err
	}

	unlockedQC := FounderUnlockedAtUnixV1(
		bc.Blocks[0].Timestamp,
		atUnix,
	)

	unlocked, err :=
		bc.founderUnlockedUnitsAtHeightV1(
			unlockedQC,
			spendHeight,
		)
	if err != nil {
		return err
	}

	nextReleased :=
		confirmedReleased + netRelease

	if nextReleased < confirmedReleased {
		return fmt.Errorf(
			"founder vesting release total overflow",
		)
	}

	if nextReleased > unlocked {
		return fmt.Errorf(
			"founder vesting spend exceeds unlocked allocation: confirmed=%d requested=%d unlocked=%d",
			confirmedReleased,
			netRelease,
			unlocked,
		)
	}

	return nil
}

func (bc *Blockchain) ValidateFounderVestingBlockV1(
	block *Block,
) error {
	lockedPubKeyHash, active :=
		founderLockedPubKeyHashFromGenesisV1(bc)

	if !active {
		return nil
	}

	if bc == nil ||
		len(bc.Blocks) == 0 ||
		bc.Blocks[0] == nil {
		return fmt.Errorf(
			"founder vesting requires a valid genesis block",
		)
	}

	if block == nil {
		return ErrNilBlock
	}

	accountingHeight := block.Index

	confirmedReleased, err :=
		bc.founderReleasedConfirmedAtHeightV1(
			lockedPubKeyHash,
			accountingHeight,
		)
	if err != nil {
		return err
	}

	blockRelease := 0

	for _, tx := range block.Transactions {
		netRelease, err :=
			bc.founderNetReleaseForTxAtHeightV1(
				tx,
				lockedPubKeyHash,
				accountingHeight,
				accountingHeight,
			)
		if err != nil {
			return err
		}

		next := blockRelease + netRelease
		if next < blockRelease {
			return fmt.Errorf(
				"founder block release overflow",
			)
		}

		blockRelease = next
	}

	unlockedQC := FounderUnlockedAtUnixV1(
		bc.Blocks[0].Timestamp,
		block.Timestamp,
	)

	unlocked, err :=
		bc.founderUnlockedUnitsAtHeightV1(
			unlockedQC,
			accountingHeight,
		)
	if err != nil {
		return err
	}

	totalReleased :=
		confirmedReleased + blockRelease

	if totalReleased < confirmedReleased {
		return fmt.Errorf(
			"founder vesting block release total overflow",
		)
	}

	if totalReleased > unlocked {
		return fmt.Errorf(
			"founder vesting block exceeds unlocked allocation: confirmed=%d block=%d unlocked=%d",
			confirmedReleased,
			blockRelease,
			unlocked,
		)
	}

	return nil
}
