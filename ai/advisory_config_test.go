package ai

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

// Each environment contract runs in a fresh process, as production's once-only
// configuration does; tests never reset a live sync.Once or mutate shared state.
func TestAdvisoryEnvironment(t *testing.T) {
	if os.Getenv("QC_AI_CONFIG_CHILD") == "1" {
		want := os.Getenv("QC_AI_TEST_ENABLED") == "1"
		if Enabled() != want || string(Level()) != os.Getenv("QC_AI_TEST_LEVEL") {
			t.Fatal("environment ignored")
		}
		SetEnabled(false)
		if Enabled() {
			t.Fatal("runtime disable ignored")
		}
		SetEnabled(true)
		if Enabled() != (Level() != AILevelOff) {
			t.Fatal("off level bypassed")
		}
		SetEnabled(want)
		amount, count := 500.0, 5
		switch Level() {
		case AILevelLow:
			amount, count = 1000, 10
		case AILevelAggressive:
			amount, count = 250, 3
		}
		txs := make([]TxLite, count)
		for i := range txs {
			txs[i] = TxLite{TxID: fmt.Sprint(i), WalletAddress: "owner", Amount: 1, Timestamp: time.Unix(10+int64(i), 0)}
		}
		if len(AnalyzeTransactions(txs[:count-1])) != 0 {
			t.Fatal("premature burst")
		}
		reports := AnalyzeTransactions(txs)
		if want && (len(reports) != 1 || reports[0].Reason != "high frequency activity") {
			t.Fatal("missing burst", reports)
		}
		if !want && len(reports) != 0 {
			t.Fatal("disabled analysis")
		}
		duplicate := make([]TxLite, count)
		for i := range duplicate {
			duplicate[i] = txs[0]
		}
		if len(AnalyzeTransactions(duplicate)) != 0 {
			t.Fatal("duplicate observation counted")
		}
		txs[count-1].Timestamp = time.Unix(1000, 0)
		if len(AnalyzeTransactions(txs)) != 0 {
			t.Fatal("out-of-window burst")
		}
		txs = []TxLite{{TxID: "amount", Amount: amount - 0.01}}
		if len(AnalyzeTransactions(txs)) != 0 {
			t.Fatal("premature amount warning")
		}
		txs[0].Amount = amount
		if got := len(AnalyzeTransactions(txs)); (got == 1) != want {
			t.Fatal("level amount threshold", got)
		}
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					SetEnabled(j%2 == 0)
					Enabled()
					Level()
					AnalyzeTransactions(txs)
				}
			}()
		}
		wg.Wait()
		return
	}
	for _, tc := range []struct{ enabled, level, want, effective string }{
		{"", "", "1", "normal"}, {"0", "normal", "0", "normal"}, {"FALSE", "aggressive", "0", "aggressive"},
		{"off", "low", "0", "low"}, {"1", "off", "0", "off"}, {"1", "low", "1", "low"},
		{"1", "aggressive", "1", "aggressive"}, {"1", "unknown", "1", "normal"},
	} {
		t.Run(tc.enabled+"_"+tc.level, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestAdvisoryEnvironment$")
			cmd.Env = append(os.Environ(), "QC_AI_CONFIG_CHILD=1", "QC_AI_ENABLED="+tc.enabled, "QC_AI_LEVEL="+tc.level, "QC_AI_TEST_ENABLED="+tc.want, "QC_AI_TEST_LEVEL="+tc.effective)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
		})
	}
}
