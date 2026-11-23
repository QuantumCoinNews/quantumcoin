package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"quantumcoin/blockchain"
	qnode "quantumcoin/node"
	"quantumcoin/p2p"
	"strings"
)

/* ====== ANSI renk sabitleri (sadece konsol görünümü için) ====== */

const (
	ansiGreen = "\x1b[32m"
	ansiCyan  = "\x1b[36m"
	ansiReset = "\x1b[0m"
)

/* ---------- Yardımcı fonksiyonlar ---------- */

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  quantumcoin [--data-dir DIR] [--config FILE] COMMAND [arguments]")
	fmt.Println("  Paths default to the caller's working directory; no automatic chdir.")
	fmt.Println("  --wallet-file FILE selects the offline newaddr/newaddr-priv store.")
	fmt.Println("  help / --help / -h       - Show help without loading config or chain")
	fmt.Println("  run [port]               - Run node (override P2P port)")
	fmt.Println("  run-mine [miner]         - Run node + API + continuous mining")
	fmt.Println("  connect [port] [addr]    - Connect to peer")
	fmt.Println("  send [from] [to] [amt]   - Sign with the stored sender wallet and send exact QC")
	fmt.Println("  sign-send FROM TO AMOUNT [--key-stdin] - Sign locally for HTTP submission")
	fmt.Println("  sign-burn FROM AMOUNT [--key-stdin]    - Sign a canonical burn locally")
	fmt.Println("  mine [miner]             - Mine one block")
	fmt.Println("  mine-forever [miner]     - Continuous mining")
	fmt.Println("  print                    - Print chain")
	fmt.Println("  newaddr                  - Generate and save wallet including private key")
	fmt.Println("  newaddr-priv             - Generate wallet + print private key (hex)")
	fmt.Println("  api                      - Start HTTP API")
}

/* ---------- main ---------- */

func runNodeCLI() {
	var bc *blockchain.Blockchain

	var err error

	paths, err := parseCLIPathsV1(os.Args[1:])
	if err == nil {
		err = validateCLICommandV1(paths.Args)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		printUsage()
		return
	}
	if runOfflineCLICommandV1(paths) {
		return
	}
	args := append([]string{os.Args[0]}, paths.Args...)
	activeCLIPathsV1 = paths

	// Canonical node bootstrap.
	nodeRuntime, err = qnode.Bootstrap(
		qnode.BootstrapOptions{
			ConfigPath:  paths.Config,
			DataDir:     paths.DataDir,
			StrictChain: true,
		},
	)
	if err != nil {
		log.Fatalf(
			"Node bootstrap failed: %v",
			err,
		)
	}

	// The CLI pins its own chain reference; concurrent HTTP adapters obtain
	// their own runtime snapshots. Only immutable startup config is shared.
	shutdownBarrier := newCLIShutdownBarrierV1()
	defer shutdownBarrier.Wait()
	bc, cfg = nodeRuntime.Snapshot()

	if bc == nil || cfg == nil {
		log.Fatal(
			"Node bootstrap returned incomplete runtime state",
		)
	}

	// Mining requires an explicit command and a supplied wallet address.
	if len(args) < 2 {
		fmt.Println("Starting node + API. Mining requires run-mine <wallet-address>.")
		go startHTTPAPI()
		go autosaveLoop()
		go trapAndShutdown(shutdownBarrier)
		p := strings.TrimPrefix(cfg.P2PPort, ":")
		if err := nodeRuntime.ServeP2P(p); err != nil {
			log.Printf("p2p server error: %v", err)
		}
		return
	}

	switch args[1] {
	case "sign-send", "sign-burn":
		runLocalSigningCommandV1(bc, args[1:])
		return
	case "run":
		if len(args) >= 3 {
			port := args[2]
			go startHTTPAPI()
			go autosaveLoop()
			go trapAndShutdown(shutdownBarrier)
			if err := nodeRuntime.ServeP2P(port); err != nil {
				log.Printf("p2p server error: %v", err)
			}
		} else {
			go startHTTPAPI()
			go autosaveLoop()
			go trapAndShutdown(shutdownBarrier)
			p := strings.TrimPrefix(cfg.P2PPort, ":")
			if err := nodeRuntime.ServeP2P(p); err != nil {
				log.Printf("p2p server error: %v", err)
			}
		}

	case "run-mine":
		if len(args) < 3 {
			fmt.Println("Usage: run-mine [miner]")
			return
		}
		miner := args[2]
		go startHTTPAPI()
		go autosaveLoop()
		go startContinuousMining(miner)
		go trapAndShutdown(shutdownBarrier)
		p := strings.TrimPrefix(cfg.P2PPort, ":")
		if err := nodeRuntime.ServeP2P(p); err != nil {
			log.Printf("p2p server error: %v", err)
		}

	case "api":
		apiReadOnlyMode = true
		nodeRuntime.SetAPIReadOnly(true)
		// API mode denies mutating routes and never autosaves/final-saves.
		// Read endpoints reload chain_data.dat when miner writes new blocks.
		go trapAPIShutdown(shutdownBarrier)
		startHTTPAPI()
		return

	case "connect":
		if len(args) < 4 {
			fmt.Println("Usage: connect [port] [address]")
			return
		}
		port := args[2]
		address := args[3]
		go startHTTPAPI()
		go autosaveLoop()
		go trapAndShutdown(shutdownBarrier)
		if err := nodeRuntime.ConnectPeer(port, address); err != nil {
			log.Printf("p2p connect error: %v", err)
		}

	case "send":
		if len(args) < 5 {
			fmt.Println("Usage: send [from] [to] [amount]")
			return
		}
		from := args[2]
		to := args[3]
		tx, err := submitCLISendV1(bc, from, to, args[4])
		if err != nil {
			log.Println("send failed:", err)
			return
		}
		nodeRuntime.BroadcastP2P(p2p.TxMessage(tx))
		fmt.Printf("✓ Transaction accepted and broadcasted (txid=%s)\n", hex.EncodeToString(tx.ID))

	case "mine":
		if len(args) < 3 {
			fmt.Println("Usage: mine [miner]")
			return
		}

		miner := args[2]
		difficulty := currentEffectiveDifficultyBits()

		candidateHeight := bc.BestHeightSnapshot() + 1
		reward := bc.ExpectedSubsidyForHeightSnapshot(
			candidateHeight,
		)

		block, err := nodeRuntime.MineOnceV1(
			miner,
			difficulty,
		)
		if err != nil {
			log.Println("mining failed:", err)
			return
		}

		nodeRuntime.BroadcastP2P(
			p2p.BlockMessage(block),
		)

		fmt.Printf(
			ansiGreen+"✅ New block mined by %s"+ansiReset+"\n",
			miner,
		)

		fmt.Printf(
			"   Hash:   %s%s%s\n",
			ansiGreen,
			hex.EncodeToString(block.Hash),
			ansiReset,
		)

		fmt.Printf(
			"   Height: %d  Reward: %d QC\n",
			block.Index,
			reward,
		)

	case "mine-forever":
		if len(args) < 3 {
			fmt.Println("Usage: mine-forever [miner]")
			return
		}
		miner := args[2]
		go autosaveLoop()
		go trapAndShutdown(shutdownBarrier)
		startContinuousMining(miner)

	case "print":
		for _, block := range bc.AllBlocksSnapshot() {
			fmt.Printf("📦 Block #%d\n", block.Index)
			fmt.Printf("⛏️  Miner     : %s\n", block.Miner)
			fmt.Printf("🧱 Hash       : %s\n", hex.EncodeToString(block.Hash))
			fmt.Printf("🔗 PrevHash   : %s\n", hex.EncodeToString(block.PrevHash))
			fmt.Println("📝 Transactions:")
			for _, tx := range block.Transactions {
				fmt.Printf("  TxID: %s\n", hex.EncodeToString(tx.ID))
				for _, out := range tx.Outputs {
					fmt.Printf("    🔸 Amount: %s QC\n", bc.OutputAmountQCAtHeightV1(out.Amount, block.Index))
				}
			}
			fmt.Println("-------------------------------")
		}

	default:
		printUsage()
		return
	}
}
