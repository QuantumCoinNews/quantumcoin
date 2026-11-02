package blockchain

import (
	"fmt"

	qconsensus "quantumcoin/consensus"
)

const FeeConsensusVersionV1 = "QC_VERIFIED_FEE_ACCOUNTING_V1"

// BlockRewardBreakdownV1 separates newly minted subsidy from transaction fees.
// Fees are conserved value from transaction inputs; they are never new mint.
type BlockRewardBreakdownV1 struct {
	Subsidy       int
	Fees          int
	CoinbaseTotal int
}

// CalculateBlockFeesV1 delegates all prevout, ownership and value checks to
// the single canonical UTXO validation engine. Fees are conserved input value.
func (bc *Blockchain) CalculateBlockFeesAtHeightV1(
	transactions []*Transaction,
	spendHeight int,
) (int, error) {
	fees, err := bc.ValidateTransactionsUTXOAtHeightV1(
		transactions,
		spendHeight,
	)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", FeeConsensusVersionV1, err)
	}

	return fees, nil
}

func (bc *Blockchain) CalculateBlockFeesV1(
	transactions []*Transaction,
) (int, error) {
	return bc.CalculateBlockFeesAtHeightV1(
		transactions,
		bc.nextSpendHeightV1(),
	)
}

// ValidateBlockRewardAgainstStateV1 requires the coinbase total to equal the
// deterministic subsidy plus fees verified from the pre-block UTXO state.
func (bc *Blockchain) ValidateBlockRewardAgainstStateV1(
	block *Block,
	totalMintedBefore int,
) (BlockRewardBreakdownV1, error) {
	var result BlockRewardBreakdownV1

	if bc == nil {
		return result, fmt.Errorf(
			"%s: nil blockchain",
			FeeConsensusVersionV1,
		)
	}
	if block == nil {
		return result, ErrNilBlock
	}

	_, coinbaseTotal, err := blockCoinbaseTotalV1(
		block,
		block.Index == 0,
	)
	if err != nil {
		return result, err
	}

	if block.Index == 0 {
		consensusResult, err := qconsensus.ValidateBlockRewardV1(
			0,
			int64(coinbaseTotal),
			0,
			0,
			int64(bc.TotalSupply),
		)
		if err != nil {
			return result, err
		}

		result.Subsidy = int(consensusResult.Subsidy)
		result.Fees = int(consensusResult.Fees)
		result.CoinbaseTotal = int(consensusResult.CoinbaseTotal)
		return result, nil
	}

	fees, err := bc.CalculateBlockFeesAtHeightV1(
		block.Transactions,
		block.Index,
	)
	if err != nil {
		return result, fmt.Errorf(
			"block fee validation failed: %w",
			err,
		)
	}

	protocol := bc.ProtocolAtHeightV1(block.Index)

	subsidy, err := ExpectedSubsidyUnitsFromStateV1(
		protocol,
		block.Index,
		totalMintedBefore,
		bc.TotalSupply,
	)
	if err != nil {
		return result, fmt.Errorf(
			"%s: expected subsidy units: %w",
			FeeConsensusVersionV1,
			err,
		)
	}

	consensusResult, err := qconsensus.ValidateBlockRewardV1(
		uint64(block.Index),
		int64(coinbaseTotal),
		int64(subsidy),
		int64(fees),
		int64(bc.TotalSupply),
	)
	if err != nil {
		return result, err
	}

	result.Subsidy = int(consensusResult.Subsidy)
	result.Fees = int(consensusResult.Fees)
	result.CoinbaseTotal = int(consensusResult.CoinbaseTotal)

	return result, nil
}
func (bc *Blockchain) ValidateBlockRewardWithFeesV1(
	block *Block,
) (BlockRewardBreakdownV1, error) {
	if bc == nil {
		return BlockRewardBreakdownV1{}, fmt.Errorf(
			"%s: nil blockchain",
			FeeConsensusVersionV1,
		)
	}

	return bc.ValidateBlockRewardAgainstStateV1(
		block,
		bc.TotalMinted(),
	)
}
