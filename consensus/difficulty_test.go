package consensus

import "testing"

func TestNormalizeDefaultDifficultyBits(t *testing.T) {
	tests := []struct {
		name string
		bits int
		want int
	}{
		{
			name: "sifir deger legacy varsayilana doner",
			bits: 0,
			want: LegacyDefaultDifficultyBits,
		},
		{
			name: "negatif deger legacy varsayilana doner",
			bits: -1,
			want: LegacyDefaultDifficultyBits,
		},
		{
			name: "gecerli deger korunur",
			bits: 20,
			want: 20,
		},
		{
			name: "maksimum ustu 255 olur",
			bits: 300,
			want: MaxDifficultyBits,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NormalizeDefaultDifficultyBits(test.bits)

			if got != test.want {
				t.Fatalf(
					"NormalizeDefaultDifficultyBits(%d) = %d, want %d",
					test.bits,
					got,
					test.want,
				)
			}
		})
	}
}

func TestNormalizeCurrentDifficultyBits(t *testing.T) {
	tests := []struct {
		name        string
		currentBits int
		defaultBits int
		want        int
	}{
		{
			name:        "gecersiz current default olur",
			currentBits: 0,
			defaultBits: 16,
			want:        16,
		},
		{
			name:        "default altina dusmez",
			currentBits: 8,
			defaultBits: 16,
			want:        16,
		},
		{
			name:        "gecerli current korunur",
			currentBits: 20,
			defaultBits: 16,
			want:        20,
		},
		{
			name:        "maksimum ustu 255 olur",
			currentBits: 300,
			defaultBits: 16,
			want:        MaxDifficultyBits,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NormalizeCurrentDifficultyBits(
				test.currentBits,
				test.defaultBits,
			)

			if got != test.want {
				t.Fatalf(
					"NormalizeCurrentDifficultyBits(%d, %d) = %d, want %d",
					test.currentBits,
					test.defaultBits,
					got,
					test.want,
				)
			}
		})
	}
}

func TestLegacyNextDifficulty(t *testing.T) {
	tests := []struct {
		name  string
		input DifficultyInput
		want  int
	}{
		{
			name: "yeterli blok yoksa default doner",
			input: DifficultyInput{
				DefaultBits:   16,
				CurrentBits:   20,
				HasFullWindow: false,
			},
			want: 16,
		},
		{
			name: "cok hizliysa bir artar",
			input: DifficultyInput{
				DefaultBits:            16,
				CurrentBits:            20,
				ActualTimespanSeconds:  2399,
				TargetBlockTimeSeconds: 30,
				HasFullWindow:          true,
			},
			want: 21,
		},
		{
			name: "yuzde seksen sinirinda degismez",
			input: DifficultyInput{
				DefaultBits:            16,
				CurrentBits:            20,
				ActualTimespanSeconds:  2400,
				TargetBlockTimeSeconds: 30,
				HasFullWindow:          true,
			},
			want: 20,
		},
		{
			name: "normal surede degismez",
			input: DifficultyInput{
				DefaultBits:            16,
				CurrentBits:            20,
				ActualTimespanSeconds:  3000,
				TargetBlockTimeSeconds: 30,
				HasFullWindow:          true,
			},
			want: 20,
		},
		{
			name: "yuzde yuz yirmi sinirinda degismez",
			input: DifficultyInput{
				DefaultBits:            16,
				CurrentBits:            20,
				ActualTimespanSeconds:  3600,
				TargetBlockTimeSeconds: 30,
				HasFullWindow:          true,
			},
			want: 20,
		},
		{
			name: "cok yavassa bir azalir",
			input: DifficultyInput{
				DefaultBits:            16,
				CurrentBits:            20,
				ActualTimespanSeconds:  3601,
				TargetBlockTimeSeconds: 30,
				HasFullWindow:          true,
			},
			want: 19,
		},
		{
			name: "default altina dusmez",
			input: DifficultyInput{
				DefaultBits:            16,
				CurrentBits:            16,
				ActualTimespanSeconds:  5000,
				TargetBlockTimeSeconds: 30,
				HasFullWindow:          true,
			},
			want: 16,
		},
		{
			name: "maksimum ustune cikmaz",
			input: DifficultyInput{
				DefaultBits:            16,
				CurrentBits:            255,
				ActualTimespanSeconds:  1000,
				TargetBlockTimeSeconds: 30,
				HasFullWindow:          true,
			},
			want: 255,
		},
		{
			name: "gecersiz hedef surede 30 saniye kullanilir",
			input: DifficultyInput{
				DefaultBits:            16,
				CurrentBits:            20,
				ActualTimespanSeconds:  2399,
				TargetBlockTimeSeconds: 0,
				HasFullWindow:          true,
			},
			want: 21,
		},
		{
			name: "gecersiz gercek sure current degeri korur",
			input: DifficultyInput{
				DefaultBits:            16,
				CurrentBits:            20,
				ActualTimespanSeconds:  0,
				TargetBlockTimeSeconds: 30,
				HasFullWindow:          true,
			},
			want: 20,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := LegacyNextDifficulty(test.input)

			if got != test.want {
				t.Fatalf(
					"LegacyNextDifficulty() = %d, want %d",
					got,
					test.want,
				)
			}
		})
	}
}
