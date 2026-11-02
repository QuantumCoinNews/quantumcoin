package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	qconsensus "quantumcoin/consensus"
)

// Ephemeral keys and verified hardfork releases are test fixtures only.
// Runtime policy dates, authorities and activation remain unconfigured.
func nftLiveAdmissionFixtureV1(t *testing.T, activationHeight int) (*Blockchain, qconsensus.NFTMonthlyProgramPolicyV1, []*ecdsa.PrivateKey) {
	t.Helper()
	chain, _, keys := nftMonthlyPolicyFixtureV1(t)
	now := time.Now().UTC()
	policy := qconsensus.NFTMonthlyProgramPolicyV1{
		NetworkID:        chain.nftAwardTrust.trust.NetworkID,
		CampaignID:       "telegram-monthly",
		FirstAwardPeriod: now.Format("2006-01"),
	}
	if err := chain.BindLocalNFTMonthlyProgramSynchronizedV1(policy); err != nil {
		t.Fatal(err)
	}
	if activationHeight > 0 {
		activation := verifiedActivationFixtureV1(t, qconsensus.ActivationPolicyV1{
			Enabled: true, Plan: qconsensus.ActivationPlan{
				NetworkID: policy.NetworkID, LegacyGenesisHex: hex.EncodeToString(chain.Blocks[0].Hash),
				ActivationHeight: activationHeight, TargetProtocol: qconsensus.ProtocolNFTV4,
				PreserveHistory: true, PreserveBalances: true,
			},
		})
		if err := chain.BindActivationPolicySynchronizedV1(activation); err != nil {
			t.Fatal(err)
		}
	}
	return chain, policy, keys
}

func nftLiveAdmissionBlockV1(t *testing.T, chain *Blockchain, record NFTMintRecordV1, keys []*ecdsa.PrivateKey, extraSubsidy int) *Block {
	t.Helper()
	// Existing helper signs actual record digests. The no-transaction block is
	// only a proof fixture; the returned candidate uses canonical QC coinbase.
	evidence := nftMonthlyPolicyBlockV1(t, chain, []NFTMintRecordV1{record}, keys)
	height := chain.Blocks[len(chain.Blocks)-1].Index + 1
	reward, err := ExpectedSubsidyUnitsFromStateV1(qconsensus.ProtocolNFTV4, height, chain.TotalMinted(), chain.TotalSupply)
	if err != nil {
		t.Fatal(err)
	}
	tx := testPeerCoinbaseTxV2(reward + extraSubsidy)
	block, err := NewNFTBlockWithAwardProofV1(height, []*Transaction{tx}, chain.Blocks[len(chain.Blocks)-1].Hash,
		"test-only-approved-miner", testExpectedNextDifficultyV1(t, chain), []NFTMintRecordV1{record}, evidence.NFTAwardProofsV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateBlockPoWV4(block); err != nil {
		t.Fatal(err)
	}
	return block
}

func nftLiveAdmissionRebindV1(t *testing.T, chain *Blockchain, source *Blockchain) {
	t.Helper()
	if err := chain.BindActivationPolicySynchronizedV1(source.activationPolicy.policy); err != nil {
		t.Fatal(err)
	}
	if err := chain.BindLocalNFTAwardTrustSynchronizedV1(source.nftAwardTrust.trust.NetworkID, cloneNFTAwardTrustV1(source.nftAwardTrust.trust)); err != nil {
		t.Fatal(err)
	}
	if err := chain.BindLocalNFTMonthlyProgramSynchronizedV1(source.nftMonthlyProgram.policy); err != nil {
		t.Fatal(err)
	}
}

func TestNFTLivePeerAdmissionAcceptsLocallyApprovedAwardV1(t *testing.T) {
	chain, policy, keys := nftLiveAdmissionFixtureV1(t, 1)
	record := nftMonthlyPolicyRecordV1(policy, 0)
	block := nftLiveAdmissionBlockV1(t, chain, record, keys, 0)
	if err := chain.ValidateNFTMonthlyAwardBlockSynchronizedV1(block); err != nil {
		t.Fatal(err)
	}
	if err := chain.ValidatePeerBlockSynchronized(block); err != nil {
		t.Fatalf("QMA0068_LIVE_RED: locally approved NFT block rejected: %v", err)
	}
	if err := chain.AddBlockFromPeerSynchronizedV2(block); err != nil {
		t.Fatal(err)
	}
	if chain.BestHeightSnapshot() != 1 || !chain.IsValidChainSynchronized() {
		t.Fatal("accepted award not in valid chain")
	}
	got := chain.AllBlocksSnapshot()[1]
	if !reflect.DeepEqual(got.NFTMintRecordsV1, block.NFTMintRecordsV1) || !reflect.DeepEqual(got.NFTAwardProofsV1, block.NFTAwardProofsV1) {
		t.Fatal("canonical append lost recipient or approval evidence")
	}
}

func TestNFTLiveAdmissionRejectsMissingLocalContextV1(t *testing.T) {
	source, policy, keys := nftLiveAdmissionFixtureV1(t, 1)
	block := nftLiveAdmissionBlockV1(t, source, nftMonthlyPolicyRecordV1(policy, 0), keys, 0)
	for _, name := range []string{"unbound", "activation_only", "trust_only", "activation_and_trust"} {
		t.Run(name, func(t *testing.T) {
			chain := DeserializeBlockchain(SerializeBlockchain(source))
			if name == "activation_only" || name == "activation_and_trust" {
				if err := chain.BindActivationPolicySynchronizedV1(source.activationPolicy.policy); err != nil {
					t.Fatal(err)
				}
			}
			if name == "trust_only" || name == "activation_and_trust" {
				if err := chain.BindLocalNFTAwardTrustSynchronizedV1(policy.NetworkID, cloneNFTAwardTrustV1(source.nftAwardTrust.trust)); err != nil {
					t.Fatal(err)
				}
			}
			if err := chain.AddBlockFromPeerSynchronizedV2(block); err == nil {
				t.Fatal("incomplete local context accepted award")
			}
			if chain.BestHeightSnapshot() != 0 {
				t.Fatal("rejection changed chain")
			}
		})
	}
}

func TestNFTLiveAdmissionRejectsBeforeActivationAndWrongActivationNetworkV1(t *testing.T) {
	chain, policy, keys := nftLiveAdmissionFixtureV1(t, 2)
	block := nftLiveAdmissionBlockV1(t, chain, nftMonthlyPolicyRecordV1(policy, 0), keys, 0)
	if err := chain.AddBlockFromPeerSynchronizedV2(block); err == nil {
		t.Fatal("award before NFT activation accepted")
	}
	wrong, p, wrongKeys := nftLiveAdmissionFixtureV1(t, 0)
	activation := verifiedActivationFixtureV1(t, qconsensus.ActivationPolicyV1{Enabled: true, Plan: qconsensus.ActivationPlan{
		NetworkID: "different-activation-network", LegacyGenesisHex: hex.EncodeToString(wrong.Blocks[0].Hash),
		ActivationHeight: 1, TargetProtocol: qconsensus.ProtocolNFTV4, PreserveHistory: true, PreserveBalances: true,
	}})
	if err := wrong.BindActivationPolicySynchronizedV1(activation); err != nil {
		t.Fatal(err)
	}
	candidate := nftLiveAdmissionBlockV1(t, wrong, nftMonthlyPolicyRecordV1(p, 0), wrongKeys, 0)
	if err := wrong.AddBlockFromPeerSynchronizedV2(candidate); err == nil {
		t.Fatal("activation and award network mismatch accepted")
	}
}

func TestNFTLiveAdmissionRejectsValidlySignedWrongMonthScopeAndQuorumV1(t *testing.T) {
	for _, name := range []string{"next_month", "previous_month", "wrong_campaign", "wrong_serial", "self_declared_signers", "missing_proof", "orphan_proof", "quorum_short"} {
		t.Run(name, func(t *testing.T) {
			chain, p, keys := nftLiveAdmissionFixtureV1(t, 1)
			r := nftMonthlyPolicyRecordV1(p, 0)
			switch name {
			case "next_month":
				r = nftMonthlyPolicyRecordV1(p, 1)
			case "previous_month":
				first, _ := time.Parse("2006-01", p.FirstAwardPeriod)
				r.AwardPeriod = first.AddDate(0, -1, 0).Format("2006-01")
			case "wrong_campaign":
				r.CampaignID = "other-campaign"
			case "wrong_serial":
				r.Serial = 2
			case "self_declared_signers":
				_, _, keys = nftLiveAdmissionFixtureV1(t, 1)
			}
			block := nftLiveAdmissionBlockV1(t, chain, r, keys, 0)
			switch name {
			case "missing_proof":
				block.NFTAwardProofsV1 = nil
			case "orphan_proof":
				block.NFTMintRecordsV1 = nil
			case "quorum_short":
				block.NFTAwardProofsV1[0].Approvals = block.NFTAwardProofsV1[0].Approvals[:1]
				// Recommit reduced quorum into a valid PoW; rejection must use local trust.
				txs := block.Transactions
				var err error
				block, err = NewNFTBlockWithAwardProofV1(block.Index, txs, block.PrevHash, block.Miner, block.Difficulty, block.NFTMintRecordsV1, block.NFTAwardProofsV1)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := chain.AddBlockFromPeerSynchronizedV2(block); err == nil {
				t.Fatal("invalid runtime award accepted")
			}
			if chain.BestHeightSnapshot() != 0 {
				t.Fatal("rejection changed height")
			}
		})
	}
}

func TestNFTLiveAdmissionRejectsDoubleAwardAndCopiesAcceptedEvidenceV1(t *testing.T) {
	chain, p, keys := nftLiveAdmissionFixtureV1(t, 1)
	first := nftLiveAdmissionBlockV1(t, chain, nftMonthlyPolicyRecordV1(p, 0), keys, 0)
	if err := chain.AddBlockFromPeerSynchronizedV2(first); err != nil {
		t.Fatal(err)
	}
	before := chain.AllBlocksSnapshot()
	first.NFTMintRecordsV1[0].RecipientPubKeyHash[0] ^= 1
	first.NFTAwardProofsV1[0].Approvals[0].Signature[0] ^= 1
	if !reflect.DeepEqual(chain.AllBlocksSnapshot(), before) {
		t.Fatal("caller mutated accepted mint evidence")
	}
	r := nftMonthlyPolicyRecordV1(p, 0)
	r.RecipientPubKeyHash = bytes.Repeat([]byte{0x22}, 20)
	duplicate := nftLiveAdmissionBlockV1(t, chain, r, keys, 0)
	if err := chain.AddBlockFromPeerSynchronizedV2(duplicate); err == nil {
		t.Fatal("second user in same month accepted")
	}
	if chain.BestHeightSnapshot() != 1 || !chain.IsValidChainSynchronized() {
		t.Fatal("duplicate rejection damaged valid chain")
	}
}

func TestNFTLiveAdmissionKeepsCanonicalQCRewardValidationV1(t *testing.T) {
	chain, p, keys := nftLiveAdmissionFixtureV1(t, 1)
	block := nftLiveAdmissionBlockV1(t, chain, nftMonthlyPolicyRecordV1(p, 0), keys, 1)
	if err := chain.ValidateNFTMonthlyAwardBlockSynchronizedV1(block); err != nil {
		t.Fatal(err)
	}
	if err := chain.AddBlockFromPeerSynchronizedV2(block); err == nil {
		t.Fatal("NFT proof bypassed excess coinbase rejection")
	}
	if chain.BestHeightSnapshot() != 0 {
		t.Fatal("bad QC reward changed chain")
	}
}

func TestNFTLiveReplacementUsesLocalContextAndIncomingHistoryV1(t *testing.T) {
	local, p, keys := nftLiveAdmissionFixtureV1(t, 1)
	genesisBytes := SerializeBlockchain(local)
	peer := DeserializeBlockchain(genesisBytes)
	nftLiveAdmissionRebindV1(t, peer, local)
	block := nftLiveAdmissionBlockV1(t, peer, nftMonthlyPolicyRecordV1(p, 0), keys, 0)
	if err := peer.AddBlockFromPeerSynchronizedV2(block); err != nil {
		t.Fatal(err)
	}
	incoming := peer.AllBlocksSnapshot()
	if err := local.ReplaceChainFromPeerSynchronizedV1(incoming); err != nil {
		t.Fatalf("locally approved longer chain rejected: %v", err)
	}
	if local.BestHeightSnapshot() != 1 || !local.IsValidChainSynchronized() {
		t.Fatal("replacement not valid")
	}
	incoming[1].NFTAwardProofsV1[0].Approvals[0].Signature[0] ^= 1
	if !local.IsValidChainSynchronized() {
		t.Fatal("caller mutated replacement proof")
	}
	// Replacement history, not abandoned local awards, decides duplicates.
	next := nftLiveAdmissionBlockV1(t, peer, nftMonthlyPolicyRecordV1(p, 0), keys, 0)
	poisoned := append(peer.AllBlocksSnapshot(), next)
	before := local.AllBlocksSnapshot()
	if err := local.ReplaceChainFromPeerSynchronizedV1(poisoned); err == nil {
		t.Fatal("duplicate award in replacement history accepted")
	}
	if !reflect.DeepEqual(local.AllBlocksSnapshot(), before) {
		t.Fatal("invalid replacement mutated chain")
	}
	// A longer approved alternate history may replace the abandoned recipient;
	// uniqueness must not combine the old local branch with the incoming one.
	branch := DeserializeBlockchain(genesisBytes)
	nftLiveAdmissionRebindV1(t, branch, local)
	replacementRecord := nftMonthlyPolicyRecordV1(p, 0)
	replacementRecord.RecipientPubKeyHash = bytes.Repeat([]byte{0x77}, 20)
	replacementAward := nftLiveAdmissionBlockV1(t, branch, replacementRecord, keys, 0)
	if err := branch.AddBlockFromPeerSynchronizedV2(replacementAward); err != nil {
		t.Fatal(err)
	}
	height := branch.BestHeightSnapshot() + 1
	reward, err := branch.ExpectedSubsidyUnitsForHeightV1(height)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := NewBlockAtProtocolV1(height, []*Transaction{testPeerCoinbaseTxV2(reward)}, branch.GetLastBlock().Hash,
		"test-only-reorg-miner", testExpectedNextDifficultyV1(t, branch), nil, qconsensus.ProtocolNFTV4)
	if err != nil {
		t.Fatal(err)
	}
	if err := branch.AddBlockFromPeerSynchronizedV2(empty); err != nil {
		t.Fatal(err)
	}
	if err := local.ReplaceChainFromPeerSynchronizedV1(branch.AllBlocksSnapshot()); err != nil {
		t.Fatalf("approved alternate history rejected: %v", err)
	}
	if !local.IsValidChainSynchronized() || !bytes.Equal(local.AllBlocksSnapshot()[1].NFTMintRecordsV1[0].RecipientPubKeyHash, replacementRecord.RecipientPubKeyHash) {
		t.Fatal("replacement kept abandoned award recipient")
	}
}

func TestNFTLiveCommitBoundaryRevalidatesAndNoContextLoadStaysClosedV1(t *testing.T) {
	chain, p, keys := nftLiveAdmissionFixtureV1(t, 1)
	block := nftLiveAdmissionBlockV1(t, chain, nftMonthlyPolicyRecordV1(p, 0), keys, 0)
	if err := chain.ValidatePeerBlockSynchronized(block); err != nil {
		t.Fatal(err)
	}
	block.NFTMintRecordsV1[0].RecipientPubKeyHash[0] ^= 1
	if err := chain.addBlockFromPeerAfterStrictV2(block); err == nil {
		t.Fatal("writer accepted changed evidence")
	}
	valid := nftLiveAdmissionBlockV1(t, chain, nftMonthlyPolicyRecordV1(p, 0), keys, 0)
	if err := chain.AddBlockFromPeerSynchronizedV2(valid); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "approved-chain.gob")
	if err := chain.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBlockchainFromFile(path); err == nil {
		t.Fatal("context-free loader trusted signed peer keys")
	}
	raw := DeserializeBlockchain(SerializeBlockchain(chain))
	if raw.IsValidChainSynchronized() {
		t.Fatal("raw gob imported local trust/program/activation")
	}
	nftLiveAdmissionRebindV1(t, raw, chain)
	if !raw.IsValidChainSynchronized() {
		t.Fatal("local rebind did not validate serialized approval evidence")
	}
	// Exercise the actual replacement writer against changed evidence too.
	empty := DeserializeBlockchain(SerializeBlockchain(chain))
	empty.Blocks = empty.Blocks[:1]
	empty.UpdateUTXOSet()
	nftLiveAdmissionRebindV1(t, empty, chain)
	bad := chain.AllBlocksSnapshot()
	bad[1].NFTAwardProofsV1[0].Approvals[0].Signature[0] ^= 1
	if err := empty.replaceChainAfterStrictV1(bad); err == nil {
		t.Fatal("replacement writer skipped local authorization")
	}
}

func TestNFTLiveAdmissionConcurrentDuplicateSubmitAcceptsOnceV1(t *testing.T) {
	chain, p, keys := nftLiveAdmissionFixtureV1(t, 1)
	block := nftLiveAdmissionBlockV1(t, chain, nftMonthlyPolicyRecordV1(p, 0), keys, 0)
	var wg sync.WaitGroup
	var accepted int32
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := chain.AddBlockFromPeerSynchronizedV2(block); err == nil {
				atomic.AddInt32(&accepted, 1)
			}
		}()
	}
	wg.Wait()
	if accepted != 1 || chain.BestHeightSnapshot() != 1 || !chain.IsValidChainSynchronized() {
		t.Fatalf("accepted=%d height=%d", accepted, chain.BestHeightSnapshot())
	}
}
