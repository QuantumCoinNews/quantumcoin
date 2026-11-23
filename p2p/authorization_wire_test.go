package p2p

import (
	"bytes"
	"encoding/gob"
	"net"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestP2PAuthorizationOverTCPRejectsForgeryAndDoubleSpend(t *testing.T) {
	t.Chdir(t.TempDir())
	cfg := config.Default()
	cfg.GenesisProfile = config.NetworkGenesisProfileV4
	chain, err := blockchain.NewNetworkBlockchainV4(cfg, config.QCNetworkProfileTestnetLocalV1)
	if err != nil {
		t.Fatal(err)
	}
	chain.SetCoinbaseMaturity(0) // isolated signature fixture; maturity has separate tests
	owner, recipient, attacker := wallet.NewWallet(), wallet.NewWallet(), wallet.NewWallet()
	if _, err = chain.MineBlock(owner.GetAddress(), 8); err != nil {
		t.Fatal(err)
	}
	makeSpend := func(from, to string, amount int) *blockchain.Transaction {
		tx, e := blockchain.NewTransaction(from, to, amount, chain)
		if e != nil {
			t.Fatal(e)
		}
		return tx
	}
	valid := makeSpend(owner.GetAddress(), recipient.GetAddress(), 5)
	conflict := makeSpend(owner.GetAddress(), recipient.GetAddress(), 6)
	forged := makeSpend(owner.GetAddress(), recipient.GetAddress(), 7)
	unsigned := makeSpend(owner.GetAddress(), recipient.GetAddress(), 8)
	theft := makeSpend(owner.GetAddress(), attacker.GetAddress(), 9)
	for _, tx := range []*blockchain.Transaction{valid, conflict, forged} {
		if err = chain.SignTransaction(tx, owner.PrivateKey); err != nil {
			t.Fatal(err)
		}
	}
	forged.Inputs[0].Signature[0] ^= 1
	theft.Sender = attacker.GetAddress()
	if err = chain.SignTransaction(theft, attacker.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if !chain.VerifyTransaction(theft) {
		t.Fatal("probe must have valid attacker crypto")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() {
		conn, e := listener.Accept()
		if e != nil {
			done <- e
			return
		}
		registerPeer(conn)
		HandleConnection(conn, chain)
		done <- nil
	}()
	client, err := net.DialTimeout("tcp", listener.Addr().String(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = listener.Close()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("peer did not stop")
		}
	})
	if err = client.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		t.Fatal(err)
	}
	encoder, decoder := gob.NewEncoder(client), gob.NewDecoder(client)
	send := func(message Message) {
		if e := encoder.Encode(message); e != nil {
			t.Fatal(e)
		}
	}
	barrier := func() {
		send(PingMessage())
		for i := 0; i < 10; i++ {
			var message Message
			if e := decoder.Decode(&message); e != nil {
				t.Fatal(e)
			}
			if message.Type == MsgPong {
				return
			}
		}
		t.Fatal("peer did not acknowledge processing barrier")
	}
	var hello bytes.Buffer
	// The supported zero sync capability avoids requesting unrelated chain data;
	// the same V4 genesis identity and handshake validation remain mandatory.
	if err = gob.NewEncoder(&hello).Encode(Handshake{Magic: networkMagicForChainV1(chain), Version: ProtocolVersion, SyncVersion: 0}); err != nil {
		t.Fatal(err)
	}
	send(Message{Type: MsgHello, Data: hello.Bytes()})
	barrier()
	for _, tx := range []*blockchain.Transaction{unsigned, forged, theft} {
		send(TxMessage(tx))
		barrier()
		if len(chain.PendingTransactionsSnapshot()) != 0 {
			t.Fatal("wire path accepted unauthorized spend")
		}
	}
	send(TxMessage(valid))
	barrier()
	pending := chain.PendingTransactionsSnapshot()
	if len(pending) != 1 || !bytes.Equal(pending[0].ID, valid.ID) {
		t.Fatal("wire path rejected positive control")
	}
	for _, tx := range []*blockchain.Transaction{conflict, valid} {
		send(TxMessage(tx))
		barrier()
		remaining := chain.PendingTransactionsSnapshot()
		if len(remaining) != 1 || !bytes.Equal(remaining[0].ID, valid.ID) {
			t.Fatal("wire path accepted conflict or replay")
		}
	}
	if chain.BestHeightSnapshot() != 1 || chain.BalanceSnapshot(attacker.GetAddress()) != 0 {
		t.Fatal("wire admission mutated confirmed balances")
	}
}
