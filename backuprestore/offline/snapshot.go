package offline

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"quantumcoin/backuprestore"
	"quantumcoin/blockchain"
)

const manifestVersion = "QC_OFFLINE_SNAPSHOT_V2"
const maxArtifactBytes int64 = 32 << 30
const maxStoredArtifactBytes = maxArtifactBytes + (maxArtifactBytes/chunkSize+2)*20 + 8

type Source struct {
	Path string `json:"path"`
	// sqlite permits logical TGWT/NFT ledger roles to share one canonical DB.
	Format           string `json:"format"`
	RotationBoundary string `json:"rotationBoundary,omitempty"`
}
type Plan struct {
	Network     string            `json:"network"`
	GenesisHash string            `json:"genesisHash"`
	Offline     bool              `json:"offline"`
	Sources     map[string]Source `json:"sources"`
}
type Entry struct {
	Name            string                     `json:"name"`
	File            string                     `json:"file"`
	Kind            backuprestore.ArtifactKind `json:"kind"`
	Format          string                     `json:"format"`
	Size            int64                      `json:"size"`
	SHA256          string                     `json:"sha256"`
	CaptureBoundary string                     `json:"captureBoundary"`
}
type Manifest struct {
	Version     string                `json:"version"`
	Network     string                `json:"network"`
	GenesisHash string                `json:"genesisHash"`
	CreatedUTC  time.Time             `json:"createdUtc"`
	Runbook     backuprestore.Runbook `json:"runbook"`
	Entries     []Entry               `json:"entries"`
}
type Engine struct {
	Python  string
	Scratch string
}

func identity(network, genesis string) error {
	if network != "local-release" && network != "testnet" && network != "mainnet" {
		return errors.New("explicit recovery network required")
	}
	b, err := hex.DecodeString(genesis)
	if err != nil || len(b) != 32 || genesis != strings.ToLower(genesis) {
		return errors.New("canonical genesis SHA256 required")
	}
	return nil
}

// noLinks rejects every existing link/reparse component (including ancestors).
// Recovery directories must additionally be private to the operator: this does
// not claim protection from another process with the same account privileges.
func noLinks(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for {
		st, err := os.Lstat(abs)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && st.Mode()&os.ModeSymlink != 0 {
			return errors.New("recovery paths must not contain links")
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			break
		}
		abs = parent
	}
	return nil
}

func readSmall(path string, limit int64) ([]byte, error) {
	if err := noLinks(path); err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() > limit {
		return nil, errors.New("invalid recovery file size or type")
	}
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if int64(len(b)) > limit {
		return nil, errors.New("recovery file too large")
	}
	return b, err
}
func writeNew(path string, data []byte) error {
	if err := noLinks(path); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func digest(path string) (int64, string, error) {
	if err := noLinks(path); err != nil {
		return 0, "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() > maxStoredArtifactBytes {
		return 0, "", errors.New("artifact size/type rejected")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, maxStoredArtifactBytes+1))
	if n > maxStoredArtifactBytes {
		return 0, "", errors.New("artifact exceeds size limit")
	}
	return n, hex.EncodeToString(h.Sum(nil)), err
}
func copyNew(source, target string) error {
	n, hash, err := digest(source)
	if err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	h := sha256.New()
	written, err := io.Copy(io.MultiWriter(out, h), io.LimitReader(in, maxStoredArtifactBytes+1))
	if err == nil {
		err = out.Sync()
	}
	cerr := out.Close()
	if err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	now, nowHash, err := digest(source)
	if err != nil {
		return err
	}
	if written != n || now != n || nowHash != hash || hex.EncodeToString(h.Sum(nil)) != hash {
		return errors.New("source changed during offline copy")
	}
	return nil
}

func formatAllowed(a backuprestore.BackupArtifact, format string) bool {
	switch a.Kind {
	case backuprestore.ArtifactSQLiteDB:
		return format == "sqlite"
	case backuprestore.ArtifactJSONLedger:
		return format == "json" || format == "sqlite"
	case backuprestore.ArtifactWalletKey:
		return format == "wallet-envelope"
	case backuprestore.ArtifactConfig:
		return format == "json"
	case backuprestore.ArtifactJSONLAppendOnly:
		return format == "jsonl" || format == "json"
	case backuprestore.ArtifactChainFile:
		return format == "file"
	}
	return false
}

// Snapshot accepts an explicit offline declaration; it never freezes or stops a
// running node. Quiesce all writers externally for cross-file consistency.
// Failed directories remain preserved without a complete manifest.
func (e Engine) capturePlain(ctx context.Context, p Plan, destination string, password []byte) (Manifest, error) {
	var m Manifest
	if !p.Offline {
		return m, errors.New("offline/quiesced sources required")
	}
	if err := identity(p.Network, p.GenesisHash); err != nil {
		return m, err
	}
	r := backuprestore.DefaultRunbookV1()
	if err := backuprestore.ValidateRunbookV1(r); err != nil {
		return m, err
	}
	if len(p.Sources) != len(r.Artifacts) {
		return m, errors.New("exact required artifact source mapping required")
	}
	if err := noLinks(destination); err != nil {
		return m, err
	}
	// Preflight before producing any output. Source paths never enter manifests.
	for _, a := range r.Artifacts {
		s, ok := p.Sources[a.Name]
		if !ok || !formatAllowed(a, s.Format) {
			return m, fmt.Errorf("source mapping/format rejected: %s", a.Name)
		}
		if s.Format == "jsonl" && s.RotationBoundary != "closed_segment" {
			return m, errors.New("JSONL requires an explicitly closed rotation segment")
		}
		if n, _, err := digest(s.Path); err != nil || n > maxArtifactBytes {
			return m, fmt.Errorf("source unavailable: %s", a.Name)
		}
	}
	if err := os.Mkdir(destination, 0700); err != nil {
		return m, err
	}
	m = Manifest{Version: manifestVersion, Network: p.Network, GenesisHash: p.GenesisHash, CreatedUTC: time.Now().UTC(), Runbook: r}
	cache := map[string]string{}
	for _, a := range r.Artifacts {
		if err := ctx.Err(); err != nil {
			return m, err
		}
		s := p.Sources[a.Name]
		name := a.Name + ".artifact"
		target := filepath.Join(destination, name)
		abs, _ := filepath.Abs(s.Path)
		cacheKey := filepath.Clean(abs) + "|" + s.Format
		var err error
		if prior, ok := cache[cacheKey]; ok {
			err = copyNew(prior, target)
		} else if a.Kind == backuprestore.ArtifactChainFile {
			err = blockchain.SnapshotPersistedChainFileV1(s.Path, target)
		} else if s.Format == "sqlite" {
			err = e.sqlite(ctx, "snapshot", s.Path, target)
		} else {
			err = copyNew(s.Path, target)
		}
		if err != nil {
			return m, fmt.Errorf("capture failed for %s: %w", a.Name, err)
		}
		cache[cacheKey] = target
		n, h, err := digest(target)
		if err != nil {
			return m, err
		}
		entry := Entry{Name: a.Name, File: name, Kind: a.Kind, Format: s.Format, Size: n, SHA256: h, CaptureBoundary: captureBoundary(a.Kind, s.Format)}
		if err := e.verifyContent(ctx, target, entry); err != nil {
			return m, err
		}
		m.Entries = append(m.Entries, entry)
	}
	if err := checkWallet(destination, password); err != nil {
		return m, err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return m, err
	}
	if err := writeNew(filepath.Join(destination, "manifest.json"), data); err != nil {
		return m, err
	}
	return m, nil
}

func checkWallet(dir string, password []byte) error {
	sealed, err := readSmall(filepath.Join(dir, "wallet_encrypted_backup.artifact"), 2*maxWalletBytes)
	if err != nil {
		return err
	}
	custody, err := readSmall(filepath.Join(dir, "wallet_custody_record.artifact"), 1<<20)
	if err != nil {
		return err
	}
	return verifyCustody(custody, sealed, password)
}

// Verify pins the expected network and genesis supplied independently by the
// operator. Hashes detect corruption, not authenticity; preserve the manifest
// SHA256 out of band. It also decrypts and validates wallet address coverage.
func (e Engine) verifyPlain(ctx context.Context, dir, network, genesis string, password []byte) (Manifest, error) {
	var m Manifest
	if err := identity(network, genesis); err != nil {
		return m, err
	}
	data, err := readSmall(filepath.Join(dir, "manifest.json"), 1<<20)
	if err != nil {
		return m, err
	}
	if err := strictJSON(data, &m); err != nil {
		return m, err
	}
	if m.Version != manifestVersion || m.Network != network || m.GenesisHash != genesis || m.CreatedUTC.IsZero() {
		return m, errors.New("snapshot identity/version mismatch")
	}
	if err := backuprestore.ValidateRunbookV1(m.Runbook); err != nil {
		return m, err
	}
	expected := backuprestore.DefaultRunbookV1().Artifacts
	if len(m.Entries) != len(expected) || len(m.Runbook.Artifacts) != len(expected) {
		return m, errors.New("incomplete snapshot")
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return m, err
	}
	if len(files) != len(expected)+1 {
		return m, errors.New("unexpected files in snapshot")
	}
	allowedFiles := map[string]bool{"manifest.json": true}
	for _, a := range expected {
		allowedFiles[a.Name+".artifact"] = true
	}
	for _, file := range files {
		if file.IsDir() || !allowedFiles[file.Name()] {
			return m, errors.New("unexpected snapshot file")
		}
	}
	seen := map[string]bool{}
	for _, entry := range m.Entries {
		var a backuprestore.BackupArtifact
		found := false
		for _, candidate := range expected {
			if candidate.Name == entry.Name {
				a = candidate
				found = true
				break
			}
		}
		if !found || seen[entry.Name] || entry.File != entry.Name+".artifact" || entry.Kind != a.Kind || !formatAllowed(a, entry.Format) {
			return m, errors.New("invalid manifest artifact mapping")
		}
		seen[entry.Name] = true
		path := filepath.Join(dir, entry.File)
		n, h, err := digest(path)
		if err != nil {
			return m, err
		}
		if n != entry.Size || h != entry.SHA256 {
			return m, errors.New("artifact hash/size mismatch: " + entry.Name)
		}
		if err := e.verifyContent(ctx, path, entry); err != nil {
			return m, err
		}
	}
	if err := checkWallet(dir, password); err != nil {
		return m, err
	}
	return m, nil
}

func (e Engine) verifyContent(ctx context.Context, path string, entry Entry) error {
	if entry.Kind == backuprestore.ArtifactChainFile && entry.Size == 0 {
		return errors.New("chain artifact must not be empty")
	}
	if entry.Format == "sqlite" {
		return e.sqlite(ctx, "verify", path, "")
	}
	if entry.Format == "json" || entry.Format == "wallet-envelope" {
		b, err := readSmall(path, 16<<20)
		if err != nil {
			return err
		}
		if !json.Valid(b) {
			return errors.New("invalid JSON artifact: " + entry.Name)
		}
	}
	if entry.Format == "jsonl" {
		b, err := readSmall(path, 16<<20)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(b), "\n") {
			if strings.TrimSpace(line) != "" && !json.Valid([]byte(line)) {
				return errors.New("invalid JSONL artifact")
			}
		}
	}
	return nil
}

const sqliteScript = `import sqlite3,sys,pathlib
action,source,target=sys.argv[1:]
uri=pathlib.Path(source).resolve().as_uri()+'?mode=ro'
with sqlite3.connect(uri,uri=True,timeout=5) as src:
 if src.execute('PRAGMA integrity_check').fetchall()!=[('ok',)]: raise RuntimeError('SQLite integrity failed')
 if action=='snapshot':
  with sqlite3.connect(target) as dst:
   src.backup(dst,pages=256)
   if dst.execute('PRAGMA integrity_check').fetchall()!=[('ok',)]: raise RuntimeError('SQLite backup integrity failed')
`

func (e Engine) sqlite(ctx context.Context, action, source, target string) error {
	if e.Python == "" {
		return errors.New("explicit Python 3 executable required for SQLite backup/verification")
	}
	if err := noLinks(source); err != nil {
		return err
	}
	if action == "snapshot" {
		if err := writeNew(target, nil); err != nil {
			return err
		}
	}
	bounded, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(bounded, e.Python, "-I", "-c", sqliteScript, action, source, target)
	// Do not forward DB contents, command environment or provider output to logs.
	if err := cmd.Run(); err != nil {
		return errors.New("SQLite backup/integrity check failed")
	}
	return nil
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
