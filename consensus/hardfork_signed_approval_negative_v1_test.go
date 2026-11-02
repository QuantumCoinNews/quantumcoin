package consensus

import "testing"

func TestHardforkSignedApprovalRejectsUnauthorizedAdminSignerV1(
	t *testing.T,
) {
	founder := hardforkApprovalTestKeyV1(t)
	admin := hardforkApprovalTestKeyV1(t)
	outsider := hardforkApprovalTestKeyV1(t)

	founderPub, _ := UncompressedPublicKeyV1(founder)
	adminPub, _ := UncompressedPublicKeyV1(admin)

	authorities := HardforkAuthoritySetV1{
		FounderPublicKeys: [][]byte{founderPub},
		AdminPublicKeys:   [][]byte{adminPub},
		FounderQuorum:     1,
		AdminQuorum:       1,
	}

	decision := hardforkApprovalTestDecisionV1()

	approvals := HardforkApprovalSetV1{
		Founder: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(t, founder, decision),
		},
		Admin: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(t, outsider, decision),
		},
	}

	if err := ValidateHardforkApprovalsV1(
		decision,
		authorities,
		approvals,
	); err == nil {
		t.Fatal("unauthorized admin signer was accepted")
	}
}

func TestHardforkSignedApprovalRejectsWrongSignatureV1(
	t *testing.T,
) {
	founder := hardforkApprovalTestKeyV1(t)
	admin := hardforkApprovalTestKeyV1(t)

	founderPub, _ := UncompressedPublicKeyV1(founder)
	adminPub, _ := UncompressedPublicKeyV1(admin)

	authorities := HardforkAuthoritySetV1{
		FounderPublicKeys: [][]byte{founderPub},
		AdminPublicKeys:   [][]byte{adminPub},
		FounderQuorum:     1,
		AdminQuorum:       1,
	}

	decision := hardforkApprovalTestDecisionV1()

	adminApproval :=
		hardforkApprovalTestSignatureV1(t, admin, decision)

	adminApproval.Signature = append(
		[]byte(nil),
		adminApproval.Signature...,
	)
	adminApproval.Signature[len(adminApproval.Signature)-1] ^= 0x01

	approvals := HardforkApprovalSetV1{
		Founder: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(t, founder, decision),
		},
		Admin: []HardforkApprovalV1{
			adminApproval,
		},
	}

	if err := ValidateHardforkApprovalsV1(
		decision,
		authorities,
		approvals,
	); err == nil {
		t.Fatal("mutated admin signature was accepted")
	}
}

func TestHardforkSignedApprovalRejectsInvalidAuthorityQuorumV1(
	t *testing.T,
) {
	founder := hardforkApprovalTestKeyV1(t)
	admin := hardforkApprovalTestKeyV1(t)

	founderPub, _ := UncompressedPublicKeyV1(founder)
	adminPub, _ := UncompressedPublicKeyV1(admin)

	authorities := HardforkAuthoritySetV1{
		FounderPublicKeys: [][]byte{founderPub},
		AdminPublicKeys:   [][]byte{adminPub},
		FounderQuorum:     1,
		AdminQuorum:       2,
	}

	decision := hardforkApprovalTestDecisionV1()

	approvals := HardforkApprovalSetV1{
		Founder: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(t, founder, decision),
		},
		Admin: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(t, admin, decision),
		},
	}

	if err := ValidateHardforkApprovalsV1(
		decision,
		authorities,
		approvals,
	); err == nil {
		t.Fatal("invalid quorum greater than authority count was accepted")
	}
}

func TestHardforkSignedApprovalRejectsDuplicateAuthorityV1(
	t *testing.T,
) {
	founder := hardforkApprovalTestKeyV1(t)
	admin := hardforkApprovalTestKeyV1(t)

	founderPub, _ := UncompressedPublicKeyV1(founder)
	adminPub, _ := UncompressedPublicKeyV1(admin)

	authorities := HardforkAuthoritySetV1{
		FounderPublicKeys: [][]byte{founderPub},
		AdminPublicKeys: [][]byte{
			adminPub,
			adminPub,
		},
		FounderQuorum: 1,
		AdminQuorum:   1,
	}

	decision := hardforkApprovalTestDecisionV1()

	approvals := HardforkApprovalSetV1{
		Founder: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(t, founder, decision),
		},
		Admin: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(t, admin, decision),
		},
	}

	if err := ValidateHardforkApprovalsV1(
		decision,
		authorities,
		approvals,
	); err == nil {
		t.Fatal("duplicate admin authority was accepted")
	}
}

func TestHardforkSignedApprovalRejectsRoleOverlapV1(
	t *testing.T,
) {
	shared := hardforkApprovalTestKeyV1(t)

	sharedPub, _ := UncompressedPublicKeyV1(shared)

	authorities := HardforkAuthoritySetV1{
		FounderPublicKeys: [][]byte{sharedPub},
		AdminPublicKeys:   [][]byte{sharedPub},
		FounderQuorum:     1,
		AdminQuorum:       1,
	}

	decision := hardforkApprovalTestDecisionV1()
	approval :=
		hardforkApprovalTestSignatureV1(t, shared, decision)

	approvals := HardforkApprovalSetV1{
		Founder: []HardforkApprovalV1{approval},
		Admin:   []HardforkApprovalV1{approval},
	}

	if err := ValidateHardforkApprovalsV1(
		decision,
		authorities,
		approvals,
	); err == nil {
		t.Fatal(
			"same signer accepted as both founder and admin authority",
		)
	}
}

func TestHardforkSignedApprovalBindsReasonProtocolNetworkAndGenesisV1(
	t *testing.T,
) {
	founder := hardforkApprovalTestKeyV1(t)
	admin := hardforkApprovalTestKeyV1(t)

	founderPub, _ := UncompressedPublicKeyV1(founder)
	adminPub, _ := UncompressedPublicKeyV1(admin)

	authorities := HardforkAuthoritySetV1{
		FounderPublicKeys: [][]byte{founderPub},
		AdminPublicKeys:   [][]byte{adminPub},
		FounderQuorum:     1,
		AdminQuorum:       1,
	}

	decision := hardforkApprovalTestDecisionV1()

	approvals := HardforkApprovalSetV1{
		Founder: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(t, founder, decision),
		},
		Admin: []HardforkApprovalV1{
			hardforkApprovalTestSignatureV1(t, admin, decision),
		},
	}

	tests := []struct {
		name   string
		mutate func(*HardforkDecisionV1)
	}{
		{
			name: "reason",
			mutate: func(d *HardforkDecisionV1) {
				d.Reasons = []string{"CRITICAL_CONSENSUS_BUG"}
			},
		},
		{
			name: "protocol",
			mutate: func(d *HardforkDecisionV1) {
				d.TargetProtocol = ProtocolContinuityV2
			},
		},
		{
			name: "network",
			mutate: func(d *HardforkDecisionV1) {
				d.NetworkID = "quantumcoin-other-network"
			},
		},
		{
			name: "genesis",
			mutate: func(d *HardforkDecisionV1) {
				d.LegacyGenesisHex =
					"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mutated := decision
			mutated.Reasons =
				append([]string(nil), decision.Reasons...)

			tc.mutate(&mutated)

			if err := ValidateHardforkApprovalsV1(
				mutated,
				authorities,
				approvals,
			); err == nil {
				t.Fatalf(
					"signed decision remained valid after %s mutation",
					tc.name,
				)
			}
		})
	}
}
