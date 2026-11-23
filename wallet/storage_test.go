package wallet

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"quantumcoin/config"
)

func configuredWalletPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wallets.json")
	previous := *config.Current()
	next := previous
	next.WalletFile = path
	config.Set(&next)
	t.Cleanup(func() { config.Set(&previous) })
	return path
}
func storeFixture(t *testing.T, wallets ...*Wallet) []byte {
	t.Helper()
	st := diskStore{Wallets: map[string]string{}}
	for _, w := range wallets {
		st.Wallets[w.GetAddress()] = w.ExportPrivateKeyHex()
	}
	raw, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestCorruptWalletStoreNeverReplaced(t *testing.T) {
	path := configuredWalletPath(t)
	one, two := NewWallet(), NewWallet()
	cases := []string{"", `{`, "null", `{}`, `{"wallets":null}`, `{"wallets":{},"unknown":1}`, `{"wallets":{},"Wallets":{}}`,
		`{"wallets":{},"wallets":{}}`,
		`{"wallets":{"` + one.GetAddress() + `":"` + one.ExportPrivateKeyHex() + `","` + one.GetAddress() + `":"` + two.ExportPrivateKeyHex() + `"}}`,
		`{"wallets":{"` + one.GetAddress() + `":"` + two.ExportPrivateKeyHex() + `"}}`,
		`{"wallets":{"` + one.GetAddress() + `":"bad-key"}}`,
		`{"wallets":{},"default":"` + one.GetAddress() + `"}`,
		string(storeFixture(t, one)) + ` {}`,
	}
	for i, raw := range cases {
		// Labels are numeric: never print private fixture values in test failures.
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if w, err := LoadWalletFromFileWithError(); w != nil || !errors.Is(err, ErrWalletStoreCorrupt) {
			t.Fatalf("case %d: corrupt store accepted", i)
		}
		if LoadWalletFromFile() != nil || LoadWallet("") != nil {
			t.Fatalf("case %d: compatibility loader created a replacement", i)
		}
		if err := SaveWalletAt(path, two); err == nil {
			t.Fatalf("case %d: corrupt store overwritten", i)
		}
		if err := SetDefaultWallet(one.GetAddress()); err == nil {
			t.Fatalf("case %d: corrupt default overwritten", i)
		}
		after, _ := os.ReadFile(path)
		if string(after) != raw {
			t.Fatalf("case %d: original data changed", i)
		}
	}
}

func TestMissingWalletRequiresExplicitCreationAndLoadIsReadOnly(t *testing.T) {
	path := configuredWalletPath(t)
	if w, err := LoadWalletFromFileWithError(); w != nil || !errors.Is(err, ErrWalletNotFound) {
		t.Fatal("missing wallet did not request explicit recovery/create")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("load created a wallet file")
	}
	one, two := NewWallet(), NewWallet()
	if err := SaveWalletAt(path, one); err != nil {
		t.Fatal(err)
	}
	if err := SaveWalletAt(path, two); err != nil {
		t.Fatal(err)
	}
	if err := SetDefaultWallet(two.GetAddress()); err != nil {
		t.Fatal(err)
	}
	restored, err := LoadWalletFromFileWithError()
	if err != nil || restored.GetAddress() != two.GetAddress() {
		t.Fatal("explicit default not loaded")
	}
	// An old store without a default is selected deterministically, without a write.
	raw := storeFixture(t, one, two)
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	addresses := []string{one.GetAddress(), two.GetAddress()}
	sort.Strings(addresses)
	for i := 0; i < 5; i++ {
		w, err := LoadStoredWalletAt(path)
		if err != nil || w.GetAddress() != addresses[0] {
			t.Fatal("unstable default selection")
		}
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(raw, after) {
		t.Fatal("read silently rewrote the default")
	}
	st, _ := readStoreAt(path)
	if len(st.Wallets) != 2 {
		t.Fatal("existing wallet lost")
	}
}

func TestDefaultProfileCorruptionAndIdentityArePreserved(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	dir, _ := appDataDir()
	path := filepath.Join(dir, "wallet.json")
	if w, err := EnsureDefaultWallet(); w != nil || !errors.Is(err, ErrWalletNotFound) {
		t.Fatal("ensure generated an implicit wallet")
	}
	one, two := NewWallet(), NewWallet()
	first := &DefaultWallet{Address: one.GetAddress(), PrivateKeyHex: one.ExportPrivateKeyHex()}
	if err := SaveDefaultWallet(first); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(path)
	if err := SaveDefaultWallet(&DefaultWallet{Address: two.GetAddress(), PrivateKeyHex: two.ExportPrivateKeyHex()}); !errors.Is(err, ErrWalletWouldReplace) {
		t.Fatal("different wallet replaced old key")
	}
	if err := SaveDefaultWallet(&DefaultWallet{Address: one.GetAddress()}); err != nil {
		t.Fatal(err)
	}
	got, err := LoadDefaultWallet()
	if err != nil || got.PrivateKeyHex != first.PrivateKeyHex {
		t.Fatal("watch-only update erased private key")
	}
	for _, raw := range []string{"", `{`, "null", `{"address":"invalid"}`, `{"address":"` + one.GetAddress() + `","privateKeyHex":"` + two.ExportPrivateKeyHex() + `"}`, `{"address":"` + one.GetAddress() + `","address":"` + two.GetAddress() + `"}`} {
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if w, err := EnsureDefaultWallet(); w != nil || err == nil {
			t.Fatal("corrupt default accepted")
		}
		if err := SaveDefaultWallet(first); err == nil {
			t.Fatal("corrupt default overwritten")
		}
		after, _ := os.ReadFile(path)
		if string(after) != raw {
			t.Fatal("corrupt file changed")
		}
	}
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	got, err = EnsureDefaultWallet()
	if err != nil || got.Address != first.Address {
		t.Fatal("old healthy profile no longer readable")
	}
}

type failingWalletFile struct {
	*os.File
	failure string
}

func (f failingWalletFile) Write(p []byte) (int, error) {
	if f.failure == "write" {
		n, _ := f.File.Write(p[:len(p)/2])
		return n, io.ErrUnexpectedEOF
	}
	if f.failure == "short" {
		return len(p) / 2, nil
	}
	return f.File.Write(p)
}
func (f failingWalletFile) Sync() error {
	if f.failure == "sync" {
		return io.ErrUnexpectedEOF
	}
	return f.File.Sync()
}
func (f failingWalletFile) Close() error {
	err := f.File.Close()
	if f.failure == "close" {
		return io.ErrUnexpectedEOF
	}
	return err
}

func TestAtomicWalletCommitFailurePreservesPreviousFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wallet.json")
	previous := storeFixture(t, NewWallet())
	next := storeFixture(t, NewWallet())
	for _, phase := range []string{"create", "write", "short", "sync", "close", "replace"} {
		t.Run(phase, func(t *testing.T) {
			if err := os.WriteFile(path, previous, 0600); err != nil {
				t.Fatal(err)
			}
			replaced := false
			ops := walletFileOps{create: func(dir string) (walletTemporaryFile, error) {
				if phase == "create" {
					return nil, io.ErrUnexpectedEOF
				}
				f, err := os.CreateTemp(dir, ".qc-wallet-store-*")
				if err != nil {
					return nil, err
				}
				return failingWalletFile{f, phase}, nil
			}, replace: func(from, to string) error { replaced = true; return io.ErrUnexpectedEOF }}
			if err := commitWalletFile(path, next, ops); err == nil {
				t.Fatal("failed write reported success")
			}
			if replaced != (phase == "replace") {
				t.Fatal("replacement ran before write/sync/close completed")
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(previous, after) {
				t.Fatal("previous keys changed after a failed commit")
			}
			leftovers, _ := filepath.Glob(filepath.Join(dir, ".qc-wallet-store-*"))
			if len(leftovers) != 0 {
				t.Fatal("owned temporary secret file left behind")
			}
		})
	}
}

func TestConcurrentWalletWritersAndCompleteReaders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallets.json")
	if err := SaveWalletAt(path, NewWallet()); err != nil {
		t.Fatal(err)
	}
	const writers = 12
	var wg sync.WaitGroup
	errs := make(chan error, writers+1)
	done := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				if _, err := readStoreAt(path); err != nil {
					errs <- err
					return
				}
			}
		}
	}()
	var writes sync.WaitGroup
	for i := 0; i < writers; i++ {
		writes.Add(1)
		go func() {
			defer writes.Done()
			if err := SaveWalletAt(path, NewWallet()); err != nil {
				errs <- err
			}
		}()
	}
	writes.Wait()
	close(done)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	st, err := readStoreAt(path)
	if err != nil || len(st.Wallets) != writers+1 {
		t.Fatal("concurrent update lost an existing wallet")
	}
}

func walletStorageChild(t *testing.T, path, mode string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestWalletStorageProcess$")
	cmd.Env = append(os.Environ(), "QC_WALLET_STORAGE_TEST_PATH="+path, "QC_WALLET_STORAGE_TEST_MODE="+mode)
	return cmd
}

func TestWalletStorageProcess(t *testing.T) {
	mode := os.Getenv("QC_WALLET_STORAGE_TEST_MODE")
	if mode == "" {
		return
	}
	path := os.Getenv("QC_WALLET_STORAGE_TEST_PATH")
	if mode == "save" {
		if err := SaveWalletAt(path, NewWallet()); err != nil {
			t.Fatal(err)
		}
		return
	}
	if mode == "crash" {
		if _, err := lockWalletStore(path); err != nil {
			t.Fatal(err)
		}
		f, err := os.CreateTemp(filepath.Dir(path), ".qc-wallet-store-*")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Write([]byte(`{"wallets":`)); err != nil {
			t.Fatal(err)
		}
		// Simulate a process exiting before temp Sync/Close/replacement. The OS
		// must release its lock, and the previous complete store must survive.
		os.Exit(0)
	}
	t.Fatal("unknown test mode")
}

func TestWalletMultiProcessUpdatesAndCrashRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallets.json")
	if err := SaveWalletAt(path, NewWallet()); err != nil {
		t.Fatal(err)
	}
	const count = 4
	children := make([]*exec.Cmd, count)
	for i := range children {
		children[i] = walletStorageChild(t, path, "save")
		if err := children[i].Start(); err != nil {
			t.Fatal(err)
		}
	}
	for _, cmd := range children {
		if err := cmd.Wait(); err != nil {
			t.Fatal("child wallet save failed:", err)
		}
	}
	st, err := readStoreAt(path)
	if err != nil || len(st.Wallets) != count+1 {
		t.Fatal("multi-process update lost wallet records")
	}
	before, _ := os.ReadFile(path)
	if raw, err := walletStorageChild(t, path, "crash").CombinedOutput(); err != nil {
		t.Fatalf("crash fixture: %v %s", err, raw)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("crashed writer changed existing store")
	}
	if err := SaveWalletAt(path, NewWallet()); err != nil {
		t.Fatal("crashed process left an unusable stale lock:", err)
	}
	st, err = readStoreAt(path)
	if err != nil || len(st.Wallets) != count+2 {
		t.Fatal("wallet not usable after writer crash")
	}
}
