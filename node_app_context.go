package main

import (
	"quantumcoin/config"
	qnode "quantumcoin/node"
)

// nodeRuntime owns the chain, configuration and service lifecycle. Adapters
// capture one chain snapshot per operation; no global chain pointer is kept.
// cfg is immutable startup configuration.

var (
	nodeRuntime     *qnode.Runtime
	cfg             *config.Config
	apiReadOnlyMode bool
)
