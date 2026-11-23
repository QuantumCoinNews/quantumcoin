package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"quantumcoin/policyexamples"
	"sort"
)

func main() {
	check := flag.Bool("check", false, "fail if generated documentation differs")
	dir := flag.String("dir", "configs", "fixture directory")
	flag.Parse()
	files, err := policyexamples.Files()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	failed := false
	for _, name := range names {
		path := filepath.Join(*dir, name)
		if *check {
			actual, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(bytes.ReplaceAll(actual, []byte("\r\n"), []byte("\n")), files[name]) {
				fmt.Fprintln(os.Stderr, "generated fixture drift:", name)
				failed = true
			}
		} else if err := os.WriteFile(path, files[name], 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}
