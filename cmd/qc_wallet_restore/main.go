package main

import (
	"flag"
	"fmt"
	"os"
	"quantumcoin/cmd/internal/walletcli"
)

func main() {
	path := flag.String("out", "", "NEW encrypted .qcvault file (never overwritten)")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Secrets must enter through local stdin JSON, never arguments")
		os.Exit(1)
	}
	if err := walletcli.Run("restore", *path, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
