package consensus

const (
	// LegacyDifficultyRetargetWindow mevcut zincirin
	// difficulty hesabında kullandığı blok penceresidir.
	LegacyDifficultyRetargetWindow int = 100

	// LegacyDefaultDifficultyBits geçersiz veya eksik başlangıç
	// difficulty değerinde kullanılan legacy varsayılandır.
	LegacyDefaultDifficultyBits int = 16

	// MaxDifficultyBits SHA-256 tabanlı mevcut PoW sisteminde
	// kabul edilen en yüksek difficulty bit değeridir.
	MaxDifficultyBits int = 255
)

// DifficultyInput legacy difficulty hesaplaması için gereken
// zincirden bağımsız verileri taşır.
//
// HasFullWindow false olduğunda yeterli blok bulunmadığı kabul edilir
// ve normalize edilmiş DefaultBits değeri döndürülür.
type DifficultyInput struct {
	DefaultBits int
	CurrentBits int

	ActualTimespanSeconds  int64
	TargetBlockTimeSeconds int64

	HasFullWindow bool
}

// NormalizeDefaultDifficultyBits legacy başlangıç difficulty
// değerini mevcut zincir kurallarına göre sınırlar.
func NormalizeDefaultDifficultyBits(defaultBits int) int {
	if defaultBits <= 0 {
		return LegacyDefaultDifficultyBits
	}

	if defaultBits > MaxDifficultyBits {
		return MaxDifficultyBits
	}

	return defaultBits
}

// NormalizeCurrentDifficultyBits mevcut blok difficulty değerini
// legacy taban ve maksimum sınırlar içinde tutar.
func NormalizeCurrentDifficultyBits(
	currentBits int,
	defaultBits int,
) int {
	defaultBits = NormalizeDefaultDifficultyBits(defaultBits)

	if currentBits <= 0 {
		currentBits = defaultBits
	}

	if currentBits < defaultBits {
		currentBits = defaultBits
	}

	if currentBits > MaxDifficultyBits {
		currentBits = MaxDifficultyBits
	}

	return currentBits
}

// LegacyNextDifficulty mevcut QuantumCoin zincirindeki
// blockchain.NextDifficulty davranışını zincirden bağımsız
// consensus fonksiyonu olarak uygular.
//
// Kurallar:
//   - Yeterli 100 blokluk pencere yoksa default difficulty döner.
//   - Hedef blok süresi geçersizse 30 saniye kullanılır.
//   - Gerçek süre hedefin yüzde 80'inden azsa difficulty 1 artar.
//   - Gerçek süre hedefin yüzde 120'sinden fazlaysa difficulty 1 azalır.
//   - Difficulty default değerin altına ve 255'in üstüne çıkamaz.
func LegacyNextDifficulty(input DifficultyInput) int {
	defaultBits := NormalizeDefaultDifficultyBits(
		input.DefaultBits,
	)

	if !input.HasFullWindow {
		return defaultBits
	}

	currentBits := NormalizeCurrentDifficultyBits(
		input.CurrentBits,
		defaultBits,
	)

	targetBlockTimeSeconds := input.TargetBlockTimeSeconds
	if targetBlockTimeSeconds <= 0 {
		targetBlockTimeSeconds = int64(
			LegacyTargetBlockTime.Seconds(),
		)
	}

	expectedTimespanSeconds :=
		targetBlockTimeSeconds *
			int64(LegacyDifficultyRetargetWindow)

	actualTimespanSeconds := input.ActualTimespanSeconds

	if actualTimespanSeconds <= 0 ||
		expectedTimespanSeconds <= 0 {
		return currentBits
	}

	fastThreshold :=
		expectedTimespanSeconds * 8 / 10

	slowThreshold :=
		expectedTimespanSeconds * 12 / 10

	if actualTimespanSeconds < fastThreshold &&
		currentBits < MaxDifficultyBits {
		return currentBits + 1
	}

	if actualTimespanSeconds > slowThreshold &&
		currentBits > defaultBits {
		return currentBits - 1
	}

	return currentBits
}
