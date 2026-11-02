package node

import (
	"errors"
	"strings"
	"testing"
)

func TestRuntimeMiningRejectsEmptyAddressV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	err := runtime.StartMining(
		"",
		MiningHooksV1{
			Difficulty: func() int {
				return 1
			},
		},
	)

	if err == nil {
		t.Fatal(
			"empty miner address was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"miner address is empty",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestRuntimeMiningRejectsMissingDifficultyProviderV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	err := runtime.StartMining(
		"miner",
		MiningHooksV1{},
	)

	if err == nil {
		t.Fatal(
			"missing difficulty provider was accepted",
		)
	}
}

func TestNilRuntimeMiningStateIsSafeV1(
	t *testing.T,
) {
	var runtime *Runtime

	if runtime.IsMining() {
		t.Fatal(
			"nil runtime reported active miner",
		)
	}

	if runtime.StopMining() {
		t.Fatal(
			"nil runtime stopped a miner",
		)
	}

	if runtime.MinerAddress() != "" {
		t.Fatal(
			"nil runtime returned miner address",
		)
	}
}

func TestMinerAlreadyRunningErrorV1(
	t *testing.T,
) {
	if !errors.Is(
		ErrMinerAlreadyRunningV1,
		ErrMinerAlreadyRunningV1,
	) {
		t.Fatal(
			"miner already running sentinel is invalid",
		)
	}
}
