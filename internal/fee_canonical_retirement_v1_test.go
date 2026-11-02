package internal

import (
	"os"
	"strings"
	"testing"
)

func TestFeeManagerLegacySurfaceIsFailClosedV1(t *testing.T) {
	fm := NewFeeManager()
	if fm == nil {
		t.Fatal("nil fee manager")
	}
	if got := fm.GetFixedFee(); got != 0 {
		t.Fatalf("fixed fee=%d want=0", got)
	}
	fm.SetFixedFee(99)
	if got := fm.GetFixedFee(); got != 0 {
		t.Fatalf("fixed fee after set=%d want=0", got)
	}
	if got := fm.ApplyFee(struct{}{}); got != 0 {
		t.Fatalf("applied fee=%d want=0", got)
	}
	if got := fm.ApplyFee(nil); got != 0 {
		t.Fatalf("nil applied fee=%d want=0", got)
	}
	if got := fm.GetTotalFees(); got != 0 {
		t.Fatalf("total fees=%d want=0", got)
	}
	fm.Reset()
	if got := fm.GetTotalFees(); got != 0 {
		t.Fatalf("total after reset=%d want=0", got)
	}
}

func TestFeeManagerSourceHasNoParallelFeeAuthorityV1(t *testing.T) {
	raw, err := os.ReadFile("fee.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, forbidden := range []string{"sync.Mutex", "totalFees", "fixedFee", "defaultFixedFee", "+= fee"} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("parallel fee marker remains: %s", forbidden)
		}
	}
	for _, required := range []string{"CalculateBlockFeesV1", "func (fm *FeeManager) ApplyFee(_ interface{}) int { return 0 }"} {
		if !strings.Contains(s, required) {
			t.Fatalf("canonical retirement marker missing: %s", required)
		}
	}
}
