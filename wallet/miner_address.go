package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"quantumcoin/config"
)

type MinerAddressOptions struct {
	Config     *config.Config
	Explicit   string
	DataDir    string
	AppDataDir string
}

type MinerAddressSelection struct {
	Address string
	Source  string
}

// ResolveMinerAddress never reads config.json, changes the environment, writes
// files or creates a key. Invalid selected input fails instead of falling back
// to a different reward recipient. Environment precedence is frozen by config.Load.
func ResolveMinerAddress(o MinerAddressOptions) (MinerAddressSelection, error) {
	selectAddress := func(address, source string) (MinerAddressSelection, error) {
		address = strings.TrimSpace(address)
		if len(address) > 128 || !ValidateAddress(address) {
			return MinerAddressSelection{}, fmt.Errorf("invalid miner address in %s", source)
		}
		return MinerAddressSelection{Address: address, Source: source}, nil
	}
	if o.Explicit != "" {
		return selectAddress(o.Explicit, "explicit")
	}
	if o.Config == nil {
		return MinerAddressSelection{}, errors.New("miner configuration unavailable")
	}
	if o.Config.Miner.Address != "" {
		source := o.Config.MinerAddressSource
		if source == "" {
			source = "config.miner.address"
		}
		return selectAddress(o.Config.Miner.Address, source)
	}
	for _, name := range []string{"miner_address.txt", "wallet_address.txt"} {
		raw, err := readMinerAddressFile(filepath.Join(o.DataDir, name), 256)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return MinerAddressSelection{}, fmt.Errorf("cannot read %s", name)
		}
		return selectAddress(string(raw), name)
	}
	if o.AppDataDir != "" {
		raw, err := readMinerAddressFile(filepath.Join(o.AppDataDir, "QuantumCoin", "wallet.json"), 4096)
		if err == nil {
			var legacy struct {
				Address string `json:"address"`
			}
			if json.Unmarshal(raw, &legacy) != nil {
				return MinerAddressSelection{}, errors.New("invalid legacy application wallet")
			}
			return selectAddress(legacy.Address, "legacy-appdata-wallet")
		}
		if !errors.Is(err, os.ErrNotExist) {
			return MinerAddressSelection{}, errors.New("legacy application wallet cannot be read")
		}
	}
	return MinerAddressSelection{}, errors.New("configure a miner address from a wallet whose private key you have backed up")
}

func readMinerAddressFile(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return nil, errors.New("miner address file exceeds limit or cannot be read")
	}
	return raw, nil
}
