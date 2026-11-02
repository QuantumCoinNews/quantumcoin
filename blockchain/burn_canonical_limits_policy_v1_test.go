package blockchain

import (
	"testing"

	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
)

func canonicalBurnLimitTestChainV1(
	t *testing.T,
) (*Blockchain, *wallet.Wallet) {
	t.Helper()

	bc := atomDenominationTestChainV1(t, 1)

	owner := wallet.NewWallet()
	if owner == nil {
		t.Fatal("create burn-limit owner wallet returned nil")
	}

	// This output originates at legacy height 0 and represents 1001 QC.
	// At AtomV3 spend height 1 the existing denomination layer normalizes it
	// to 1,001,000,000 atoms.
	funding := &Transaction{
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{
				Amount:     1001,
				PubKeyHash: wallet.HashPubKey(owner.PublicKey),
			},
		},
		Sender: "COINBASE",
		Amount: 1001,
	}
	funding.ID = funding.Hash()

	bc.Blocks[0].Transactions = []*Transaction{
		funding,
	}

	bc.coinbaseMaturity = 0
	bc.UpdateUTXOSet()

	if got := bc.nextSpendHeightV1(); got != 1 {
		t.Fatalf(
			"burn-limit fixture next spend height=%d want=1",
			got,
		)
	}

	if got := bc.ProtocolAtHeightV1(1); got != qconsensus.ProtocolAtomV3 {
		t.Fatalf(
			"burn-limit fixture protocol at height 1=%v want=%v",
			got,
			qconsensus.ProtocolAtomV3,
		)
	}

	return bc, owner
}

func canonicalBurnForQCAmountV1(
	t *testing.T,
	bc *Blockchain,
	owner *wallet.Wallet,
	rawQC string,
) *Transaction {
	t.Helper()

	tx, err :=
		NewCanonicalBurnTransactionFromQCAmountV1(
			owner.GetAddress(),
			rawQC,
			bc,
		)
	if err != nil {
		t.Fatalf(
			"create canonical burn amount %q QC: %v",
			rawQC,
			err,
		)
	}

	if err := tx.Sign(owner.PrivateKey); err != nil {
		t.Fatalf(
			"sign canonical burn amount %q QC: %v",
			rawQC,
			err,
		)
	}

	if !tx.Verify() {
		t.Fatalf(
			"canonical burn amount %q QC failed verification",
			rawQC,
		)
	}

	return tx
}

func TestValidateCanonicalBurnLimitsAtHeightV1(t *testing.T) {
	bc, owner := canonicalBurnLimitTestChainV1(t)

	tx :=
		canonicalBurnForQCAmountV1(
			t,
			bc,
			owner,
			"1000",
		)

	if err :=
		bc.validateCanonicalBurnLimitsAtHeightV1(
			tx,
			1,
			wallet.HashPubKey(owner.PublicKey),
			0,
			0,
		); err != nil {
		t.Fatalf(
			"canonical burn at exact per-tx limit rejected: %v",
			err,
		)
	}
}

func TestValidateCanonicalBurnLimitsRejectsAbovePerTransactionV1(
	t *testing.T,
) {
	bc, owner := canonicalBurnLimitTestChainV1(t)

	tx :=
		canonicalBurnForQCAmountV1(
			t,
			bc,
			owner,
			"1000.000001",
		)

	if err :=
		bc.validateCanonicalBurnLimitsAtHeightV1(
			tx,
			1,
			wallet.HashPubKey(owner.PublicKey),
			0,
			0,
		); err == nil {
		t.Fatal(
			"burn one atom above canonical per-tx maximum unexpectedly accepted",
		)
	}
}

func TestValidateCanonicalBurnLimitsRejectsOwnerLifetimeOverflowV1(
	t *testing.T,
) {
	bc, owner := canonicalBurnLimitTestChainV1(t)

	tx :=
		canonicalBurnForQCAmountV1(
			t,
			bc,
			owner,
			"0.000001",
		)

	if err :=
		bc.validateCanonicalBurnLimitsAtHeightV1(
			tx,
			1,
			wallet.HashPubKey(owner.PublicKey),
			qconsensus.CanonicalBurnOwnerLifetimeMaxAtomsV1,
			0,
		); err == nil {
		t.Fatal(
			"owner lifetime burn one atom above maximum unexpectedly accepted",
		)
	}
}

func TestValidateCanonicalBurnLimitsRejectsNetworkLifetimeOverflowV1(
	t *testing.T,
) {
	bc, owner := canonicalBurnLimitTestChainV1(t)

	tx :=
		canonicalBurnForQCAmountV1(
			t,
			bc,
			owner,
			"0.000001",
		)

	if err :=
		bc.validateCanonicalBurnLimitsAtHeightV1(
			tx,
			1,
			wallet.HashPubKey(owner.PublicKey),
			0,
			qconsensus.CanonicalBurnNetworkLifetimeMaxAtomsV1,
		); err == nil {
		t.Fatal(
			"network lifetime burn one atom above maximum unexpectedly accepted",
		)
	}
}
