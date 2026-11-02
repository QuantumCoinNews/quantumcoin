package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"quantumcoin/blockchain"
	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
)

func TestActualMuxAtomExactOneAtomSendV1(
	t *testing.T,
) {
	clearActualMuxTestEnvV1(t)
	t.Setenv(
		"QC_API_TOKEN",
		actualMuxTestTokenV1,
	)

	owner := wallet.NewWallet()
	receiver := wallet.NewWallet()

	chain := blockchain.NewBlockchain(
		blockchain.CanonicalInitialMiningRewardV1,
		int(blockchain.QCTotalSupplyHardCap),
	)
	if chain == nil {
		t.Fatal("NewBlockchain returned nil")
	}

	if len(chain.Blocks) == 0 ||
		chain.Blocks[0] == nil ||
		len(chain.Blocks[0].Hash) == 0 {
		t.Fatal("test chain missing genesis hash")
	}

	const activationHeight = 12

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID: "atom-api-exact-send-test",

			LegacyGenesisHex: hex.EncodeToString(
				chain.Blocks[0].Hash,
			),

			ActivationHeight: activationHeight,
			PreserveHistory:  true,
			PreserveBalances: true,
			TargetProtocol:   qconsensus.ProtocolAtomV3,
		},
	}

	if err := chain.BindActivationPolicyV1(
		verifiedActivationFixtureV1(t, policy),
	); err != nil {
		t.Fatalf(
			"bind AtomV3 activation: %v",
			err,
		)
	}

	// Height 1 funds owner.
	// Heights 2..11 use unique miner addresses so repeated
	// legacy coinbase transactions cannot share the same txid.
	if _, err := chain.MineBlock(
		owner.GetAddress(),
		1,
	); err != nil {
		t.Fatalf(
			"mine owner funding block: %v",
			err,
		)
	}

	for height := 2; height < activationHeight; height++ {
		filler := wallet.NewWallet()

		if _, err := chain.MineBlock(
			filler.GetAddress(),
			1,
		); err != nil {
			t.Fatalf(
				"mine legacy filler block %d: %v",
				height,
				err,
			)
		}
	}

	if got := chain.GetBestHeight(); got != activationHeight-1 {
		t.Fatalf(
			"best height=%d want=%d",
			got,
			activationHeight-1,
		)
	}

	if got := chain.ProtocolAtHeightV1(
		activationHeight,
	); got != qconsensus.ProtocolAtomV3 {
		t.Fatalf(
			"protocol at height %d=%q want=%q",
			activationHeight,
			got,
			qconsensus.ProtocolAtomV3,
		)
	}

	oldBC := bc
	oldRuntime := nodeRuntime

	bc = chain

	// BroadcastP2P is nil-receiver safe. Keep networking
	// disabled for this integration contract.
	nodeRuntime = nil

	t.Cleanup(func() {
		bc = oldBC
		nodeRuntime = oldRuntime
	})

	payload, err := json.Marshal(
		SendRequest{
			From: owner.GetAddress(),
			To:   receiver.GetAddress(),

			Amount: FlexibleAmount(
				"0.000001",
			),

			PrivHex: owner.ExportPrivateKeyHex(),
		},
	)
	if err != nil {
		t.Fatalf(
			"marshal send payload: %v",
			err,
		)
	}

	handler := newActualMuxTestHandlerV1(t)

	request := newActualMuxRequestV1(
		http.MethodPost,
		"/api/tx/send",
		bytes.NewReader(payload),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	addActualMuxTokenV1(request)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status=%d want=%d body=%q",
			recorder.Code,
			http.StatusOK,
			recorder.Body.String(),
		)
	}

	var response SendResponse

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatalf(
			"decode send response: %v body=%q",
			err,
			recorder.Body.String(),
		)
	}

	if !response.Success {
		t.Fatalf(
			"send failed: %+v",
			response,
		)
	}

	if response.TxID == "" {
		t.Fatal("successful send returned empty txid")
	}

	if len(chain.PendingTxs()) != 1 {
		t.Fatalf(
			"pending transactions=%d want=1",
			len(chain.PendingTxs()),
		)
	}

	tx := chain.PendingTxs()[0]
	if tx == nil {
		t.Fatal("pending transaction is nil")
	}

	if got := hex.EncodeToString(tx.ID); got != response.TxID {
		t.Fatalf(
			"response txid=%q pending txid=%q",
			response.TxID,
			got,
		)
	}

	receiverHash := wallet.HashPubKey(
		receiver.PublicKey,
	)

	recipientOutputs := 0
	recipientAmount := 0

	for _, output := range tx.Outputs {
		if bytes.Equal(
			output.PubKeyHash,
			receiverHash,
		) {
			recipientOutputs++
			recipientAmount += output.Amount
		}
	}

	if recipientOutputs != 1 {
		t.Fatalf(
			"recipient outputs=%d want=1",
			recipientOutputs,
		)
	}

	if recipientAmount != 1 {
		t.Fatalf(
			"recipient amount=%d want=1 atom",
			recipientAmount,
		)
	}

	if tx.Amount != 1 {
		t.Fatalf(
			"transaction wire amount=%v want=1 atom",
			tx.Amount,
		)
	}
}
