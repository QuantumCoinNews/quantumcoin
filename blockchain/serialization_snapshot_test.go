package blockchain

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestSerializeBlockchainConcurrentMiningUsesCoherentSnapshot(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "4")
	t.Setenv("QC_NODE_DIR", t.TempDir())
	chain := NewBlockchain(config.Default().InitialReward, config.Default().TotalSupply)
	failures := make(chan error, 100)
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		for i := 0; i < 8; i++ {
			if _, err := chain.MineBlock(wallet.NewWallet().GetAddress(), 4); err != nil {
				failures <- err
				return
			}
		}
	}()
	for i := 0; i < 3; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 16; j++ {
				copy := DeserializeBlockchain(SerializeBlockchain(chain))
				if copy == nil || len(copy.Blocks) == 0 {
					failures <- fmt.Errorf("invalid wire snapshot")
					continue
				}
				// Rebuilding from the detached history must reproduce its UTXO.
				before := cloneUTXOForState(copy.UTXO)
				copy.UpdateUTXOSet()
				if !reflect.DeepEqual(before, copy.UTXO) {
					failures <- fmt.Errorf("wire history and UTXO came from different tips")
				}
			}
		}()
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
}
