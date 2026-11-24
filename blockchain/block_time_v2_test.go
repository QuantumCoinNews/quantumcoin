package blockchain

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestNetworkV3MedianRetargetIgnoresSingleOutlierAndLocalOverride(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "4")
	chain := &Blockchain{networkIdentity: &NetworkIdentityV1{GenesisDomain: TestnetGenesisDomainV3}}
	for i := 0; i < 111; i++ {
		chain.Blocks = append(chain.Blocks, &Block{Index: i, Timestamp: int64(i+1) * 30, Difficulty: 9})
	}
	chain.Blocks[110].Timestamp += 7_200
	difficulty, err := chain.ExpectedNextDifficultyV1()
	if err != nil || difficulty != 9 {
		t.Fatalf("median retarget accepted single outlier or override: %d %v", difficulty, err)
	}
	for i, block := range chain.Blocks {
		block.Timestamp = int64(i+1) * 20
	}
	if got := chain.NextDifficulty(1); got != 10 {
		t.Fatalf("median retarget missed sustained faster pace: %d", got)
	}
}

func TestNextMiningTimeAdvancesPastRepeatedCoinbaseClockV1(t *testing.T) {
	chain := NewBlockchain(50, 0)
	previous := time.Date(2026, time.November, 24, 3, 0, 0, 123_456_789, time.UTC)
	coinbase := chain.Blocks[0].Transactions[0]
	coinbase.Timestamp = previous
	coinbase.ID = coinbase.Hash()
	chain.Blocks[0].Timestamp = previous.Unix()

	got, err := chain.nextMiningTimeV1(previous)
	if err != nil {
		t.Fatal("advance repeated coinbase clock:", err)
	}
	if want := previous.Add(time.Nanosecond); !got.Equal(want) {
		t.Fatalf("next mining timestamp=%s want=%s", got.Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
	}
}

func TestNetworkV3MedianTimeAppliesToPeerReplayAndMining(t *testing.T) {
	cfg := config.Default()
	cfg.GenesisProfile = config.NetworkGenesisProfileV3
	chain, err := NewNetworkBlockchainV3(cfg, config.QCNetworkProfileTestnetLocalV1)
	if err != nil {
		t.Fatal(err)
	}
	miner := wallet.NewWallet().GetAddress()
	genesis := chain.Blocks[0]
	bits := config.QCTestnetDefaultDifficultyBitsV1
	cb, err := newCoinbaseTxWithRewardV1(miner, 50, time.Unix(genesis.Timestamp, 0))
	if err != nil {
		t.Fatal(err)
	}
	bad, err := newBlockAtTimeV1(1, []*Transaction{cb}, genesis.Hash, miner, bits, genesis.Timestamp)
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range []error{
		chain.ValidatePeerBlockStrictV2(bad),
		chain.ValidateReplacementChainStrictV1([]*Block{genesis, bad}),
	} {
		if err == nil || !strings.Contains(err.Error(), "median time past") {
			t.Fatal("expected median time rejection", err)
		}
	}
	for i := 0; i < 3; i++ {
		if _, err := chain.MineBlock(miner, bits); err != nil {
			t.Fatal("valid in-process mining failed", err)
		}
	}
	template, _, err := chain.PrepareExternalMiningTemplate(miner, bits)
	if err != nil || template.Timestamp <= medianBlockTimeV2(chain.Blocks, len(chain.Blocks)-1) {
		t.Fatal("external template violated median time", err)
	}
	path := filepath.Join(t.TempDir(), "chain.dat")
	if err := chain.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBlockchainFromFileWithLocalContextV1(path, chain); err != nil {
		t.Fatal("canonical replay rejected mined timestamps", err)
	}
	cfg.GenesisProfile = config.NetworkGenesisProfileV2
	previousProfile, err := NewNetworkBlockchainV2(cfg, config.QCNetworkProfileTestnetLocalV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := previousProfile.BindRuntimeNetworkIdentityV1("testnet-local"); err != nil {
		t.Fatal(err)
	}
	if previousProfile.usesMedianBlockTimeV2() || previousProfile.NetworkIdentitySnapshotV1().GenesisHash == chain.NetworkIdentitySnapshotV1().GenesisHash {
		t.Fatal("median rule was retroactively applied to the V2 profile")
	}
	if _, err := LoadBlockchainFromFileWithLocalContextV1(path, previousProfile); err == nil {
		t.Fatal("V2 context accepted the V3 genesis")
	}
}
