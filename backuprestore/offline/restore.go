package offline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RestoreEvent is an immutable selection record. Files are never overwritten.
// A crash leaves either a complete valid event, no event, or an invalid record
// that blocks further selection. Operators inspect preserved files explicitly.
type RestoreEvent struct {
	Version        string    `json:"version"`
	Action         string    `json:"action"`
	From           string    `json:"from"`
	To             string    `json:"to"`
	Network        string    `json:"network"`
	GenesisHash    string    `json:"genesisHash"`
	PreviousSHA256 string    `json:"previousSha256"`
	UTC            time.Time `json:"utc"`
}

func lockWorkspace(root string) (func(), error) {
	if err := noLinks(root); err != nil {
		return nil, err
	}
	if err := os.Mkdir(root, 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	lock := filepath.Join(root, ".offline-recovery-lock")
	if err := os.Mkdir(lock, 0700); err != nil {
		return nil, errors.New("recovery workspace locked; inspect stale lock after crash before manual removal")
	}
	return func() { _ = os.Remove(lock) }, nil
}

func journal(root, network, genesis string) ([]RestoreEvent, string, error) {
	files, err := os.ReadDir(root)
	if err != nil {
		return nil, "", err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	var events []RestoreEvent
	previous := ""
	selected := ""
	for _, file := range files {
		name := file.Name()
		if name == ".offline-recovery-lock" {
			continue
		}
		if strings.HasPrefix(name, "generation-") {
			if !validGeneration(name) || !file.IsDir() {
				return nil, "", errors.New("invalid restore generation")
			}
			continue
		}
		if file.IsDir() || name != fmt.Sprintf("event-%08d.json", len(events)+1) {
			return nil, "", errors.New("unmanaged files or invalid recovery journal")
		}
		b, err := readSmall(filepath.Join(root, name), 64<<10)
		if err != nil {
			return nil, "", err
		}
		var event RestoreEvent
		if err := strictJSON(b, &event); err != nil {
			return nil, "", err
		}
		if event.Version != "QC_OFFLINE_RESTORE_EVENT_V1" || event.Network != network || event.GenesisHash != genesis || event.PreviousSHA256 != previous || event.From != selected || event.UTC.IsZero() || !validGeneration(event.To) && event.To != "" {
			return nil, "", errors.New("recovery journal chain/identity mismatch")
		}
		if event.Action != "RESTORE" && event.Action != "ROLLBACK" {
			return nil, "", errors.New("unknown recovery journal action")
		}
		if event.Action == "RESTORE" && event.To == "" {
			return nil, "", errors.New("restore target required")
		}
		h := sha256.Sum256(b)
		previous = hex.EncodeToString(h[:])
		selected = event.To
		events = append(events, event)
	}
	return events, previous, nil
}

func validGeneration(name string) bool {
	if !strings.HasPrefix(name, "generation-") {
		return false
	}
	id := strings.TrimPrefix(name, "generation-")
	b, err := hex.DecodeString(id)
	return err == nil && len(b) == 16 && strings.ToLower(id) == id
}

func appendEvent(root, action, from, to, network, genesis, previous string, count int) error {
	event := RestoreEvent{Version: "QC_OFFLINE_RESTORE_EVENT_V1", Action: action, From: from, To: to, Network: network, GenesisHash: genesis, PreviousSHA256: previous, UTC: time.Now().UTC()}
	b, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return err
	}
	return writeNew(filepath.Join(root, fmt.Sprintf("event-%08d.json", count+1)), b)
}

// DryRun copies into a NEW isolated directory and verifies the result. This is
// a real restore rehearsal, never a live path replacement or wallet unlock.
func (e Engine) DryRun(ctx context.Context, snapshot, target, network, genesis string, password []byte) error {
	m, err := e.Verify(ctx, snapshot, network, genesis, password)
	if err != nil {
		return err
	}
	if err := noLinks(target); err != nil {
		return err
	}
	if err := os.Mkdir(target, 0700); err != nil {
		return err
	}
	for _, entry := range m.Entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := copyNew(filepath.Join(snapshot, entry.File), filepath.Join(target, entry.File)); err != nil {
			return err
		}
	}
	if err := copyNew(filepath.Join(snapshot, "manifest.json"), filepath.Join(target, "manifest.json")); err != nil {
		return err
	}
	_, err = e.Verify(ctx, target, network, genesis, password)
	return err
}

// Restore publishes a verified isolated generation in an offline workspace.
// The journal is the selection authority; partial generations are never selected.
// It does not reconfigure services or copy keys into an active wallet.
func (e Engine) Restore(ctx context.Context, snapshot, root, network, genesis string, password []byte, offline bool) (string, error) {
	if !offline {
		return "", errors.New("offline recovery declaration required")
	}
	if err := identity(network, genesis); err != nil {
		return "", err
	}
	if _, err := e.Verify(ctx, snapshot, network, genesis, password); err != nil {
		return "", err
	}
	unlock, err := lockWorkspace(root)
	if err != nil {
		return "", err
	}
	defer unlock()
	events, previous, err := journal(root, network, genesis)
	if err != nil {
		return "", err
	}
	from := ""
	if len(events) > 0 {
		from = events[len(events)-1].To
	}
	if from != "" {
		if _, err := e.Verify(ctx, filepath.Join(root, from), network, genesis, password); err != nil {
			return "", err
		}
	}
	id, err := randomID()
	if err != nil {
		return "", err
	}
	name := "generation-" + id
	if err := e.DryRun(ctx, snapshot, filepath.Join(root, name), network, genesis, password); err != nil {
		return "", err
	}
	if err := appendEvent(root, "RESTORE", from, name, network, genesis, previous, len(events)); err != nil {
		return "", err
	}
	return filepath.Join(root, name), nil
}

// Rollback re-selects the previous validated generation (or no generation after
// the first restore). Both old and new files remain intact for investigation.
func (e Engine) Rollback(ctx context.Context, root, network, genesis string, password []byte, offline bool) (string, error) {
	if !offline {
		return "", errors.New("offline recovery declaration required")
	}
	if err := identity(network, genesis); err != nil {
		return "", err
	}
	unlock, err := lockWorkspace(root)
	if err != nil {
		return "", err
	}
	defer unlock()
	events, previous, err := journal(root, network, genesis)
	if err != nil {
		return "", err
	}
	if len(events) == 0 {
		return "", errors.New("no restore to roll back")
	}
	last := events[len(events)-1]
	target := last.From
	if target != "" {
		if _, err := e.Verify(ctx, filepath.Join(root, target), network, genesis, password); err != nil {
			return "", err
		}
	}
	if err := appendEvent(root, "ROLLBACK", last.To, target, network, genesis, previous, len(events)); err != nil {
		return "", err
	}
	if target == "" {
		return "", nil
	}
	return filepath.Join(root, target), nil
}

func (e Engine) Current(ctx context.Context, root, network, genesis string, password []byte) (string, error) {
	if err := identity(network, genesis); err != nil {
		return "", err
	}
	if err := noLinks(root); err != nil {
		return "", err
	}
	events, _, err := journal(root, network, genesis)
	if err != nil {
		return "", err
	}
	if len(events) == 0 || events[len(events)-1].To == "" {
		return "", nil
	}
	target := filepath.Join(root, events[len(events)-1].To)
	if _, err := e.Verify(ctx, target, network, genesis, password); err != nil {
		return "", err
	}
	return target, nil
}

// SealWalletFile and UnsealWalletFile only create new files. Unsealing is an
// explicit offline recovery step; stdout never receives private wallet material.
func SealWalletFile(source, target string, password []byte) error {
	raw, err := readSmall(source, maxWalletBytes)
	if err != nil {
		return err
	}
	defer wipe(raw)
	sealed, err := SealWallet(raw, password)
	if err != nil {
		return err
	}
	return writeNew(target, sealed)
}
func UnsealWalletFile(source, target string, password []byte) error {
	sealed, err := readSmall(source, 2*maxWalletBytes)
	if err != nil {
		return err
	}
	raw, err := OpenWallet(sealed, password)
	if err != nil {
		return err
	}
	defer wipe(raw)
	return writeNew(target, raw)
}
