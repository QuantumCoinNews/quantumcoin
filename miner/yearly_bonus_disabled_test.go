package miner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	qinternal "quantumcoin/internal"
)

const yearlyBonusChildProcessEnvV1 = "QC_YEARLY_BONUS_CHILD_V1"
const yearlyBonusChildFileEnvV1 = "QC_YEARLY_BONUS_FILE_V1"

func TestYearlyBonusCompatibilityHookIsNoOpV1(t *testing.T) {
	before := qinternal.ListBonuses("")

	for i := 0; i < 100; i++ {
		checkYearlyBonus(
			fmt.Sprintf("QC-disabled-bonus-%d", i),
		)
	}

	after := qinternal.ListBonuses("")

	if !reflect.DeepEqual(before, after) {
		t.Fatalf(
			"disabled yearly bonus hook changed bonus records: before=%d after=%d",
			len(before),
			len(after),
		)
	}
}

func TestYearlyBonusCompatibilityHookIsNoOpAcrossRestartV1(
	t *testing.T,
) {
	if os.Getenv(yearlyBonusChildProcessEnvV1) == "1" {
		bonusPath := os.Getenv(yearlyBonusChildFileEnvV1)
		if bonusPath == "" {
			t.Fatal("child bonus file path required")
		}

		qinternal.SetBonusFile(bonusPath)

		for i := 0; i < 100; i++ {
			checkYearlyBonus(
				fmt.Sprintf("QC-restart-bonus-%d", i),
			)
		}

		if bonuses := qinternal.ListBonuses(""); len(bonuses) != 0 {
			t.Fatalf(
				"disabled yearly bonus created %d in-memory records",
				len(bonuses),
			)
		}

		if _, err := os.Stat(bonusPath); !os.IsNotExist(err) {
			t.Fatalf(
				"disabled yearly bonus created persistent data: %v",
				err,
			)
		}

		return
	}

	bonusPath := filepath.Join(
		t.TempDir(),
		"yearly-bonus.jsonl",
	)

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve test executable: %v", err)
	}

	for processRun := 1; processRun <= 2; processRun++ {
		command := exec.Command(
			executable,
			"-test.run=^TestYearlyBonusCompatibilityHookIsNoOpAcrossRestartV1$",
			"-test.count=1",
		)

		command.Env = append(
			os.Environ(),
			yearlyBonusChildProcessEnvV1+"=1",
			yearlyBonusChildFileEnvV1+"="+bonusPath,
		)

		output, runErr := command.CombinedOutput()
		if runErr != nil {
			t.Fatalf(
				"yearly bonus child process %d failed: %v\n%s",
				processRun,
				runErr,
				output,
			)
		}
	}

	if _, err := os.Stat(bonusPath); !os.IsNotExist(err) {
		t.Fatalf(
			"disabled yearly bonus produced data across restart: %v",
			err,
		)
	}
}
