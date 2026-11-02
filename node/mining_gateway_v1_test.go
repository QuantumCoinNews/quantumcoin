package node

import (
	"errors"
	"sync"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/wallet"
)

func TestMineOnceV1RejectsUnavailableRuntimeV1(
	t *testing.T,
) {
	var runtime *Runtime

	_, err := runtime.MineOnceV1("miner", 1)
	if !errors.Is(
		err,
		ErrMiningBlockchainUnavailableV1,
	) {
		t.Fatalf(
			"nil runtime error = %v, want %v",
			err,
			ErrMiningBlockchainUnavailableV1,
		)
	}
}

func TestMineOnceV1RejectsUnavailableBlockchainV1(
	t *testing.T,
) {
	runtime := NewRuntime(nil, nil)

	_, err := runtime.MineOnceV1("miner", 1)
	if !errors.Is(
		err,
		ErrMiningBlockchainUnavailableV1,
	) {
		t.Fatalf(
			"missing blockchain error = %v, want %v",
			err,
			ErrMiningBlockchainUnavailableV1,
		)
	}
}

func TestMineOnceV1ProducesBlockV1(
	t *testing.T,
) {
	chain := blockchain.NewBlockchain(1, 0)
	runtime := NewRuntime(chain, nil)

	block, err := runtime.MineOnceV1("miner", 1)
	if err != nil {
		t.Fatalf("mine one block: %v", err)
	}
	if block == nil {
		t.Fatal("mine one block returned nil block")
	}
	if block.Index != 1 {
		t.Fatalf(
			"mined block index = %d, want 1",
			block.Index,
		)
	}
	if len(chain.Blocks) != 2 {
		t.Fatalf(
			"chain block count = %d, want 2",
			len(chain.Blocks),
		)
	}
}

func TestMineOnceV1SerializesConcurrentMiningV1(
	t *testing.T,
) {
	chain := blockchain.NewBlockchain(1, 0)
	runtime := NewRuntime(chain, nil)

	minerAddresses := []string{
		wallet.GetAddressFromPub([]byte("miner1")),
		wallet.GetAddressFromPub([]byte("miner2")),
	}
	errorsCh := make(chan error, len(minerAddresses))
	start := make(chan struct{})

	var wait sync.WaitGroup
	wait.Add(len(minerAddresses))

	for _, miner := range minerAddresses {
		miner := miner

		go func() {
			defer wait.Done()
			<-start

			_, err := runtime.MineOnceV1(
				miner,
				1,
			)
			errorsCh <- err
		}()
	}

	close(start)
	wait.Wait()
	close(errorsCh)

	for err := range errorsCh {
		if err != nil {
			t.Fatalf(
				"concurrent mining failed: %v",
				err,
			)
		}
	}

	if len(chain.Blocks) != 3 {
		t.Fatalf(
			"chain block count = %d, want 3",
			len(chain.Blocks),
		)
	}
	if chain.Blocks[1].Index != 1 ||
		chain.Blocks[2].Index != 2 {
		t.Fatalf(
			"mined indexes = %d, %d; want 1, 2",
			chain.Blocks[1].Index,
			chain.Blocks[2].Index,
		)
	}
}
