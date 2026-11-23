package backuprestore

import "testing"

func TestEveryRecoveryArtifactRequiresEncryption(t *testing.T) {
	for i, a := range DefaultRunbookV1().Artifacts {
		t.Run(a.Name, func(t *testing.T) {
			r := DefaultRunbookV1()
			r.Artifacts[i].RequiresEncryption = false
			if ValidateRunbookV1(r) == nil {
				t.Fatal("plaintext backup opt-out accepted")
			}
		})
	}
	r := DefaultRunbookV1()
	if r.Artifacts[0].AppendOnlyPreferred {
		t.Fatal("full-file chain store advertised append-only")
	}
}

func TestSingleValidationRequiresEveryRecoveryRole(t *testing.T) {
	for _, name := range RequiredArtifactNamesV1() {
		t.Run(name, func(t *testing.T) {
			r := DefaultRunbookV1()
			for i, a := range r.Artifacts {
				if a.Name == name {
					r.Artifacts = append(r.Artifacts[:i], r.Artifacts[i+1:]...)
					break
				}
			}
			if ValidateRunbookV1(r) == nil {
				t.Fatal("incomplete recovery accepted")
			}
		})
	}
}
func TestRecoveryRejectsUnknownKindsAndUnencryptedWallet(t *testing.T) {
	for _, mutate := range []func(*Runbook){
		func(r *Runbook) { r.Artifacts[0].Kind = "chain_flie" },
		func(r *Runbook) { r.Artifacts[0].Kind = ArtifactConfig },
		func(r *Runbook) { r.Artifacts[len(r.Artifacts)-2].RequiresEncryption = false },
		func(r *Runbook) {
			a := r.Artifacts[0]
			a.Name = "additional"
			a.Kind = "future_unknown"
			r.Artifacts = append(r.Artifacts, a)
		},
		func(r *Runbook) { r.Artifacts[0].Name = " quantumcoin_chain_file " },
	} {
		r := DefaultRunbookV1()
		mutate(&r)
		if ValidateRunbookV1(r) == nil {
			t.Fatal("unsafe recovery accepted")
		}
	}
}
