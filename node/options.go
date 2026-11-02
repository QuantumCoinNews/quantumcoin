package node

const BootstrapVersionV1 = "QC_NODE_BOOTSTRAP_V1"

// BootstrapOptions controls creation of the canonical node runtime.
type BootstrapOptions struct {
	ConfigPath string

	// StrictChain validates the complete loaded or newly created chain.
	StrictChain bool
}

// DefaultBootstrapOptions returns the legacy-compatible startup defaults.
func DefaultBootstrapOptions() BootstrapOptions {
	return BootstrapOptions{
		ConfigPath:  "config.json",
		StrictChain: false,
	}
}
