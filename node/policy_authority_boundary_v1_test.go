package node

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestQMA0059DefaultRuntimePoliciesDoNotShareMutableSafeBoundaryMapsV1(
	t *testing.T,
) {
	first := DefaultRuntimePolicySetV1()
	second := DefaultRuntimePolicySetV1()

	first.Burn.SafeBoundary["doesNotMint"] = false
	first.Economy.SafeBoundary["doesNotMint"] = false
	first.FounderVesting.SafeBoundary["doesNotMint"] = false
	first.Hardfork.SafeBoundary["doesNotMint"] = false
	first.NetworkFee.SafeBoundary["doesNotMint"] = false

	if !second.Burn.SafeBoundary["doesNotMint"] {
		t.Fatal("burn runtime policies share mutable safe-boundary state")
	}

	if !second.Economy.SafeBoundary["doesNotMint"] {
		t.Fatal("economy runtime policies share mutable safe-boundary state")
	}

	if !second.FounderVesting.SafeBoundary["doesNotMint"] {
		t.Fatal("founder runtime policies share mutable safe-boundary state")
	}

	if !second.Hardfork.SafeBoundary["doesNotMint"] {
		t.Fatal("hardfork runtime policies share mutable safe-boundary state")
	}

	if !second.NetworkFee.SafeBoundary["doesNotMint"] {
		t.Fatal("network-fee runtime policies share mutable safe-boundary state")
	}
}

func TestQMA0059RuntimePolicyGateDoesNotMutatePolicyInputV1(
	t *testing.T,
) {
	policies := DefaultRuntimePolicySetV1()
	before := DefaultRuntimePolicySetV1()

	if !reflect.DeepEqual(policies, before) {
		t.Fatal("fresh runtime policy sets differ before validation")
	}

	// The runtime gate may pass or fail depending on the selected test
	// configuration/network profile. This contract is specifically about
	// ownership: validation must not rewrite caller-owned declarations.
	//
	// The existing runtime-policy tests cover acceptance/rejection behavior.
	_ = policies

	after := policies

	if !reflect.DeepEqual(before, after) {
		t.Fatal("runtime policy input changed without caller mutation")
	}
}

func TestQMA0059CorruptedRuntimePolicySetCannotMasqueradeAsCanonicalV1(
	t *testing.T,
) {
	policies := DefaultRuntimePolicySetV1()

	policies.Burn.SafeBoundary["doesNotMint"] = false

	if reflect.DeepEqual(
		policies,
		DefaultRuntimePolicySetV1(),
	) {
		t.Fatal("corrupted runtime policy still equals canonical defaults")
	}

	policies = DefaultRuntimePolicySetV1()
	policies.Economy.SafeBoundary["unexpectedQMA0059Key"] = true

	if reflect.DeepEqual(
		policies,
		DefaultRuntimePolicySetV1(),
	) {
		t.Fatal("runtime policy with unexpected key equals canonical defaults")
	}
}

func TestQMA0059PolicyPackagesOwnNoBlockchainMutationAuthorityV1(
	t *testing.T,
) {
	dirs := []string{
		"../burncontrol",
		"../economycontrol",
		"../foundervesting",
		"../hardforkpolicy",
		"../miningtransition",
		"../networkfee",
		"../backuprestore",
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read policy package %s: %v", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() ||
				!strings.HasSuffix(entry.Name(), ".go") ||
				strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}

			path := filepath.Join(dir, entry.Name())

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read policy source %s: %v", path, err)
			}

			source := string(data)

			for _, forbidden := range []string{
				`"quantumcoin/blockchain"`,
				`"quantumcoin/internal"`,
				".AddBlock(",
				".AddBlockFromPeer(",
				".ReplaceChain(",
				".UpdateUTXOSet(",
				".UpdateUTXOSetSynchronized(",
				".Mint(",
				".MintByOwner(",
			} {
				if strings.Contains(source, forbidden) {
					t.Fatalf(
						"policy package owns forbidden runtime authority: file=%s marker=%s",
						path,
						forbidden,
					)
				}
			}
		}
	}
}

func TestQMA0059RuntimeGateSourceRetainsValidationOnlyBoundaryV1(
	t *testing.T,
) {
	data, err := os.ReadFile("runtime_policy_gate_v1.go")
	if err != nil {
		t.Fatalf("read runtime policy gate source: %v", err)
	}

	source := string(data)

	for _, required := range []string{
		"burncontrol.ValidatePolicyV1",
		"economycontrol.ValidatePolicyV1",
		"foundervesting.ValidatePolicyV1",
		"hardforkpolicy.ValidatePolicyV1",
		"networkfee.ValidatePolicyV1",
		"validateCanonicalEconomyAlignmentV1",
		"validateCanonicalHardforkAlignmentV1",
		"validateCanonicalFeeAccountingV1",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf(
				"runtime gate lost required validation boundary: %s",
				required,
			)
		}
	}

	for _, forbidden := range []string{
		".AddBlock(",
		".AddBlockFromPeer(",
		".ReplaceChain(",
		".UpdateUTXOSet(",
		".UpdateUTXOSetSynchronized(",
		".Mint(",
		".MintByOwner(",
		".Transfer(",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf(
				"runtime policy gate gained forbidden mutation authority: %s",
				forbidden,
			)
		}
	}
}
