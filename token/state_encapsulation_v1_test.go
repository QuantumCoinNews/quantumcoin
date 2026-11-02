package token

import (
	"encoding/json"
	"sync"
	"testing"
)

func TestStateEncapsulationV1SnapshotsAreDefensive(
	t *testing.T,
) {
	base := New(
		"BAS",
		"Base Token",
		8,
		1000,
		"owner",
	)

	base.mu.Lock()
	base.balances["owner"] = 100
	base.mu.Unlock()

	baseSnapshot := base.BalancesSnapshotV1()
	baseSnapshot["owner"] = 999
	baseSnapshot["attacker"] = 50

	if balance := base.BalanceOf("owner"); balance != 100 {
		t.Fatalf(
			"base balance changed through snapshot: %d",
			balance,
		)
	}

	if balance := base.BalanceOf("attacker"); balance != 0 {
		t.Fatalf(
			"base snapshot created attacker balance: %d",
			balance,
		)
	}

	qc20 := NewQC20TokenWithOwner(
		"Factory Token",
		"FCT",
		1000,
		8,
		"owner",
	)

	qc20.mu.Lock()
	qc20.allowances["owner"] =
		map[string]uint64{
			"spender": 25,
		}
	qc20.mu.Unlock()

	qc20Balances := qc20.BalancesSnapshotV1()
	qc20Balances["owner"] = 1
	qc20Balances["attacker"] = 999

	qc20Allowances := qc20.AllowancesSnapshotV1()
	qc20Allowances["owner"]["spender"] = 999
	qc20Allowances["attacker"] =
		map[string]uint64{
			"spender": 1000,
		}

	if balance := qc20.BalanceOf("owner"); balance != 1000 {
		t.Fatalf(
			"QC20 balance changed through snapshot: %d",
			balance,
		)
	}

	if allowance := qc20.Allowance(
		"owner",
		"spender",
	); allowance != 25 {
		t.Fatalf(
			"QC20 allowance changed through snapshot: %d",
			allowance,
		)
	}

	if allowance := qc20.Allowance(
		"attacker",
		"spender",
	); allowance != 0 {
		t.Fatalf(
			"QC20 snapshot created attacker allowance: %d",
			allowance,
		)
	}

	qc721 := NewQC721TokenWithOwner(
		"Collection",
		"NFT",
		"owner",
	)

	qc721.mu.Lock()
	qc721.owners[1] = "holder"
	qc721.tokenURIs[1] = "ipfs://original"
	qc721.mu.Unlock()

	owners := qc721.OwnersSnapshotV1()
	uris := qc721.TokenURIsSnapshotV1()

	owners[1] = "attacker"
	owners[2] = "attacker"
	uris[1] = "ipfs://tampered"
	uris[2] = "ipfs://attacker"

	owner, err := qc721.OwnerOf(1)
	if err != nil {
		t.Fatalf("read QC721 owner: %v", err)
	}

	if owner != "holder" {
		t.Fatalf(
			"QC721 owner changed through snapshot: %q",
			owner,
		)
	}

	uri, err := qc721.TokenURI(1)
	if err != nil {
		t.Fatalf("read QC721 URI: %v", err)
	}

	if uri != "ipfs://original" {
		t.Fatalf(
			"QC721 URI changed through snapshot: %q",
			uri,
		)
	}

	if _, err := qc721.OwnerOf(2); err == nil {
		t.Fatal("QC721 snapshot created a new token owner")
	}
}

func requireExactJSONKeysV1(
	t *testing.T,
	raw []byte,
	expected ...string,
) {
	t.Helper()

	var object map[string]json.RawMessage

	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatalf("decode JSON object: %v", err)
	}

	if len(object) != len(expected) {
		t.Fatalf(
			"JSON key count = %d, want %d: %s",
			len(object),
			len(expected),
			string(raw),
		)
	}

	for _, key := range expected {
		if _, exists := object[key]; !exists {
			t.Fatalf(
				"historical JSON key missing: %q",
				key,
			)
		}
	}
}

func TestStateEncapsulationV1HistoricalJSONKeys(
	t *testing.T,
) {
	owner := canonicalAddressFixture(
		"owner",
	)

	base := New(
		"BAS",
		"Base Token",
		8,
		1000,
		owner,
	)

	baseRaw, err := json.Marshal(base)
	if err != nil {
		t.Fatalf("marshal base token: %v", err)
	}

	requireExactJSONKeysV1(
		t,
		baseRaw,
		"symbol",
		"name",
		"decimals",
		"max_supply",
		"total_supply",
		"owner",
		"balances",
	)

	var loadedBase Token

	if err := json.Unmarshal(
		baseRaw,
		&loadedBase,
	); err != nil {
		t.Fatalf("reload base token: %v", err)
	}

	if loadedBase.Symbol() != "BAS" ||
		loadedBase.Name() != "Base Token" ||
		loadedBase.Decimals() != 8 ||
		loadedBase.MaxSupply() != 1000 ||
		loadedBase.Owner() != owner {
		t.Fatal("base token state changed after JSON round trip")
	}

	qc20 := NewQC20TokenWithOwner(
		"Factory Token",
		"FCT",
		1000,
		8,
		owner,
	)

	qc20Raw, err := json.Marshal(qc20)
	if err != nil {
		t.Fatalf("marshal QC20 token: %v", err)
	}

	requireExactJSONKeysV1(
		t,
		qc20Raw,
		"Name",
		"Symbol",
		"Owner",
		"MaxSupply",
		"TotalSupply",
		"Decimals",
		"Balances",
		"Allowances",
	)

	var loadedQC20 QC20Token

	if err := json.Unmarshal(
		qc20Raw,
		&loadedQC20,
	); err != nil {
		t.Fatalf("reload QC20 token: %v", err)
	}

	if loadedQC20.Name() != "Factory Token" ||
		loadedQC20.Symbol() != "FCT" ||
		loadedQC20.Owner() != owner ||
		loadedQC20.MaxSupply() != 1000 ||
		loadedQC20.TotalSupply() != 1000 ||
		loadedQC20.Decimals() != 8 {
		t.Fatal("QC20 state changed after JSON round trip")
	}

	qc721 := NewQC721TokenWithOwner(
		"Collection",
		"NFT",
		owner,
	)

	qc721Raw, err := json.Marshal(qc721)
	if err != nil {
		t.Fatalf("marshal QC721 token: %v", err)
	}

	requireExactJSONKeysV1(
		t,
		qc721Raw,
		"Name",
		"Symbol",
		"Owner",
		"Owners",
		"TokenURIs",
	)

	var loadedQC721 QC721Token

	if err := json.Unmarshal(
		qc721Raw,
		&loadedQC721,
	); err != nil {
		t.Fatalf("reload QC721 token: %v", err)
	}

	if loadedQC721.Name() != "Collection" ||
		loadedQC721.Symbol() != "NFT" ||
		loadedQC721.Owner() != owner {
		t.Fatal("QC721 state changed after JSON round trip")
	}
}

func TestStateEncapsulationV1ConcurrentReads(
	t *testing.T,
) {
	base := New(
		"BAS",
		"Base Token",
		8,
		1000,
		"owner",
	)

	qc20 := NewQC20TokenWithOwner(
		"Factory Token",
		"FCT",
		1000,
		8,
		"owner",
	)

	qc721 := NewQC721TokenWithOwner(
		"Collection",
		"NFT",
		"owner",
	)

	start := make(chan struct{})

	var wait sync.WaitGroup
	wait.Add(2)

	go func() {
		defer wait.Done()
		<-start

		for index := 0; index < 200; index++ {
			base.mu.Lock()
			base.balances["owner"] =
				uint64(index)
			base.mu.Unlock()

			qc20.mu.Lock()
			qc20.balances["owner"] =
				uint64(index)
			qc20.allowances["owner"] =
				map[string]uint64{
					"spender": uint64(index),
				}
			qc20.mu.Unlock()

			qc721.mu.Lock()
			qc721.owners[uint64(index)] =
				"owner"
			qc721.tokenURIs[uint64(index)] =
				"ipfs://state"
			qc721.mu.Unlock()
		}
	}()

	go func() {
		defer wait.Done()
		<-start

		for index := 0; index < 200; index++ {
			_ = base.Symbol()
			_ = base.Name()
			_ = base.Decimals()
			_ = base.MaxSupply()
			_ = base.TotalSupply()
			_ = base.Owner()
			_ = base.BalancesSnapshotV1()

			_ = qc20.Name()
			_ = qc20.Symbol()
			_ = qc20.Owner()
			_ = qc20.MaxSupply()
			_ = qc20.TotalSupply()
			_ = qc20.Decimals()
			_ = qc20.BalancesSnapshotV1()
			_ = qc20.AllowancesSnapshotV1()

			_ = qc721.Name()
			_ = qc721.Symbol()
			_ = qc721.Owner()
			_ = qc721.OwnersSnapshotV1()
			_ = qc721.TokenURIsSnapshotV1()
		}
	}()

	close(start)
	wait.Wait()
}
