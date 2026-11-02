package blockchain

import (
	"path/filepath"
	"reflect"
	"testing"

	"quantumcoin/consensus"
	"quantumcoin/wallet"
)

// QMA-0073: build/sign a real one-atom burn, pay a one-atom fee and mint an
// approved NFT in the SAME canonical block. Check peer admission and disk
// reload, rather than only burn policy booleans or a fabricated event record.
func TestNFTBurnOneAtomFeePeerPersistenceAndSupplyV1(t *testing.T) {
	chain, policy, keys := nftLiveAdmissionFixtureV1(t, 2)
	// The shared authorization fixture uses an uncapped zero TotalSupply;
	// this economy scenario explicitly exercises the canonical cap instead.
	chain.TotalSupply = int(consensus.CanonicalTotalSupplyQCV1)
	chain.SetCoinbaseMaturity(0)
	owner, miner := wallet.NewWallet(), wallet.NewWallet()
	funding, err := chain.MineBlock(owner.GetAddress(), testExpectedNextDifficultyV1(t, chain))
	if err != nil {
		t.Fatal(err)
	}
	inputUnits, err := consensus.WholeQCAmountToProtocolUnitsV1(consensus.ProtocolNFTV4, int64(funding.Transactions[0].Outputs[0].Amount))
	if err != nil {
		t.Fatal(err)
	}
	burn, err := NewCanonicalBurnTransactionFromQCAmountV1(owner.GetAddress(), "0.000001", chain)
	if err != nil {
		t.Fatal(err)
	}
	if len(burn.Outputs) != 2 || burn.Outputs[0].Amount != 1 || burn.Outputs[1].Amount != int(inputUnits)-1 {
		t.Fatal("exact one-atom burn/change did not use canonical protocol units")
	}
	burn.Outputs[1].Amount-- // A real one-atom transaction fee, outside burn.
	burn.ID = burn.Hash()
	if err := burn.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if err := chain.AddTransaction(burn); err != nil {
		t.Fatal(err)
	}
	fees, err := chain.CalculateBlockFeesSnapshot(chain.PendingTxs())
	if err != nil || fees != 1 {
		t.Fatalf("burn fee=%d want=1 atom: %v", fees, err)
	}
	peer := DeserializeBlockchain(SerializeBlockchain(chain))
	nftLiveAdmissionRebindV1(t, peer, chain)
	peer.SetCoinbaseMaturity(0)
	record := nftMonthlyPolicyRecordV1(policy, 0)
	proof := nftMonthlyPolicyBlockV1(t, chain, []NFTMintRecordV1{record}, keys).NFTAwardProofsV1[0]
	before, subsidyQC := chain.TotalMinted(), chain.ExpectedSubsidyForHeightV1(2)
	subsidyUnits, err := chain.ExpectedSubsidyUnitsForHeightV1(2)
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(t.TempDir(), "burn-nft-chain.dat")
	block, err := chain.MineBlockWithNFTAwardAndPersistSynchronizedV1(miner.GetAddress(), record, proof, filename)
	if err != nil {
		t.Fatal(err)
	}
	_, paid, err := blockCoinbaseTotalV1(block, false)
	if err != nil || paid != subsidyUnits+1 || chain.TotalMinted() != before+subsidyQC {
		t.Fatalf("burn/NFT changed subsidy or minted the fee: paid=%d error=%v", paid, err)
	}
	if err := peer.AddBlockFromPeerSynchronizedV2(block); err != nil {
		t.Fatalf("real peer rejected burn/NFT block: %v", err)
	}
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(filename, chain)
	if err != nil {
		t.Fatal(err)
	}
	for name, actual := range map[string]*Blockchain{"miner": chain, "peer": peer, "trusted-reload": loaded} {
		burned, err := actual.TotalBurnedAtomsCheckedV1()
		if err != nil || burned != 1 {
			t.Fatalf("%s burned=%d want=1 atom: %v", name, burned, err)
		}
		circulating, err := actual.CirculatingSupplyAtomsCheckedV1()
		issued, conversionErr := consensus.WholeQCToAtomsV1(int64(actual.TotalMinted()))
		if err != nil || conversionErr != nil || circulating != issued-1 || actual.TotalSupply != int(consensus.CanonicalTotalSupplyQCV1) {
			t.Fatalf("%s burn changed issuance/cap/circulating: circulating=%d issued=%d cap=%d errors=%v/%v", name, circulating, issued, actual.TotalSupply, err, conversionErr)
		}
		if actual.GetBalance(owner.GetAddress()) != int(inputUnits)-2 {
			t.Fatalf("%s owner balance did not lose exactly burn+fee", name)
		}
		if !reflect.DeepEqual(chain.StateSnapshot().UTXO, actual.StateSnapshot().UTXO) {
			t.Fatalf("%s UTXO diverged", name)
		}
		ownership, err := actual.NFTOwnershipSnapshotSynchronizedV1()
		if err != nil || len(ownership.NFTs) != 1 {
			t.Fatalf("%s lost canonical NFT ownership: %v", name, err)
		}
	}
	if err := chain.AddTransaction(burn); err == nil {
		t.Fatal("confirmed burn was admitted for a second spend")
	}
}
