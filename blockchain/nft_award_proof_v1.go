package blockchain

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	qconsensus "quantumcoin/consensus"
)

// NFTAwardProofV1 is public evidence, not an authority policy. Trusted keys
// remain private local Blockchain state and are never carried in this type.
type NFTAwardProofV1 struct {
	Decision  qconsensus.NFTAwardDecisionV1
	Approvals []qconsensus.NFTAwardApprovalV1
}

func cloneNFTAwardProofsV1(proofs []NFTAwardProofV1) []NFTAwardProofV1 {
	if proofs == nil {
		return nil
	}
	result := make([]NFTAwardProofV1, len(proofs))
	for i, proof := range proofs {
		result[i].Decision = proof.Decision
		if proof.Approvals != nil {
			result[i].Approvals = make([]qconsensus.NFTAwardApprovalV1, len(proof.Approvals))
			for j, approval := range proof.Approvals {
				result[i].Approvals[j] = qconsensus.NFTAwardApprovalV1{
					PublicKey: append([]byte(nil), approval.PublicKey...),
					Signature: append([]byte(nil), approval.Signature...),
				}
			}
		}
	}
	return result
}

// NFTBlockCommitmentDigestV1 binds ordered record/proof pairs and each proof's
// ordered canonical public-key/signature bytes. It validates representation,
// not local authority/quorum, game eligibility, campaign limits or ownership.
// Missing proofs retain the previous record-only digest for compatibility;
// runtime admission continues to reject all nonempty NFT data in this wave.
func NFTBlockCommitmentDigestV1(block *Block) ([32]byte, error) {
	if block == nil {
		return [32]byte{}, ErrNilBlock
	}
	recordsDigest, err := NFTMintRecordsDigestV1(block.NFTMintRecordsV1)
	if err != nil {
		return [32]byte{}, err
	}
	if len(block.NFTAwardProofsV1) == 0 {
		return recordsDigest, nil
	}
	if len(block.NFTAwardProofsV1) != len(block.NFTMintRecordsV1) {
		return [32]byte{}, fmt.Errorf("NFT block commitment: record/proof count mismatch")
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("QC_NFT_APPROVED_BLOCK_COMMITMENT_V1\x00"))
	_, _ = hasher.Write(recordsDigest[:])
	writeCount := func(value int) {
		var encoded [8]byte
		binary.BigEndian.PutUint64(encoded[:], uint64(value))
		_, _ = hasher.Write(encoded[:])
	}
	writeCount(len(block.NFTAwardProofsV1))
	for i, proof := range block.NFTAwardProofsV1 {
		record := block.NFTMintRecordsV1[i]
		recordDigest, err := NFTMintRecordDigestV1(record)
		if err != nil {
			return [32]byte{}, err
		}
		if proof.Decision.NetworkID != record.NetworkID || proof.Decision.RecordDigest != recordDigest {
			return [32]byte{}, fmt.Errorf("NFT block commitment: proof %d differs from actual record", i)
		}
		decisionDigest, err := qconsensus.NFTAwardDecisionDigestV1(proof.Decision)
		if err != nil {
			return [32]byte{}, fmt.Errorf("NFT block commitment: proof %d: %w", i, err)
		}
		if len(proof.Approvals) == 0 {
			return [32]byte{}, fmt.Errorf("NFT block commitment: proof %d has no approvals", i)
		}
		_, _ = hasher.Write(decisionDigest[:])
		writeCount(len(proof.Approvals))
		for j, approval := range proof.Approvals {
			if _, err := qconsensus.ParseUncompressedPublicKeyV1(approval.PublicKey); err != nil {
				return [32]byte{}, fmt.Errorf("NFT block commitment: proof %d signer %d: %w", i, j, err)
			}
			if _, _, err := qconsensus.DecodeCanonicalTransactionSignatureV1(approval.Signature); err != nil {
				return [32]byte{}, fmt.Errorf("NFT block commitment: proof %d signature %d: %w", i, j, err)
			}
			writeCount(len(approval.PublicKey))
			_, _ = hasher.Write(approval.PublicKey)
			writeCount(len(approval.Signature))
			_, _ = hasher.Write(approval.Signature)
		}
	}
	var result [32]byte
	copy(result[:], hasher.Sum(nil))
	return result, nil
}
