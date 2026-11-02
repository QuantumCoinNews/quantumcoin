package consensus

import "testing"

func TestNFTProtocolActivationHeightBoundaryV4(t *testing.T) {
	const activationHeight = 100

	tests := []struct {
		name             string
		height           int
		activationHeight int
		want             ProtocolVersion
	}{
		{
			name:             "before activation",
			height:           activationHeight - 1,
			activationHeight: activationHeight,
			want:             ProtocolLegacyV1,
		},
		{
			name:             "at activation",
			height:           activationHeight,
			activationHeight: activationHeight,
			want:             ProtocolNFTV4,
		},
		{
			name:             "after activation",
			height:           activationHeight + 1,
			activationHeight: activationHeight,
			want:             ProtocolNFTV4,
		},
		{
			name:             "activation unset",
			height:           activationHeight,
			activationHeight: 0,
			want:             ProtocolLegacyV1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ProtocolAtHeightTargetV1(
				test.height,
				test.activationHeight,
				ProtocolNFTV4,
			)
			if got != test.want {
				t.Fatalf("protocol at height %d = %q, want %q",
					test.height, got, test.want)
			}
		})
	}
}
