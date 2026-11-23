package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	bip39 "github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/pbkdf2"
)

// MnemonicProfileV1 Preserve the derivation already used by qc_wallet_create/qc_wallet_restore.
// This is a QC single-address profile, not an Ethereum/BIP32 wallet.
const MnemonicProfileV1 = "qc-canonical-deterministic-wallet-v1"
const RecoveryVaultVersionV1 = "QC_ENCRYPTED_MNEMONIC_V1"
const recoveryIterations = 600000

var ErrRecoveryAddressMismatch = errors.New("recovery words do not match the expected QC address")

type RecoveryMaterial struct {
	Profile  string `json:"profile"`
	Address  string `json:"address"`
	Mnemonic string `json:"mnemonic"`
}

// NewRecoveryMaterial creates a fresh 256-bit, 24-word wallet. No fixed seed.
func NewRecoveryMaterial() (*RecoveryMaterial, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return nil, errors.New("could not generate wallet entropy")
	}
	defer clear(entropy)
	words, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil, errors.New("could not encode recovery words")
	}
	return recoveryMaterialExact(words)
}

func recoveryMaterialExact(words string) (*RecoveryMaterial, error) {
	if len(words) > 2048 || !bip39.IsMnemonicValid(words) {
		return nil, errors.New("invalid recovery word count or checksum")
	}
	w := deriveRecoveryWallet(words)
	return &RecoveryMaterial{Profile: MnemonicProfileV1, Address: w.GetAddress(), Mnemonic: words}, nil
}

func deriveRecoveryWallet(words string) *Wallet {
	seed := bip39.NewSeed(words, "")
	defer clear(seed)
	digest := sha256.Sum256(seed)
	defer clear(digest[:])
	n := new(big.Int).Sub(secp256k1.S256().Params().N, big.NewInt(1))
	d := new(big.Int).SetBytes(digest[:])
	d.Mod(d, n).Add(d, big.NewInt(1))
	var scalar [32]byte
	d.FillBytes(scalar[:])
	defer clear(scalar[:])
	key := secp256k1.PrivKeyFromBytes(scalar[:])
	defer key.Zero()
	return &Wallet{PrivateKey: key.ToECDSA(), PublicKey: key.PubKey().SerializeUncompressed()}
}

// RestoreRecoveryMaterial never falls back to creating a different wallet.
// English word spacing is normalized for ordinary recovery. If an old restore
// tool derived from noncanonical spacing, retain its exact input only when it
// proves control of the independently supplied old address.
func RestoreRecoveryMaterial(words, expectedAddress string) (*RecoveryMaterial, error) {
	if len(words) > 2048 {
		return nil, errors.New("recovery input too large")
	}
	if expectedAddress != "" && !ValidateAddress(expectedAddress) {
		return nil, errors.New("invalid expected QC address")
	}
	canonical := strings.Join(strings.Fields(words), " ")
	m, err := recoveryMaterialExact(canonical)
	if err != nil {
		return nil, err
	}
	if expectedAddress == "" || m.Address == expectedAddress {
		return m, nil
	}
	raw := strings.TrimSpace(words)
	if raw != canonical {
		if legacy, e := recoveryMaterialExact(raw); e == nil && legacy.Address == expectedAddress {
			return legacy, nil
		}
	}
	return nil, ErrRecoveryAddressMismatch
}

func (m RecoveryMaterial) Wallet() (*Wallet, error) {
	if m.Profile != MnemonicProfileV1 || !ValidateAddress(m.Address) {
		return nil, errors.New("unsupported recovery profile or QC address")
	}
	verified, err := recoveryMaterialExact(m.Mnemonic)
	if err != nil || verified.Address != m.Address {
		return nil, ErrRecoveryAddressMismatch
	}
	return deriveRecoveryWallet(m.Mnemonic), nil
}

type recoveryEnvelope struct {
	Version    string `json:"version"`
	Profile    string `json:"profile"`
	Address    string `json:"address"`
	KDF        string `json:"kdf"`
	Iterations int    `json:"iterations"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func recoveryPassword(password string) error {
	if len(password) < 14 || len(password) > 1024 {
		return errors.New("use a local wallet password of 14 to 1024 bytes")
	}
	return nil
}

func recoveryAAD(v recoveryEnvelope) []byte {
	return []byte(RecoveryVaultVersionV1 + "\x00" + v.Profile + "\x00" + v.Address)
}

func recoveryGCM(password string, salt []byte) (cipher.AEAD, error) {
	key := pbkdf2.Key([]byte(password), salt, recoveryIterations, 32, sha256.New)
	defer clear(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("wallet encryption unavailable")
	}
	return cipher.NewGCM(block)
}

func SealRecoveryMaterial(m RecoveryMaterial, password string) ([]byte, error) {
	if err := recoveryPassword(password); err != nil {
		return nil, err
	}
	if _, err := m.Wallet(); err != nil {
		return nil, err
	}
	plain, err := json.Marshal(m)
	if err != nil {
		return nil, errors.New("could not encode wallet")
	}
	defer clear(plain)
	salt, nonce := make([]byte, 16), make([]byte, 12)
	if _, err = rand.Read(salt); err != nil {
		return nil, errors.New("wallet encryption entropy unavailable")
	}
	if _, err = rand.Read(nonce); err != nil {
		return nil, errors.New("wallet encryption entropy unavailable")
	}
	gcm, err := recoveryGCM(password, salt)
	if err != nil {
		return nil, err
	}
	v := recoveryEnvelope{Version: RecoveryVaultVersionV1, Profile: m.Profile, Address: m.Address,
		KDF: "PBKDF2-HMAC-SHA256", Iterations: recoveryIterations,
		Salt: base64.StdEncoding.EncodeToString(salt), Nonce: base64.StdEncoding.EncodeToString(nonce)}
	v.Ciphertext = base64.StdEncoding.EncodeToString(gcm.Seal(nil, nonce, plain, recoveryAAD(v)))
	return json.Marshal(v)
}

func decodeRecoveryJSON(raw []byte, target any) error {
	if len(raw) == 0 || len(raw) > 16384 {
		return errors.New("invalid encrypted wallet size")
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	if d.Decode(target) != nil || d.Decode(new(any)) != io.EOF {
		return errors.New("invalid encrypted wallet format")
	}
	return nil
}

func recoveryBytes(text string, size int) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(text)
	if err != nil || base64.StdEncoding.EncodeToString(b) != text || (size > 0 && len(b) != size) {
		return nil, errors.New("invalid encrypted wallet encoding")
	}
	return b, nil
}

func OpenRecoveryMaterial(raw []byte, password string) (*RecoveryMaterial, error) {
	if err := recoveryPassword(password); err != nil {
		return nil, err
	}
	var v recoveryEnvelope
	if err := decodeRecoveryJSON(raw, &v); err != nil {
		return nil, err
	}
	if v.Version != RecoveryVaultVersionV1 || v.Profile != MnemonicProfileV1 ||
		!ValidateAddress(v.Address) || v.KDF != "PBKDF2-HMAC-SHA256" || v.Iterations != recoveryIterations {
		return nil, errors.New("unsupported encrypted wallet format")
	}
	salt, err := recoveryBytes(v.Salt, 16)
	if err != nil {
		return nil, err
	}
	nonce, err := recoveryBytes(v.Nonce, 12)
	if err != nil {
		return nil, err
	}
	ciphertext, err := recoveryBytes(v.Ciphertext, 0)
	if err != nil || len(ciphertext) < 17 || len(ciphertext) > 8192 {
		return nil, errors.New("invalid wallet ciphertext")
	}
	gcm, err := recoveryGCM(password, salt)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, recoveryAAD(v))
	if err != nil {
		return nil, errors.New("wallet password or encrypted backup is incorrect")
	}
	defer clear(plain)
	var m RecoveryMaterial
	if err = decodeRecoveryJSON(plain, &m); err != nil {
		return nil, err
	}
	if m.Address != v.Address || m.Profile != v.Profile {
		return nil, errors.New("wallet identity does not match encrypted backup")
	}
	if _, err = m.Wallet(); err != nil {
		return nil, err
	}
	return &m, nil
}

// SaveRecoveryVaultNew Publish a complete new encrypted file without replacing any existing wallet.
func SaveRecoveryVaultNew(path string, raw []byte) error {
	if path == "" {
		return errors.New("wallet file path required")
	}
	var envelope recoveryEnvelope
	if err := decodeRecoveryJSON(raw, &envelope); err != nil {
		return err
	}
	if envelope.Version != RecoveryVaultVersionV1 || !ValidateAddress(envelope.Address) {
		return errors.New("invalid wallet envelope")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return errors.New("wallet directory unavailable")
	}
	f, err := os.CreateTemp(dir, ".qc-wallet-*")
	if err != nil {
		return errors.New("could not stage encrypted wallet")
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(raw); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return errors.New("could not persist encrypted wallet")
	}
	if err = os.Link(f.Name(), path); err != nil {
		return errors.New("wallet destination already exists or cannot be safely created")
	}
	return nil
}
