package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecurityEventsBoundDiskReadsAndPreserveOversizedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "security_events.json")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(MaxSecurityEventLogBytes + 1); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	result := ReadEvents(dir, int(^uint(0)>>1))
	if !result.Truncated || len(result.Events) != 0 {
		t.Fatal("oversized file read")
	}
	if err := AppendEvent(dir, NewEvent("x", SeverityInfo, "test", "", "", nil)); err == nil {
		t.Fatal("oversized store was overwritten")
	}
	stat, _ := os.Stat(path)
	if stat.Size() != MaxSecurityEventLogBytes+1 {
		t.Fatal("existing log changed")
	}
}
