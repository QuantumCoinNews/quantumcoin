package node

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"quantumcoin/consensus"
)

const NFTGameProgramCapabilityVersionV1 = "QC_NFT_GAME_PROGRAM_HTTP_V1"

type nftGameProgramHTTPResponseV1 struct {
	Success            bool     `json:"success"`
	Version            string   `json:"version"`
	Verified           bool     `json:"verified"`
	GameActivatedAt    int64    `json:"gameActivatedAt"`
	NetworkID          string   `json:"networkID"`
	LegacyGenesisHex   string   `json:"legacyGenesisHex"`
	CampaignID         string   `json:"campaignID"`
	FirstAwardPeriod   string   `json:"firstAwardPeriod"`
	AwardPeriods       []string `json:"awardPeriods"`
	AwardCount         int      `json:"awardCount"`
	MaxAwardsPerMonth  int      `json:"maxAwardsPerMonth"`
	MintProtocolActive bool     `json:"mintProtocolActive"`
}

// ServeNFTGameProgramV1 is installed behind the canonical main API's existing
// header authentication. This read does not select, approve, mine or persist.
func (r *Runtime) ServeNFTGameProgramV1(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if request.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeNFTGameProgramHTTPV1(w, http.StatusMethodNotAllowed, map[string]any{"success": false, "message": "method not allowed"})
		return
	}
	if r == nil {
		writeNFTGameProgramHTTPV1(w, http.StatusServiceUnavailable, map[string]any{"success": false, "message": "node runtime unavailable"})
		return
	}
	r.mu.RLock()
	program, chain := r.nftGameProgram, r.Blockchain
	r.mu.RUnlock()
	response := nftGameProgramHTTPResponseV1{Success: true, Version: NFTGameProgramCapabilityVersionV1, AwardPeriods: []string{}}
	if !program.VerifiedV1() {
		writeNFTGameProgramHTTPV1(w, http.StatusOK, response)
		return
	}
	if chain == nil {
		writeNFTGameProgramHTTPV1(w, http.StatusServiceUnavailable, map[string]any{"success": false, "message": "program chain unavailable"})
		return
	}
	activation := chain.ActivationPolicySnapshotV1()
	policy, trust := program.PolicySnapshotV1(), program.TrustSnapshotV1()
	if err := activation.Validate(); err != nil || !activation.Enabled || activation.Plan.EffectiveTargetProtocol() != consensus.ProtocolNFTV4 ||
		activation.Plan.NetworkID != policy.NetworkID || !strings.EqualFold(activation.Plan.LegacyGenesisHex, trust.LegacyGenesisHex) || !chain.NFTAwardTrustBoundSnapshotV1() {
		writeNFTGameProgramHTTPV1(w, http.StatusServiceUnavailable, map[string]any{"success": false, "message": "verified program context unavailable"})
		return
	}
	periods, err := consensus.MonthlyNFTAwardPeriodsFromGameActivationV1(time.Unix(program.GameActivatedAtV1(), 0))
	if err != nil || len(periods) != consensus.MonthlyRareNFTAwardCountV1 || periods[0] != policy.FirstAwardPeriod {
		writeNFTGameProgramHTTPV1(w, http.StatusServiceUnavailable, map[string]any{"success": false, "message": "verified program periods unavailable"})
		return
	}
	response.Verified, response.GameActivatedAt = true, program.GameActivatedAtV1()
	response.NetworkID, response.LegacyGenesisHex = policy.NetworkID, trust.LegacyGenesisHex
	response.CampaignID, response.FirstAwardPeriod, response.AwardPeriods = policy.CampaignID, policy.FirstAwardPeriod, periods
	response.AwardCount, response.MaxAwardsPerMonth = consensus.MonthlyRareNFTAwardCountV1, 1
	response.MintProtocolActive = chain.ProtocolAtHeightV1(chain.BestHeightSnapshot()+1) == consensus.ProtocolNFTV4
	writeNFTGameProgramHTTPV1(w, http.StatusOK, response)
}

func writeNFTGameProgramHTTPV1(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		status, data = http.StatusInternalServerError, []byte(`{"success":false,"message":"program response unavailable"}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(data, '\n'))
}
