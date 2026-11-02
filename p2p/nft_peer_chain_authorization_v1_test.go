package p2p

import (
	"bytes"
	"net"
	"reflect"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/consensus"
	"quantumcoin/wallet"
)

func TestP2PChainRejectsUnauthorizedNFTRecordsV1(t *testing.T) {
	previousSecurity := DefaultPeerSecurity
	DefaultPeerSecurity = nil
	t.Cleanup(func() { DefaultPeerSecurity = previousSecurity })

	for _, location := range []string{"genesis", "new_block"} {
		t.Run(location, func(t *testing.T) {
			local := blockchain.NewBlockchain(
				blockchain.CanonicalInitialMiningRewardV1,
				int(blockchain.QCTotalSupplyHardCap),
			)
			if err := local.BindCanonicalActivationPolicyV1(); err != nil {
				t.Fatalf("bind canonical local policy: %v", err)
			}
			if local.ProtocolAtHeightV1(1) != consensus.ProtocolLegacyV1 {
				t.Fatal("fixture requires the currently disabled canonical activation")
			}

			clone := func() *blockchain.Blockchain {
				t.Helper()
				copied := blockchain.DeserializeBlockchain(
					blockchain.SerializeBlockchain(local),
				)
				if err := copied.BindCanonicalActivationPolicyV1(); err != nil {
					t.Fatalf("bind canonical fixture policy: %v", err)
				}
				return copied
			}

			control := clone()
			peerChain := clone()
			difficulty, err := peerChain.ExpectedNextDifficultyV1()
			if err != nil {
				t.Fatalf("resolve difficulty: %v", err)
			}
			if _, err := peerChain.MineBlock(
				wallet.NewWallet().GetAddress(), difficulty,
			); err != nil {
				t.Fatalf("mine clean peer baseline: %v", err)
			}
			if !peerChain.IsValidChainSynchronized() {
				t.Fatal("clean peer fixture failed validation")
			}

			src, other := net.Pipe()
			defer src.Close()
			defer other.Close()
			remote := src.RemoteAddr().String()

			handshakeMu.Lock()
			previousHandshake, existed := handshakeOK[remote]
			handshakeOK[remote] = true
			handshakeMu.Unlock()
			defer func() {
				handshakeMu.Lock()
				defer handshakeMu.Unlock()
				if existed {
					handshakeOK[remote] = previousHandshake
				} else {
					delete(handshakeOK, remote)
				}
			}()

			handleMessage(Message{
				Type: MsgChain,
				Data: blockchain.SerializeBlockchain(peerChain),
			}, control, src)

			if control.BestHeightSnapshot() != 1 {
				t.Fatal("clean canonical peer chain was rejected by the actual handler")
			}
			actual := control.AllBlocksSnapshot()
			expected := peerChain.AllBlocksSnapshot()
			if len(actual) != len(expected) {
				t.Fatal("clean acceptance changed block count")
			}
			for index, want := range expected {
				got := actual[index]
				if got == nil || want == nil {
					t.Fatalf("nil baseline block at index %d", index)
				}
				if got.Index != want.Index ||
					got.Timestamp != want.Timestamp ||
					got.Nonce != want.Nonce ||
					got.Difficulty != want.Difficulty ||
					got.Miner != want.Miner ||
					!bytes.Equal(got.Hash, want.Hash) ||
					!bytes.Equal(got.PrevHash, want.PrevHash) ||
					!bytes.Equal(got.HashTransactions(), want.HashTransactions()) {
					t.Fatalf("clean acceptance changed canonical block content at index %d", index)
				}
				gotNFT, err := blockchain.NFTMintRecordsDigestV1(got.NFTMintRecordsV1)
				if err != nil {
					t.Fatalf("accepted NFT digest: %v", err)
				}
				wantNFT, err := blockchain.NFTMintRecordsDigestV1(want.NFTMintRecordsV1)
				if err != nil {
					t.Fatalf("expected NFT digest: %v", err)
				}
				if gotNFT != wantNFT {
					t.Fatalf("clean acceptance changed NFT records at index %d", index)
				}
			}

			attackedBlock := peerChain.Blocks[0]
			if location == "new_block" {
				attackedBlock = peerChain.Blocks[1]
			}

			hashBefore, err := blockchain.ComputePoWHashV1(
				attackedBlock, attackedBlock.Nonce,
			)
			if err != nil {
				t.Fatalf("baseline header hash: %v", err)
			}

			attackedBlock.NFTMintRecordsV1 = []blockchain.NFTMintRecordV1{
				{
					NetworkID:           "test-only-p2p-nft",
					CampaignID:          "telegram-monthly",
					AwardPeriod:         "2026-10",
					Serial:              1,
					RecipientPubKeyHash: bytes.Repeat([]byte{0x44}, 20),
					MetadataHash:        [32]byte{0x7a},
				},
			}

			hashAfter, err := blockchain.ComputePoWHashV1(
				attackedBlock, attackedBlock.Nonce,
			)
			if err != nil || hashBefore != hashAfter {
				t.Fatalf("fixture unexpectedly changed legacy header hash: %v", err)
			}
			if peerChain.IsValidChainSynchronized() {
				t.Fatal("unauthorized NFT fixture passed chain validation")
			}

			before := local.AllBlocksSnapshot()
			handleMessage(Message{
				Type: MsgChain,
				Data: blockchain.SerializeBlockchain(peerChain),
			}, local, src)

			if !reflect.DeepEqual(before, local.AllBlocksSnapshot()) {
				t.Fatal("unauthorized NFT message changed the local chain")
			}
			if local.BestHeightSnapshot() != 0 {
				t.Fatal("unauthorized NFT peer chain was accepted")
			}
			if local.ProtocolAtHeightV1(1) != consensus.ProtocolLegacyV1 {
				t.Fatal("peer message changed the local activation policy")
			}
		})
	}
}
