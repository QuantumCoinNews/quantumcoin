package blockchain

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type chainFileLockV1 struct {
	mu    sync.Mutex
	users int
}

var chainFileLocksV1 = struct {
	sync.Mutex
	entries map[string]*chainFileLockV1
}{entries: make(map[string]*chainFileLockV1)}

func canonicalChainFilenameV1(filename string) (string, string, error) {
	if strings.TrimSpace(filename) == "" {
		return "", "", fmt.Errorf("chain persistence: filename is empty")
	}
	absolute, err := filepath.Abs(filename)
	if err != nil {
		return "", "", err
	}
	// Resolve existing symlinks, including a configured file symlink, so path
	// aliases use one writer lock and the target keeps its configured identity.
	if resolved, err := filepath.EvalSymlinks(absolute); err == nil {
		absolute = resolved
	} else if parent, err := filepath.EvalSymlinks(filepath.Dir(absolute)); err == nil {
		absolute = filepath.Join(parent, filepath.Base(absolute))
	}
	key := filepath.Clean(absolute)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	return absolute, fmt.Sprintf("%x", sha256.Sum256([]byte(key))), nil
}

func lockChainFilenameV1(key string) func() {
	chainFileLocksV1.Lock()
	entry := chainFileLocksV1.entries[key]
	if entry == nil {
		entry = &chainFileLockV1{}
		chainFileLocksV1.entries[key] = entry
	}
	entry.users++
	chainFileLocksV1.Unlock()
	entry.mu.Lock()
	return func() {
		entry.mu.Unlock()
		chainFileLocksV1.Lock()
		entry.users--
		if entry.users == 0 {
			delete(chainFileLocksV1.entries, key)
		}
		chainFileLocksV1.Unlock()
	}
}

// persistChainFileV1 is shared by ordinary saves, local-miner commits and NFT
// commits. Caller holds the chain state lock before entering this function.
// Encoding starts AFTER acquiring the destination lock; a delayed ordinary
// save cannot install bytes captured before a newer commit on the same chain.
func persistChainFileV1(filename string, encode func(io.Writer) error, identities ...*NetworkIdentityV1) error {
	absolute, key, err := canonicalChainFilenameV1(filename)
	if err != nil {
		return err
	}
	unlock := lockChainFilenameV1(key)
	defer unlock()
	unlockOS, err := lockChainFileOSV1(absolute, key)
	if err != nil {
		return fmt.Errorf("chain persistence lock: %w", err)
	}
	defer unlockOS()
	var expected *NetworkIdentityV1
	if len(identities) > 0 {
		expected = identities[0]
	}
	if len(identities) > 0 {
		if err := validateDestinationNetworkV1(absolute, expected); err != nil {
			return err
		}
	}
	file, err := os.CreateTemp(filepath.Dir(absolute), ".qc-chain-commit-*")
	if err != nil {
		return fmt.Errorf("chain persistence create: %w", err)
	}
	temporary := file.Name()
	defer func() { _ = file.Close(); _ = os.Remove(temporary) }()
	if err := encode(file); err != nil {
		return fmt.Errorf("chain persistence encode: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("chain persistence flush: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("chain persistence close: %w", err)
	}
	if err := replaceChainFileOSV1(temporary, absolute); err != nil {
		return fmt.Errorf("chain persistence install: %w", err)
	}
	return nil
}

func readChainFileV1(filename string) ([]byte, error) {
	file, err := openChainFileOSV1(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

// Called while holding the destination OS lock. Two profiles booting before the
// first file exists cannot race to replace each other's subsequently saved data.
func validateDestinationNetworkV1(path string, expected *NetworkIdentityV1) error {
	file, err := openChainFileOSV1(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	payload, header, err := diskChainPayloadV1(file)
	if err != nil {
		return fmt.Errorf("chain persistence: damaged destination preserved: %w", err)
	}
	var stored struct{ StoredNetworkIdentityV1 *NetworkIdentityV1 }
	if err := decodeChainGobV1(payload, &stored); err != nil {
		return fmt.Errorf("chain persistence: damaged destination preserved: %w", err)
	}
	if header != nil && !header.matchesStoredIdentity(stored.StoredNetworkIdentityV1) {
		return fmt.Errorf("chain persistence: destination header and stored identity differ")
	}
	if header != nil && expected != nil && fmt.Sprintf("%x", header.Genesis) != expected.GenesisHash {
		return fmt.Errorf("chain persistence: destination belongs to a different genesis")
	}
	if expected == nil {
		if stored.StoredNetworkIdentityV1 != nil {
			return fmt.Errorf("chain persistence: unbound writer cannot overwrite network-bound destination")
		}
		return nil
	}
	if stored.StoredNetworkIdentityV1 == nil {
		if expected.Network == "mainnet" {
			return fmt.Errorf("chain persistence: mainnet refuses untagged destination")
		}
		return nil
	}
	if *stored.StoredNetworkIdentityV1 != *expected {
		return fmt.Errorf("chain persistence: destination belongs to a different network/genesis")
	}
	return nil
}
