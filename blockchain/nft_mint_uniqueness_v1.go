package blockchain

import "fmt"

const nftMintUniquenessVersionV1 = "QC_NFT_MINT_UNIQUENESS_V1"

type nftAwardPeriodKeyV1 struct {
	networkID   string
	awardPeriod string
}

func validateNFTMintUniquenessV1(
	existingBlocks []*Block,
	candidate *Block,
) error {
	if candidate == nil {
		return ErrNilBlock
	}

	seen := make(map[nftAwardPeriodKeyV1]string)

	checkBlock := func(block *Block, location string) error {
		if block == nil {
			return fmt.Errorf(
				"%s: nil block at %s",
				nftMintUniquenessVersionV1,
				location,
			)
		}

		for recordIndex, record := range block.NFTMintRecordsV1 {
			if err := validateNFTMintRecordV1(record); err != nil {
				return fmt.Errorf(
					"%s: invalid NFT record at %s record[%d]: %w",
					nftMintUniquenessVersionV1,
					location,
					recordIndex,
					err,
				)
			}

			key := nftAwardPeriodKeyV1{
				networkID:   record.NetworkID,
				awardPeriod: record.AwardPeriod,
			}
			if previous, exists := seen[key]; exists {
				return fmt.Errorf(
					"%s: duplicate NFT award period network=%q period=%q at %s; already present at %s",
					nftMintUniquenessVersionV1,
					key.networkID,
					key.awardPeriod,
					location,
					previous,
				)
			}

			seen[key] = location
		}

		return nil
	}

	for index, block := range existingBlocks {
		if err := checkBlock(block, fmt.Sprintf("existing block[%d]", index)); err != nil {
			return err
		}
	}

	return checkBlock(candidate, "candidate block")
}

func validateNFTMintChainUniquenessV1(blocks []*Block) error {
	if len(blocks) == 0 {
		return nil
	}

	return validateNFTMintUniquenessV1(
		blocks[:len(blocks)-1],
		blocks[len(blocks)-1],
	)
}
