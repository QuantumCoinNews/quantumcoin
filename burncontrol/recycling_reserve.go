package burncontrol

import "fmt"

// RecyclingReservePolicy describes a distinct future transfer mechanism.
// Funds held by a spendable reserve are not burned. This proposal is disabled;
// it never authorizes minting or spending the permanent burn sink.
type RecyclingReservePolicy struct {
	Mechanism                  string `json:"mechanism"`
	Enabled                    bool   `json:"enabled"`
	TransfersExistingQCOnly    bool   `json:"transfersExistingQcOnly"`
	RequiresOwnerSignature     bool   `json:"requiresOwnerSignature"`
	CountsAsBurnedSupply       bool   `json:"countsAsBurnedSupply"`
	MayMintReplacement         bool   `json:"mayMintReplacement"`
	MaySpendBurnSink           bool   `json:"maySpendBurnSink"`
	RequiresSeparateActivation bool   `json:"requiresSeparateActivation"`
}

func DefaultRecyclingReservePolicyV1() RecyclingReservePolicy {
	return RecyclingReservePolicy{Mechanism: "RECYCLING_RESERVE_PROPOSAL", TransfersExistingQCOnly: true, RequiresOwnerSignature: true, RequiresSeparateActivation: true}
}
func ValidateRecyclingReservePolicyV1(p RecyclingReservePolicy) error {
	if p != DefaultRecyclingReservePolicyV1() {
		return fmt.Errorf("recycling reserve is an inactive proposal; permanent burn cannot be recycled or re-minted")
	}
	return nil
}
