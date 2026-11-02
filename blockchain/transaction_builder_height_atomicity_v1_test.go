package blockchain

import (
	"testing"
	"time"

	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
)

func TestTransactionBuilderSnapshotsCandidateHeightWithUTXOV1(
	t *testing.T,
) {
	const activationHeight = 2

	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			activationHeight,
		)

	receiver := wallet.NewWallet()

	if got := bc.ProtocolAtHeightV1(1); got != qconsensus.ProtocolLegacyV1 {
		t.Fatalf(
			"protocol at height 1=%q want=%q",
			got,
			qconsensus.ProtocolLegacyV1,
		)
	}

	if got := bc.ProtocolAtHeightV1(2); got != qconsensus.ProtocolAtomV3 {
		t.Fatalf(
			"protocol at height 2=%q want=%q",
			got,
			qconsensus.ProtocolAtomV3,
		)
	}

	type builderResult struct {
		tx  *Transaction
		err error
	}

	started := make(chan struct{})
	done := make(chan builderResult, 1)

	bc.stateMu.Lock()
	locked := true

	defer func() {
		if locked {
			bc.stateMu.Unlock()
		}
	}()

	go func() {
		close(started)

		tx, err :=
			NewTransactionFromQCAmountV1(
				owner.GetAddress(),
				receiver.GetAddress(),
				"1",
				bc,
			)

		done <- builderResult{
			tx:  tx,
			err: err,
		}
	}()

	<-started

	// Give the builder ample opportunity to reach the state boundary.
	// Old behavior reads candidate height before waiting for stateMu.
	time.Sleep(250 * time.Millisecond)

	// Advance the canonical tip while stateMu is held.
	// The next candidate height is now 2, exactly the AtomV3 boundary.
	bc.Blocks = append(
		bc.Blocks,
		&Block{
			Index: 1,
		},
	)

	bc.stateMu.Unlock()
	locked = false

	var result builderResult

	select {
	case result = <-done:
	case <-time.After(3 * time.Second):
		t.Fatal(
			"transaction builder did not complete after state lock release",
		)
	}

	if result.err != nil {
		t.Fatalf(
			"transaction builder returned error: %v",
			result.err,
		)
	}

	if result.tx == nil {
		t.Fatal("transaction builder returned nil transaction")
	}

	if len(result.tx.Outputs) == 0 {
		t.Fatal("transaction builder returned no outputs")
	}

	const wantAtomUnits = 1_000_000

	if got := result.tx.Outputs[0].Amount; got != wantAtomUnits {
		t.Fatalf(
			"activation-boundary transaction amount=%d want=%d atoms",
			got,
			wantAtomUnits,
		)
	}
}
