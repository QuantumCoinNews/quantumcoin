package blockchain

import (
	"fmt"
	"time"

	qconsensus "quantumcoin/consensus"
)

const (
	EconomyConsensusVersionV1 = "QC_ECONOMY_CONSENSUS_V1"

	CanonicalGenesisUnixV1         int64 = 1_725_158_400
	CanonicalBlockTimeSecondsV1    int64 = 30
	CanonicalHalvingIntervalSecsV1 int64 = 63_072_000
	CanonicalBlocksPerHalvingV1          = int(CanonicalHalvingIntervalSecsV1 / CanonicalBlockTimeSecondsV1)
	CanonicalInitialMiningRewardV1       = int(qconsensus.CanonicalInitialMiningRewardQCV1)
	CanonicalMinimumMiningRewardV1       = 1
)

func CanonicalGenesisTimeV1() time.Time {
	return time.Unix(
		CanonicalGenesisUnixV1,
		0,
	).UTC()
}

// RewardAtHeightWithParamsV1 calculates a deterministic integer subsidy.
//
// Height 1 is the first block after genesis. Every canonical two-year block
// interval halves the reward. Because the current transaction amount model is
// integer QC, the reward floors at 1 QC and continues until the hard cap.
func RewardAtHeightWithParamsV1(
	height int,
	initialReward int,
	blocksPerHalving int,
) int {
	if initialReward <= 0 {
		return 0
	}

	if blocksPerHalving <= 0 {
		blocksPerHalving = CanonicalBlocksPerHalvingV1
	}

	if height < 1 {
		height = 1
	}

	halvings := (height - 1) / blocksPerHalving
	reward := initialReward

	for i := 0; i < halvings && reward > CanonicalMinimumMiningRewardV1; i++ {
		reward /= 2

		if reward < CanonicalMinimumMiningRewardV1 {
			reward = CanonicalMinimumMiningRewardV1
		}
	}

	return reward
}

func RewardAtHeightV1(height int) int {
	return RewardAtHeightWithParamsV1(
		height,
		CanonicalInitialMiningRewardV1,
		CanonicalBlocksPerHalvingV1,
	)
}

// CapRewardByRemainingSupplyV1 implements the final partial block reward.
func CapRewardByRemainingSupplyV1(
	baseReward int,
	totalMinted int,
	totalSupply int,
) int {
	return int(
		qconsensus.CapSubsidyByRemainingSupply(
			int64(baseReward),
			int64(totalMinted),
			int64(totalSupply),
		),
	)
}

func ExpectedSubsidyFromStateV1(
	height int,
	totalMinted int,
	totalSupply int,
) int {
	effectiveHeight := height
	if effectiveHeight < 1 {
		effectiveHeight = 1
	}

	baseSubsidy, err := qconsensus.LegacySubsidyAtHeight(
		uint64(effectiveHeight),
	)
	if err != nil {
		return 0
	}

	return CapRewardByRemainingSupplyV1(
		int(baseSubsidy),
		totalMinted,
		totalSupply,
	)
}

func (bc *Blockchain) ExpectedSubsidyForHeightV1(
	height int,
) int {
	if bc == nil {
		return 0
	}

	return ExpectedSubsidyFromStateV1(
		height,
		bc.TotalMinted(),
		bc.TotalSupply,
	)
}

func blockCoinbaseTotalV1(
	block *Block,
	allowMultiple bool,
) (int, int, error) {
	if block == nil {
		return 0, 0, ErrNilBlock
	}

	transactions := make(
		[]*qconsensus.CoinbaseTransactionViewV1,
		len(block.Transactions),
	)

	for transactionIndex, transaction := range block.Transactions {
		if transaction == nil {
			transactions[transactionIndex] = nil
			continue
		}

		outputAmounts := make([]int64, len(transaction.Outputs))
		for outputIndex, output := range transaction.Outputs {
			outputAmounts[outputIndex] = int64(output.Amount)
		}

		transactions[transactionIndex] =
			&qconsensus.CoinbaseTransactionViewV1{
				Coinbase:      transaction.IsCoinbase(),
				OutputAmounts: outputAmounts,
			}
	}

	result, err := qconsensus.ValidateCoinbaseLayoutV1(
		transactions,
		allowMultiple,
	)
	if err != nil {
		return 0, 0, err
	}

	coinbaseTotal := int(result.Total)
	if int64(coinbaseTotal) != result.Total {
		return 0, 0, fmt.Errorf(
			"coinbase total does not fit int: %d",
			result.Total,
		)
	}

	return result.Count, coinbaseTotal, nil
}

func ValidateBlockSubsidyAgainstMintedV1(
	block *Block,
	totalMintedBefore int,
	totalSupply int,
) (int, error) {
	_, coinbaseTotal, err := blockCoinbaseTotalV1(
		block,
		block != nil && block.Index == 0,
	)
	if err != nil {
		return 0, err
	}

	if block.Index == 0 {
		if totalSupply > 0 && coinbaseTotal > totalSupply {
			return 0, fmt.Errorf(
				"genesis allocation exceeds total supply: allocation=%d supply=%d",
				coinbaseTotal,
				totalSupply,
			)
		}

		return coinbaseTotal, nil
	}

	expected := ExpectedSubsidyFromStateV1(
		block.Index,
		totalMintedBefore,
		totalSupply,
	)

	if expected <= 0 {
		return 0, fmt.Errorf(
			"emission is complete at block height %d",
			block.Index,
		)
	}

	if coinbaseTotal != expected {
		return 0, fmt.Errorf(
			"block subsidy mismatch at height %d: got %d want %d",
			block.Index,
			coinbaseTotal,
			expected,
		)
	}

	return coinbaseTotal, nil
}

func (bc *Blockchain) ValidateBlockSubsidyV1(
	block *Block,
) error {
	if bc == nil {
		return fmt.Errorf(
			"nil blockchain",
		)
	}

	_, err := bc.ValidateBlockRewardWithFeesV1(block)
	return err
}

func NewCanonicalGenesisBlockV1(
	transactions []*Transaction,
) (*Block, error) {
	block := &Block{
		Index:        0,
		Timestamp:    CanonicalGenesisUnixV1,
		Transactions: transactions,
		PrevHash:     []byte{},
		Hash:         nil,
		Nonce:        0,
		Miner:        "genesis",
		Difficulty:   1,
		Metadata: map[string]string{
			"economyConsensus": EconomyConsensusVersionV1,
			"genesisVersion":   "QC_CANONICAL_GENESIS_V1",
		},
	}

	pow := NewProofOfWork(block)
	if pow == nil {
		return nil, fmt.Errorf(
			"canonical genesis proof of work is unavailable",
		)
	}

	nonce, hash, err := pow.Run()
	if err != nil {
		return nil, fmt.Errorf(
			"canonical genesis proof of work: %w",
			err,
		)
	}

	block.Nonce = nonce
	block.Hash = append([]byte(nil), hash...)

	if err := ValidateBlockPoWV1(block); err != nil {
		return nil, fmt.Errorf(
			"canonical genesis validation: %w",
			err,
		)
	}

	return block, nil
}
