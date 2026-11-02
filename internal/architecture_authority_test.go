package internal

import (
	"os"
	"strings"
	"testing"
)

func TestInternalHasSingleCanonicalMonetaryAuthority(t *testing.T) {
	tests := []struct {
		file      string
		required  []string
		forbidden []string
	}{
		{
			file: "fee.go",
			required: []string{
				"type FeeManager struct{}",
				"ApplyFee(_ interface{}) int { return 0 }",
			},
			forbidden: []string{
				"sync.Mutex",
				"totalFees",
				"fixedFee",
				"+= fee",
			},
		},
		{
			file: "reward.go",
			required: []string{
				"blockchain.RewardAtHeightV1",
				"blockchain.ExpectedSubsidyFromStateV1",
				"blockchain.CapRewardByRemainingSupplyV1",
			},
			forbidden: []string{
				"time.Now(",
				"time.Since(",
				"MiningPeriodSecs <",
			},
		},
		{
			file: "mempool.go",
			required: []string{
				"canonical *blockchain.Blockchain",
				"canonical.AddTransaction(tx)",
				"PendingTransactionsSnapshot",
			},
			forbidden: []string{
				"transactions []*blockchain.Transaction",
				"append(mp.transactions",
				"index map[string]struct{}",
				"AnalyzeTransactions",
			},
		},
		{
			file: "utxo.go",
			required: []string{
				"canonical *blockchain.Blockchain",
				"canonical.UpdateUTXOSetSynchronized()",
				"canonical.FindUTXOsSnapshot",
			},
			forbidden: []string{
				"UTXOs map[string]",
				"set.UTXOs",
				"make(map[string][]blockchain.TransactionOutput)",
			},
		},
		{
			file: "bonus_core.go",
			required: []string{
				"reward_system.go içindeki memBonusLog",
			},
			forbidden: []string{
				"bonusLog []string",
				"append(bonusLog",
			},
		},
		{
			file: "reward_system.go",
			required: []string{
				`Status:    "pending_settlement"`,
				"maxMemBonusLines = 1000",
			},
			forbidden: []string{
				"blockchain.NewTransaction",
				"Mint(",
				"ChangeBalance(",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.file, func(t *testing.T) {
			data, err := os.ReadFile(test.file)
			if err != nil {
				t.Fatalf("read %s: %v", test.file, err)
			}

			source := string(data)

			for _, marker := range test.required {
				if !strings.Contains(source, marker) {
					t.Fatalf(
						"%s missing authority marker: %s",
						test.file,
						marker,
					)
				}
			}

			for _, marker := range test.forbidden {
				if strings.Contains(source, marker) {
					t.Fatalf(
						"%s contains parallel authority marker: %s",
						test.file,
						marker,
					)
				}
			}
		})
	}
}
