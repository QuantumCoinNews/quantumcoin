// Package security contains QuantumCoin project code for this module.
package security

import "strings"

const AdminSecurityAlertsVersion = "ADMIN_SECURITY_ALERTS_V1"

type AdminSecurityAlertInput struct {
	AdminReport map[string]any `json:"adminReport"`
	EventLog    map[string]any `json:"eventLog"`
}

func BuildAdminSecurityAlerts(input AdminSecurityAlertInput) map[string]any {
	adminRisk := adminAlertString(input.AdminReport, "overallRisk", "UNKNOWN")
	adminStatus := adminAlertString(input.AdminReport, "overallStatus", "UNKNOWN")
	securityScore := adminAlertInt(input.AdminReport, "securityScore", 0)

	eventRisk := adminAlertString(input.EventLog, "overallRisk", "UNKNOWN")
	totalEvents := adminAlertInt(input.EventLog, "totalEvents", 0)

	alerts := []map[string]any{}

	if adminRisk == "HIGH" {
		alerts = append(alerts, adminSecurityAlert("ADMIN_REPORT_HIGH_RISK", "HIGH", "admin_report", "Admin security report is HIGH risk. Review required before testnet/public exposure."))
	} else if adminRisk == "MEDIUM" {
		alerts = append(alerts, adminSecurityAlert("ADMIN_REPORT_MEDIUM_RISK", "WARNING", "admin_report", "Admin security report is MEDIUM risk. Review recommended before testnet/public exposure."))
	} else if adminRisk == "WATCH" {
		alerts = append(alerts, adminSecurityAlert("ADMIN_REPORT_WATCH", "INFO", "admin_report", "Admin security report is in WATCH mode. Continue monitoring."))
	}

	if eventRisk == "CRITICAL" {
		alerts = append(alerts, adminSecurityAlert("SECURITY_EVENT_CRITICAL", "CRITICAL", "security_event_log", "Critical security event exists in local event log."))
	} else if eventRisk == "HIGH" {
		alerts = append(alerts, adminSecurityAlert("SECURITY_EVENT_HIGH", "HIGH", "security_event_log", "High severity security event exists in local event log."))
	} else if eventRisk == "WARNING" {
		alerts = append(alerts, adminSecurityAlert("SECURITY_EVENT_WARNING", "WARNING", "security_event_log", "Warning severity security event exists in local event log."))
	}

	if totalEvents > 0 && eventRisk == "UNKNOWN" {
		alerts = append(alerts, adminSecurityAlert("SECURITY_EVENTS_PRESENT", "INFO", "security_event_log", "Security events exist; review event details."))
	}

	if securityScore > 0 && securityScore < 70 {
		alerts = append(alerts, adminSecurityAlert("LOW_SECURITY_SCORE", "WARNING", "admin_report", "Security score is below recommended threshold."))
	}

	if len(alerts) == 0 {
		alerts = append(alerts, adminSecurityAlert("NO_IMMEDIATE_ADMIN_ALERT", "INFO", "admin_alerts", "No immediate admin security alert. Continue monitoring."))
	}

	overallAlertLevel := "INFO"
	for _, a := range alerts {
		severity := strings.ToUpper(adminAlertString(a, "severity", "INFO"))
		if severity == "CRITICAL" {
			overallAlertLevel = "CRITICAL"
			break
		}
		if severity == "HIGH" && overallAlertLevel != "CRITICAL" {
			overallAlertLevel = "HIGH"
		}
		if severity == "WARNING" && overallAlertLevel != "CRITICAL" && overallAlertLevel != "HIGH" {
			overallAlertLevel = "WARNING"
		}
	}

	return map[string]any{
		"version":           AdminSecurityAlertsVersion,
		"mode":              "READ_ONLY_ADMIN_SECURITY_ALERTS",
		"overallAlertLevel": overallAlertLevel,
		"adminRisk":         adminRisk,
		"adminStatus":       adminStatus,
		"securityScore":     securityScore,
		"eventRisk":         eventRisk,
		"totalEvents":       totalEvents,
		"alerts":            alerts,
		"policyNote":        "Admin alerts and security logs are advisory/read-only. They do not ban clients, block requests, change rate limits, change mempool, change chain, change balances, change consensus, change total supply, perform transfers, start mining, stop mining, kill processes, or modify release executables.",
		"safeBoundary": map[string]bool{
			"readOnly":                  true,
			"advisoryOnly":              true,
			"blocksRequest":             false,
			"bansClient":                false,
			"changesRateLimits":         false,
			"changesMempool":            false,
			"changesChain":              false,
			"replacesChain":             false,
			"changesBalances":           false,
			"changesConsensus":          false,
			"changesTotalSupply":        false,
			"performsTransfer":          false,
			"startsMining":              false,
			"stopsMining":               false,
			"killsProcess":              false,
			"modifiesReleaseExecutable": false,
		},
	}
}

func adminSecurityAlert(alertType, severity, source, message string) map[string]any {
	return map[string]any{
		"type":     alertType,
		"severity": severity,
		"source":   source,
		"message":  message,
	}
}

func adminAlertString(m map[string]any, key string, fallback string) string {
	if m == nil {
		return fallback
	}
	if v, ok := m[key].(string); ok && v != "" {
		return v
	}
	return fallback
}

func adminAlertInt(m map[string]any, key string, fallback int) int {
	if m == nil {
		return fallback
	}
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	}
	return fallback
}
