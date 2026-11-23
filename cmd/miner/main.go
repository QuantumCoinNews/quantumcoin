package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"quantumcoin/miner"
)

var (
	flagAddr    = flag.String("address", "", "Coinbase reward address (required)")
	flagThreads = flag.Int("threads", runtime.NumCPU(), "Number of CPU threads")
	flagMode    = flag.String("mode", "node", "Mode: node | mock")
	flagLog     = flag.String("log", "", "Log file (e.g., miner.log)")
	flagAPI     = flag.String("api", "", "Node API origin (default QC_API_BASE or loopback port 8082)")
)

func main() {
	flag.Parse()

	if *flagAddr == "" {
		fmt.Println("Usage: miner -address <QC_ADDRESS> [-threads N] [-mode node|mock] [-api NODE_ORIGIN] [-log miner.log]")
		os.Exit(2)
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
	case "node":
		base := strings.TrimSpace(*flagAPI)
		if base == "" {
			base = strings.TrimSpace(os.Getenv("QC_API_BASE"))
		}
		if base == "" {
			base = "http://127.0.0.1:8082"
		}
		ad, err := miner.NewNodeBackend(base, os.Getenv("QC_API_TOKEN"))
		if err != nil {
			log.Fatalf("adapter init failed: %v", err)
		}
		backend = ad
	case "local":
		log.Fatal("local chain-file mining is retired in this command; select -mode node")
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
