// Package monitoring contains QuantumCoin project code for this module.
package monitoring

import (
	"errors"
	"strings"
	"time"
)

const EventLogAlertRunbookVersionV1 = "QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_V1"

type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityWarning  Severity = "WARNING"
	SeverityCritical Severity = "CRITICAL"
)

type Event struct {
	ID       string            `json:"id"`
	Source   string            `json:"source"`
	Category string            `json:"category"`
	Severity Severity          `json:"severity"`
	Message  string            `json:"message"`
	At       string            `json:"at"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type Alert struct {
	Open         bool     `json:"open"`
	Severity     Severity `json:"severity"`
	Category     string   `json:"category"`
	Action       string   `json:"action"`
	Escalate     bool     `json:"escalate"`
	SafeBoundary string   `json:"safeBoundary"`
}

type RunbookEntry struct {
	Category    string   `json:"category"`
	Severity    Severity `json:"severity"`
	Summary     string   `json:"summary"`
	AdminAction string   `json:"adminAction"`
	Forbidden   []string `json:"forbidden"`
}

func NewEvent(id, source, category string, severity Severity, message string) Event {
	return Event{
		ID:       strings.TrimSpace(id),
		Source:   strings.TrimSpace(source),
		Category: strings.TrimSpace(category),
		Severity: NormalizeSeverity(string(severity)),
		Message:  strings.TrimSpace(message),
		At:       time.Now().UTC().Format(time.RFC3339),
		Metadata: map[string]string{},
	}
}

func NormalizeSeverity(raw string) Severity {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "CRITICAL", "FATAL", "PANIC", "HIGH":
		return SeverityCritical
	case "WARNING", "WARN", "MEDIUM":
		return SeverityWarning
	default:
		return SeverityInfo
	}
}

func ValidateEvent(e Event) error {
	if strings.TrimSpace(e.Source) == "" {
		return errors.New("monitoring event source is required")
	}
	if strings.TrimSpace(e.Category) == "" {
		return errors.New("monitoring event category is required")
	}
	if strings.TrimSpace(e.Message) == "" {
		return errors.New("monitoring event message is required")
	}
	sev := NormalizeSeverity(string(e.Severity))
	if sev != e.Severity {
		return errors.New("monitoring event severity must be normalized")
	}
	return nil
}

func ClassifyAlert(e Event) Alert {
	sev := NormalizeSeverity(string(e.Severity))
	entry := RunbookForCategory(e.Category)

	alert := Alert{
		Open:         sev != SeverityInfo,
		Severity:     sev,
		Category:     strings.TrimSpace(e.Category),
		Action:       entry.AdminAction,
		Escalate:     sev == SeverityCritical,
		SafeBoundary: "READ_ONLY_ALERT_DOES_NOT_CHANGE_CHAIN_BALANCE_SUPPLY_MINING_OR_PAYOUT",
	}

	if alert.Action == "" {
		alert.Action = "Review event details, check service health, and record admin note before any manual action."
	}

	return alert
}

func DefaultRunbookEntries() []RunbookEntry {
	forbidden := []string{
		"do_not_mint",
		"do_not_change_qc_supply",
		"do_not_change_balances",
		"do_not_send_funds",
		"do_not_start_or_stop_miners",
		"do_not_auto_ban_without_review",
	}

	return []RunbookEntry{
		{
			Category:    "api_down",
			Severity:    SeverityCritical,
			Summary:     "An expected API is unreachable or repeatedly failing.",
			AdminAction: "Check process status, port, env mode, logs, and last deployment before restart decision.",
			Forbidden:   forbidden,
		},
		{
			Category:    "auth_failure",
			Severity:    SeverityWarning,
			Summary:     "Admin/API authentication failed or token is missing.",
			AdminAction: "Verify admin token configuration, reject query-token usage, and review source IP/client.",
			Forbidden:   forbidden,
		},
		{
			Category:    "chain_mismatch",
			Severity:    SeverityCritical,
			Summary:     "Mainnet/testnet chain file, network label, or height looks inconsistent.",
			AdminAction: "Stop promotion, verify chain file separation, compare node labels, and keep release files untouched.",
			Forbidden:   forbidden,
		},
		{
			Category:    "payout_guard",
			Severity:    SeverityWarning,
			Summary:     "TGWT/claim/withdraw payout requires admin review.",
			AdminAction: "Review claim status, settlement precheck, and ensure backend has no private key/gas-payment path.",
			Forbidden:   forbidden,
		},
		{
			Category:    "nft_mint_guard",
			Severity:    SeverityWarning,
			Summary:     "NFT monthly flow reached a review/mint-sensitive state.",
			AdminAction: "Verify AI review, admin approval, uniqueness, final metadata hash, and manual approval token.",
			Forbidden:   forbidden,
		},
		{
			Category:    "watch_verification_failure",
			Severity:    SeverityWarning,
			Summary:     "Watch & Earn completion is not server-side verified.",
			AdminAction: "Reject reward, keep final revenue mode disabled, and review provider verification logs.",
			Forbidden:   forbidden,
		},
	}
}

func RunbookForCategory(category string) RunbookEntry {
	clean := strings.ToLower(strings.TrimSpace(category))
	for _, entry := range DefaultRunbookEntries() {
		if strings.ToLower(entry.Category) == clean {
			return entry
		}
	}
	return RunbookEntry{
		Category:    clean,
		Severity:    SeverityInfo,
		Summary:     "General monitoring event.",
		AdminAction: "Review event details and record admin note.",
		Forbidden: []string{
			"do_not_mint",
			"do_not_change_qc_supply",
			"do_not_change_balances",
			"do_not_send_funds",
		},
	}
}

func SafeBoundaryV1() map[string]bool {
	return map[string]bool{
		"doesNotMint":              true,
		"doesNotSendFunds":         true,
		"doesNotChangeBalances":    true,
		"doesNotChangeChain":       true,
		"doesNotChangeConsensus":   true,
		"doesNotChangeSupply":      true,
		"doesNotStartMining":       true,
		"doesNotStopMining":        true,
		"requiresAdminReview":      true,
		"readOnlyAlertingStandard": true,
	}
}
