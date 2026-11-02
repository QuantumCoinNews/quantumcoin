package blockchain

import "errors"

var (
	ErrNFTMintDisabled      = errors.New("blockchain nft: chain NFT mint is disabled by safe boundary")
	ErrNFTRecipientRequired = errors.New("blockchain nft: recipient address is required")
	ErrNFTTypeRequired      = errors.New("blockchain nft: nft type is required")
)

func (bc *Blockchain) MintNFT(toAddress, nftType string, meta map[string]string) (string, error) {
	return "", ErrNFTMintDisabled
}
