package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// Source contracts cover every canonical command/HTTP adapter after the split.
// Tool entry points such as cmd/miner/main.go remain separate, literal files.
func readNodeContractSourceV1(path string) ([]byte, error) {
	if filepath.Clean(path) == "main.go" {
		return readNodeApplicationSourcesV1()
	}
	return os.ReadFile(path)
}

func readNodeApplicationSourcesV1() ([]byte, error) {
	files, err := filepath.Glob("node_app_*.go")
	if err != nil {
		return nil, err
	}
	var source bytes.Buffer
	for _, path := range append([]string{"main.go"}, files...) {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		source.Write(data)
		source.WriteByte('\n')
	}
	return source.Bytes(), nil
}
