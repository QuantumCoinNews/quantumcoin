// recovery is an explicit offline utility. Node startup never calls it.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"quantumcoin/backuprestore/offline"
)

func run(args []string, in *os.File, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: recovery seal-wallet|unseal-wallet|recover-wallet|materialize|snapshot|verify|dry-run|restore|rollback|current [flags]")
	}
	f := flag.NewFlagSet("recovery", flag.ContinueOnError)
	f.SetOutput(out)
	source := f.String("source", "", "source wallet/envelope or snapshot directory")
	target := f.String("target", "", "NEW output file/directory, or offline restore workspace")
	planPath := f.String("plan", "", "snapshot source plan JSON")
	network := f.String("network", "", "expected local-release, testnet, or mainnet")
	genesis := f.String("genesis", "", "independently pinned genesis hash")
	python := f.String("python", "", "explicit Python 3 executable for SQLite")
	scratch := f.String("scratch", "", "operator-controlled scratch directory on protected/encrypted storage")
	isOffline := f.Bool("offline", false, "operator declares sources quiesced / recovery workspace offline")
	passwordStdin := f.Bool("password-stdin", false, "read password from non-terminal stdin; never pass passwords as arguments")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return errors.New("unexpected recovery arguments")
	}
	switch args[0] {
	case "seal-wallet", "unseal-wallet", "recover-wallet", "materialize", "snapshot", "verify", "dry-run", "restore", "rollback", "current":
	default:
		return errors.New("unknown recovery command")
	}
	if !*passwordStdin {
		return errors.New("--password-stdin required; use a secret manager or secure prompt pipe")
	}
	st, err := in.Stat()
	if err != nil {
		return err
	}
	if st.Mode()&os.ModeCharDevice != 0 {
		return errors.New("refusing echoed password input from terminal")
	}
	password, err := io.ReadAll(io.LimitReader(in, 1025))
	if err != nil {
		return err
	}
	defer func() {
		for i := range password {
			password[i] = 0
		}
	}()
	if len(password) > 1024 {
		return errors.New("password input too large")
	}
	password = bytes.TrimSuffix(bytes.TrimSuffix(password, []byte("\n")), []byte("\r"))
	if len(password) < 12 {
		return errors.New("password requires at least 12 bytes")
	}
	e := offline.Engine{Python: *python, Scratch: *scratch}
	ctx := context.Background()
	selected := ""
	switch args[0] {
	case "seal-wallet":
		err = offline.SealWalletFile(*source, *target, password)
	case "unseal-wallet":
		if !*isOffline {
			return errors.New("--offline required to unseal a wallet")
		}
		err = offline.UnsealWalletFile(*source, *target, password)
	case "recover-wallet":
		if !*isOffline {
			return errors.New("--offline required to recover a wallet")
		}
		err = e.RecoverWallet(ctx, *source, *target, *network, *genesis, password)
	case "materialize":
		_, err = e.Materialize(ctx, *source, *target, *network, *genesis, password, *isOffline)
	case "snapshot":
		file, openErr := os.Open(*planPath)
		if openErr != nil {
			return openErr
		}
		defer file.Close()
		var p offline.Plan
		d := json.NewDecoder(io.LimitReader(file, 1<<20))
		d.DisallowUnknownFields()
		if err = d.Decode(&p); err != nil {
			return errors.New("invalid snapshot plan")
		}
		if d.Decode(new(any)) != io.EOF {
			return errors.New("trailing snapshot plan")
		}
		if !*isOffline || !p.Offline {
			return errors.New("--offline and offline plan required")
		}
		_, err = e.Snapshot(ctx, p, *target, password)
	case "verify":
		_, err = e.Verify(ctx, *source, *network, *genesis, password)
	case "dry-run":
		err = e.DryRun(ctx, *source, *target, *network, *genesis, password)
	case "restore":
		selected, err = e.Restore(ctx, *source, *target, *network, *genesis, password, *isOffline)
	case "rollback":
		selected, err = e.Rollback(ctx, *target, *network, *genesis, password, *isOffline)
	case "current":
		selected, err = e.Current(ctx, *target, *network, *genesis, password)
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%s completed; selected generation: %s\n", args[0], selected)
	return err
}
func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
