package consensus

import "testing"

func TestProtocolAtHeight(t *testing.T) {
	tests := []struct {
		name             string
		height           int
		activationHeight int
		want             ProtocolVersion
	}{
		{
			name:             "aktivasyon öncesi legacy",
			height:           99,
			activationHeight: 100,
			want:             ProtocolLegacyV1,
		},
		{
			name:             "aktivasyon bloğu continuity",
			height:           100,
			activationHeight: 100,
			want:             ProtocolContinuityV2,
		},
		{
			name:             "aktivasyon kapalı",
			height:           500,
			activationHeight: 0,
			want:             ProtocolLegacyV1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ProtocolAtHeight(
				test.height,
				test.activationHeight,
			)

			if got != test.want {
				t.Fatalf(
					"got %q, want %q",
					got,
					test.want,
				)
			}
		})
	}
}

func TestActivationPlanValidate(t *testing.T) {
	plan := ActivationPlan{
		NetworkID: "quantumcoin-mainnet-v1",

		// Şimdilik test değeri. Gerçek genesis hash daha sonra
		// aktif chain_data.dat üzerinden çıkarılacak.
		LegacyGenesisHex: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",

		ActivationHeight: 100,
		PreserveHistory:  true,
		PreserveBalances: true,
	}

	if err := plan.Validate(); err != nil {
		t.Fatalf("valid plan rejected: %v", err)
	}
}

func TestActivationPlanRejectsBalanceReset(t *testing.T) {
	plan := ActivationPlan{
		NetworkID:        "quantumcoin-mainnet-v1",
		LegacyGenesisHex: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ActivationHeight: 100,
		PreserveHistory:  true,
		PreserveBalances: false,
	}

	if err := plan.Validate(); err == nil {
		t.Fatal("balance reset plan must be rejected")
	}
}
