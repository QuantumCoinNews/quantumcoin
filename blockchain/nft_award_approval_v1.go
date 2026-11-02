package blockchain

import (
	"fmt"

	qconsensus "quantumcoin/consensus"
)

// ValidateNFTAwardRecordApprovalV1 binds an approval to the actual canonical
// record, including its recipient and metadata. Trust is local configuration.
// This standalone verifier does not enable MintNFT or change block admission.
func ValidateNFTAwardRecordApprovalV1(
	record NFTMintRecordV1,
	decision qconsensus.NFTAwardDecisionV1,
	trust qconsensus.NFTAwardTrustV1,
	approvals []qconsensus.NFTAwardApprovalV1,
) error {
	digest, err := NFTMintRecordDigestV1(record)
	if err != nil {
		return fmt.Errorf("NFT award record approval: %w", err)
	}
	if record.NetworkID != decision.NetworkID {
		return fmt.Errorf("NFT award record approval: record network differs from decision")
	}
	return qconsensus.ValidateNFTAwardApprovalsV1(decision, trust, approvals, digest)
}
