package main

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	qblockchain "quantumcoin/blockchain"
	qconsensus "quantumcoin/consensus"
	qwallet "quantumcoin/wallet"
)

func TestBalanceAPIReportsOneAtomAsMicroQCV1(t *testing.T) {
	chain := qblockchain.NewBlockchain(
		qblockchain.CanonicalInitialMiningRewardV1,
		int(qblockchain.QCTotalSupplyHardCap),
	)
	if chain == nil {
		t.Fatal("NewBlockchain returned nil")
	}

	genesis := chain.Blocks[0]
	if genesis == nil || len(genesis.Hash) == 0 {
		t.Fatal("test chain missing genesis hash")
	}

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID:        "atom-balance-api-display-test",
			LegacyGenesisHex: hex.EncodeToString(genesis.Hash),
			ActivationHeight: 1,
			PreserveHistory:  true,
			PreserveBalances: true,
			TargetProtocol:   qconsensus.ProtocolAtomV3,
		},
	}

	if err := chain.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err != nil {
		t.Fatalf("bind AtomV3 activation: %v", err)
	}

	owner := qwallet.NewWallet()
	address := owner.GetAddress()

	funding := &qblockchain.Transaction{
		Inputs: []qblockchain.TransactionInput{
			{
				TxID:     []byte{0x01},
				OutIndex: 0,
			},
		},
		Outputs: []qblockchain.TransactionOutput{
			{
				Amount:     1,
				PubKeyHash: qwallet.Base58DecodeAddress(address),
			},
		},
		Sender: "atom-api-fixture",
		Amount: 1,
	}
	funding.ID = funding.Hash()

	chain.Blocks = append(
		chain.Blocks,
		&qblockchain.Block{
			Index:        1,
			Transactions: []*qblockchain.Transaction{funding},
		},
	)

	chain.UpdateUTXOSetSynchronized()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/wallet/balance/"+address,
		nil,
	)
	rec := httptest.NewRecorder()

	handleBalanceForChain(rec, req, chain)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status=%d want=%d body=%q",
			rec.Code,
			http.StatusOK,
			rec.Body.String(),
		)
	}

	var response BalanceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf(
			"decode balance response: %v body=%q",
			err,
			rec.Body.String(),
		)
	}

	if response.Balance != 0.000001 {
		t.Fatalf(
			"AtomV3 API balance=%v want=0.000001 QC",
			response.Balance,
		)
	}

	if response.Spendable != 0.000001 {
		t.Fatalf(
			"AtomV3 API spendable=%v want=0.000001 QC",
			response.Spendable,
		)
	}
}
