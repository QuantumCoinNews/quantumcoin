package consensus

import "testing"

func TestHardforkDecisionAcceptsNFTProtocolV4V1(t *testing.T) {
	decision := hardforkApprovalTestDecisionV1()
	decision.TargetProtocol = ProtocolVersion("QC_NFT_PROTOCOL_V4")

	if _, err := CanonicalHardforkDecisionPayloadV1(decision); err != nil {
		t.Fatalf("NFT protocol target was rejected by hardfork decision validation: %v", err)
	}
}
