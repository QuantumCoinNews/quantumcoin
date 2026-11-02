package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"net"
	"quantumcoin/blockchain"
	"quantumcoin/consensus"
	"reflect"
	"testing"
	"time"
)

type nftPeerContextFixtureV1 struct {
	local, peer *blockchain.Blockchain
	activation  consensus.ActivationPolicyV1
	trust       consensus.NFTAwardTrustV1
	program     consensus.NFTMonthlyProgramPolicyV1
	record      blockchain.NFTMintRecordV1
	keys        []*ecdsa.PrivateKey
}

func newNFTPeerContextFixtureV1(t *testing.T) nftPeerContextFixtureV1 {
	t.Helper()
	f := nftPeerContextFixtureV1{local: blockchain.NewBlockchain(1, 0)}
	network := "test-only-peer-nft-context"
	genesis := hex.EncodeToString(f.local.Blocks[0].Hash)
	f.activation = verifiedActivationFixtureV1(t, consensus.ActivationPolicyV1{Enabled: true, Plan: consensus.ActivationPlan{
		NetworkID: network, LegacyGenesisHex: genesis, ActivationHeight: 1, TargetProtocol: consensus.ProtocolNFTV4,
		PreserveHistory: true, PreserveBalances: true,
	}})
	f.trust = consensus.NFTAwardTrustV1{NetworkID: network, LegacyGenesisHex: genesis, AdminQuorum: 2}
	for i := 0; i < 2; i++ {
		key, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		pub, err := consensus.UncompressedPublicKeyV1(key)
		if err != nil {
			t.Fatal(err)
		}
		f.keys = append(f.keys, key)
		f.trust.AdminPublicKeys = append(f.trust.AdminPublicKeys, pub)
	}
	f.program = consensus.NFTMonthlyProgramPolicyV1{NetworkID: network, CampaignID: "telegram-monthly", FirstAwardPeriod: time.Now().UTC().Format("2006-01")}
	bind := func(chain *blockchain.Blockchain) {
		if err := chain.BindActivationPolicySynchronizedV1(f.activation); err != nil {
			t.Fatal(err)
		}
		if err := chain.BindLocalNFTAwardTrustSynchronizedV1(network, f.trust); err != nil {
			t.Fatal(err)
		}
		if err := chain.BindLocalNFTMonthlyProgramSynchronizedV1(f.program); err != nil {
			t.Fatal(err)
		}
	}
	bind(f.local)
	f.peer = blockchain.DeserializeBlockchain(blockchain.SerializeBlockchain(f.local))
	bind(f.peer)
	f.record = blockchain.NFTMintRecordV1{NetworkID: network, CampaignID: f.program.CampaignID, AwardPeriod: f.program.FirstAwardPeriod,
		Serial: 1, RecipientPubKeyHash: bytes.Repeat([]byte{0x11}, 20), MetadataHash: [32]byte{0x44}}
	return f
}

func nftPeerContextBlockV1(t *testing.T, f nftPeerContextFixtureV1, keys []*ecdsa.PrivateKey) *blockchain.Block {
	t.Helper()
	commitment, err := blockchain.NFTMintRecordDigestV1(f.record)
	if err != nil {
		t.Fatal(err)
	}
	decision := consensus.NFTAwardDecisionV1{NetworkID: f.record.NetworkID, LegacyGenesisHex: f.trust.LegacyGenesisHex, RecordDigest: commitment}
	digest, err := consensus.NFTAwardDecisionDigestV1(decision)
	if err != nil {
		t.Fatal(err)
	}
	proof := blockchain.NFTAwardProofV1{Decision: decision}
	for _, key := range keys {
		pub, err := consensus.UncompressedPublicKeyV1(key)
		if err != nil {
			t.Fatal(err)
		}
		sig, err := consensus.SignTransactionDigestV1(key, digest)
		if err != nil {
			t.Fatal(err)
		}
		proof.Approvals = append(proof.Approvals, consensus.NFTAwardApprovalV1{PublicKey: pub, Signature: sig})
	}
	reward, err := f.peer.ExpectedSubsidyUnitsForHeightV1(1)
	if err != nil {
		t.Fatal(err)
	}
	tx := nftPeerContextCoinbaseV1(reward)
	difficulty, err := f.peer.ExpectedNextDifficultyV1()
	if err != nil {
		t.Fatal(err)
	}
	block, err := blockchain.NewNFTBlockWithAwardProofV1(1, []*blockchain.Transaction{tx}, f.peer.Blocks[0].Hash,
		"test-only-peer-nft-miner", difficulty, []blockchain.NFTMintRecordV1{f.record}, []blockchain.NFTAwardProofV1{proof})
	if err != nil {
		t.Fatal(err)
	}
	if err := blockchain.ValidateBlockPoWV4(block); err != nil {
		t.Fatal(err)
	}
	return block
}

func nftPeerContextCoinbaseV1(amount int) *blockchain.Transaction {
	tx := &blockchain.Transaction{Inputs: []blockchain.TransactionInput{}, Outputs: []blockchain.TransactionOutput{{Amount: amount, PubKeyHash: []byte("peer-context-miner")}},
		Timestamp: time.Now(), Sender: "COINBASE", Amount: float64(amount)}
	tx.ID = tx.Hash()
	return tx
}

func nftPeerContextReceiveV1(t *testing.T, local, peer *blockchain.Blockchain) {
	t.Helper()
	src, other := net.Pipe()
	defer src.Close()
	defer other.Close()
	remote := src.RemoteAddr().String()
	setHandshakeOK(remote)
	defer func() { handshakeMu.Lock(); delete(handshakeOK, remote); handshakeMu.Unlock() }()
	handleMessage(Message{Type: MsgChain, Data: blockchain.SerializeBlockchain(peer)}, local, src)
}

func TestNFTPeerContextAcceptsLocallyApprovedChainV1(t *testing.T) {
	f := newNFTPeerContextFixtureV1(t)
	block := nftPeerContextBlockV1(t, f, f.keys)
	if err := f.peer.AddBlockFromPeerSynchronizedV2(block); err != nil {
		t.Fatal(err)
	}
	if !f.peer.IsValidChainSynchronized() {
		t.Fatal("approved source fixture invalid")
	}
	nftPeerContextReceiveV1(t, f.local, f.peer)
	if f.local.BestHeightSnapshot() != 1 {
		t.Fatal("QMA0068_P2P_CONTEXT_RED: actual MsgChain handler rejected locally approved NFT chain")
	}
	got := f.local.AllBlocksSnapshot()[1]
	if !bytes.Equal(got.Hash, block.Hash) || !reflect.DeepEqual(got.NFTMintRecordsV1, block.NFTMintRecordsV1) || !reflect.DeepEqual(got.NFTAwardProofsV1, block.NFTAwardProofsV1) {
		t.Fatal("actual handler lost approved recipient/proof evidence")
	}
	if !f.local.IsValidChainSynchronized() || f.local.ProtocolAtHeightV1(1) != consensus.ProtocolNFTV4 {
		t.Fatal("handler changed local validation policy")
	}
}

func TestNFTPeerContextRejectsUnknownSignersV1(t *testing.T) {
	f := newNFTPeerContextFixtureV1(t)
	var rogue []*ecdsa.PrivateKey
	rogueTrust := f.trust
	rogueTrust.AdminPublicKeys = nil
	for i := 0; i < 2; i++ {
		key, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		pub, err := consensus.UncompressedPublicKeyV1(key)
		if err != nil {
			t.Fatal(err)
		}
		rogue = append(rogue, key)
		rogueTrust.AdminPublicKeys = append(rogueTrust.AdminPublicKeys, pub)
	}
	block := nftPeerContextBlockV1(t, f, rogue)
	proof := block.NFTAwardProofsV1[0]
	if err := blockchain.ValidateNFTAwardRecordApprovalV1(f.record, proof.Decision, rogueTrust, proof.Approvals); err != nil {
		t.Fatal(err)
	}
	f.peer.Blocks = append(f.peer.Blocks, block) // Attacker payload: valid PoW, self-declared signers.
	nftPeerContextReceiveV1(t, f.local, f.peer)
	if f.local.BestHeightSnapshot() != 0 {
		t.Fatal("peer's self-declared NFT signers replaced local authority")
	}
}

func TestNFTPeerContextRejectsIncompleteLocalPolicyV1(t *testing.T) {
	for _, name := range []string{"no_local_policy", "activation_and_authorities_without_program"} {
		t.Run(name, func(t *testing.T) {
			f := newNFTPeerContextFixtureV1(t)
			fresh := blockchain.DeserializeBlockchain(blockchain.SerializeBlockchain(f.local))
			if name != "no_local_policy" {
				if err := fresh.BindActivationPolicySynchronizedV1(f.activation); err != nil {
					t.Fatal(err)
				}
				if err := fresh.BindLocalNFTAwardTrustSynchronizedV1(f.trust.NetworkID, f.trust); err != nil {
					t.Fatal(err)
				}
			}
			block := nftPeerContextBlockV1(t, f, f.keys)
			if err := f.peer.AddBlockFromPeerSynchronizedV2(block); err != nil {
				t.Fatal(err)
			}
			nftPeerContextReceiveV1(t, fresh, f.peer)
			if fresh.BestHeightSnapshot() != 0 {
				t.Fatal("peer payload supplied missing local program/activation")
			}
		})
	}
}

func TestNFTPeerContextPreservesLegacyChainAcceptanceV1(t *testing.T) {
	local := blockchain.NewBlockchain(1, 0)
	peer := blockchain.DeserializeBlockchain(blockchain.SerializeBlockchain(local))
	reward, err := peer.ExpectedSubsidyUnitsForHeightV1(1)
	if err != nil {
		t.Fatal(err)
	}
	difficulty, err := peer.ExpectedNextDifficultyV1()
	if err != nil {
		t.Fatal(err)
	}
	block, err := blockchain.NewBlockAtProtocolV1(1, []*blockchain.Transaction{nftPeerContextCoinbaseV1(reward)}, peer.Blocks[0].Hash,
		"test-only-legacy-context-miner", difficulty, nil, consensus.ProtocolLegacyV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := peer.AddBlockFromPeerSynchronizedV2(block); err != nil {
		t.Fatal(err)
	}
	nftPeerContextReceiveV1(t, local, peer)
	if local.BestHeightSnapshot() != 1 || !local.IsValidChainSynchronized() {
		t.Fatal("local default LegacyV1 clean chain rejected")
	}
}
