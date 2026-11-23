// AI_SAFE_BOUNDARY_V1

// Package ai contains QuantumCoin project code for this module.
package ai

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AISecurityReview is the safe AI output model.
// It is intentionally advisory-only.
// It must never mint coins, change balances, change mining rewards, or execute transfers.
type AISecurityReview struct {
	WalletAddress string  `json:"wallet_address"`
	TxID          string  `json:"tx_id,omitempty"`
	Scope         string  `json:"scope"`
	RiskLevel     string  `json:"risk_level"`
	Score         float64 `json:"score"`
	Action        string  `json:"action"`
	Reason        string  `json:"reason"`
	AdminReview   bool    `json:"admin_review"`
	CreatedAt     int64   `json:"created_at"`
}

// AISecurityReviewFile is the persisted report format.
type AISecurityReviewFile struct {
	Timestamp      int64              `json:"ts"`
	Reviews        []AISecurityReview `json:"ai_security_reviews"`
	DroppedReviews uint64             `json:"dropped_reviews"`
}

// BuildSecurityReviews converts anomaly analysis into advisory-only security reviews.
func BuildSecurityReviews(txs []TxLite) []AISecurityReview {
	if !Enabled() || len(txs) == 0 {
		return nil
	}

	anoms := AnalyzeTransactions(txs)
	if len(anoms) == 0 {
		return nil
	}

	out := make([]AISecurityReview, 0, len(anoms))
	now := time.Now().Unix()

	for _, a := range anoms {
		if strings.TrimSpace(a.WalletAddress) == "" {
			continue
		}

		level := "low"
		action := "monitor"
		adminReview := false

		if a.Score >= 0.90 {
			level = "high"
			action = "admin_review"
			adminReview = true
		} else if a.Score >= 0.70 {
			level = "medium"
			action = "monitor_plus_review"
			adminReview = true
		}

		reason := strings.TrimSpace(a.Reason)
		if reason == "" {
			reason = "AI security signal"
		}

		out = append(out, AISecurityReview{
			WalletAddress: a.WalletAddress,
			TxID:          a.TxID,
			Scope:         "block_transaction_monitoring",
			RiskLevel:     level,
			Score:         a.Score,
			Action:        action,
			Reason:        reason,
			AdminReview:   adminReview,
			CreatedAt:     now,
		})
	}

	return out
}

// WriteAISecurityReviewsTo writes advisory-only AI security review output.
// This file is for monitoring/admin review only.
func WriteAISecurityReviewsTo(dir string, reviews []AISecurityReview) error {
	if len(reviews) == 0 {
		return nil
	}

	if strings.TrimSpace(dir) == "" {
		dir = "."
	}
	path := filepath.Join(dir, "ai_security_reviews.json")
	storeMu.Lock()
	defer storeMu.Unlock()
	unlock, lockErr := lockRecordDirectory(dir)
	if lockErr != nil {
		return lockErr
	}
	defer unlock()
	var payload AISecurityReviewFile
	data, err := readBoundedFile(path)
	if err == nil {
		if err = json.Unmarshal(data, &payload); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	const maxReviews = 1000
	if len(payload.Reviews) > maxReviews {
		payload.DroppedReviews += uint64(len(payload.Reviews) - maxReviews)
		payload.Reviews = payload.Reviews[len(payload.Reviews)-maxReviews:]
	}
	if len(reviews) > maxReviews {
		payload.DroppedReviews += uint64(len(reviews) - maxReviews)
		reviews = reviews[len(reviews)-maxReviews:]
	}
	for _, review := range reviews {
		review.WalletAddress = boundedText(review.WalletAddress)
		review.TxID = boundedText(review.TxID)
		review.Scope = boundedText(review.Scope)
		review.RiskLevel = boundedText(review.RiskLevel)
		review.Action = boundedText(review.Action)
		review.Reason = boundedText(review.Reason)
		payload.Reviews = append(payload.Reviews, review)
	}
	if len(payload.Reviews) > maxReviews {
		payload.DroppedReviews += uint64(len(payload.Reviews) - maxReviews)
		payload.Reviews = payload.Reviews[len(payload.Reviews)-maxReviews:]
	}
	payload.Timestamp = time.Now().Unix()
	return writeJSONAtomic(path, payload)
}
