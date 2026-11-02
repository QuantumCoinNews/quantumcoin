// TOKEN_AUTHORIZATION_V1

package token

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
)

const TokenAuthorizationVersionV1 = "QC_TOKEN_AUTHORIZATION_V1"

const (
	TokenAuthorizationModelLegacyV1 = "TOKEN"
	TokenAuthorizationModelQC20V1   = "QC20"
	TokenAuthorizationModelQC721V1  = "QC721"
)

const (
	TokenOperationMintV1                = "MINT"
	TokenOperationTransferV1            = "TRANSFER"
	TokenOperationBurnV1                = "BURN"
	TokenOperationApproveV1             = "APPROVE"
	TokenOperationTransferFromV1        = "TRANSFER_FROM"
	TokenOperationRegistryLoadV1        = "REGISTRY_LOAD"
	TokenOperationFungiblePersistenceV1 = "FUNGIBLE_PERSISTENCE_LOAD"
	TokenOperationMinerRewardV1         = "MINER_TOKEN_REWARD"
)

var (
	ErrTokenAuthorizationRequired = errors.New(
		"token: signed authorization required",
	)
	ErrTokenAuthorizationInvalid = errors.New(
		"token: invalid signed authorization",
	)
	ErrTokenAuthorizationNonce = errors.New(
		"token: invalid authorization nonce",
	)
	ErrTokenAuthorizationReplay = errors.New(
		"token: authorization replay rejected",
	)
)

// TokenAuthorizationClaimV1 is the complete operation that a wallet signs.
// Every state-changing field is bound to the signature. Nonce starts at one
// and advances independently for each verified caller and token instance.
type TokenAuthorizationClaimV1 struct {
	Version   string `json:"version"`
	Domain    string `json:"domain"`
	Operation string `json:"operation"`
	Caller    string `json:"caller"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	Spender   string `json:"spender,omitempty"`
	Amount    uint64 `json:"amount,omitempty"`
	TokenID   uint64 `json:"token_id,omitempty"`
	Nonce     uint64 `json:"nonce"`
}

// TokenAuthorizationV1 contains only public signing material.
// Private keys are never stored in this structure.
type TokenAuthorizationV1 struct {
	Claim     TokenAuthorizationClaimV1 `json:"claim"`
	PublicKey []byte                    `json:"public_key"`
	Signature []byte                    `json:"signature"`
}

func writeTokenAuthorizationStringV1(
	buffer *bytes.Buffer,
	value string,
) error {
	if uint64(len(value)) > uint64(^uint32(0)) {
		return fmt.Errorf(
			"token authorization string is too large",
		)
	}

	var length [4]byte
	binary.BigEndian.PutUint32(
		length[:],
		uint32(len(value)),
	)

	if _, err := buffer.Write(length[:]); err != nil {
		return err
	}
	if _, err := buffer.WriteString(value); err != nil {
		return err
	}

	return nil
}

func canonicalTokenAuthorizationPayloadV1(
	claim TokenAuthorizationClaimV1,
) ([]byte, error) {
	if claim.Version != TokenAuthorizationVersionV1 {
		return nil, fmt.Errorf(
			"%w: unsupported version",
			ErrTokenAuthorizationInvalid,
		)
	}

	if strings.TrimSpace(claim.Domain) == "" ||
		strings.TrimSpace(claim.Operation) == "" ||
		strings.TrimSpace(claim.Caller) == "" {
		return nil, fmt.Errorf(
			"%w: domain, operation, and caller are required",
			ErrTokenAuthorizationInvalid,
		)
	}

	if !validTokenAddress(claim.Caller) {
		return nil, fmt.Errorf(
			"%w: caller is not a canonical token address",
			ErrTokenAuthorizationInvalid,
		)
	}

	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "from", value: claim.From},
		{name: "to", value: claim.To},
		{name: "spender", value: claim.Spender},
	} {
		if field.value != "" &&
			!validTokenAddress(field.value) {
			return nil, fmt.Errorf(
				"%w: %s is not a canonical token address",
				ErrTokenAuthorizationInvalid,
				field.name,
			)
		}
	}

	if claim.Nonce == 0 {
		return nil, fmt.Errorf(
			"%w: nonce must start at one",
			ErrTokenAuthorizationNonce,
		)
	}

	var buffer bytes.Buffer

	for _, value := range []string{
		claim.Version,
		claim.Domain,
		claim.Operation,
		claim.Caller,
		claim.From,
		claim.To,
		claim.Spender,
	} {
		if err := writeTokenAuthorizationStringV1(
			&buffer,
			value,
		); err != nil {
			return nil, err
		}
	}

	var number [8]byte

	for _, value := range []uint64{
		claim.Amount,
		claim.TokenID,
		claim.Nonce,
	} {
		binary.BigEndian.PutUint64(
			number[:],
			value,
		)

		if _, err := buffer.Write(number[:]); err != nil {
			return nil, err
		}
	}

	return buffer.Bytes(), nil
}

// TokenAuthorizationDigestV1 returns the canonical digest signed by wallets.
func TokenAuthorizationDigestV1(
	claim TokenAuthorizationClaimV1,
) ([32]byte, error) {
	var zero [32]byte

	payload, err :=
		canonicalTokenAuthorizationPayloadV1(claim)
	if err != nil {
		return zero, err
	}

	return sha256.Sum256(payload), nil
}

// SignTokenAuthorizationV1 is a wallet-side helper. It derives the caller
// address from the public key and refuses a mismatching caller string.
func SignTokenAuthorizationV1(
	claim TokenAuthorizationClaimV1,
	privateKey *ecdsa.PrivateKey,
) (TokenAuthorizationV1, error) {
	var authorization TokenAuthorizationV1

	publicKey, err :=
		qconsensus.UncompressedPublicKeyV1(privateKey)
	if err != nil {
		return authorization, fmt.Errorf(
			"sign token authorization: %w",
			err,
		)
	}

	caller := wallet.GetAddressFromPub(publicKey)

	if strings.TrimSpace(claim.Caller) == "" {
		claim.Caller = caller
	}

	if claim.Caller != caller {
		return authorization, fmt.Errorf(
			"%w: caller does not match signing key",
			ErrTokenAuthorizationInvalid,
		)
	}

	digest, err := TokenAuthorizationDigestV1(claim)
	if err != nil {
		return authorization, err
	}

	signature, err :=
		qconsensus.SignTransactionDigestV1(
			privateKey,
			digest,
		)
	if err != nil {
		return authorization, fmt.Errorf(
			"sign token authorization digest: %w",
			err,
		)
	}

	return TokenAuthorizationV1{
		Claim: claim,
		PublicKey: append(
			[]byte(nil),
			publicKey...,
		),
		Signature: append(
			[]byte(nil),
			signature...,
		),
	}, nil
}

func verifyTokenAuthorizationV1(
	authorization TokenAuthorizationV1,
	expected TokenAuthorizationClaimV1,
) error {
	if authorization.Claim != expected {
		return fmt.Errorf(
			"%w: signed operation does not match requested operation",
			ErrTokenAuthorizationInvalid,
		)
	}

	derivedCaller := wallet.GetAddressFromPub(
		authorization.PublicKey,
	)

	if derivedCaller == "" ||
		derivedCaller != expected.Caller {
		return fmt.Errorf(
			"%w: caller does not match public key",
			ErrTokenAuthorizationInvalid,
		)
	}

	digest, err :=
		TokenAuthorizationDigestV1(expected)
	if err != nil {
		return err
	}

	if err := qconsensus.VerifyTransactionDigestSignatureV1(
		authorization.PublicKey,
		digest,
		authorization.Signature,
	); err != nil {
		return fmt.Errorf(
			"%w: signature verification failed",
			ErrTokenAuthorizationInvalid,
		)
	}

	return nil
}

// TokenAuthorizationDomainV1 separates signatures belonging to different
// token models and token definitions.
func TokenAuthorizationDomainV1(
	model string,
	name string,
	symbol string,
	owner string,
	decimals uint8,
	maxSupply uint64,
) string {
	var buffer bytes.Buffer

	for _, value := range []string{
		TokenAuthorizationVersionV1,
		model,
		name,
		symbol,
		owner,
	} {
		_ = writeTokenAuthorizationStringV1(
			&buffer,
			value,
		)
	}

	_ = buffer.WriteByte(decimals)

	var supply [8]byte
	binary.BigEndian.PutUint64(
		supply[:],
		maxSupply,
	)
	_, _ = buffer.Write(supply[:])

	digest := sha256.Sum256(buffer.Bytes())

	return TokenAuthorizationVersionV1 +
		":" +
		hex.EncodeToString(digest[:])
}

func checkTokenAuthorizationNonceV1(
	nonces map[string]uint64,
	caller string,
	nonce uint64,
) error {
	last := nonces[caller]

	if nonce <= last {
		return fmt.Errorf(
			"%w: caller nonce %d was already consumed",
			ErrTokenAuthorizationReplay,
			nonce,
		)
	}

	if last == ^uint64(0) || nonce != last+1 {
		return fmt.Errorf(
			"%w: got %d, want %d",
			ErrTokenAuthorizationNonce,
			nonce,
			last+1,
		)
	}

	return nil
}

func commitTokenAuthorizationNonceV1(
	nonces map[string]uint64,
	caller string,
	nonce uint64,
) {
	nonces[caller] = nonce
}
