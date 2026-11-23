package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func isolatedBonusAuditV1(t *testing.T, path string) {
	t.Helper()
	previousPath := currentBonusFilePath()
	memBonusMu.Lock()
	previous := memBonusLog
	memBonusLog = nil
	memBonusMu.Unlock()
	SetBonusFile(path)
	t.Cleanup(func() { SetBonusFile(previousPath); memBonusMu.Lock(); memBonusLog = previous; memBonusMu.Unlock() })
}

func TestBonusConcurrentAuditWritesAreCompleteAndNonMonetary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit", "bonus.jsonl")
	isolatedBonusAuditV1(t, path)
	const count = 32
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if !GiveBonus(fmt.Sprintf("test-audit-%d", i), "record-only", 1, "not a payment", "") {
				t.Errorf("audit write %d failed", i)
			}
			_ = ListBonuses("")
			_ = ListBonusLog()
		}(i)
	}
	wg.Wait()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	seen := map[string]bool{}
	scan := bufio.NewScanner(file)
	for scan.Scan() {
		var b Bonus
		if err := json.Unmarshal(scan.Bytes(), &b); err != nil {
			t.Fatal(err)
		}
		if seen[b.Address] || b.Status != "pending_settlement" || b.Amount != 1 || b.Timestamp.IsZero() {
			t.Fatal("duplicate, incomplete or payment-like audit record")
		}
		seen[b.Address] = true
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	if len(seen) != count || len(ListBonuses("")) != count {
		t.Fatal("file and memory record counts differ")
	}
	for _, b := range ListBonuses("") {
		if !seen[b.Address] {
			t.Fatal("memory/file audit identity mismatch")
		}
	}
}

func TestBonusFailedAuditWriteLeavesMemoryUnchanged(t *testing.T) {
	isolatedBonusAuditV1(t, t.TempDir()) // Directory cannot be opened as a log file.
	before := ListBonuses("")
	if GiveBonus("test-audit", "record-only", 1, "", "") {
		t.Fatal("failed disk write reported success")
	}
	if !reflect.DeepEqual(before, ListBonuses("")) {
		t.Fatal("failed disk write changed memory")
	}
}
