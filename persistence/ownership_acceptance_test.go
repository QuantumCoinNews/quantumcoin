package persistence

import (
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"testing"

	"quantumcoin/blockchain"
)

func TestPersistenceRegressionLegacyChainAuthority(t *testing.T) {
	path := filepath.Join(t.TempDir(), "canonical.dat")
	original := []byte("existing-canonical-artifact")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	blocks := []*blockchain.Block{{Index: 0, Hash: []byte("unverified")}}
	if err := SaveChainV1(path, blocks); err == nil {
		t.Error("standalone chain writer accepted unverified blocks")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, after) {
		t.Error("standalone chain writer overwrote existing artifact")
	}
	var blob bytes.Buffer
	if err := gob.NewEncoder(&blob).Encode(ChainEnvelopeV1{Version: ChainStorageVersionV1, Blocks: blocks}); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(t.TempDir(), "old.gob")
	if err := os.WriteFile(legacy, blob.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadChainV1(legacy); err == nil {
		t.Error("standalone loader accepted unverified legacy chain")
	}
}

func TestPersistenceRegressionPathValidation(t *testing.T) {
	cfg := StorageConfig{RootDir: t.TempDir(), Network: NetworkTestnet}
	for _, row := range []struct {
		name   string
		mutate func(*DataDomain)
	}{
		{"parent-name", func(d *DataDomain) { d.Name = "../../escaped" }},
		{"separator", func(d *DataDomain) { d.Name = "nested/domain" }},
		{"windows-device", func(d *DataDomain) { d.Name = "con" }},
		{"parent-schema", func(d *DataDomain) { d.SchemaVersion = "../../outside" }},
		{"unknown-mode", func(d *DataDomain) { d.StorageMode = "not-a-storage-mode" }},
		{"opposite-network-not-separated", func(d *DataDomain) { d.MainnetSeparated = false }},
	} {
		t.Run(row.name, func(t *testing.T) {
			d := DefaultPlanV1().Domains[0]
			row.mutate(&d)
			if path, err := ResolveDomainPath(cfg, d); err == nil {
				t.Fatalf("invalid domain resolved: %s", path)
			}
		})
	}
	plan := DefaultPlanV1()
	extra := plan.Domains[0]
	extra.Name = "../../unregistered"
	plan.Domains = append(plan.Domains, extra)
	if err := ValidatePlanV1(plan); err == nil {
		t.Error("plan accepted traversing domain name")
	}
}

func TestPersistenceRegressionReleasedLockCannotUnlockNewOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	first, err := AcquireFileLock(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err = first.Release(); err != nil {
		t.Fatal(err)
	}
	second, err := AcquireFileLock(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Release()
	if err = first.Release(); err != nil {
		t.Fatal(err)
	}
	third, err := AcquireFileLock(path, 0)
	if err == nil {
		defer third.Release()
		t.Fatal("old release unlocked a different owner")
	}
}

func TestPersistenceRegressionDryRunIsNotRestoreEvidence(t *testing.T) {
	plan, cfg, evidence, hooks := runtimeReadinessFixtureV1(t)
	if err := os.Remove(evidence[0].RestoreTarget); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := ValidateRuntimeReadinessV1(plan, cfg, evidence, hooks); err == nil {
		t.Fatal("dry-run-only target accepted as executed restore evidence")
	}
}

func TestPersistenceRegressionChainIsSnapshotStorage(t *testing.T) {
	if DefaultPlanV1().Domains[0].AppendOnlyLedger {
		t.Fatal("chain snapshot incorrectly declared append-only")
	}
}
