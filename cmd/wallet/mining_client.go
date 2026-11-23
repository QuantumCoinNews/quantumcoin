package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"quantumcoin/config"
	"quantumcoin/miner"
	"quantumcoin/node"
	"quantumcoin/wallet"
)

var guiMining struct {
	sync.Mutex
	cancel   context.CancelFunc
	done     chan struct{}
	notified chan struct{}
	log      func(string)
}

var errNativeMinerAlreadyRunningV1 = errors.New("mining worker is already running; stop it before restarting")

func nativeMinerAlive() bool {
	guiMining.Lock()
	defer guiMining.Unlock()
	return guiMining.cancel != nil
}

func nativeMiningMessage(message string) {
	guiMining.Lock()
	hook := guiMining.log
	guiMining.Unlock()
	if hook != nil {
		hook(message)
	}
}

func nativeMiningAcceptanceHandler(dispatch func(func())) func(int, string) {
	return func(height int, hash string) {
		nativeMiningMessage(fmt.Sprintf("Block %d accepted: %s", height, hash))
		dispatch(func() {
			if walletRefreshHook != nil {
				walletRefreshHook()
			}
		})
	}
}

func startNativeMining() error {
	guiMining.Lock()
	if guiMining.cancel != nil {
		guiMining.Unlock()
		return errNativeMinerAlreadyRunningV1
	}
	guiMining.Unlock()
	data := strings.TrimSpace(os.Getenv("QC_DATA_DIR"))
	if data == "" {
		data = nodeDir()
	}
	path := strings.TrimSpace(os.Getenv("QC_CONFIG_FILE"))
	if path == "" {
		path = filepath.Join(data, "config.json")
	}
	cfg, err := config.LoadWithDataDir(path, data)
	if err != nil {
		return err
	}
	if cfg.GenesisProfile != config.NetworkGenesisProfileV4 {
		return fmt.Errorf("select the fresh network-v4 testnet/mainnet profile before mining")
	}
	identity, err := node.TrustedMiningIdentity(cfg)
	if err != nil {
		return err
	}
	address := strings.TrimSpace(os.Getenv("ADDR"))
	if selected, selectErr := selectedRecoveryAddress(); selectErr != nil {
		return selectErr
	} else if selected != "" {
		if address == "" {
			address = selected
		}
	}
	selection, err := wallet.ResolveMinerAddress(wallet.MinerAddressOptions{Config: cfg, Explicit: address, DataDir: data, AppDataDir: os.Getenv("APPDATA")})
	if err != nil {
		return err
	}
	base := strings.TrimSpace(os.Getenv("QC_API_BASE"))
	if base == "" {
		base = "http://127.0.0.1" + cfg.HTTPPort
	}
	preflight, cancelPreflight := context.WithTimeout(context.Background(), 5*time.Second)
	worker, err := miner.PrepareNodeWorker(preflight, base, os.Getenv("QC_API_TOKEN"), identity.GenesisHash, miner.WorkerConfig{
		Address: selection.Address, Threads: runtime.NumCPU(), OnAccepted: nativeMiningAcceptanceHandler(ui),
	})
	cancelPreflight()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	if !installNativeMiningRunV1(cancel, done) {
		cancel()
		return errNativeMinerAlreadyRunningV1
	}
	nativeMiningMessage("Solo PoW started on " + identity.Network + "; genesis " + identity.GenesisHash)
	go func() {
		err := worker.Run(ctx)
		completeNativeMiningRunV1(done)
		if err != nil && ctx.Err() == nil {
			nativeMiningMessage("Mining stopped: " + err.Error())
		}
		cancel()
		notifyNativeMiningStoppedV1(done, ui)
	}()
	return nil
}

func stopNativeMining() error {
	return stopNativeMiningWithDispatchV1(ui)
}

func stopNativeMiningWithDispatchV1(dispatch func(func())) error {
	guiMining.Lock()
	cancel, done := guiMining.cancel, guiMining.done
	guiMining.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		return fmt.Errorf("mining stop is still pending")
	}
	notifyNativeMiningStoppedV1(done, dispatch)
	return nil
}

func installNativeMiningRunV1(cancel context.CancelFunc, done chan struct{}) bool {
	if cancel == nil || done == nil {
		return false
	}
	guiMining.Lock()
	defer guiMining.Unlock()
	if guiMining.cancel != nil {
		return false
	}
	guiMining.cancel = cancel
	guiMining.done = done
	return true
}

func completeNativeMiningRunV1(done chan struct{}) bool {
	guiMining.Lock()
	defer guiMining.Unlock()
	if guiMining.done != done || guiMining.cancel == nil {
		return false
	}
	guiMining.cancel = nil
	close(done) // Close completion before releasing ownership to a replacement.
	return true
}

func notifyNativeMiningStoppedV1(done chan struct{}, dispatch func(func())) {
	dispatch(func() {
		guiMining.Lock()
		if guiMining.done != done || guiMining.cancel != nil || guiMining.notified == done {
			guiMining.Unlock()
			return
		}
		guiMining.notified = done
		minerRunningState = false
		guiMining.Unlock()
		if onMinerStateUpdate != nil {
			onMinerStateUpdate(false)
		}
	})
}

// A render callback can be queued again by the UI. Read current ownership when
// that final callback runs, rather than rendering an old requested boolean.
func nativeMiningUIStateHandlerV1(dispatch func(func()), render func(bool)) func(bool) {
	return func(_ bool) {
		dispatch(func() {
			running := nativeMinerAlive()
			minerRunningState = running
			render(running)
		})
	}
}
