package miner

import (
	"bytes"
	"log"
	"strings"
	"testing"
	"time"

	"quantumcoin/blockchain"
)

func TestMinerNFTFailureCannotRecordOrNotifyV1(t *testing.T) {
	oldSave, oldNotify := SaveNFTRewardHook, NotifyTelegramHook
	t.Cleanup(func() {
		SaveNFTRewardHook, NotifyTelegramHook = oldSave, oldNotify
	})
	chain := blockchain.NewBlockchain(1, 0)
	for _, flag := range []string{"", "0", "1", "true", "yes", "on"} {
		t.Run("miner_env_"+flag, func(t *testing.T) {
			t.Setenv("QC_ENABLE_MINER_NFT_REWARD", flag)
			for _, chainFlag := range []string{"0", "1"} {
				t.Run("chain_env_"+chainFlag, func(t *testing.T) {
					t.Setenv("QC_ENABLE_CHAIN_NFT_MINT", chainFlag)
					for _, tc := range []struct {
						name  string
						chain *blockchain.Blockchain
					}{
						{"missing_chain", nil},
						{"disabled_chain_mint", chain},
					} {
						t.Run(tc.name, func(t *testing.T) {
							saves, notices := 0, 0
							SaveNFTRewardHook = func(string, string, string, time.Time) error {
								saves++
								return nil
							}
							NotifyTelegramHook = func(string, string) { notices++ }
							GrantNFTBonus("QC_TEST_RECIPIENT", "rare", tc.chain)
							if saves != 0 || notices != 0 {
								t.Fatalf("QMA0069_RED: miner path produced an unminted award: db=%d telegram=%d", saves, notices)
							}
						})
					}
				})
			}
		})
	}
}

func TestMinerNFTStubDoesNotClaimAwardV1(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	GrantNFTReward("QC_TEST_RECIPIENT")
	if strings.Contains(strings.ToLower(output.String()), "reward granted") {
		t.Fatalf("QMA0066_STUB_RED: legacy stub claimed an NFT award: %s", output.String())
	}
}
