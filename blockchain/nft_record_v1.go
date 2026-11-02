package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

type NFTMintRecordV1 struct {
	NetworkID           string
	CampaignID          string
	AwardPeriod         string
	Serial              uint32
	RecipientPubKeyHash []byte
	MetadataHash        [32]byte
}

func validateNFTMintRecordV1(record NFTMintRecordV1) error {
	if !validNFTRecordIdentifierV1(record.NetworkID) {
		return fmt.Errorf("NFT mint record V1: invalid network ID")
	}
	if !validNFTRecordIdentifierV1(record.CampaignID) {
		return fmt.Errorf("NFT mint record V1: invalid campaign ID")
	}
	if _, err := time.Parse("2006-01", record.AwardPeriod); err != nil {
		return fmt.Errorf("NFT mint record V1: invalid award period: %w", err)
	}
	if record.Serial == 0 {
		return fmt.Errorf("NFT mint record V1: serial must be greater than zero")
	}
	if len(record.RecipientPubKeyHash) != 20 {
		return fmt.Errorf("NFT mint record V1: recipient public-key hash must be 20 bytes")
	}
	if record.MetadataHash == ([32]byte{}) {
		return fmt.Errorf("NFT mint record V1: metadata hash is required")
	}
	return nil
}

func validNFTRecordIdentifierV1(value string) bool {
	if value == "" || len(value) > 128 || strings.TrimSpace(value) != value {
		return false
	}
	for i := 0; i < len(value); i++ {
		b := value[i]
		if (b >= 'a' && b <= 'z') ||
			(b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') ||
			b == '-' || b == '_' || b == '.' || b == ':' {
			continue
		}
		return false
	}
	return true
}

func canonicalNFTMintIdentityPayloadV1(record NFTMintRecordV1) ([]byte, error) {
	if err := validateNFTMintRecordV1(record); err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	buffer.WriteString("QC_NFT_TOKEN_ID_V1\x00")

	for _, value := range []string{
		record.NetworkID,
		record.CampaignID,
		record.AwardPeriod,
	} {
		if err := binary.Write(&buffer, binary.BigEndian, uint16(len(value))); err != nil {
			return nil, err
		}
		buffer.WriteString(value)
	}

	if err := binary.Write(&buffer, binary.BigEndian, record.Serial); err != nil {
		return nil, err
	}
	buffer.Write(record.MetadataHash[:])

	return buffer.Bytes(), nil
}

func NFTTokenIDV1(record NFTMintRecordV1) ([32]byte, error) {
	payload, err := canonicalNFTMintIdentityPayloadV1(record)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(payload), nil
}

func NFTMintRecordDigestV1(record NFTMintRecordV1) ([32]byte, error) {
	payload, err := canonicalNFTMintIdentityPayloadV1(record)
	if err != nil {
		return [32]byte{}, err
	}

	var buffer bytes.Buffer
	buffer.WriteString("QC_NFT_MINT_RECORD_V1\x00")
	buffer.Write(payload)
	buffer.Write(record.RecipientPubKeyHash)

	return sha256.Sum256(buffer.Bytes()), nil
}
