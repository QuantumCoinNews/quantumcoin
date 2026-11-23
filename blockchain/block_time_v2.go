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
	if !bc.usesMedianBlockTimeV2() || len(bc.Blocks) == 0 {
		return now, nil
	}
	last := len(bc.Blocks) - 1
	minimum := max(medianBlockTimeV2(bc.Blocks, last), bc.Blocks[last].Timestamp) + 1
	if minimum > now.Add(MaxFutureDrift).Unix() {
		return time.Time{}, fmt.Errorf("median time past exceeds the allowed mining time")
	}
	if minimum > now.Unix() {
		return time.Unix(minimum, 0).UTC(), nil
	}
	return now, nil
}
