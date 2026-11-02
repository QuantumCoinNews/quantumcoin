package p2p

import (
	"encoding/hex"
	"net"
	"quantumcoin/blockchain"
	"quantumcoin/consensus"
	"quantumcoin/wallet"
	"testing"
)

func TestHardforkPeerChainUsesLocalVerifiedActivationV1(t *testing.T) {
	local := blockchain.NewBlockchain(blockchain.CanonicalInitialMiningRewardV1, int(blockchain.QCTotalSupplyHardCap))
	peerChain := blockchain.DeserializeBlockchain(blockchain.SerializeBlockchain(local))
	p := verifiedActivationFixtureV1(t, consensus.ActivationPolicyV1{Enabled: true, Plan: consensus.ActivationPlan{
		NetworkID: "test-only-peer-activation", LegacyGenesisHex: hex.EncodeToString(local.Blocks[0].Hash),
		ActivationHeight: 1, TargetProtocol: consensus.ProtocolAtomV3, PreserveHistory: true, PreserveBalances: true,
	}})
	if err := local.BindActivationPolicyV1(p); err != nil {
		t.Fatal(err)
	}
	if err := peerChain.BindActivationPolicyV1(p); err != nil {
		t.Fatal(err)
	}
	difficulty, err := peerChain.ExpectedNextDifficultyV1()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := peerChain.MineBlock(wallet.NewWallet().GetAddress(), difficulty); err != nil {
		t.Fatal(err)
	}
	if !peerChain.IsValidChainSynchronized() {
		t.Fatal("fixture signed chain is invalid")
	}
	src, other := net.Pipe()
	defer src.Close()
	defer other.Close()
	remote := src.RemoteAddr().String()
	setHandshakeOK(remote)
	defer func() { handshakeMu.Lock(); delete(handshakeOK, remote); handshakeMu.Unlock() }()
	handleMessage(Message{Type: MsgChain, Data: blockchain.SerializeBlockchain(peerChain)}, local, src)
	if got := local.BestHeightSnapshot(); got != 1 {
		t.Fatalf("valid signed-protocol peer chain rejected: height=%d", got)
	}
	if local.ProtocolAtHeightV1(1) != consensus.ProtocolAtomV3 {
		t.Fatal("peer chain changed local activation")
	}
}
