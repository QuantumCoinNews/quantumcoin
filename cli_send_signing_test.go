package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
)

func cliSendFixture(t *testing.T) (*blockchain.Blockchain, *wallet.Wallet, *wallet.Wallet, *config.Config) {
	t.Helper()
	clearActualMuxTestEnvV1(t)
	t.Setenv("QC_NODE_DIR", t.TempDir())
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	previous := config.Current()
	c := config.Default()
	dir := t.TempDir()
	c.WalletFile = filepath.Join(dir, "wallet_data.json")
	c.ChainFile = filepath.Join(dir, config.QCIsolatedLocalChainFileV1)
	c.HTTPPort = config.QCIsolatedLocalHTTPPortV1
	c.P2PPort = config.QCIsolatedLocalP2PPortV1
	if err := config.SetValidated(c); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { config.Set(previous) })
	chain := blockchain.NewIsolatedLocalBlockchainV1(c.InitialReward, c.TotalSupply)
	chain.SetCoinbaseMaturity(0)
	owner, recipient := wallet.NewWallet(), wallet.NewWallet()
	if _, err := chain.MineBlock(owner.GetAddress(), 8); err != nil {
		t.Fatal(err)
	}
	return chain, owner, recipient, c
}

func TestCLISendSignsStoredOwnerAndAdmitsVerifiedTransaction(t *testing.T) {
	chain, owner, to, c := cliSendFixture(t)
	if err := wallet.SaveWallet(owner); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(c.WalletFile)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := submitCLISendV1(chain, owner.GetAddress(), to.GetAddress(), "2")
	if err != nil {
		t.Fatal(err)
	}
	if !tx.Verify() || len(tx.Inputs) == 0 {
		t.Fatal("CLI submitted unsigned transaction")
	}
	for _, input := range tx.Inputs {
		if len(input.Signature) == 0 || len(input.PubKey) != 65 {
			t.Fatal("missing input authorization")
		}
	}
	if tx.Outputs[0].Amount != 2 || !bytes.Equal(tx.Outputs[0].PubKeyHash, wallet.Base58DecodeAddress(to.GetAddress())) {
		t.Fatal("wrong recipient/amount")
	}
	if state := chain.TransactionLocationSnapshot(hex.EncodeToString(tx.ID)); !state.InMempool {
		t.Fatal("signed send missing from canonical mempool")
	}
	after, _ := os.ReadFile(c.WalletFile)
	if !bytes.Equal(before, after) {
		t.Fatal("send changed wallet store")
	}
}

func TestCLISendMissingCorruptAndMismatchedKeysFailWithoutMutation(t *testing.T) {
	for _, kind := range []string{"missing-file", "missing-sender", "invalid-key", "mismatched-key", "malformed-store"} {
		t.Run(kind, func(t *testing.T) {
			chain, owner, to, c := cliSendFixture(t)
			foreign := wallet.NewWallet()
			var data []byte
			switch kind {
			case "missing-sender":
				if err := wallet.SaveWallet(foreign); err != nil {
					t.Fatal(err)
				}
			case "invalid-key", "mismatched-key":
				key := "invalid"
				if kind == "mismatched-key" {
					key = foreign.ExportPrivateKeyHex()
				}
				data, _ = json.Marshal(map[string]any{"wallets": map[string]string{owner.GetAddress(): key}, "default": owner.GetAddress()})
			case "malformed-store":
				data = []byte("not JSON")
			}
			if data != nil {
				if err := os.WriteFile(c.WalletFile, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			before, readErr := os.ReadFile(c.WalletFile)
			if tx, err := submitCLISendV1(chain, owner.GetAddress(), to.GetAddress(), "2"); err == nil || tx != nil {
				t.Fatal("unauthorized send accepted")
			}
			if len(chain.PendingTransactionsSnapshot()) != 0 {
				t.Fatal("failed send entered mempool")
			}
			after, afterErr := os.ReadFile(c.WalletFile)
			if !bytes.Equal(before, after) || os.IsNotExist(readErr) != os.IsNotExist(afterErr) {
				t.Fatal("failure generated/changed wallet")
			}
		})
	}
}

func TestCLISendUsesExactAtomAmountAndRejectsInvalidAmounts(t *testing.T) {
	chain, owner, to, _ := cliSendFixture(t)
	policy := qconsensus.ActivationPolicyV1{Enabled: true, Plan: qconsensus.ActivationPlan{NetworkID: "cli-send-atom-test", LegacyGenesisHex: hex.EncodeToString(chain.TipSnapshot().PrevHash), ActivationHeight: 2, TargetProtocol: qconsensus.ProtocolAtomV3, PreserveHistory: true, PreserveBalances: true}}
	if err := chain.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err != nil {
		t.Fatal(err)
	}
	if err := wallet.SaveWallet(owner); err != nil {
		t.Fatal(err)
	}
	for _, amount := range []string{"0", "-1", "NaN", "1e3", "0.0000001", "999999999999999999999999999999999999999"} {
		if tx, err := submitCLISendV1(chain, owner.GetAddress(), to.GetAddress(), amount); err == nil || tx != nil {
			t.Fatal("invalid amount accepted", amount)
		}
	}
	tx, err := submitCLISendV1(chain, owner.GetAddress(), to.GetAddress(), "0.000001")
	if err != nil {
		t.Fatal(err)
	}
	if tx.Outputs[0].Amount != 1 || !tx.Verify() {
		t.Fatal("CLI atom precision or signature failed")
	}
}

func TestCLISendCannotBypassCoreSignatureOrUTXOOwnership(t *testing.T) {
	chain, owner, to, _ := cliSendFixture(t)
	unsigned, err := blockchain.NewTransactionFromQCAmountV1(owner.GetAddress(), to.GetAddress(), "2", chain)
	if err != nil {
		t.Fatal(err)
	}
	if err := chain.AddTransaction(unsigned); err == nil {
		t.Fatal("unsigned transaction accepted")
	}
	attacker := wallet.NewWallet()
	unsigned.Sender = attacker.GetAddress()
	unsigned.ID = unsigned.Hash()
	if err := unsigned.Sign(attacker.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if !unsigned.Verify() {
		t.Fatal("attack fixture must have a valid attacker signature")
	}
	if err := chain.AddTransaction(unsigned); err == nil {
		t.Fatal("attacker spent another owner's output")
	}
	if len(chain.PendingTransactionsSnapshot()) != 0 {
		t.Fatal("unauthorized spend admitted")
	}
}

// Child process executes the real CLI dispatcher/bootstrap rather than a
// substitute command. It prints only verification state, never private keys.
func TestCLISendCommandChild(t *testing.T) {
	if os.Getenv("QC_CLI_SEND_TEST_CHILD") != "1" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("QC_CLI_SEND_TEST_ARGS")), &args); err != nil {
		t.Fatal(err)
	}
	os.Args = append([]string{os.Args[0]}, args...)
	runNodeCLI()
	chain, _ := nodeRuntime.Snapshot()
	pending := chain.PendingTransactionsSnapshot()
	valid := len(pending) == 1 && pending[0].Verify()
	if valid {
		for _, in := range pending[0].Inputs {
			valid = valid && len(in.Signature) > 0
		}
	}
	state, _ := json.Marshal(map[string]any{"pending": len(pending), "valid": valid})
	t.Log("CLI_SEND_STATE=" + string(state))
}

func TestCLISendRealCommandSignsAndMissingKeyDoesNotClaimSuccess(t *testing.T) {
	chain, owner, to, c := cliSendFixture(t)
	// Trusted startup applies the canonical maturity, so mature the funding
	// block using actual consensus-valid blocks before launching the child.
	chain.SetCoinbaseMaturity(c.CoinbaseMaturity)
	for i := 0; i < c.CoinbaseMaturity; i++ {
		if _, err := chain.MineBlock(wallet.NewWallet().GetAddress(), 8); err != nil {
			t.Fatal(err)
		}
	}
	if err := chain.SaveToFile(c.ChainFile); err != nil {
		t.Fatal(err)
	}
	if err := wallet.SaveWallet(owner); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, present := range []bool{true, false} {
		store := c.WalletFile
		if !present {
			store = filepath.Join(filepath.Dir(c.WalletFile), "absent-wallet.json")
		}
		args, _ := json.Marshal([]string{"send", owner.GetAddress(), to.GetAddress(), "2"})
		cmd := exec.Command(exe, "-test.run=^TestCLISendCommandChild$", "-test.v")
		cmd.Dir = filepath.Dir(c.ChainFile)
		cmd.Env = append(os.Environ(), "QC_CLI_SEND_TEST_CHILD=1", "QC_CLI_SEND_TEST_ARGS="+string(args),
			"QC_CHAIN_FILE="+c.ChainFile, "QC_WALLET_FILE="+store, "QC_HTTP_PORT="+c.HTTPPort, "QC_P2P_PORT="+c.P2PPort)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI child failed: %v\n%s", err, output)
		}
		text := string(output)
		if strings.Contains(text, owner.ExportPrivateKeyHex()) {
			t.Fatal("private key leaked in command output")
		}
		if present {
			if !strings.Contains(text, `CLI_SEND_STATE={"pending":1,"valid":true}`) || !strings.Contains(text, "Transaction accepted") {
				t.Fatal(text)
			}
		} else if strings.Contains(text, "Transaction accepted") || !strings.Contains(text, `CLI_SEND_STATE={"pending":0,"valid":false}`) {
			t.Fatal(text)
		}
	}
}
