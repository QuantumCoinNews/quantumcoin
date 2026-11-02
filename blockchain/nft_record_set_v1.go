package blockchain

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

const nftMintRecordSetDomainV1 = "QC_NFT_MINT_RECORD_SET_V1\x00"

// NFTMintRecordsDigestV1 returns an ordered, domain-separated digest of
// canonical NFT mint records. Each record digest includes its validated
// network, campaign, period, serial, recipient, and metadata commitment.
func NFTMintRecordsDigestV1(records []NFTMintRecordV1) ([32]byte, error) {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(nftMintRecordSetDomainV1))

	var count [8]byte
	binary.BigEndian.PutUint64(count[:], uint64(len(records)))
	_, _ = hasher.Write(count[:])

	for index, record := range records {
		recordDigest, err := NFTMintRecordDigestV1(record)
		if err != nil {
			return [32]byte{}, fmt.Errorf(
				"NFT mint record set V1: record %d: %w",
				index,
				err,
			)
		}
		_, _ = hasher.Write(recordDigest[:])
	}

	var digest [32]byte
	copy(digest[:], hasher.Sum(nil))
	return digest, nil
}
