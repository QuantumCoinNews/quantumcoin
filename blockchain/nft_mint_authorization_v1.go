package blockchain

import (
	"fmt"

	qconsensus "quantumcoin/consensus"
)

const nftMintAuthorizationVersionV1 = "QC_NFT_MINT_AUTHORIZATION_V1"

// validateNFTMintAuthorizationV1 fails closed until local program rules,
// ownership and all admission paths use the evidence carried by the block.
func validateNFTMintAuthorizationV1(
	block *Block,
	protocol qconsensus.ProtocolVersion,
) error {
	if block == nil {
		return ErrNilBlock
	}

	if len(block.NFTMintRecordsV1) == 0 && len(block.NFTAwardProofsV1) == 0 {
		return nil
	}

	if protocol != qconsensus.ProtocolNFTV4 {
		return fmt.Errorf(
			"%s: NFT mint records require protocol %q",
			nftMintAuthorizationVersionV1,
			qconsensus.ProtocolNFTV4,
		)
	}

	return fmt.Errorf(
		"%s: NFT mint authorization is missing for %d record(s)",
		nftMintAuthorizationVersionV1,
		len(block.NFTMintRecordsV1),
	)
}
