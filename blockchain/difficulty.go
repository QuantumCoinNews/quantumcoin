package blockchain

import (
	"fmt"

	"quantumcoin/config"
	qconsensus "quantumcoin/consensus"
)

const (
	// Eski blockchain içi adları koruyoruz; değerlerin kanonik kaynağı
	// artık consensus paketidir.
	difficultyRetargetWindow = qconsensus.LegacyDifficultyRetargetWindow
	maxDifficultyBits        = qconsensus.MaxDifficultyBits
)

// NextDifficulty mevcut zincir temposuna göre önerilen dinamik PoW
// difficulty değerini döndürür.
//
// Dış sözleşme geriye uyumluluk için blockchain paketinde korunur;
// deterministik retarget hesabı kanonik consensus motoruna devredilir.
func (bc *Blockchain) NextDifficulty(
	defaultBits int,
) int {
	input := qconsensus.DifficultyInput{
		DefaultBits: defaultBits,
	}

	if bc == nil ||
		len(bc.Blocks) <= difficultyRetargetWindow {
		return qconsensus.LegacyNextDifficulty(
			input,
		)
	}

	lastIndex := len(bc.Blocks) - 1
	prevIndex := lastIndex - difficultyRetargetWindow

	last := bc.Blocks[lastIndex]
	prev := bc.Blocks[prevIndex]

	if last == nil || prev == nil {
		return qconsensus.LegacyNextDifficulty(
			input,
		)
	}

	targetBlockTimeSeconds :=
		config.Current().TargetBlockTimeSecs

	input.CurrentBits = last.Difficulty
	input.ActualTimespanSeconds =
		last.Timestamp - prev.Timestamp
	input.TargetBlockTimeSeconds =
		int64(targetBlockTimeSeconds)
	input.HasFullWindow = true

	return qconsensus.LegacyNextDifficulty(
		input,
	)
}

// ExpectedNextDifficultyV1 returns the effective difficulty required for the
// next block after the current chain tip.
//
// It uses the same resolution path as the node and local miner:
// canonical base difficulty, deterministic chain retarget and the temporary
// fixed override where the selected runtime profile explicitly permits it.
//
// Public testnet and mainnet profiles reject fixed overrides.
func (bc *Blockchain) ExpectedNextDifficultyV1() (
	int,
	error,
) {
	if bc == nil {
		return 0, fmt.Errorf(
			"expected next difficulty: nil blockchain",
		)
	}

	cfg := config.Current()
	if cfg == nil {
		return 0, fmt.Errorf(
			"expected next difficulty: nil config",
		)
	}

	dynamicBits := bc.NextDifficulty(
		cfg.DefaultDifficultyBits,
	)

	info, err := config.ResolveEffectiveDifficultyBitsV1(
		cfg,
		dynamicBits,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"expected next difficulty: %w",
			err,
		)
	}

	return info.EffectiveBits, nil
}

// ValidateExpectedDifficultyV1 requires a candidate next block to use exactly
// the difficulty selected for the current chain state and runtime profile.
func (bc *Blockchain) ValidateExpectedDifficultyV1(
	blk *Block,
) error {
	if bc == nil {
		return fmt.Errorf(
			"expected difficulty validation: nil blockchain",
		)
	}

	if blk == nil {
		return ErrNilBlock
	}

	expected, err := bc.ExpectedNextDifficultyV1()
	if err != nil {
		return err
	}

	if blk.Difficulty != expected {
		return fmt.Errorf(
			"block difficulty mismatch at height %d: got %d, expected %d",
			blk.Index,
			blk.Difficulty,
			expected,
		)
	}

	return nil
}
