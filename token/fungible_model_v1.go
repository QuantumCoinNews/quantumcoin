package token

// FungibleTokenModelV1 identifies the concrete fungible token family.
type FungibleTokenModelV1 string

const (
	FungibleTokenModelControlledEmissionV1 FungibleTokenModelV1 = FungibleTokenModelV1(TokenAuthorizationModelLegacyV1)
	FungibleTokenModelFixedSupplyV1        FungibleTokenModelV1 = FungibleTokenModelV1(TokenAuthorizationModelQC20V1)
)

// FungibleSupplyPolicyV1 describes whether supply can be issued after
// construction. It is intentionally separate from the common read contract.
type FungibleSupplyPolicyV1 string

const (
	FungibleSupplyPolicyControlledEmissionV1 FungibleSupplyPolicyV1 = "CONTROLLED_EMISSION"
	FungibleSupplyPolicyFixedSupplyV1        FungibleSupplyPolicyV1 = "FIXED_SUPPLY"
)

// FungibleTokenReadV1 exposes only locked scalar reads and defensive
// snapshots shared by Token and QC20Token. Mutating methods are deliberately
// excluded because the two models have different supply contracts.
type FungibleTokenReadV1 interface {
	FungibleModelV1() FungibleTokenModelV1
	FungibleSupplyPolicyV1() FungibleSupplyPolicyV1
	Name() string
	Symbol() string
	Decimals() uint8
	MaxSupply() uint64
	TotalSupply() uint64
	Owner() string
	BalanceOf(string) uint64
	BalancesSnapshotV1() map[string]uint64
}

func (t *Token) FungibleModelV1() FungibleTokenModelV1 {
	return FungibleTokenModelControlledEmissionV1
}

func (t *Token) FungibleSupplyPolicyV1() FungibleSupplyPolicyV1 {
	return FungibleSupplyPolicyControlledEmissionV1
}

func (t *QC20Token) FungibleModelV1() FungibleTokenModelV1 {
	return FungibleTokenModelFixedSupplyV1
}

func (t *QC20Token) FungibleSupplyPolicyV1() FungibleSupplyPolicyV1 {
	return FungibleSupplyPolicyFixedSupplyV1
}

func validFungibleTokenModelV1(
	model FungibleTokenModelV1,
) bool {
	switch model {
	case FungibleTokenModelControlledEmissionV1,
		FungibleTokenModelFixedSupplyV1:
		return true
	default:
		return false
	}
}

var (
	_ FungibleTokenReadV1 = (*Token)(nil)
	_ FungibleTokenReadV1 = (*QC20Token)(nil)
)
