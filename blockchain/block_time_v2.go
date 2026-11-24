package blockchain

import (
	"fmt"
	"slices"
	"time"

	"quantumcoin/config"
	"quantumcoin/consensus"
)

const medianBlockTimeWindowV2 = 11

func (bc *Blockchain) usesMedianBlockTimeV2() bool {
	if bc == nil || bc.networkIdentity == nil {
		return false
	}
	domain := bc.networkIdentity.GenesisDomain
	return domain == TestnetGenesisDomainV3 || domain == MainnetGenesisDomainV3 || domain == TestnetGenesisDomainV4 || domain == MainnetGenesisDomainV4
}

func medianBlockTimeV2(blocks []*Block, end int) int64 {
	var window [medianBlockTimeWindowV2]int64
	start := max(0, end+1-len(window))
	times := window[:end-start+1]
	for i := range times {
		times[i] = blocks[start+i].Timestamp
	}
	slices.Sort(times)
	return times[len(times)/2]
}

func (bc *Blockchain) medianNextDifficultyV2() int {
	defaultBits := consensus.CanonicalDefaultDifficultyBitsV1
	if bc.networkIdentity.GenesisDomain == TestnetGenesisDomainV3 || bc.networkIdentity.GenesisDomain == TestnetGenesisDomainV4 {
		defaultBits = config.QCTestnetDefaultDifficultyBitsV1
	}
	input := consensus.DifficultyInput{DefaultBits: defaultBits}
	last := len(bc.Blocks) - 1
	previous := last - consensus.LegacyDifficultyRetargetWindow
	if previous >= medianBlockTimeWindowV2-1 {
		input.CurrentBits = bc.Blocks[last].Difficulty
		input.ActualTimespanSeconds = medianBlockTimeV2(bc.Blocks, last) - medianBlockTimeV2(bc.Blocks, previous)
		input.TargetBlockTimeSeconds = consensus.CanonicalBlockTimeSecondsV1
		input.HasFullWindow = true
	}
	return consensus.LegacyNextDifficulty(input)
}

func (bc *Blockchain) validateNextBlockTimestampV1(block *Block) error {
	if len(bc.Blocks) == 0 {
		return nil
	}
	last := len(bc.Blocks) - 1
	if bc.usesMedianBlockTimeV2() {
		if block.Timestamp <= medianBlockTimeV2(bc.Blocks, last) {
			return fmt.Errorf("block timestamp must exceed median time past at height %d", block.Index)
		}
	} else if block.Timestamp < bc.Blocks[last].Timestamp {
		return fmt.Errorf("block timestamp goes backwards at height %d", block.Index)
	}
	return nil
}

func (bc *Blockchain) nextMiningTimeV1(now time.Time) (time.Time, error) {
	if bc == nil || len(bc.Blocks) == 0 {
		return now, nil
	}
	wallNow := now
	if bc.usesMedianBlockTimeV2() {
		last := len(bc.Blocks) - 1
		minimum := max(medianBlockTimeV2(bc.Blocks, last), bc.Blocks[last].Timestamp) + 1
		if minimum > wallNow.Add(MaxFutureDrift).Unix() {
			return time.Time{}, fmt.Errorf("median time past exceeds the allowed mining time")
		}
		if minimum > now.Unix() {
			now = time.Unix(minimum, 0).UTC()
		}
	}

	// Legacy transaction IDs include the full coinbase timestamp. Keep local
	// coinbase timestamps strictly increasing so a repeated or coarse system
	// clock cannot recreate a confirmed coinbase ID, especially for zero-fee
	// blocks after emission ends.
	if previous, ok := lastCoinbaseTimestampV1(bc.Blocks); ok && !now.After(previous) {
		now = previous.Add(time.Nanosecond)
	}
	if bc.usesMedianBlockTimeV2() && now.After(wallNow.Add(MaxFutureDrift)) {
		return time.Time{}, fmt.Errorf("median time past exceeds the allowed mining time")
	}
	return now, nil
}

func lastCoinbaseTimestampV1(blocks []*Block) (time.Time, bool) {
	for blockIndex := len(blocks) - 1; blockIndex >= 0; blockIndex-- {
		block := blocks[blockIndex]
		if block == nil {
			continue
		}
		for txIndex := len(block.Transactions) - 1; txIndex >= 0; txIndex-- {
			tx := block.Transactions[txIndex]
			if tx == nil || !tx.IsCoinbase() || tx.Timestamp.IsZero() {
				continue
			}
			return tx.Timestamp.UTC(), true
		}
	}
	return time.Time{}, false
}
