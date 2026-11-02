package blockchain

import "fmt"

// ValidateBlockPoWAtHeightV1 selects PoW validation using the blockchain's
// bound activation policy and the candidate block height.
func (bc *Blockchain) ValidateBlockPoWAtHeightV1(block *Block) error {
	if bc == nil {
		return fmt.Errorf("canonical PoW activation dispatch: blockchain cannot be nil")
	}
	if block == nil {
		return fmt.Errorf("canonical PoW activation dispatch: block cannot be nil")
	}

	protocol := bc.ProtocolAtHeightV1(block.Index)
	return ValidateBlockPoWAtProtocolV1(block, protocol)
}
