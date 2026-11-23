package consensus

import (
	"math/bits"
	"testing"
)

func TestHardforkQuorumEveryFounderAdminSubset(t *testing.T) {
	decision := hardforkApprovalTestDecisionV1()
	authorities := HardforkAuthoritySetV1{FounderQuorum: 2, AdminQuorum: 2}
	var founders, admins []HardforkApprovalV1
	for i := 0; i < 3; i++ {
		f, a := hardforkApprovalTestKeyV1(t), hardforkApprovalTestKeyV1(t)
		fp, err := UncompressedPublicKeyV1(f)
		if err != nil {
			t.Fatal(err)
		}
		ap, err := UncompressedPublicKeyV1(a)
		if err != nil {
			t.Fatal(err)
		}
		authorities.FounderPublicKeys = append(authorities.FounderPublicKeys, fp)
		authorities.AdminPublicKeys = append(authorities.AdminPublicKeys, ap)
		founders = append(founders, hardforkApprovalTestSignatureV1(t, f, decision))
		admins = append(admins, hardforkApprovalTestSignatureV1(t, a, decision))
	}
	for fmask := 0; fmask < 8; fmask++ {
		for amask := 0; amask < 8; amask++ {
			approvals := HardforkApprovalSetV1{}
			for i := 0; i < 3; i++ {
				if fmask&(1<<i) != 0 {
					approvals.Founder = append(approvals.Founder, founders[i])
				}
				if amask&(1<<i) != 0 {
					approvals.Admin = append(approvals.Admin, admins[i])
				}
			}
			want := bits.OnesCount(uint(fmask)) >= 2 && bits.OnesCount(uint(amask)) >= 2
			if err := ValidateHardforkApprovalsV1(decision, authorities, approvals); (err == nil) != want {
				t.Fatalf("founders=%03b admins=%03b valid=%v want=%v err=%v", fmask, amask, err == nil, want, err)
			}
		}
	}
}

func TestHardforkSignedDecisionRejectsEveryFieldMutation(t *testing.T) {
	decision := hardforkApprovalTestDecisionV1()
	founder, admin := hardforkApprovalTestKeyV1(t), hardforkApprovalTestKeyV1(t)
	fp, _ := UncompressedPublicKeyV1(founder)
	ap, _ := UncompressedPublicKeyV1(admin)
	authority := HardforkAuthoritySetV1{FounderPublicKeys: [][]byte{fp}, AdminPublicKeys: [][]byte{ap}, FounderQuorum: 1, AdminQuorum: 1}
	approvals := HardforkApprovalSetV1{Founder: []HardforkApprovalV1{hardforkApprovalTestSignatureV1(t, founder, decision)}, Admin: []HardforkApprovalV1{hardforkApprovalTestSignatureV1(t, admin, decision)}}
	cases := map[string]func(*HardforkDecisionV1){
		"network": func(d *HardforkDecisionV1) { d.NetworkID += "-other" },
		"genesis": func(d *HardforkDecisionV1) {
			d.LegacyGenesisHex = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
		},
		"height":   func(d *HardforkDecisionV1) { d.ActivationHeight++ },
		"protocol": func(d *HardforkDecisionV1) { d.TargetProtocol = ProtocolNFTV4 },
		"reason":   func(d *HardforkDecisionV1) { d.Reasons = []string{"EMERGENCY_CONSENSUS_SECURITY_FIX"} },
		"snapshot": func(d *HardforkDecisionV1) {
			d.SnapshotSHA256 = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
		},
		"binary": func(d *HardforkDecisionV1) {
			d.ReleaseBinarySHA256 = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
		},
	}
	if err := ValidateHardforkApprovalsV1(decision, authority, approvals); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := decision
			mutate(&d)
			if err := ValidateHardforkApprovalsV1(d, authority, approvals); err == nil {
				t.Fatal("changed signed field retained approval")
			}
		})
	}
}
