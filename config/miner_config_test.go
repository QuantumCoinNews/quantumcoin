package config

import (
	"testing"
)

func TestMinerConfigTypedLoadingAndFrozenEnvPrecedence(t *testing.T) {
	clearConfigLoaderEnvV1(t)
	t.Setenv("QC_MINER", "")
	previous := Current()
	t.Cleanup(func() { Set(previous) })
	path := writeConfigFixtureV1(t, `{"Miner":{"Address":"file-address"}}`)
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Miner.Address != "file-address" || c.MinerAddressSource != "config.miner.address" {
		t.Fatal("legacy typed config not decoded")
	}
	t.Setenv("QC_MINER", " env-address ")
	c, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Miner.Address != "env-address" || c.MinerAddressSource != "QC_MINER" {
		t.Fatal("environment precedence wrong")
	}
	t.Setenv("QC_MINER", "later-address")
	if c.Miner.Address != "env-address" || Current().Miner.Address != "env-address" {
		t.Fatal("loaded config changed after environment mutation")
	}
	bad := writeConfigFixtureV1(t, `{"miner":{"address":"x","unknown":true}}`)
	if _, err := Load(bad); err == nil {
		t.Fatal("loose miner configuration accepted")
	}
}
