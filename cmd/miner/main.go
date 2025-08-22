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
	flagAddr    = flag.String("address", "", "Coinbase ödül adresi (zorunlu)")
	flagThreads = flag.Int("threads", runtime.NumCPU(), "CPU iş parçacığı sayısı")
	flagMode    = flag.String("mode", "local", "Çalışma modu: mock | local")
	flagLog     = flag.String("log", "", "Log dosyası (örn: miner.log)")
	flagConfig  = flag.String("config", "config.json", "Config dosyası yolu")
	flagChain   = flag.String("chain", "chain_data.dat", "Chain dosyası yolu")
	flagP2P     = flag.String("p2p", ":3001", "P2P dinleme portu (adapter broadcast için)")
)

func main() {
	flag.Parse()
	if *flagAddr == "" {
		fmt.Println("Kullanım: miner -address <QC_ADDRESS> [-threads N] [-mode mock|local] [-config config.json] [-chain chain_data.dat] [-p2p :3001] [-log miner.log]")
		os.Exit(2)
	}

	// Çift tık desteği: exe klasörünü çalışma dizini yap
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

	// CTRL+C
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigc := make(chan os.Signal, 2)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func() { <-sigc; log.Println("SIGINT: kapanıyor"); cancel() }()

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
			log.Fatalf("adapter init: %v", err)
		}
		backend = ad
	default:
		log.Fatalf("bilinmeyen mode: %s", *flagMode)
	}

	w := miner.NewWorker(backend, miner.WorkerConfig{
		Threads:      *flagThreads,
		Address:      *flagAddr,
		GreenOnFound: true,
	})
	log.Printf("Miner başlıyor | addr=%s threads=%d mode=%s", *flagAddr, *flagThreads, *flagMode)

	t0 := time.Now()
	if err := w.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("miner hata: %v", err)
	}
	log.Printf("Durduruldu. Süre=%s, toplam hash=%d, bulunan=%d",
		time.Since(t0).Truncate(time.Millisecond), w.HashCount, w.Found())
}
