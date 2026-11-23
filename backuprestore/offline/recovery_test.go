package offline

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"quantumcoin/backuprestore"
	"quantumcoin/wallet"
	"strings"
	"testing"
	"time"
)

var testPassword = []byte("synthetic-test-password-only")
var testGenesis = strings.Repeat("a", 64)

func pythonForTest(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("QC_RECOVERY_TEST_PYTHON"); p != "" {
		return p
	}
	p, err := exec.LookPath("python")
	if err != nil {
		t.Fatal("Python required for real SQLite recovery tests", err)
	}
	return p
}
func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
func syntheticWallet(t *testing.T) ([]byte, string) {
	t.Helper()
	w := wallet.NewWallet()
	address := w.GetAddress()
	raw, err := json.Marshal(map[string]any{"wallets": map[string]string{address: w.ExportPrivateKeyHex()}, "default": address})
	if err != nil {
		t.Fatal(err)
	}
	return raw, address
}
func fixture(t *testing.T) (Engine, Plan, string) {
	t.Helper()
	dir := t.TempDir()
	e := Engine{Python: pythonForTest(t), Scratch: dir}
	raw, address := syntheticWallet(t)
	sealed, err := SealWallet(raw, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	custody, _ := json.Marshal(Custody{Custodians: []string{"test-custodian-reference"}, RecoveryProcedure: "Test-only recovery drill; no real keys", Addresses: []string{address}, LastDrillUTC: time.Now().UTC()})
	db := filepath.Join(dir, "source.db")
	if out, err := exec.Command(e.Python, "-I", "-c", "import sqlite3,sys; c=sqlite3.connect(sys.argv[1]); c.execute('create table ledger (amount integer)'); c.execute('insert into ledger values (1)'); c.commit(); c.close()", db).CombinedOutput(); err != nil {
		t.Fatal("synthetic SQLite creation", err, string(out))
	}
	p := Plan{Network: "testnet", GenesisHash: testGenesis, Offline: true, Sources: map[string]Source{}}
	for _, a := range backuprestore.DefaultRunbookV1().Artifacts {
		path := filepath.Join(dir, a.Name)
		format := "file"
		data := []byte("synthetic chain bytes")
		switch a.Kind {
		case backuprestore.ArtifactSQLiteDB:
			path = db
			format = "sqlite"
		case backuprestore.ArtifactJSONLedger:
			path = db
			format = "sqlite"
		case backuprestore.ArtifactJSONLAppendOnly:
			format = "jsonl"
			data = []byte("{\"synthetic\":true}\n")
		case backuprestore.ArtifactConfig:
			format = "json"
			data = []byte("{\"synthetic\":true}")
		case backuprestore.ArtifactWalletKey:
			format = "wallet-envelope"
			data = sealed
		}
		if a.Name == "wallet_custody_record" {
			data = custody
		}
		if path != db {
			mustWrite(t, path, data)
		}
		p.Sources[a.Name] = Source{Path: path, Format: format, RotationBoundary: "closed_segment"}
	}
	return e, p, dir
}
func makeSnapshot(t *testing.T, e Engine, p Plan, root, name string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if _, err := e.Snapshot(context.Background(), p, dir, testPassword); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSnapshotRehearsalRestoreRollbackAndWalletRecovery(t *testing.T) {
	e, p, root := fixture(t)
	ctx := context.Background()
	first := makeSnapshot(t, e, p, root, "snapshot-one")
	m, err := e.Verify(ctx, first, p.Network, p.GenesisHash, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	var dbHash string
	for _, entry := range m.Entries {
		if entry.Format == "sqlite" {
			if dbHash != "" && entry.SHA256 != dbHash {
				t.Fatal("logical roles captured different DB generations")
			}
			dbHash = entry.SHA256
		}
	}
	if err := e.DryRun(ctx, first, filepath.Join(root, "rehearsal"), p.Network, p.GenesisHash, testPassword); err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(root, "restore-workspace")
	one, err := e.Restore(ctx, first, workspace, p.Network, p.GenesisHash, testPassword, true)
	if err != nil {
		t.Fatal(err)
	}
	chain := p.Sources["quantumcoin_chain_file"].Path
	mustWrite(t, chain, []byte("second synthetic chain generation"))
	second := makeSnapshot(t, e, p, root, "snapshot-two")
	two, err := e.Restore(ctx, second, workspace, p.Network, p.GenesisHash, testPassword, true)
	if err != nil {
		t.Fatal(err)
	}
	if one == two {
		t.Fatal("restore overwrote an existing generation")
	}
	selected, err := e.Rollback(ctx, workspace, p.Network, p.GenesisHash, testPassword, true)
	if err != nil || selected != one {
		t.Fatal("rollback failed", err)
	}
	current, err := e.Current(ctx, workspace, p.Network, p.GenesisHash, testPassword)
	if err != nil || current != one {
		t.Fatal("journal selection failed", err)
	}
	if _, err := os.Stat(two); err != nil {
		t.Fatal("rollback deleted newer generation")
	}
	unlocked := filepath.Join(root, "recovered-wallet.json")
	if err := e.RecoverWallet(ctx, one, unlocked, p.Network, p.GenesisHash, testPassword); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(unlocked)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WalletAddresses(raw); err != nil {
		t.Fatal("unusable recovered wallet", err)
	}
	if err := e.RecoverWallet(ctx, one, unlocked, p.Network, p.GenesisHash, testPassword); err == nil {
		t.Fatal("existing wallet overwritten")
	}
}

func TestRecoveryRejectsCorruptionNetworkMismatchAndUnsafeMappings(t *testing.T) {
	e, p, root := fixture(t)
	ctx := context.Background()
	snapshot := makeSnapshot(t, e, p, root, "snapshot")
	for _, network := range []string{"mainnet", "local-release", ""} {
		if _, err := e.Verify(ctx, snapshot, network, p.GenesisHash, testPassword); err == nil {
			t.Fatal("wrong network accepted")
		}
	}
	if _, err := e.Verify(ctx, snapshot, p.Network, strings.Repeat("b", 64), testPassword); err == nil {
		t.Fatal("wrong genesis accepted")
	}
	if _, err := e.Verify(ctx, snapshot, p.Network, p.GenesisHash, []byte("wrong-test-password")); err == nil {
		t.Fatal("wrong wallet password accepted")
	}
	manifestPath := filepath.Join(snapshot, "manifest.json")
	original, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Manifest){
		func(m *Manifest) { m.Entries[0].File = "../source.db" },
		func(m *Manifest) { m.Entries = m.Entries[:len(m.Entries)-1] },
		func(m *Manifest) { m.Entries[0].Size++ },
		func(m *Manifest) { m.Entries[0].SHA256 = strings.Repeat("0", 64) },
		func(m *Manifest) { m.Entries[0] = m.Entries[1] },
		func(m *Manifest) { m.Entries[0].Kind = "unknown" },
	} {
		var m Manifest
		plain, key, err := openManifest(original, testPassword)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(plain, &m); err != nil {
			t.Fatal(err)
		}
		mutate(&m)
		raw, _ := json.Marshal(m)
		var header protectedManifest
		_ = json.Unmarshal(original, &header)
		b, err := sealManifest(raw, key, header.Salt)
		wipe(key)
		if err != nil {
			t.Fatal(err)
		}
		mustWrite(t, manifestPath, b)
		if _, err := e.Verify(ctx, snapshot, p.Network, p.GenesisHash, testPassword); err == nil {
			t.Fatal("damaged manifest accepted")
		}
	}
	mustWrite(t, manifestPath, original)
	mustWrite(t, filepath.Join(snapshot, "unexpected-plaintext-key"), []byte("synthetic test"))
	if _, err := e.Verify(ctx, snapshot, p.Network, p.GenesisHash, testPassword); err == nil {
		t.Fatal("unlisted file accepted")
	}
	if err := os.Remove(filepath.Join(snapshot, "unexpected-plaintext-key")); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(snapshot, "quantumcoin_chain_file.artifact"), []byte("tampered"))
	if _, err := e.Restore(ctx, snapshot, filepath.Join(root, "rejected-restore"), p.Network, p.GenesisHash, testPassword, true); err == nil {
		t.Fatal("corrupted restore accepted")
	}
	if _, err := os.Stat(filepath.Join(root, "rejected-restore")); !os.IsNotExist(err) {
		t.Fatal("restore wrote before verifying")
	}
	if _, err := e.Snapshot(ctx, p, filepath.Join(root, "snapshot"), testPassword); err == nil {
		t.Fatal("snapshot overwritten")
	}
	p.Offline = false
	if _, err := e.Snapshot(ctx, p, filepath.Join(root, "live-rejected"), testPassword); err == nil {
		t.Fatal("live snapshot accepted")
	}
}

func TestWalletEncryptionAuthenticationAndCustodyCoverage(t *testing.T) {
	raw, address := syntheticWallet(t)
	one, err := SealWallet(raw, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	two, err := SealWallet(raw, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(one, two) || bytes.Contains(one, raw) {
		t.Fatal("encryption did not randomize")
	}
	opened, err := OpenWallet(one, testPassword)
	if err != nil || !bytes.Equal(opened, raw) {
		t.Fatal("wallet recovery failed", err)
	}
	var env walletEnvelope
	if err := json.Unmarshal(one, &env); err != nil {
		t.Fatal(err)
	}
	env.Ciphertext[0] ^= 1
	bad, _ := json.Marshal(env)
	if _, err := OpenWallet(bad, testPassword); err == nil {
		t.Fatal("ciphertext tamper accepted")
	}
	if _, err := SealWallet(raw, []byte("short")); err == nil {
		t.Fatal("weak password accepted")
	}
	if _, err := OpenWallet(raw, testPassword); err == nil {
		t.Fatal("renamed plaintext accepted")
	}
	for _, data := range [][]byte{[]byte(`{"wallets":{}}`), []byte(`{"address":"` + address + `","privateKeyHex":"` + strings.Repeat("0", 64) + `"}`), append(append([]byte{}, raw...), []byte("{}")...)} {
		if _, err := WalletAddresses(data); err == nil {
			t.Fatal("invalid wallet accepted")
		}
	}
	c := Custody{Custodians: []string{"operator-reference"}, RecoveryProcedure: "offline test drill", Addresses: []string{address}, LastDrillUTC: time.Now().UTC()}
	b, _ := json.Marshal(c)
	if err := verifyCustody(b, one, testPassword); err != nil {
		t.Fatal(err)
	}
	c.Addresses = []string{"wrong-address"}
	b, _ = json.Marshal(c)
	if verifyCustody(b, one, testPassword) == nil {
		t.Fatal("unrecoverable custody accepted")
	}
	c.Addresses = []string{address}
	c.LastDrillUTC = time.Time{}
	b, _ = json.Marshal(c)
	if verifyCustody(b, one, testPassword) == nil {
		t.Fatal("untested custody accepted")
	}
}

func TestPartialGenerationLockAndBrokenJournalFailClosed(t *testing.T) {
	e, p, root := fixture(t)
	ctx := context.Background()
	snapshot := makeSnapshot(t, e, p, root, "snapshot")
	workspace := filepath.Join(root, "workspace")
	one, err := e.Restore(ctx, snapshot, workspace, p.Network, p.GenesisHash, testPassword, true)
	if err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(workspace, "generation-"+strings.Repeat("0", 32))
	if err := os.Mkdir(partial, 0700); err != nil {
		t.Fatal(err)
	}
	current, err := e.Current(ctx, workspace, p.Network, p.GenesisHash, testPassword)
	if err != nil || current != one {
		t.Fatal("partial generation selected", err)
	}
	unlock, err := lockWorkspace(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Rollback(ctx, workspace, p.Network, p.GenesisHash, testPassword, true); err == nil {
		t.Fatal("parallel mutation bypassed lock")
	}
	unlock()
	mustWrite(t, filepath.Join(workspace, "event-00000002.json"), []byte(`{"partial":`))
	if _, err := e.Current(ctx, workspace, p.Network, p.GenesisHash, testPassword); err == nil {
		t.Fatal("broken journal accepted")
	}
	if _, err := e.Rollback(ctx, workspace, p.Network, p.GenesisHash, testPassword, true); err == nil {
		t.Fatal("broken journal overwritten")
	}
	if _, err := os.Stat(one); err != nil {
		t.Fatal("valid generation lost")
	}
}

func TestSQLiteBackupIncludesCommittedWALPages(t *testing.T) {
	e := Engine{Python: pythonForTest(t)}
	root := t.TempDir()
	db := filepath.Join(root, "wal.db")
	cmd := exec.Command(e.Python, "-I", "-u", "-c", `import sqlite3,sys
c=sqlite3.connect(sys.argv[1]);c.execute('PRAGMA journal_mode=WAL');c.execute('PRAGMA wal_autocheckpoint=0');c.execute('create table rewards (amount integer)');c.execute('insert into rewards values (42)');c.commit();print('READY',flush=True);sys.stdin.readline();c.close()`, db)
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = input.Close(); _ = cmd.Wait() }()
	line, err := bufio.NewReader(output).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "READY" {
		t.Fatal("WAL fixture failed", err)
	}
	if st, err := os.Stat(db + "-wal"); err != nil || st.Size() == 0 {
		t.Fatal("fixture has no WAL pages")
	}
	backup := filepath.Join(root, "snapshot.db")
	if err := e.sqlite(context.Background(), "snapshot", db, backup); err != nil {
		t.Fatal(err)
	}
	check := exec.Command(e.Python, "-I", "-c", "import sqlite3,sys; c=sqlite3.connect(sys.argv[1]); assert c.execute('select amount from rewards').fetchall()==[(42,)]; assert c.execute('pragma integrity_check').fetchall()==[('ok',)]", backup)
	if err := check.Run(); err != nil {
		t.Fatal("WAL data absent from restored DB", err)
	}
	mustWrite(t, filepath.Join(root, "corrupt.db"), []byte("not a database"))
	if err := e.sqlite(context.Background(), "verify", filepath.Join(root, "corrupt.db"), ""); err == nil {
		t.Fatal("corrupt SQLite accepted")
	}
}
