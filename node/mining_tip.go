package node

import "quantumcoin/blockchain"

func (r *Runtime) MiningTip() blockchain.MiningTip {
	if r == nil {
		return (*blockchain.Blockchain)(nil).MiningTipSnapshot()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Blockchain.MiningTipSnapshot()
}
