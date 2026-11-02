// Package security contains QuantumCoin project code for this module.
package security

const GUICLIAPIGuardrailsAdminVisibilityVersion = "GUI_CLI_API_GUARDRAILS_ADMIN_VISIBILITY_V1"

func BuildGUICLIAPIGuardrailsAdminVisibility(guiCliAPIGuardrails map[string]any) map[string]any {
	guiCliAPIRisk := "UNKNOWN"
	if guiCliAPIGuardrails != nil {
		if v, ok := guiCliAPIGuardrails["overallRisk"].(string); ok && v != "" {
			guiCliAPIRisk = v
		}
	}

	return map[string]any{
		"version":             GUICLIAPIGuardrailsAdminVisibilityVersion,
		"guiCliApiRisk":       guiCliAPIRisk,
		"guiCliApiGuardrails": guiCliAPIGuardrails,
		"recommendedAction":   "Review GUI/CLI/API guardrail warnings before release/testnet exposure.",
		"changesMempool":      false,
		"changesChain":        false,
		"replacesChain":       false,
		"changesBalances":     false,
		"changesConsensus":    false,
		"changesTotalSupply":  false,
		"performsTransfer":    false,
		"startsMining":        false,
		"changesGUI":          false,
		"changesCLI":          false,
		"changesAPI":          false,
		"blocksRequest":       false,
	}
}
