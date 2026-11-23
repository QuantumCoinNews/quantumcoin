// Package offline implements explicit operator-driven recovery into isolated
// directories. It is never invoked by node startup or an HTTP handler.
package offline

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"quantumcoin/utils"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
	"quantumcoin/wallet"
)

const maxWalletBytes = 4 << 20
const walletHeader = "QC_OFFLINE_WALLET_AES256GCM_PBKDF2_SHA256_600000_V1"

type walletEnvelope struct {
	Version    string `json:"version"`
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

// Custody contains public recovery instructions only. Closed-schema parsing
// rejects unexpected secret-bearing fields; private keys belong in the envelope.
type Custody struct {
	Custodians        []string  `json:"custodians"`
	RecoveryProcedure string    `json:"recoveryProcedure"`
	Addresses         []string  `json:"addresses"`
	LastDrillUTC      time.Time `json:"lastDrillUtc"`
}

func strictJSON(data []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return errors.New("invalid recovery JSON schema")
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("trailing recovery JSON")
	}
	return nil
}

// WalletAddresses validates the two current on-disk QC wallet JSON formats.
// No private material is returned or included in error messages.
func WalletAddresses(raw []byte) ([]string, error) {
	if len(raw) == 0 || len(raw) > maxWalletBytes {
		return nil, errors.New("invalid wallet size")
	}
	var store struct {
		Wallets       map[string]string `json:"wallets"`
		Default       string            `json:"default"`
		PrivateKeyHex string            `json:"privateKeyHex"`
		Address       string            `json:"address"`
	}
	if err := strictJSON(raw, &store); err != nil {
		return nil, err
	}
	keys := store.Wallets
	if store.PrivateKeyHex != "" {
		if len(keys) != 0 || store.Default != "" {
			return nil, errors.New("ambiguous wallet format")
		}
		keys = map[string]string{store.Address: store.PrivateKeyHex}
	}
	if len(keys) == 0 {
		return nil, errors.New("wallet contains no recoverable keys")
	}
	addresses := make([]string, 0, len(keys))
	for address, hexKey := range keys {
		if len(hexKey) != 64 {
			return nil, errors.New("invalid wallet key")
		}
		key, err := wallet.ImportPrivateKeyHex(hexKey)
		if err != nil {
			return nil, errors.New("invalid wallet key")
		}
		pub, err := utils.SecpPublicBytesV1(&key.PublicKey)
		if err != nil {
			return nil, err
		}
		if wallet.GetAddressFromPub(pub) != address {
			return nil, errors.New("wallet address/key mismatch")
		}
		addresses = append(addresses, address)
	}
	if store.Default != "" {
		if _, ok := keys[store.Default]; !ok {
			return nil, errors.New("wallet default address missing")
		}
	}
	sort.Strings(addresses)
	return addresses, nil
}

func SealWallet(raw, password []byte) ([]byte, error) {
	if len(password) < 12 {
		return nil, errors.New("wallet password requires at least 12 bytes")
	}
	if _, err := WalletAddresses(raw); err != nil {
		return nil, err
	}
	e := walletEnvelope{Version: walletHeader, Salt: make([]byte, 16), Nonce: make([]byte, 12)}
	if _, err := rand.Read(e.Salt); err != nil {
		return nil, err
	}
	if _, err := rand.Read(e.Nonce); err != nil {
		return nil, err
	}
	key := pbkdf2.Key(password, e.Salt, 600000, 32, sha256.New)
	defer wipe(key)
	b, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	g, err := cipher.NewGCM(b)
	if err != nil {
		return nil, err
	}
	e.Ciphertext = g.Seal(nil, e.Nonce, raw, []byte(walletHeader))
	return json.MarshalIndent(e, "", "  ")
}

func OpenWallet(sealed, password []byte) ([]byte, error) {
	if len(sealed) > 2*maxWalletBytes || len(password) < 12 {
		return nil, errors.New("invalid encrypted wallet or password")
	}
	var e walletEnvelope
	if err := strictJSON(sealed, &e); err != nil {
		return nil, err
	}
	if e.Version != walletHeader || len(e.Salt) != 16 || len(e.Nonce) != 12 || len(e.Ciphertext) < 16 {
		return nil, errors.New("invalid encrypted wallet envelope")
	}
	key := pbkdf2.Key(password, e.Salt, 600000, 32, sha256.New)
	defer wipe(key)
	b, _ := aes.NewCipher(key)
	g, _ := cipher.NewGCM(b)
	raw, err := g.Open(nil, e.Nonce, e.Ciphertext, []byte(walletHeader))
	if err != nil {
		return nil, errors.New("wallet authentication failed")
	}
	if _, err := WalletAddresses(raw); err != nil {
		wipe(raw)
		return nil, err
	}
	return raw, nil
}

func verifyCustody(data, sealed, password []byte) error {
	var c Custody
	if err := strictJSON(data, &c); err != nil {
		return err
	}
	if len(c.Custodians) == 0 || strings.TrimSpace(c.RecoveryProcedure) == "" || c.LastDrillUTC.IsZero() || c.LastDrillUTC.After(time.Now().UTC().Add(5*time.Minute)) {
		return errors.New("custody instructions and completed recovery drill required")
	}
	for _, ref := range c.Custodians {
		if strings.TrimSpace(ref) == "" {
			return errors.New("custodian reference required")
		}
	}
	raw, err := OpenWallet(sealed, password)
	if err != nil {
		return err
	}
	defer wipe(raw)
	addresses, err := WalletAddresses(raw)
	if err != nil {
		return err
	}
	sort.Strings(c.Addresses)
	if len(addresses) != len(c.Addresses) {
		return errors.New("custody address coverage mismatch")
	}
	for i := range addresses {
		if addresses[i] != c.Addresses[i] {
			return errors.New("custody address coverage mismatch")
		}
	}
	return nil
}

func wipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
