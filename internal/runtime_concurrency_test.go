package internal

import (
	"fmt"
	"sync"
	"testing"

	"quantumcoin/blockchain"
)

func TestMempoolConcurrentCanonicalAdapterV1(t *testing.T) {
	chain := blockchain.NewBlockchain(1, 0)
	mp := NewCanonicalMempool(chain)

	if mp == nil {
		t.Fatal("canonical mempool adapter is nil")
	}

	var wg sync.WaitGroup
	errs := make(chan error, 64)

	for worker := 0; worker < 32; worker++ {
		wg.Add(1)

		go func(worker int) {
			defer wg.Done()

			for iteration := 0; iteration < 100; iteration++ {
				if !mp.BindCanonicalBlockchain(chain) {
					errs <- fmt.Errorf(
						"worker %d canonical rebinding failed",
						worker,
					)
					return
				}

				mp.SetCapacity(1000)
				_ = mp.Len()
				_ = mp.Has([]byte("missing-transaction"))
				_ = mp.GetTransactions()
			}
		}(worker)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

func TestRewardConcurrentCanonicalDeterminismV1(t *testing.T) {
	height := blockchain.CanonicalBlocksPerHalvingV1 + 7
	totalSupply := int(blockchain.QCTotalSupplyHardCap)

	cfg := RewardConfig{
		InitialReward:       999,
		HalvingPeriodBlocks: 1,
		HalvingPeriodSecs:   1,
		MiningPeriodSecs:    1,
		GenesisUnix:         1,
		TotalSupply:         totalSupply,
	}

	wantHeight := blockchain.RewardAtHeightV1(height)
	wantState := blockchain.ExpectedSubsidyFromStateV1(
		height,
		0,
		totalSupply,
	)

	var wg sync.WaitGroup
	errs := make(chan error, 64)

	for worker := 0; worker < 32; worker++ {
		wg.Add(1)

		go func(worker int) {
			defer wg.Done()

			for iteration := 0; iteration < 100; iteration++ {
				gotHeight := CalculateRewardByHeight(
					blockchain.CanonicalInitialMiningRewardV1,
					blockchain.CanonicalBlocksPerHalvingV1,
					height,
				)
				if gotHeight != wantHeight {
					errs <- fmt.Errorf(
						"worker %d height reward=%d want=%d",
						worker,
						gotHeight,
						wantHeight,
					)
					return
				}

				gotState := ComputeReward(
					cfg,
					height,
					0,
					int64(iteration),
				)
				if gotState != wantState {
					errs <- fmt.Errorf(
						"worker %d state reward=%d want=%d",
						worker,
						gotState,
						wantState,
					)
					return
				}
			}
		}(worker)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

func TestFeeManagerConcurrentFailClosedV1(t *testing.T) {
	manager := NewFeeManager()

	var wg sync.WaitGroup
	errs := make(chan error, 64)

	for worker := 0; worker < 32; worker++ {
		wg.Add(1)

		go func(worker int) {
			defer wg.Done()

			for iteration := 0; iteration < 100; iteration++ {
				manager.SetFixedFee(iteration)

				if got := manager.GetFixedFee(); got != 0 {
					errs <- fmt.Errorf(
						"worker %d fixed fee=%d want=0",
						worker,
						got,
					)
					return
				}

				if got := manager.ApplyFee(struct{}{}); got != 0 {
					errs <- fmt.Errorf(
						"worker %d applied fee=%d want=0",
						worker,
						got,
					)
					return
				}

				if got := manager.GetTotalFees(); got != 0 {
					errs <- fmt.Errorf(
						"worker %d total fees=%d want=0",
						worker,
						got,
					)
					return
				}

				manager.Reset()
			}
		}(worker)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}
