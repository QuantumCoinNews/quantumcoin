package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"
)

func autosaveLoop() {
	if nodeRuntime == nil {
		log.Println(
			"autosave error: node runtime unavailable",
		)
		return
	}

	err := nodeRuntime.RunAutosave(
		context.Background(),
		10*time.Second,
		func(err error) {
			log.Println(
				"autosave error:",
				err,
			)
		},
	)
	if err != nil {
		log.Println(
			"autosave stopped:",
			err,
		)
	}
}

func reloadAPIChainFromDiskIfChanged() {
	if nodeRuntime == nil {
		return
	}

	nodeRuntime.SetAPIReadOnly(apiReadOnlyMode)

	reloaded, err := nodeRuntime.ReloadChainFromDiskIfChanged()
	if err != nil {
		log.Printf(
			"api live reload error: %v",
			err,
		)
		return
	}

	if !reloaded {
		return
	}

	loadedBC, _ := nodeRuntime.Snapshot()
	if loadedBC == nil {
		log.Print(
			"api live reload returned nil blockchain",
		)
		return
	}

	log.Printf(
		"api live reload: chain height=%d file=%s",
		loadedBC.BestHeightSnapshot(),
		cfg.ChainFile,
	)
}

func shutdownAPIHTTPServer(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("API shutdown context is nil")
	}

	if nodeRuntime == nil {
		return fmt.Errorf("node runtime unavailable")
	}

	server := nodeRuntime.HTTP()
	if server == nil {
		return fmt.Errorf("API HTTP server unavailable")
	}

	return server.Shutdown(ctx)
}

func trapAPIShutdown(barrier *cliShutdownBarrierV1) {
	signalChannel := make(chan os.Signal, 1)

	signal.Notify(
		signalChannel,
		os.Interrupt,
	)
	defer signal.Stop(signalChannel)

	select {
	case <-barrier.stop:
		return
	case <-signalChannel:
	}
	if !barrier.Begin() {
		return
	}
	defer barrier.Finish()

	fmt.Println()
	fmt.Println("Shutting down API...")

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := shutdownAPIHTTPServer(ctx); err != nil {
		log.Printf(
			"API graceful shutdown error: %v",
			err,
		)
	}
}

/* ---------- Graceful shutdown ---------- */

func trapAndShutdown(barrier *cliShutdownBarrierV1) {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(
		signalChannel,
		os.Interrupt,
	)

	defer signal.Stop(signalChannel)
	select {
	case <-barrier.stop:
		return
	case <-signalChannel:
	}
	if !barrier.Begin() {
		return
	}
	defer barrier.Finish()

	fmt.Println()
	fmt.Println("Shutting down...")

	if nodeRuntime == nil {
		log.Print(
			"shutdown error: node runtime unavailable",
		)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer cancel()

	if err := nodeRuntime.Shutdown(ctx); err != nil {
		log.Printf(
			"shutdown error: %v",
			err,
		)
		os.Exit(1)
	}

	os.Exit(0)
}
