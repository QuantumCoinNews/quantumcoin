package node

import "testing"

func TestMiningLoopNaturalExitReleasesRestartLock(
	t *testing.T,
) {
	runtime := &Runtime{}

	stop := make(chan struct{})
	done := make(chan struct{})

	runtime.minerStop = stop
	runtime.minerDone = done
	runtime.minerAddress = "test-miner"
	runtime.minerStopping = true

	// A nil blockchain makes MineOnceV1 return
	// ErrMiningBlockchainUnavailableV1. That represents an internal,
	// non-stop-channel loop termination and exercises the natural-exit
	// cleanup path.
	runtime.runMiningLoopV1(
		"test-miner",
		stop,
		done,
		MiningHooksV1{
			Difficulty: func() int {
				return 1
			},
		},
	)

	select {
	case <-done:
	default:
		t.Fatal(
			"mining loop returned without closing done channel",
		)
	}

	runtime.mu.RLock()
	defer runtime.mu.RUnlock()

	if runtime.minerStop != nil {
		t.Fatal(
			"natural mining-loop exit left restart lock active",
		)
	}

	if runtime.minerDone != nil {
		t.Fatal(
			"natural mining-loop exit left done state active",
		)
	}

	if runtime.minerAddress != "" {
		t.Fatalf(
			"natural mining-loop exit left miner address %q",
			runtime.minerAddress,
		)
	}

	if runtime.minerStopping {
		t.Fatal(
			"natural mining-loop exit left stopping state active",
		)
	}
}
