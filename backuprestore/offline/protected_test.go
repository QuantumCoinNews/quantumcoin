package offline

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"quantumcoin/blockchain"
	"strings"
	"testing"
)

func TestProtectedSnapshotAuthenticatesAllArtifactsAndManifest(t *testing.T) {
	e, p, root := fixture(t)
	ctx := context.Background()
	snapshot := makeSnapshot(t, e, p, root, "protected")
	m, err := e.Verify(ctx, snapshot, p.Network, p.GenesisHash, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range m.Entries {
		stored, err := os.ReadFile(filepath.Join(snapshot, entry.File))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(stored, []byte("QCSEAL2\n")) {
			t.Fatal("unencrypted artifact", entry.Name)
		}
		if bytes.Contains(stored, []byte("synthetic")) || bytes.Contains(stored, []byte("custodians")) {
			t.Fatal("plaintext leak", entry.Name)
		}
	}
	manifestPath := filepath.Join(snapshot, "manifest.json")
	original, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(original, []byte("sha256")) || bytes.Contains(original, []byte("custody")) {
		t.Fatal("plaintext manifest metadata leaked")
	}
	var header protectedManifest
	_ = json.Unmarshal(original, &header)
	header.Ciphertext[0] ^= 1
	bad, _ := json.Marshal(header)
	mustWrite(t, manifestPath, bad)
	// An attacker can replace both the artifact and the manifest, but cannot
	// produce a valid MAC without independent custody of the passphrase.
	artifact := filepath.Join(snapshot, m.Entries[0].File)
	saved, _ := os.ReadFile(artifact)
	mustWrite(t, artifact, []byte("attacker replacement"))
	if _, err := e.Verify(ctx, snapshot, p.Network, p.GenesisHash, testPassword); err == nil {
		t.Fatal("artifact+manifest substitution accepted")
	}
	mustWrite(t, manifestPath, original)
	mustWrite(t, artifact, saved)
	for _, damaged := range [][]byte{saved[:len(saved)-1], append(append([]byte{}, saved...), 0), append(append([]byte{}, saved[:20]...), saved[40:]...)} {
		mustWrite(t, artifact, damaged)
		if _, err := e.Verify(ctx, snapshot, p.Network, p.GenesisHash, testPassword); err == nil {
			t.Fatal("truncated/reordered/appended ciphertext accepted")
		}
	}
	mustWrite(t, artifact, saved)
	second := makeSnapshot(t, e, p, root, "protected-second")
	other, _ := os.ReadFile(filepath.Join(second, m.Entries[0].File))
	mustWrite(t, artifact, other)
	if _, err := e.Verify(ctx, snapshot, p.Network, p.GenesisHash, testPassword); err == nil {
		t.Fatal("cross-snapshot artifact substitution accepted")
	}
	mustWrite(t, artifact, saved)
	plainDir := filepath.Join(root, "plain-must-not-pass")
	if _, err := e.Materialize(ctx, snapshot, plainDir, p.Network, p.GenesisHash, testPassword, true); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Verify(ctx, plainDir, p.Network, p.GenesisHash, testPassword); err == nil {
		t.Fatal("plaintext materialization treated as protected backup")
	}
	files, err := os.ReadDir(e.Scratch)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "qc-recovery-private-") {
			t.Fatal("private scratch was not cleaned")
		}
	}
}

func TestRecoveryMaterializedChainReplaysWithTrustedContext(t *testing.T) {
	e, p, root := fixture(t)
	ctx := context.Background()
	chain := blockchain.NewBlockchain(blockchain.CanonicalInitialMiningRewardV1, int(blockchain.QCTotalSupplyHardCap))
	if err := chain.BindRuntimeNetworkIdentityV1("testnet-local"); err != nil {
		t.Fatal(err)
	}
	p.GenesisHash = hex.EncodeToString(chain.BlockByIndexSnapshot(0).Hash)
	source := p.Sources["quantumcoin_chain_file"]
	source.Path = filepath.Join(root, "real-chain.dat")
	p.Sources["quantumcoin_chain_file"] = source
	if err := chain.SaveToFile(source.Path); err != nil {
		t.Fatal(err)
	}
	snapshot := makeSnapshot(t, e, p, root, "chain-backup")
	target := filepath.Join(root, "materialized")
	if _, err := e.Materialize(ctx, snapshot, target, p.Network, p.GenesisHash, testPassword, false); err == nil {
		t.Fatal("live materialization accepted")
	}
	if _, err := e.Materialize(ctx, snapshot, target, p.Network, p.GenesisHash, testPassword, true); err != nil {
		t.Fatal(err)
	}
	loaded, err := blockchain.LoadBlockchainFromFileWithLocalContextV1(filepath.Join(target, "quantumcoin_chain_file.artifact"), chain)
	if err != nil {
		t.Fatal("restored chain did not pass trusted replay", err)
	}
	if loaded.NetworkIdentitySnapshotV1().Network != "testnet" {
		t.Fatal("restored network changed")
	}
	if err := e.RecoverWallet(ctx, snapshot, filepath.Join(root, "restored-wallet.json"), p.Network, p.GenesisHash, testPassword); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotRequiresClosedJSONLSegmentAndProtectedScratch(t *testing.T) {
	e, p, root := fixture(t)
	s := p.Sources["admin_event_log"]
	s.RotationBoundary = ""
	p.Sources["admin_event_log"] = s
	if _, err := e.Snapshot(context.Background(), p, filepath.Join(root, "unrotated"), testPassword); err == nil {
		t.Fatal("open log segment accepted")
	}
	e.Scratch = ""
	if _, err := e.Snapshot(context.Background(), p, filepath.Join(root, "no-private-scratch"), testPassword); err == nil {
		t.Fatal("implicit plaintext scratch accepted")
	}
}

func TestSQLiteSnapshotWhileTransactionsAreCommitting(t *testing.T) {
	e := Engine{Python: pythonForTest(t)}
	root := t.TempDir()
	source := filepath.Join(root, "live.db")
	cmd := exec.Command(e.Python, "-I", "-u", "-c", `import sqlite3,threading,time,sys
path=sys.argv[1]
c=sqlite3.connect(path);c.execute('PRAGMA journal_mode=WAL');c.execute('create table balances (id integer primary key, value integer)');c.executemany('insert into balances values (?,?)',[(1,500),(2,500)]);c.commit();c.close()
stop=threading.Event();ready=threading.Event()
def write():
 db=sqlite3.connect(path)
 for i in range(100000):
  if stop.is_set(): break
  with db:
   db.execute('update balances set value=? where id=1',(i%1000,));time.sleep(.001);db.execute('update balances set value=? where id=2',(1000-i%1000,))
  if i==5: ready.set()
  time.sleep(.001)
 db.close()
t=threading.Thread(target=write);t.start();ready.wait(10);print('READY',flush=True);sys.stdin.readline();stop.set();t.join()`, source)
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = in.Close(); _ = cmd.Wait() }()
	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "READY" {
		t.Fatal("live SQLite fixture did not start", err)
	}
	target := filepath.Join(root, "snapshot.db")
	if err := e.sqlite(context.Background(), "snapshot", source, target); err != nil {
		t.Fatal(err)
	}
	check := exec.Command(e.Python, "-I", "-c", "import sqlite3,sys; c=sqlite3.connect(sys.argv[1]); assert c.execute('select sum(value) from balances').fetchone()[0]==1000; assert c.execute('pragma integrity_check').fetchall()==[('ok',)]", target)
	if err := check.Run(); err != nil {
		t.Fatal("snapshot contains a partial transaction", err)
	}
}
