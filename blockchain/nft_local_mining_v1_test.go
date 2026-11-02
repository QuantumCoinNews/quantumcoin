package blockchain

import (
	"bytes"
	"reflect"
	"sync"
	"testing"
	"time"

	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
)

// Interface assertion lets the diagnostic test compile before the new API is
// installed; an absent entry is a test failure, never a compilation failure.
type nftLocalMiningForTestV1 interface {
	MineBlockWithNFTAwardSynchronizedV1(string, int, NFTMintRecordV1, NFTAwardProofV1) (*Block, error)
}

func nftLocalMiningEntryForTestV1(t *testing.T, bc *Blockchain) nftLocalMiningForTestV1 {
	t.Helper()
	entry, ok := any(bc).(nftLocalMiningForTestV1)
	if !ok {
		t.Fatal("local approved NFT mining entry is missing")
	}
	return entry
}

func TestNFTLocalMiningAllowsActivatedEmptyBlockV1(t *testing.T) {
	bc, _, _ := nftLiveAdmissionFixtureV1(t, 1)
	before := bc.TotalMinted()
	expectedQC := bc.ExpectedSubsidyForHeightV1(1)
	expectedUnits, err := bc.ExpectedSubsidyUnitsForHeightV1(1)
	if err != nil {
		t.Fatal(err)
	}
	block, err := bc.MineBlock(wallet.NewWallet().GetAddress(), testExpectedNextDifficultyV1(t, bc))
	if err != nil {
		t.Fatalf("QMA0068_MINING_RED: verified NFTV4 local mining rejected: %v", err)
	}
	if len(block.NFTMintRecordsV1) != 0 || len(block.NFTAwardProofsV1) != 0 {
		t.Fatal("ordinary mining automatically awarded an NFT")
	}
	_, paid, err := blockCoinbaseTotalV1(block, false)
	if err != nil || paid != expectedUnits || bc.TotalMinted() != before+expectedQC {
		t.Fatalf("canonical NFTV4 subsidy changed: paid=%d error=%v", paid, err)
	}
	if err := ValidateBlockPoWV4(block); err != nil {
		t.Fatal(err)
	}
	if !bc.IsValidChainSynchronized() {
		t.Fatal("ordinary NFTV4 mined chain is invalid")
	}
}

func TestNFTLocalMiningCommitsApprovedAwardAndOneAtomFeeV1(t *testing.T) {
	bc, p, keys := nftLiveAdmissionFixtureV1(t, 2)
	bc.SetCoinbaseMaturity(0)
	sender, receiver, miner := wallet.NewWallet(), wallet.NewWallet(), wallet.NewWallet()
	// A real legacy block funds the spend before the verified NFTV4 height.
	funding, err := bc.MineBlock(sender.GetAddress(), testExpectedNextDifficultyV1(t, bc))
	if err != nil {
		t.Fatal(err)
	}
	fundingTx := funding.Transactions[0]
	inputUnits, err := qconsensus.WholeQCAmountToProtocolUnitsV1(qconsensus.ProtocolNFTV4, int64(fundingTx.Outputs[0].Amount))
	if err != nil {
		t.Fatal(err)
	}
	spend := &Transaction{Inputs: []TransactionInput{{TxID: cloneBytesV1(fundingTx.ID), OutIndex: 0}},
		Outputs:   []TransactionOutput{{Amount: int(inputUnits) - 1, PubKeyHash: wallet.HashPubKey(receiver.PublicKey)}},
		Timestamp: time.Now().UTC(), Sender: sender.GetAddress(), Amount: float64(inputUnits - 1)}
	spend.ID = spend.Hash()
	if err := spend.Sign(sender.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if err := bc.AddTransaction(spend); err != nil {
		t.Fatal(err)
	}
	fees, err := bc.CalculateBlockFeesSnapshot(bc.PendingTxs())
	if err != nil || fees != 1 {
		t.Fatalf("real pending fee=%d want=1 atom error=%v", fees, err)
	}
	peer := DeserializeBlockchain(SerializeBlockchain(bc))
	nftLiveAdmissionRebindV1(t, peer, bc)
	peer.SetCoinbaseMaturity(0)
	record := nftMonthlyPolicyRecordV1(p, 0)
	proof := nftMonthlyPolicyBlockV1(t, bc, []NFTMintRecordV1{record}, keys).NFTAwardProofsV1[0]
	entry := nftLocalMiningEntryForTestV1(t, bc)
	before, expectedQC := bc.TotalMinted(), bc.ExpectedSubsidyForHeightV1(2)
	expectedUnits, err := bc.ExpectedSubsidyUnitsForHeightV1(2)
	if err != nil {
		t.Fatal(err)
	}
	bad := record
	bad.RecipientPubKeyHash = cloneBytesV1(record.RecipientPubKeyHash)
	bad.RecipientPubKeyHash[0] ^= 1
	if _, err := entry.MineBlockWithNFTAwardSynchronizedV1(miner.GetAddress(), testExpectedNextDifficultyV1(t, bc), bad, proof); err == nil {
		t.Fatal("changed recipient accepted with the original approval")
	}
	if bc.BestHeightSnapshot() != 1 || bc.TotalMinted() != before || len(bc.PendingTxs()) != 1 {
		t.Fatal("failed award changed the chain or cleared the pending spend")
	}
	block, err := entry.MineBlockWithNFTAwardSynchronizedV1(miner.GetAddress(), testExpectedNextDifficultyV1(t, bc), record, proof)
	if err != nil {
		t.Fatal(err)
	}
	_, paid, err := blockCoinbaseTotalV1(block, false)
	if err != nil || paid != expectedUnits+1 || bc.TotalMinted() != before+expectedQC {
		t.Fatalf("NFT mining changed canonical fee/subsidy: paid=%d expected=%d error=%v", paid, expectedUnits+1, err)
	}
	if len(bc.PendingTxs()) != 0 || len(block.Transactions) != 2 || len(block.NFTMintRecordsV1) != 1 || len(block.NFTAwardProofsV1) != 1 {
		t.Fatal("accepted mining did not commit the spend and one approved award")
	}
	if err := peer.AddBlockFromPeerSynchronizedV2(block); err != nil {
		t.Fatalf("locally mined award rejected by canonical peer gate: %v", err)
	}
	saved := bc.AllBlocksSnapshot()[2]
	if !bytes.Equal(saved.NFTMintRecordsV1[0].RecipientPubKeyHash, record.RecipientPubKeyHash) || !reflect.DeepEqual(saved.NFTAwardProofsV1[0], proof) {
		t.Fatal("selected recipient or approvals lost")
	}
	record.RecipientPubKeyHash[0] ^= 1
	proof.Approvals[0].Signature[0] ^= 1
	block.NFTMintRecordsV1[0].RecipientPubKeyHash[0] ^= 1
	block.NFTAwardProofsV1[0].Approvals[0].PublicKey[0] ^= 1
	block.Transactions[0].Outputs[0].Amount++
	block.Hash[0] ^= 1
	if !reflect.DeepEqual(saved, bc.AllBlocksSnapshot()[2]) || !bc.IsValidChainSynchronized() {
		t.Fatal("caller evidence/returned block mutation altered canonical chain")
	}
}

func TestNFTLocalMiningRejectsInvalidAwardOrContextAtomicallyV1(t *testing.T) {
	cases := []string{"no_activation", "activation_only", "activation_and_trust", "unknown_signers", "quorum_short", "changed_recipient", "wrong_campaign", "wrong_month", "wrong_difficulty"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			source, p, keys := nftLiveAdmissionFixtureV1(t, 1)
			target := source
			if name == "no_activation" || name == "activation_only" || name == "activation_and_trust" {
				target = DeserializeBlockchain(SerializeBlockchain(source))
				if name != "no_activation" {
					if err := target.BindActivationPolicySynchronizedV1(source.activationPolicy.policy); err != nil {
						t.Fatal(err)
					}
				}
				if name == "activation_and_trust" {
					if err := target.BindLocalNFTAwardTrustSynchronizedV1(p.NetworkID, source.nftAwardTrust.trust); err != nil {
						t.Fatal(err)
					}
				}
			}
			record := nftMonthlyPolicyRecordV1(p, 0)
			if name == "wrong_campaign" {
				record.CampaignID = "different-campaign"
			}
			if name == "wrong_month" {
				record = nftMonthlyPolicyRecordV1(p, 1)
			}
			if name == "unknown_signers" {
				_, _, keys = nftLiveAdmissionFixtureV1(t, 1)
			}
			proof := nftMonthlyPolicyBlockV1(t, source, []NFTMintRecordV1{record}, keys).NFTAwardProofsV1[0]
			if name == "quorum_short" {
				proof.Approvals = proof.Approvals[:1]
			}
			if name == "changed_recipient" {
				record.RecipientPubKeyHash[0] ^= 1
			}
			difficulty := testExpectedNextDifficultyV1(t, target)
			if name == "wrong_difficulty" {
				if difficulty > 1 {
					difficulty--
				} else {
					difficulty++
				}
			}
			before, beforeMinted := target.AllBlocksSnapshot(), target.TotalMinted()
			_, err := nftLocalMiningEntryForTestV1(t, target).MineBlockWithNFTAwardSynchronizedV1(wallet.NewWallet().GetAddress(), difficulty, record, proof)
			if err == nil {
				t.Fatal("invalid local award/context accepted")
			}
			if !reflect.DeepEqual(before, target.AllBlocksSnapshot()) || target.TotalMinted() != beforeMinted {
				t.Fatal("rejected award changed canonical state")
			}
		})
	}
}

func TestNFTLocalMiningSerializesSamePeriodAwardAdmissionV1(t *testing.T) {
	bc, p, keys := nftLiveAdmissionFixtureV1(t, 1)
	entry := nftLocalMiningEntryForTestV1(t, bc)
	difficulty := testExpectedNextDifficultyV1(t, bc)
	before, expectedQC := bc.TotalMinted(), bc.ExpectedSubsidyForHeightV1(1)
	records := []NFTMintRecordV1{nftMonthlyPolicyRecordV1(p, 0), nftMonthlyPolicyRecordV1(p, 0)}
	records[1].RecipientPubKeyHash[0] ^= 1
	proofs := []NFTAwardProofV1{
		nftMonthlyPolicyBlockV1(t, bc, records[:1], keys).NFTAwardProofsV1[0],
		nftMonthlyPolicyBlockV1(t, bc, records[1:], keys).NFTAwardProofsV1[0],
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	miner := wallet.NewWallet().GetAddress()
	var workers sync.WaitGroup
	for i := 0; i < 2; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			<-start
			_, err := entry.MineBlockWithNFTAwardSynchronizedV1(miner, difficulty, records[i], proofs[i])
			results <- err
		}(i)
	}
	close(start)
	workers.Wait()
	close(results)
	accepted, rejected := 0, 0
	for err := range results {
		if err == nil {
			accepted++
		} else {
			rejected++
		}
	}
	if accepted != 1 || rejected != 1 || bc.BestHeightSnapshot() != 1 || bc.TotalMinted() != before+expectedQC {
		t.Fatalf("same month admitted twice: accepted=%d rejected=%d", accepted, rejected)
	}
	if len(bc.AllBlocksSnapshot()[1].NFTMintRecordsV1) != 1 || !bc.IsValidChainSynchronized() {
		t.Fatal("same-month writer race produced invalid awards")
	}
}
