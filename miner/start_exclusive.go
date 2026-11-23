package miner

import (
	"errors"

	"quantumcoin/blockchain"
)

// ErrMinerAlreadyActiveV1 distinguishes a rejected exclusive start from a new
// mining loop. The existing Start API retains its idempotent behavior.
var ErrMinerAlreadyActiveV1 = errors.New("miner: already active; stop before exclusive restart")

// StartExclusiveV1 runs prepare only after validating the start request and
// finding the lifecycle idle. The lifecycle mutex covers preparation and loop
// installation, so another Start cannot make the result a silent no-op.
// prepare must not call Start, Stop or another lifecycle-changing miner method.
// A preparation error is returned without starting a loop.
func StartExclusiveV1(bc *blockchain.Blockchain, address string, difficulty int, prepare func() (Options, error)) error {
	if prepare == nil {
		return errors.New("miner: exclusive start requires preparation")
	}
	return startPreparedV1(bc, address, difficulty, true, func() (Options, error) {
		options, err := prepare()
		if err != nil {
			return Options{}, err
		}
		return MergeOptionsV1(options), nil
	})
}
