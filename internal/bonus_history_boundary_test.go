package internal

import (
	"fmt"
	"strings"
	"testing"
)

func TestBonusHistoryIsBoundedSingleAndNeverClaimsPayment(t *testing.T) {
	previousPath := currentBonusFilePath()
	SetBonusFile("")
	memBonusMu.Lock()
	previous := memBonusLog
	memBonusLog = nil
	memBonusMu.Unlock()
	t.Cleanup(func() {
		SetBonusFile(previousPath)
		memBonusMu.Lock()
		memBonusLog = previous
		memBonusMu.Unlock()
	})
	for i := 0; i < 1003; i++ {
		if !GiveBonus(fmt.Sprintf("test-only-%d", i), "audit-record-only", 1, "no payment", "") {
			t.Fatal("record rejected")
		}
	}
	structured, legacy := ListBonuses(""), ListBonusLog()
	if len(structured) != 1000 || len(legacy) != len(structured) {
		t.Fatal("history grew without bound or views diverged")
	}
	if structured[0].Address != "test-only-3" || structured[999].Address != "test-only-1002" {
		t.Fatal("wrong retention order")
	}
	for i, record := range structured {
		if record.Status != "pending_settlement" || !strings.Contains(legacy[i], record.Address) || !strings.Contains(legacy[i], "pending_settlement") {
			t.Fatal("record views imply a payment or disagree")
		}
	}
	GiveBonusCore("test-only-core", "audit-record-only", 1, "", "")
	if len(ListBonuses("")) != 1000 || len(ListBonusLog()) != 1000 {
		t.Fatal("compatibility logger created a second history")
	}
}
