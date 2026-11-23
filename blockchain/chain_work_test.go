package blockchain

import (
	"bytes"
	"testing"
	"time"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestChainWorkSelection(t *testing.T) {
	makeChain := func(bits ...int) []*Block {
		blocks := []*Block{{Index: 0}}
		for i, difficulty := range bits {
			blocks = append(blocks, &Block{Index: i + 1, Difficulty: difficulty})
		}
		return blocks
	}
	for _, test := range []struct {
		name             string
		candidate, local []*Block
		want             bool
	}{
		{"shorter more work", makeChain(5), makeChain(3, 3, 3), true},
		{"longer less work", makeChain(3, 3, 3), makeChain(5), false},
		{"same height more work", makeChain(5), makeChain(4), true},
		{"equal work keeps local", makeChain(4), makeChain(3, 3), false},
		{"large work does not overflow", makeChain(255), makeChain(63, 63, 63), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := greaterChainWork(test.candidate, test.local)
			if err != nil || got != test.want {
				t.Fatalf("got %v, %v; want %v", got, err, test.want)
			}
		})
	}
	for _, blocks := range [][]*Block{nil, {nil}, {{}, nil}, {{}, {Difficulty: 256}}, {{}, {Difficulty: -1}}} {
		if _, err := chainWork(blocks); err == nil {
			t.Fatal("invalid work input accepted")
		}
	}
}

func TestReplacementChoosesValidatedShorterHigherWorkFork(t *testing.T) {
	cfg := config.Default()
	cfg.GenesisProfile = config.NetworkGenesisProfileV3
	build := func(count int, spacing int64) *Blockchain {
		chain, err := NewNetworkBlockchainV3(cfg, config.QCNetworkProfileTestnetLocalV1)
		if err != nil {
			t.Fatal(err)
		}
		miner := wallet.NewWallet().GetAddress()
		for height := 1; height <= count; height++ {
			timestamp := chain.Blocks[0].Timestamp + int64(height)*spacing
			coinbase, err := newCoinbaseTxWithRewardV1(miner, 50, time.Unix(timestamp, 0).UTC())
			if err != nil {
				t.Fatal(err)
			}
			block, err := newBlockAtTimeV1(height, []*Transaction{coinbase}, chain.GetLastBlock().Hash,
				miner, testExpectedNextDifficultyV1(t, chain), timestamp)
			if err != nil {
				t.Fatal(err)
			}
			chain.Blocks = append(chain.Blocks, block)
		}
		chain.UpdateUTXOSet()
		return chain
	}
	shorter := build(114, 20)
	longer := build(118, 30)
	if err := shorter.validateCanonicalChainWithLocalContextV1(shorter.Blocks); err != nil {
		t.Fatalf("higher work fixture is not canonical: %v", err)
	}
	if err := longer.validateCanonicalChainWithLocalContextV1(longer.Blocks); err != nil {
		t.Fatalf("longer fixture is not canonical: %v", err)
	}
	if err := shorter.ReplaceChain(longer.BlocksSnapshot()); err == nil {
		t.Fatal("longer lower-work chain replaced the higher-work tip")
	}
	if err := longer.ReplaceChain(shorter.BlocksSnapshot()); err != nil {
		t.Fatalf("shorter higher-work chain rejected: %v", err)
	}
	if !bytes.Equal(longer.GetLastBlock().Hash, shorter.GetLastBlock().Hash) {
		t.Fatal("higher-work tip not installed")
	}
	if err := longer.ReplaceChain(shorter.BlocksSnapshot()); err == nil {
		t.Fatal("equal work caused a replacement")
	}
	forged := shorter.BlocksSnapshot()
	forged[len(forged)-1].Difficulty = 255
	if err := longer.ReplaceChain(forged); err == nil {
		t.Fatal("unmined claimed difficulty counted as work")
	}
}
