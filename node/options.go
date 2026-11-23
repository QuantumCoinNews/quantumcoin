package node

const BootstrapVersionV1 = "QC_NODE_BOOTSTRAP_V1"

// BootstrapOptions controls creation of the canonical node runtime.
type BootstrapOptions struct {
	ConfigPath string
	// DataDir anchors configured storage paths without changing the process cwd.
	// Empty preserves the existing library caller's relative-path behavior.
	DataDir string

	// Deprecated: full chain validation is mandatory in every startup mode.
	// Retained for source compatibility; false does not bypass validation.
	StrictChain bool
}

// DefaultBootstrapOptions returns safe canonical startup defaults.
func DefaultBootstrapOptions() BootstrapOptions {
	return BootstrapOptions{
		ConfigPath:  "config.json",
		StrictChain: true,
	}
}
