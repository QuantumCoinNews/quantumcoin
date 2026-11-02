package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readMinerContractSource(
	t *testing.T,
	parts ...string,
) string {
	t.Helper()

	path := filepath.Join(parts...)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(
			"read %s: %v",
			path,
			err,
		)
	}

	return string(data)
}

func sourceSection(
	t *testing.T,
	source string,
	start string,
	end string,
) string {
	t.Helper()

	startIndex := strings.Index(
		source,
		start,
	)

	if startIndex < 0 {
		t.Fatalf(
			"start marker missing: %q",
			start,
		)
	}

	endIndex := strings.Index(
		source[startIndex+len(start):],
		end,
	)

	if endIndex < 0 {
		t.Fatalf(
			"end marker missing: %q",
			end,
		)
	}

	endIndex += startIndex + len(start)

	return source[startIndex:endIndex]
}

func TestMinerAPIUsesInProcessRuntimeContract(
	t *testing.T,
) {
	source := readMinerContractSource(
		t,
		"main.go",
	)

	handler := sourceSection(
		t,
		source,
		"func handleMinerStart",
		"func handleMinerStop",
	)

	for _, forbidden := range []string{
		"exec.Command",
		"run-mine",
		"cmd/miner",
	} {
		if strings.Contains(
			handler,
			forbidden,
		) {
			t.Fatalf(
				"miner API returned to child-process contract %q",
				forbidden,
			)
		}
	}

	for _, required := range []string{
		"nodeRuntime.IsMining()",
		"go startContinuousMining(addr)",
	} {
		if !strings.Contains(
			handler,
			required,
		) {
			t.Fatalf(
				"miner API in-process contract missing %q",
				required,
			)
		}
	}

	if !strings.Contains(
		source,
		"nodeRuntime.StartMining(",
	) {
		t.Fatal(
			"continuous mining no longer delegates to node runtime",
		)
	}
}

func TestMinerEntrypointContractsRemainSeparated(
	t *testing.T,
) {
	rootSource := readMinerContractSource(
		t,
		"main.go",
	)

	minerSource := readMinerContractSource(
		t,
		"cmd",
		"miner",
		"main.go",
	)

	if !strings.Contains(
		rootSource,
		`case "run-mine":`,
	) {
		t.Fatal(
			"root run-mine CLI entrypoint missing",
		)
	}

	if !strings.Contains(
		rootSource,
		"miner := os.Args[2]",
	) {
		t.Fatal(
			"root run-mine positional address contract missing",
		)
	}

	if !strings.Contains(
		minerSource,
		`flag.String("address"`,
	) {
		t.Fatal(
			"standalone miner -address contract missing",
		)
	}

	if !strings.Contains(
		minerSource,
		`Usage: miner -address <QC_ADDRESS>`,
	) {
		t.Fatal(
			"standalone miner usage contract missing",
		)
	}

	// They are separate executable entrypoints. The HTTP API must not
	// translate one CLI syntax into the other.
	handler := sourceSection(
		t,
		rootSource,
		"func handleMinerStart",
		"func handleMinerStop",
	)

	if strings.Contains(
		handler,
		"run-mine",
	) ||
		strings.Contains(
			handler,
			"-address",
		) ||
		strings.Contains(
			handler,
			"exec.Command",
		) {
		t.Fatal(
			"HTTP miner control conflates independent CLI contracts",
		)
	}
}

func TestWalletMinerRestartDoesNotDependOnCommandHandle(
	t *testing.T,
) {
	source := readMinerContractSource(
		t,
		"cmd",
		"wallet",
		"main_fyne.go",
	)

	if strings.Contains(
		source,
		"minerCmd",
	) {
		t.Fatal(
			"wallet returned to stale miner command-handle ownership",
		)
	}

	startBlock := sourceSection(
		t,
		source,
		"func startMinerVisible",
		"func stopMinerVisible",
	)

	for _, required := range []string{
		"qcResetStaleMinerState()",
		"readMinerPID()",
		"isProcessAlive(pid)",
		"clearMinerPID()",
	} {
		if !strings.Contains(
			source,
			required,
		) {
			t.Fatalf(
				"wallet lifecycle recovery missing %q",
				required,
			)
		}
	}

	if strings.Contains(
		startBlock,
		"cmd.Wait()",
	) {
		t.Fatal(
			"wallet start path returned to launcher Wait ownership",
		)
	}

	resetBlock := sourceSection(
		t,
		source,
		"func qcResetStaleMinerState",
		"var (",
	)

	for _, required := range []string{
		"minerRunningState = false",
		`"miner_pid.txt"`,
		`"miner_cmd_pid.txt"`,
		`"miner_stop.flag"`,
	} {
		if !strings.Contains(
			resetBlock,
			required,
		) {
			t.Fatalf(
				"stale miner cleanup missing %q",
				required,
			)
		}
	}
}
