package internal

// FeeManager is a retired compatibility surface. Transaction fees are
// consensus values derived from verified canonical UTXO input/output balance
// by blockchain.CalculateBlockFeesV1. This type must not maintain a second
// fee policy, mutate transactions, or accumulate fee-like RAM state.
type FeeManager struct{}

func NewFeeManager() *FeeManager { return &FeeManager{} }

// SetFixedFee is retained for source compatibility. Node-local fixed fees are
// not consensus-authoritative and therefore cannot be enabled here.
func (fm *FeeManager) SetFixedFee(_ int) {}

// GetFixedFee returns zero because this retired adapter owns no fee policy.
func (fm *FeeManager) GetFixedFee() int { return 0 }

// ApplyFee is fail-closed. Its legacy interface lacks canonical chain and UTXO
// context, so accepting or charging any value here would create a second,
// unverifiable fee model. Canonical fee accounting occurs during transaction
// and block validation in package blockchain.
func (fm *FeeManager) ApplyFee(_ interface{}) int { return 0 }

// GetTotalFees returns zero because canonical fees are never accumulated here.
func (fm *FeeManager) GetTotalFees() int { return 0 }

// Reset is retained for source compatibility and intentionally has no state.
func (fm *FeeManager) Reset() {}
