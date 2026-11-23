package main

import (
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
)

/* ---------- Explorer ---------- */

const (
	defaultBlocksPageLimit = 20
	maxBlocksPageLimit     = 200
)

func parseBlocksPagination(
	r *http.Request,
) (limit int, offset int) {
	limit = defaultBlocksPageLimit
	offset = 0

	if r == nil {
		return limit, offset
	}

	query := r.URL.Query()

	if raw :=
		strings.TrimSpace(
			query.Get("limit"),
		); raw != "" {
		if parsed, err :=
			strconv.Atoi(raw); err == nil &&
			parsed > 0 {
			limit = parsed
		}
	}

	if limit > maxBlocksPageLimit {
		limit = maxBlocksPageLimit
	}

	if raw :=
		strings.TrimSpace(
			query.Get("offset"),
		); raw != "" {
		if parsed, err :=
			strconv.Atoi(raw); err == nil &&
			parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}

func handleBlocksList(w http.ResponseWriter, r *http.Request) {
	if !requireAPIGet(w, r) {
		return
	}
	limit, offset :=
		parseBlocksPagination(
			r,
		)

	bc, _ := nodeRuntime.Snapshot()
	blocks, total :=
		bc.BlockSummaryPageFromTipSnapshot(
			offset,
			limit,
		)

	w.Header().Set(
		"X-Total-Count",
		strconv.Itoa(total),
	)
	w.Header().Set(
		"X-Page-Limit",
		strconv.Itoa(limit),
	)
	w.Header().Set(
		"X-Page-Offset",
		strconv.Itoa(offset),
	)

	nextOffset :=
		offset + len(blocks)

	hasMore :=
		nextOffset < total

	w.Header().Set(
		"X-Has-More",
		strconv.FormatBool(hasMore),
	)

	if hasMore {
		w.Header().Set(
			"X-Next-Offset",
			strconv.Itoa(nextOffset),
		)
	}

	type bsum struct {
		Index      int    `json:"index"`
		Hash       string `json:"hash"`
		PrevHash   string `json:"prev_hash"`
		Timestamp  int64  `json:"timestamp"`
		Miner      string `json:"miner"`
		Difficulty int    `json:"difficulty"`
		TxCount    int    `json:"tx_count"`
	}

	summaries :=
		make(
			[]bsum,
			0,
			len(blocks),
		)

	for _, b := range blocks {
		summaries = append(
			summaries,
			bsum{
				Index:      b.Index,
				Hash:       hex.EncodeToString(b.Hash),
				PrevHash:   hex.EncodeToString(b.PrevHash),
				Timestamp:  b.Timestamp,
				Miner:      b.Miner,
				Difficulty: b.Difficulty,
				TxCount:    b.TxCount,
			},
		)
	}

	writeOK(w, summaries)
}

func handleBlockDetail(w http.ResponseWriter, r *http.Request) {
	if !requireAPIGet(w, r) {
		return
	}
	q := r.URL.Query()
	bc, _ := nodeRuntime.Snapshot()
	if bc == nil {
		writeError(w, http.StatusServiceUnavailable, "blockchain is unavailable")
		return
	}
	if idxStr := q.Get("index"); idxStr != "" {
		if idx, err := strconv.Atoi(idxStr); err == nil {
			if blk := bc.BlockByIndexSnapshot(idx); blk != nil {
				writeOK(w, newBlockDetailPresentationV1(bc, blk))
				return
			}
			writeError(w, http.StatusNotFound, "block not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid index")
		return
	}
	if h := q.Get("hash"); h != "" {
		raw, err := hex.DecodeString(strings.TrimPrefix(h, "0x"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid hash hex")
			return
		}
		if blk := bc.BlockByHashSnapshot(raw); blk != nil {
			writeOK(w, newBlockDetailPresentationV1(bc, blk))
			return
		}
		writeError(w, http.StatusNotFound, "block not found")
		return
	}
	writeError(w, http.StatusBadRequest, "index or hash required")
}
