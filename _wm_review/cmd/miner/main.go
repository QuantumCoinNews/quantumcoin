package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"quantumcoin/miner"
)

var (
	flagAddr    = flag.String("address", "", "Coinbase reward address (required)")
	flagThreads = flag.Int("threads", runtime.NumCPU(), "Number of CPU threads")
	flagMode    = flag.String("mode", "local", "Mode: mock | local")
	flagLog     = flag.String("log", "", "Log file (e.g., miner.log)")
	flagConfig  = flag.String("config", "config.json", "Config file path")
	flagChain   = flag.String("chain", "chain_data.dat", "Chain data file path")
	flagP2P     = flag.String("p2p", ":3001", "P2P listen port (for adapter broadcast)")
)

func main() {
	flag.Parse()
	if *flagAddr == "" {
		fmt.Println("Usage: miner -address <QC_ADDRESS> [-threads N] [-mode mock|local] [-config config.json] [-chain chain_data.dat] [-p2p :3001] [-log miner.log]")
		os.Exit(2)
	}

	// Double-click support: use exe directory as working directory
	if exe, err := os.Executable(); err == nil {
		_ = os.Chdir(filepath.Dir(exe))
	}

	if cl := *flagLog; cl != "" {
		f, err := os.OpenFile(cl, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			log.SetOutput(f)
			defer f.Close()
		}
	}
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// CTRL+C / SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigc := make(chan os.Signal, 2)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func() { <-sigc; log.Println("SIGINT: shutting down"); cancel() }()

	// Backend
	var backend miner.Backend
	switch *flagMode {
	case "mock":
		backend = miner.NewMockBackend(*flagAddr)
	case "local":
		ad, err := miner.NewQCLocalAdapter(miner.QCLocalOpts{
			ConfigPath: *flagConfig,
			ChainPath:  *flagChain,
			P2PPort:    *flagP2P,
		})
		if err != nil {
			log.Fatalf("adapter init failed: %v", err)
		}
		backend = ad
	default:
		log.Fatalf("unknown mode: %s", *flagMode)
	}

	w := miner.NewWorker(backend, miner.WorkerConfig{
		Threads:      *flagThreads,
		Address:      *flagAddr,
		GreenOnFound: true, // keep ANSI green for block found, cyan for hash (handled in worker)
	})

	log.Printf("Miner starting | addr=%s threads=%d mode=%s", *flagAddr, *flagThreads, *flagMode)

	t0 := time.Now()
	if err := w.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("miner error: %v", err)
	}
	log.Printf("Stopped. duration=%s total_hash=%d found=%d",
		time.Since(t0).Truncate(time.Millisecond), w.HashCount, w.Found())
}
