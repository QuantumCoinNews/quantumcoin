package token_test

import (
	"reflect"
	"testing"

	qtoken "quantumcoin/token"
)

func TestStateEncapsulationV1ExternalStructsExposeNoFields(
	t *testing.T,
) {
	types := []reflect.Type{
		reflect.TypeOf(qtoken.Token{}),
		reflect.TypeOf(qtoken.QC20Token{}),
		reflect.TypeOf(qtoken.QC721Token{}),
	}

	for _, currentType := range types {
		for index := 0; index < currentType.NumField(); index++ {
			field := currentType.Field(index)

			if field.IsExported() {
				t.Fatalf(
					"%s exposes mutable field %s",
					currentType.Name(),
					field.Name,
				)
			}
		}
	}
}

func TestStateEncapsulationV1ExternalReadAPIs(
	t *testing.T,
) {
	base := qtoken.New(
		"BAS",
		"Base Token",
		8,
		1000,
		"owner",
	)

	if base.Symbol() != "BAS" ||
		base.Name() != "Base Token" ||
		base.Decimals() != 8 ||
		base.MaxSupply() != 1000 ||
		base.TotalSupply() != 0 ||
		base.Owner() != "owner" {
		t.Fatal("base token scalar API returned incorrect state")
	}

	baseSnapshot := base.BalancesSnapshotV1()
	baseSnapshot["attacker"] = 1000

	if base.BalanceOf("attacker") != 0 {
		t.Fatal("external base snapshot mutated token state")
	}

	qc20 := qtoken.NewQC20TokenWithOwner(
		"Factory Token",
		"FCT",
		1000,
		8,
		"owner",
	)

	if qc20.Name() != "Factory Token" ||
		qc20.Symbol() != "FCT" ||
		qc20.Owner() != "owner" ||
		qc20.MaxSupply() != 1000 ||
		qc20.TotalSupply() != 1000 ||
		qc20.Decimals() != 8 {
		t.Fatal("QC20 scalar API returned incorrect state")
	}

	qc20Balances := qc20.BalancesSnapshotV1()
	qc20Balances["owner"] = 0
	qc20Balances["attacker"] = 1000

	if qc20.BalanceOf("owner") != 1000 ||
		qc20.BalanceOf("attacker") != 0 {
		t.Fatal("external QC20 balance snapshot mutated state")
	}

	qc20Allowances := qc20.AllowancesSnapshotV1()
	qc20Allowances["owner"] =
		map[string]uint64{
			"attacker": 1000,
		}

	if qc20.Allowance("owner", "attacker") != 0 {
		t.Fatal("external QC20 allowance snapshot mutated state")
	}

	qc721 := qtoken.NewQC721TokenWithOwner(
		"Collection",
		"NFT",
		"owner",
	)

	if qc721.Name() != "Collection" ||
		qc721.Symbol() != "NFT" ||
		qc721.Owner() != "owner" {
		t.Fatal("QC721 scalar API returned incorrect state")
	}

	owners := qc721.OwnersSnapshotV1()
	uris := qc721.TokenURIsSnapshotV1()

	owners[1] = "attacker"
	uris[1] = "ipfs://attacker"

	if _, err := qc721.OwnerOf(1); err == nil {
		t.Fatal("external QC721 owner snapshot mutated state")
	}

	if _, err := qc721.TokenURI(1); err == nil {
		t.Fatal("external QC721 URI snapshot mutated state")
	}
}
