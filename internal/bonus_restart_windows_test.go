package internal

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestBonusAuditRestartDoesNotRestoreSpendableEntitlement(t *testing.T) {
	if path := os.Getenv("QC_INTERNAL_AUDIT_CHILD"); path != "" {
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		SetBonusFile(path)
		if len(ListBonuses("")) != 0 || len(ListBonusLog()) != 0 {
			t.Fatal("audit file loaded as entitlement")
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("read/setup rewrote historical audit")
		}
		return
	}
	path := filepath.Join(t.TempDir(), "bonus.jsonl")
	isolatedBonusAuditV1(t, path)
	if !GiveBonus("test-audit", "record-only", 1, "no payment", "") {
		t.Fatal("fixture write failed")
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		child := exec.Command(os.Args[0], "-test.run=^TestBonusAuditRestartDoesNotRestoreSpendableEntitlement$", "-test.count=1")
		child.Env = append(os.Environ(), "QC_INTERNAL_AUDIT_CHILD="+path)
		child.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if output, err := child.CombinedOutput(); err != nil {
			t.Fatalf("restart child: %v: %s", err, output)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("restart changed audit file")
	}
}
