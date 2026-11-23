package main

import (
	"encoding/hex"
	"path/filepath"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/consensus"
	qnode "quantumcoin/node"
	"quantumcoin/wallet"
)

func networkSigningFixture(t *testing.T) (*blockchain.Blockchain, *wallet.Wallet, *wallet.Wallet, *config.Config) {
	t.Helper()
	previous := config.Current()
	// Restore the process config after all test-only environment selectors.
	// Its validation must see the original environment, not this testnet's.
	t.Cleanup(func() { config.Set(previous) })
	clearActualMuxTestEnvV1(t)
	t.Setenv("QC_NODE_DIR", t.TempDir())
	t.Setenv("QC_NETWORK", "testnet-local")
	t.Setenv("QC_DIFFICULTY_BITS", "")
	c := config.Default()
	dir := t.TempDir()
	c.GenesisProfile = config.NetworkGenesisProfileV4
	c.ChainFile = filepath.Join(dir, config.QCTestnetChainFileV1)
	c.WalletFile = filepath.Join(dir, "wallets.json")
	c.HTTPPort = ":18082"
	c.P2PPort = ":13001"
	if err := config.SetValidated(c); err != nil {
		t.Fatal(err)
	}
	chain, err := blockchain.NewNetworkBlockchainV4(c, config.QCNetworkProfileTestnetLocalV1)
	if err != nil {
		t.Fatal(err)
	}
	chain.SetCoinbaseMaturity(0)
	owner, to := wallet.NewWallet(), wallet.NewWallet()
	if _, err = chain.MineBlock(owner.GetAddress(), 8); err != nil {
		t.Fatal(err)
	}
	return chain, owner, to, c
}

func TestNetworkV4CLISendUsesCanonicalSigningContext(t *testing.T) {
	chain, owner, to, _ := networkSigningFixture(t)
	if err := wallet.SaveWallet(owner); err != nil {
		t.Fatal(err)
	}
	tx, err := submitCLISendV1(chain, owner.GetAddress(), to.GetAddress(), "1")
	if err != nil {
		t.Fatal(err)
	}
	if !chain.VerifyTransaction(tx) || tx.Verify() || len(chain.PendingTransactionsSnapshot()) != 1 {
		t.Fatal("CLI did not use network-bound signatures")
	}
}

func TestNetworkV4WalletSignedHTTPTransferAndBurn(t *testing.T) {
	for _, burn := range []bool{false, true} {
		t.Run(map[bool]string{false: "transfer", true: "burn"}[burn], func(t *testing.T) {
			chain, owner, to, c := networkSigningFixture(t)
			amount := "1"
			path := "/api/tx/send"
			if burn {
				p := consensus.ActivationPolicyV1{Enabled: true, Plan: consensus.ActivationPlan{NetworkID: "signature-testnet", LegacyGenesisHex: hex.EncodeToString(chain.BlockByIndexSnapshot(0).Hash), ActivationHeight: 2, TargetProtocol: consensus.ProtocolAtomV3, PreserveHistory: true, PreserveBalances: true}}
				if err := chain.BindActivationPolicyV1(verifiedActivationFixtureV1(t, p)); err != nil {
					t.Fatal(err)
				}
				amount = "0.000001"
				path = "/api/tx/burn"
			}
			tx, err := signWalletIntentV1(chain, owner.GetAddress(), to.GetAddress(), amount, owner.ExportPrivateKeyHex(), burn)
			if err != nil {
				t.Fatal(err)
			}
			if !chain.VerifyTransaction(tx) || tx.Verify() {
				t.Fatal("local wallet signer used legacy context")
			}
			previousRuntime, previousConfig, previousMode := nodeRuntime, cfg, apiReadOnlyMode
			nodeRuntime = qnode.NewRuntime(chain, c)
			cfg = c
			apiReadOnlyMode = false
			t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)
			t.Cleanup(func() { nodeRuntime, cfg, apiReadOnlyMode = previousRuntime, previousConfig, previousMode })
			handler := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
			// Same public transaction contents with a legacy signature must be rejected.
			legacy := *tx
			legacy.Inputs = append([]blockchain.TransactionInput(nil), tx.Inputs...)
			if err = legacy.Sign(owner.PrivateKey); err != nil {
				t.Fatal(err)
			}
			if response := webRequest(handler, "POST", path, signedEnvelope(t, &legacy)); response.Code != 400 {
				t.Fatal("HTTP accepted legacy signature", response.Code)
			}
			foreign, err := blockchain.NewNetworkBlockchainV4(c, config.QCNetworkProfileMainnetV1)
			if err != nil {
				t.Fatal(err)
			}
			if err = foreign.SignTransaction(&legacy, owner.PrivateKey); err != nil {
				t.Fatal(err)
			}
			if response := webRequest(handler, "POST", path, signedEnvelope(t, &legacy)); response.Code != 400 {
				t.Fatal("HTTP accepted another genesis signature", response.Code)
			}
			if response := webRequest(handler, "POST", path, signedEnvelope(t, tx)); response.Code != 200 {
				t.Fatal("HTTP rejected correctly scoped signature", response.Code, response.Body.String())
			}
		})
	}
}
