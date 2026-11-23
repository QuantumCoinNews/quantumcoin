package wallet

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"quantumcoin/utils"
	"regexp"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	secpECDSA "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

type RecoveryLinkChallenge struct {
	Version     string `json:"version"`
	NetworkID   string `json:"networkID"`
	GenesisHex  string `json:"genesisHex"`
	PlayerID    string `json:"playerId"`
	Address     string `json:"address"`
	WalletOwner string `json:"walletOwner"`
	Nonce       string `json:"nonce"`
	IssuedAt    int64  `json:"issuedAt"`
	ExpiresAt   int64  `json:"expiresAt"`
}

type RecoveryLinkProof struct {
	PlayerID  string `json:"playerId"`
	Address   string `json:"address"`
	Nonce     string `json:"nonce"`
	PublicKey string `json:"publicKey"`
	Signature string `json:"signature"`
}

var linkPlayerPattern = regexp.MustCompile(`^tg_[0-9]{1,20}$`)
var linkNetworkPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,128}$`)
var linkHexPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// SignRecoveryLink This signs only the existing, short-lived wallet-link domain. It cannot sign
// arbitrary messages, transactions, social claims or rewards.
func SignRecoveryLink(m RecoveryMaterial, c RecoveryLinkChallenge, playerID string, now int64) (*RecoveryLinkProof, error) {
	if c.Version != "QC_GAME_WALLET_LINK_V1" || !linkPlayerPattern.MatchString(playerID) ||
		c.PlayerID != playerID || c.Address != m.Address || !linkNetworkPattern.MatchString(c.NetworkID) ||
		!linkHexPattern.MatchString(c.GenesisHex) || c.GenesisHex == strings.Repeat("0", 64) ||
		!linkHexPattern.MatchString(c.Nonce) || c.IssuedAt < 0 || c.ExpiresAt-c.IssuedAt != 300 ||
		now < c.IssuedAt || now >= c.ExpiresAt {
		return nil, errors.New("invalid, expired or different-account wallet link challenge")
	}
	w, err := m.Wallet()
	if err != nil {
		return nil, err
	}
	defer utils.ZeroSecpPrivateV1(w.PrivateKey)
	if c.WalletOwner != hex.EncodeToString(HashPubKey(w.PublicKey)) {
		return nil, errors.New("wallet does not own challenge address")
	}
	// ASCII-only validated values and sorted map keys match the Python bridge's
	// canonical JSON, independently of Go struct field ordering.
	data, err := json.Marshal(map[string]any{
		"version": c.Version, "networkID": c.NetworkID, "genesisHex": c.GenesisHex,
		"playerId": c.PlayerID, "address": c.Address, "walletOwner": c.WalletOwner,
		"nonce": c.Nonce, "issuedAt": c.IssuedAt, "expiresAt": c.ExpiresAt,
	})
	if err != nil {
		return nil, errors.New("invalid public challenge")
	}
	digest := sha256.Sum256(append([]byte("QC_GAME_WALLET_LINK_V1\x00"), data...))
	scalar, err := utils.SecpPrivateBytesV1(w.PrivateKey)
	if err != nil {
		return nil, err
	}
	defer clear(scalar)
	private := secp256k1.PrivKeyFromBytes(scalar)
	defer private.Zero()
	proof := &RecoveryLinkProof{PlayerID: playerID, Address: m.Address, Nonce: c.Nonce,
		PublicKey: hex.EncodeToString(w.PublicKey), Signature: hex.EncodeToString(secpECDSA.Sign(private, digest[:]).Serialize())}
	return proof, nil
}
