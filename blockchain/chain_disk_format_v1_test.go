package blockchain

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestChainDiskEnvelopeIntegrityAndLegacyUpgrade(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chain.dat")
	chain := identityTestChain(t, "testnet-local")
	if err := os.WriteFile(path, SerializeBlockchain(chain), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(path, chain)
	if err != nil {
		t.Fatal("legacy load", err)
	}
	if err := loaded.SaveToFile(path); err != nil {
		t.Fatal("legacy upgrade", err)
	}
	valid, err := os.ReadFile(path)
	if err != nil || !bytes.HasPrefix(valid, []byte(chainDiskMagicV1)) {
		t.Fatal("missing versioned header", err)
	}
	if _, err := LoadBlockchainFromFileWithLocalContextV1(path, chain); err != nil {
		t.Fatal("framed reload", err)
	}
	foreign := identityTestChain(t, "mainnet")
	if _, err := LoadBlockchainFromFileWithLocalContextV1(path, foreign); err == nil {
		t.Fatal("different network accepted the same file")
	}
	if DeserializeBlockchain(valid) != nil {
		t.Fatal("disk format accepted as a P2P payload")
	}

	for name, corrupt := range map[string]func([]byte) []byte{
		"bad version": func(b []byte) []byte { b[7]++; return b },
		"truncated":   func(b []byte) []byte { return b[:len(b)-1] },
		"trailing":    func(b []byte) []byte { return append(b, 1) },
		"payload":     func(b []byte) []byte { b[len(b)-1] ^= 1; return b },
		"header identity": func(b []byte) []byte {
			var h chainDiskHeaderV1
			if err := binary.Read(bytes.NewReader(b), binary.BigEndian, &h); err != nil {
				t.Fatal(err)
			}
			h.Genesis[0] ^= 1
			h.ChainID = h.chainID()
			var out bytes.Buffer
			if err := binary.Write(&out, binary.BigEndian, h); err != nil {
				t.Fatal(err)
			}
			return append(out.Bytes(), b[binary.Size(h):]...)
		},
	} {
		t.Run(name, func(t *testing.T) {
			data := corrupt(bytes.Clone(valid))
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadBlockchainFromFile(path); err == nil {
				t.Fatal("malformed chain file loaded")
			}
			if err := chain.SaveToFile(path); err == nil {
				t.Fatal("save overwrote damaged chain file")
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(after, data) {
				t.Fatal("damaged destination not preserved", err)
			}
		})
	}
	if err := os.WriteFile(path, []byte("malformed gob"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBlockchainFromFile(path); err == nil {
		t.Fatal("bad legacy file loaded")
	}
}
