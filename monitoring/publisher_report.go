package monitoring

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
)

type PublisherDelivery struct {
	Platform string `json:"platform"`
	Status   string `json:"status"`
}

type PublisherReport struct {
	Version    string              `json:"version"`
	JobID      string              `json:"jobId"`
	Editorial  string              `json:"editorial"`
	Deliveries []PublisherDelivery `json:"deliveries"`
}

func (r PublisherReport) Events() ([]Event, error) {
	date, slot, ok := strings.Cut(r.JobID, ".")
	if _, err := time.Parse("2006-01-02", date); err != nil || !ok ||
		(slot != "slot1" && slot != "slot2") || r.Version != "QC_PUBLISHER_REPORT_V1" {
		return nil, fmt.Errorf("invalid publisher job")
	}
	event := func(stage, subject, outcome string, severity Severity) Event {
		id := sha256.Sum256([]byte(r.JobID + "/" + stage + "/" + subject + "/" + outcome))
		e := NewEvent(fmt.Sprintf("publisher-%x", id), "autopublisher", stage, severity, outcome)
		e.Metadata["subject"] = subject
		e.Metadata["jobId"] = r.JobID
		return e
	}
	events := []Event{event("publisher_topic", r.JobID, "selected", SeverityInfo)}
	switch r.Editorial {
	case "pending":
	case "passed":
		events = append(events, event("publisher_editorial", r.JobID, "passed", SeverityInfo))
	case "rejected":
		events = append(events, event("publisher_editorial", r.JobID, "rejected", SeverityWarning))
	default:
		return nil, fmt.Errorf("invalid editorial status")
	}
	seen := make(map[string]bool)
	for _, d := range r.Deliveries {
		switch d.Platform {
		case "x", "youtube", "telegram", "instagram", "tiktok":
		default:
			return nil, fmt.Errorf("invalid publisher platform")
		}
		if seen[d.Platform] {
			return nil, fmt.Errorf("duplicate publisher platform")
		}
		seen[d.Platform] = true
		severity := SeverityInfo
		switch d.Status {
		case "confirmed":
		case "failed":
			severity = SeverityWarning
		case "uncertain":
			severity = SeverityCritical
		default:
			return nil, fmt.Errorf("invalid delivery status")
		}
		events = append(events, event("publisher_delivery", r.JobID+"/"+d.Platform, d.Status, severity))
	}
	return events, nil
}
