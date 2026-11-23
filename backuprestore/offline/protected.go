package offline

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"quantumcoin/backuprestore"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const protectedVersion = "QC_AUTHENTICATED_ENCRYPTED_BACKUP_V2"
const chunkSize = 1 << 20

type protectedManifest struct {
	Version    string `json:"version"`
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
	MAC        []byte `json:"mac"`
}

// Keys are domain-separated from wallet envelopes and never stored in a backup.
// Custody of the passphrase outside backup storage is the trust anchor.
func backupKey(password, salt []byte) ([]byte, error) {
	if len(password) < 12 || len(salt) != 32 {
		return nil, errors.New("backup custody password/salt rejected")
	}
	return pbkdf2.Key(password, append([]byte(protectedVersion), salt...), 600000, 32, sha256.New), nil
}
func subkey(master []byte, purpose string) []byte {
	h := hmac.New(sha256.New, master)
	_, _ = h.Write([]byte(purpose))
	return h.Sum(nil)
}
func manifestMAC(master []byte, p protectedManifest) []byte {
	k := subkey(master, "manifest-hmac")
	defer wipe(k)
	h := hmac.New(sha256.New, k)
	for _, b := range [][]byte{[]byte(p.Version), p.Salt, p.Nonce, p.Ciphertext} {
		_, _ = h.Write(b)
	}
	return h.Sum(nil)
}
func backupAEAD(master []byte, purpose string) (cipher.AEAD, error) {
	k := subkey(master, purpose)
	defer wipe(k)
	b, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(b)
}
func sealManifest(raw, master, salt []byte) ([]byte, error) {
	g, err := backupAEAD(master, "manifest-encryption")
	if err != nil {
		return nil, err
	}
	p := protectedManifest{Version: protectedVersion, Salt: salt, Nonce: make([]byte, 12)}
	if _, err := rand.Read(p.Nonce); err != nil {
		return nil, err
	}
	p.Ciphertext = g.Seal(nil, p.Nonce, raw, []byte(p.Version))
	p.MAC = manifestMAC(master, p)
	return json.MarshalIndent(p, "", "  ")
}
func openManifest(raw, password []byte) ([]byte, []byte, error) {
	var p protectedManifest
	if err := strictJSON(raw, &p); err != nil {
		return nil, nil, err
	}
	if p.Version != protectedVersion || len(p.Nonce) != 12 || len(p.MAC) != 32 || len(p.Ciphertext) > 2<<20 {
		return nil, nil, errors.New("authenticated V2 snapshot required; plaintext V1 is not accepted")
	}
	key, err := backupKey(password, p.Salt)
	if err != nil {
		return nil, nil, err
	}
	if !hmac.Equal(p.MAC, manifestMAC(key, p)) {
		wipe(key)
		return nil, nil, errors.New("backup manifest authentication failed")
	}
	g, err := backupAEAD(key, "manifest-encryption")
	if err != nil {
		wipe(key)
		return nil, nil, err
	}
	plain, err := g.Open(nil, p.Nonce, p.Ciphertext, []byte(p.Version))
	if err != nil {
		wipe(key)
		return nil, nil, errors.New("backup manifest decryption failed")
	}
	return plain, key, nil
}

// Each artifact has its own derived key. Counter nonces are unique under that
// key; a new random snapshot salt creates new keys for every snapshot. Frames
// authenticate name, index and length; an authenticated EOF prevents truncation.
func cryptArtifact(ctx context.Context, source, target, name string, master []byte, encrypt bool) error {
	if err := noLinks(source); err != nil {
		return err
	}
	if err := noLinks(target); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer out.Close()
	g, err := backupAEAD(master, "artifact:"+name)
	if err != nil {
		return err
	}
	magic := []byte("QCSEAL2\n")
	if encrypt {
		if _, err = out.Write(magic); err != nil {
			return err
		}
	} else {
		got := make([]byte, len(magic))
		if _, err := io.ReadFull(in, got); err != nil || !bytes.Equal(got, magic) {
			return errors.New("encrypted artifact header rejected")
		}
	}
	var total int64
	for counter := uint64(0); ; counter++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		nonce := make([]byte, 12)
		binary.BigEndian.PutUint64(nonce[4:], counter)
		var length [4]byte
		var body []byte
		terminal := false
		if encrypt {
			plain := make([]byte, chunkSize)
			n, readErr := io.ReadFull(in, plain)
			if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
				return readErr
			}
			plain = plain[:n]
			terminal = n == 0
			binary.BigEndian.PutUint32(length[:], uint32(n))
			aad := append(append([]byte(name), nonce...), length[:]...)
			body = g.Seal(nil, nonce, plain, aad)
			wipe(plain)
			if _, err := out.Write(length[:]); err != nil {
				return err
			}
			if _, err := out.Write(body); err != nil {
				return err
			}
			total += int64(n)
		} else {
			if _, err := io.ReadFull(in, length[:]); err != nil {
				return errors.New("truncated encrypted artifact")
			}
			n := binary.BigEndian.Uint32(length[:])
			if n > chunkSize {
				return errors.New("encrypted frame exceeds limit")
			}
			terminal = n == 0
			body = make([]byte, int(n)+g.Overhead())
			if _, err := io.ReadFull(in, body); err != nil {
				return errors.New("truncated encrypted artifact")
			}
			aad := append(append([]byte(name), nonce...), length[:]...)
			plain, err := g.Open(nil, nonce, body, aad)
			if err != nil {
				return errors.New("artifact authentication failed")
			}
			_, err = out.Write(plain)
			total += int64(len(plain))
			wipe(plain)
			if err != nil {
				return err
			}
		}
		if total > maxArtifactBytes {
			return errors.New("decrypted artifact exceeds size limit")
		}
		if terminal {
			if !encrypt {
				extra := make([]byte, 1)
				if n, err := in.Read(extra); n != 0 || err != io.EOF {
					return errors.New("trailing encrypted artifact data")
				}
			}
			break
		}
	}
	return out.Sync()
}

// Scratch must reside on an operator-controlled encrypted volume. Temporary
// plaintext is needed by SQLite integrity/recovery tools; files are cleaned on
// ordinary completion/failure, never advertised as securely erased after crash.
func (e Engine) scratch() (string, error) {
	if e.Scratch == "" {
		return "", errors.New("explicit protected --scratch directory required")
	}
	if err := noLinks(e.Scratch); err != nil {
		return "", err
	}
	st, err := os.Stat(e.Scratch)
	if err != nil || !st.IsDir() {
		return "", errors.New("protected scratch directory unavailable")
	}
	return os.MkdirTemp(e.Scratch, "qc-recovery-private-*")
}

func (e Engine) Snapshot(ctx context.Context, p Plan, destination string, password []byte) (Manifest, error) {
	var m Manifest
	if err := noLinks(destination); err != nil {
		return m, err
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		return m, errors.New("new snapshot destination required")
	}
	stage, err := e.scratch()
	if err != nil {
		return m, err
	}
	defer e.cleanupScratch(stage)
	plain := filepath.Join(stage, "capture")
	m, err = e.capturePlain(ctx, p, plain, password)
	if err != nil {
		return m, err
	}
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return m, err
	}
	key, err := backupKey(password, salt)
	if err != nil {
		return m, err
	}
	defer wipe(key)
	if err := os.Mkdir(destination, 0700); err != nil {
		return m, err
	}
	for _, entry := range m.Entries {
		if err := cryptArtifact(ctx, filepath.Join(plain, entry.File), filepath.Join(destination, entry.File), entry.Name, key, true); err != nil {
			return m, err
		}
	}
	raw, err := readSmall(filepath.Join(plain, "manifest.json"), 1<<20)
	if err != nil {
		return m, err
	}
	sealed, err := sealManifest(raw, key, salt)
	if err != nil {
		return m, err
	}
	if err := writeNew(filepath.Join(destination, "manifest.json"), sealed); err != nil {
		return m, err
	}
	return m, nil
}

func (e Engine) decryptSnapshot(ctx context.Context, dir, target, network, genesis string, password []byte) (Manifest, error) {
	var m Manifest
	if err := identity(network, genesis); err != nil {
		return m, err
	}
	raw, err := readSmall(filepath.Join(dir, "manifest.json"), 3<<20)
	if err != nil {
		return m, err
	}
	plain, key, err := openManifest(raw, password)
	if err != nil {
		return m, err
	}
	defer wipe(key)
	defer wipe(plain)
	if err := strictJSON(plain, &m); err != nil {
		return m, err
	}
	if m.Network != network || m.GenesisHash != genesis || m.Version != manifestVersion {
		return m, errors.New("backup network/genesis mismatch")
	}
	// Authenticate first, validate all paths/roles before decrypting any artifacts.
	if err := validateManifestMapping(m); err != nil {
		return m, err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return m, err
	}
	if len(files) != len(m.Entries)+1 {
		return m, errors.New("unexpected snapshot files")
	}
	for _, entry := range m.Entries {
		if err := cryptArtifact(ctx, filepath.Join(dir, entry.File), filepath.Join(target, entry.File), entry.Name, key, false); err != nil {
			return m, fmt.Errorf("decrypt %s: %w", entry.Name, err)
		}
	}
	if err := writeNew(filepath.Join(target, "manifest.json"), plain); err != nil {
		return m, err
	}
	return e.verifyPlain(ctx, target, network, genesis, password)
}

func (e Engine) Verify(ctx context.Context, dir, network, genesis string, password []byte) (Manifest, error) {
	stage, err := e.scratch()
	if err != nil {
		return Manifest{}, err
	}
	defer e.cleanupScratch(stage)
	return e.decryptSnapshot(ctx, dir, stage, network, genesis, password)
}

func (e Engine) RecoverWallet(ctx context.Context, dir, target, network, genesis string, password []byte) error {
	stage, err := e.scratch()
	if err != nil {
		return err
	}
	defer e.cleanupScratch(stage)
	if _, err := e.decryptSnapshot(ctx, dir, stage, network, genesis, password); err != nil {
		return err
	}
	return UnsealWalletFile(filepath.Join(stage, "wallet_encrypted_backup.artifact"), target, password)
}

// Materialize creates verified plaintext working files only as an explicit
// offline operation. Target storage must be private and encrypted at rest.
// Inner wallet custody encryption is retained; RecoverWallet is a separate step.
func (e Engine) Materialize(ctx context.Context, dir, target, network, genesis string, password []byte, offline bool) (Manifest, error) {
	if !offline {
		return Manifest{}, errors.New("offline materialization required")
	}
	if _, err := e.Verify(ctx, dir, network, genesis, password); err != nil {
		return Manifest{}, err
	}
	if err := noLinks(target); err != nil {
		return Manifest{}, err
	}
	if err := os.Mkdir(target, 0700); err != nil {
		return Manifest{}, err
	}
	return e.decryptSnapshot(ctx, dir, target, network, genesis, password)
}

func (e Engine) cleanupScratch(stage string) {
	base, err := filepath.Abs(e.Scratch)
	if err != nil {
		return
	}
	target, err := filepath.Abs(stage)
	if err != nil {
		return
	}
	if filepath.Dir(target) != base || !strings.HasPrefix(filepath.Base(target), "qc-recovery-private-") {
		return
	}
	_ = os.RemoveAll(target)
}
func captureBoundary(kind backuprestore.ArtifactKind, format string) string {
	if kind == backuprestore.ArtifactChainFile {
		return "committed_file_under_writer_lock"
	}
	if format == "sqlite" {
		return "sqlite_backup_transaction"
	}
	if format == "jsonl" {
		return "closed_segment"
	}
	return "quiesced_stable_file"
}
func validateManifestMapping(m Manifest) error {
	if err := backuprestore.ValidateRunbookV1(m.Runbook); err != nil {
		return err
	}
	expected := backuprestore.DefaultRunbookV1().Artifacts
	if len(m.Entries) != len(expected) || len(m.Runbook.Artifacts) != len(expected) || m.CreatedUTC.IsZero() {
		return errors.New("incomplete protected manifest")
	}
	seen := map[string]bool{}
	for _, entry := range m.Entries {
		found := false
		for _, a := range expected {
			if a.Name == entry.Name && entry.Kind == a.Kind && formatAllowed(a, entry.Format) && entry.CaptureBoundary == captureBoundary(a.Kind, entry.Format) {
				found = true
				break
			}
		}
		if !found || seen[entry.Name] || entry.File != entry.Name+".artifact" || entry.Size < 0 || entry.Size > maxArtifactBytes || len(entry.SHA256) != 64 {
			return errors.New("invalid protected artifact mapping")
		}
		seen[entry.Name] = true
	}
	return nil
}
