package node

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"

	"quantumcoin/blockchain"
	"quantumcoin/consensus"
	"quantumcoin/wallet"
)

const NFTMonthlyMintRequestVersionV1 = "QC_NFT_MONTHLY_MINT_REQUEST_V1"
const NFTMonthlyMintReceiptVersionV1 = "QC_NFT_MONTHLY_MINT_RECEIPT_V1"
const nftMonthlyMintMaxBodyV1 = 64 * 1024

// Only public record commitments and public approval evidence cross HTTP.
// Authorities, quorum, activation, chain path and miner reward address are local.
type nftMonthlyMintHTTPRequestV1 struct {
	Version string `json:"version"`
	Record  struct {
		NetworkID           string `json:"networkID"`
		CampaignID          string `json:"campaignID"`
		AwardPeriod         string `json:"awardPeriod"`
		Serial              uint32 `json:"serial"`
		RecipientPubKeyHash string `json:"recipientPubKeyHash"`
		MetadataHash        string `json:"metadataHash"`
	} `json:"record"`
	Decision struct {
		NetworkID        string `json:"networkID"`
		LegacyGenesisHex string `json:"legacyGenesisHex"`
		RecordDigest     string `json:"recordDigest"`
	} `json:"decision"`
	Approvals []struct {
		PublicKey string `json:"publicKey"`
		Signature string `json:"signature"`
	} `json:"approvals"`
}

type nftMonthlyMintHTTPReceiptV1 struct {
	Success             bool   `json:"success"`
	Version             string `json:"version"`
	Persisted           bool   `json:"persisted"`
	AlreadyMinted       bool   `json:"alreadyMinted"`
	TokenID             string `json:"tokenID"`
	RecordDigest        string `json:"recordDigest"`
	NetworkID           string `json:"networkID"`
	LegacyGenesisHex    string `json:"legacyGenesisHex"`
	CampaignID          string `json:"campaignID"`
	AwardPeriod         string `json:"awardPeriod"`
	Serial              uint32 `json:"serial"`
	RecipientPubKeyHash string `json:"recipientPubKeyHash"`
	MetadataHash        string `json:"metadataHash"`
	MintHeight          int    `json:"mintHeight"`
	MintBlockHash       string `json:"mintBlockHash"`
	ChainHeight         int    `json:"chainHeight"`
	ChainTipHash        string `json:"chainTipHash"`
	PersistedHeight     int    `json:"persistedHeight"`
	PersistedTipHash    string `json:"persistedTipHash"`
}

// The actual main mux supplies header authentication and its existing limiter.
// Receipt lookup uses the same signed evidence and never writes chain state.
func (r *Runtime) ServeNFTMonthlyMintV1(w http.ResponseWriter, request *http.Request) {
	r.serveNFTMonthlyAwardHTTPV1(w, request, true)
}

func (r *Runtime) ServeNFTMonthlyReceiptV1(w http.ResponseWriter, request *http.Request) {
	r.serveNFTMonthlyAwardHTTPV1(w, request, false)
}

func nftMonthlyHTTPFailureV1(w http.ResponseWriter, status int, code string) {
	writeNFTGameProgramHTTPV1(w, status, map[string]any{"success": false, "version": NFTMonthlyMintReceiptVersionV1, "code": code})
}

func (r *Runtime) serveNFTMonthlyAwardHTTPV1(w http.ResponseWriter, request *http.Request, mint bool) {
	w.Header().Set("Cache-Control", "no-store")
	if request.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		nftMonthlyHTTPFailureV1(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if r == nil {
		nftMonthlyHTTPFailureV1(w, http.StatusServiceUnavailable, "runtime_unavailable")
		return
	}
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		nftMonthlyHTTPFailureV1(w, http.StatusUnsupportedMediaType, "json_required")
		return
	}
	if request.Body == nil {
		nftMonthlyHTTPFailureV1(w, http.StatusBadRequest, "invalid_request")
		return
	}
	data, err := io.ReadAll(io.LimitReader(request.Body, nftMonthlyMintMaxBodyV1+1))
	if len(data) > nftMonthlyMintMaxBodyV1 {
		nftMonthlyHTTPFailureV1(w, http.StatusRequestEntityTooLarge, "request_too_large")
		return
	}
	if err != nil || !utf8.Valid(data) || !nftMonthlyUniqueJSONV1(data) {
		nftMonthlyHTTPFailureV1(w, http.StatusBadRequest, "invalid_request")
		return
	}
	var payload nftMonthlyMintHTTPRequestV1
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || payload.Version != NFTMonthlyMintRequestVersionV1 {
		nftMonthlyHTTPFailureV1(w, http.StatusBadRequest, "invalid_request")
		return
	}
	record, proof, err := payload.awardV1()
	if err != nil {
		nftMonthlyHTTPFailureV1(w, http.StatusUnprocessableEntity, "invalid_award")
		return
	}
	receipt, status, code := r.nftMonthlyHTTPAwardV1(record, proof, mint)
	if code != "" {
		nftMonthlyHTTPFailureV1(w, status, code)
		return
	}
	// Response I/O holds no runtime/mining lock. A lost response is reconciled
	// through the same signed lookup on retry, without another block.
	writeNFTGameProgramHTTPV1(w, status, receipt)
}

func (r *Runtime) nftMonthlyHTTPAwardV1(record blockchain.NFTMintRecordV1, proof blockchain.NFTAwardProofV1, mint bool) (nftMonthlyMintHTTPReceiptV1, int, string) {
	// Serialize the complete lookup/mint/receipt path with ordinary runtime
	// mining. Hold runtime state stable; do not call the locking runtime mint
	// method from here. Both paths use the same canonical blockchain entry.
	r.miningMu.Lock()
	defer r.miningMu.Unlock()
	r.mu.RLock()
	defer r.mu.RUnlock()
	if mint && r.APIReadOnly {
		return nftMonthlyMintHTTPReceiptV1{}, http.StatusForbidden, "runtime_read_only"
	}
	if !r.nftMonthlyHTTPContextLockedV1(record) {
		return nftMonthlyMintHTTPReceiptV1{}, http.StatusServiceUnavailable, "verified_context_unavailable"
	}
	chain, filename := r.Blockchain, strings.TrimSpace(r.Config.ChainFile)
	if err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(record, proof.Decision, proof.Approvals); err != nil {
		return nftMonthlyMintHTTPReceiptV1{}, http.StatusUnprocessableEntity, "award_approval_rejected"
	}
	id, _ := blockchain.NFTTokenIDV1(record)
	actualDigest, _ := blockchain.NFTMintRecordDigestV1(record)
	owned, found, err := chain.NFTByTokenIDSynchronizedV1(id)
	if err != nil {
		return nftMonthlyMintHTTPReceiptV1{}, http.StatusServiceUnavailable, "canonical_receipt_unavailable"
	}
	alreadyMinted := found
	if found {
		digest, err := blockchain.NFTMintRecordDigestV1(owned.Record)
		if err != nil || digest != actualDigest {
			return nftMonthlyMintHTTPReceiptV1{}, http.StatusConflict, "token_record_conflict"
		}
	} else if !mint {
		return nftMonthlyMintHTTPReceiptV1{}, http.StatusNotFound, "award_not_found"
	} else {
		// No fallback to founder/pool addresses, wallet creation or request
		// supplied reward destinations. Replays do not need a miner address.
		miner := strings.TrimSpace(r.Config.RewardAddrMiner)
		if miner == "" || len(miner) > 128 || !wallet.ValidateAddress(miner) {
			return nftMonthlyMintHTTPReceiptV1{}, http.StatusServiceUnavailable, "local_miner_address_required"
		}
		_, mintErr := chain.MineBlockWithNFTAwardAndPersistSynchronizedV1(miner, record, proof, filename)
		if mintErr != nil {
			// A competing canonical writer can win after the lookup. Reconcile
			// only the exact approved record, and still demand disk confirmation.
			entry, exists, lookupErr := chain.NFTByTokenIDSynchronizedV1(id)
			digest, digestErr := blockchain.NFTMintRecordDigestV1(entry.Record)
			if lookupErr != nil || !exists || digestErr != nil || digest != actualDigest {
				return nftMonthlyMintHTTPReceiptV1{}, http.StatusServiceUnavailable, "mint_not_confirmed"
			}
			alreadyMinted = true
		}
	}
	receipt, err := nftMonthlyPersistedReceiptV1(chain, filename, record, proof, alreadyMinted)
	if err != nil {
		return nftMonthlyMintHTTPReceiptV1{}, http.StatusServiceUnavailable, "persistence_confirmation_unavailable"
	}
	return receipt, http.StatusOK, ""
}

func (r *Runtime) nftMonthlyHTTPContextLockedV1(record blockchain.NFTMintRecordV1) bool {
	if r.Blockchain == nil || r.Config == nil || strings.TrimSpace(r.Config.ChainFile) == "" || !r.nftGameProgram.VerifiedV1() {
		return false
	}
	policy, trust := r.nftGameProgram.PolicySnapshotV1(), r.nftGameProgram.TrustSnapshotV1()
	activation := r.Blockchain.ActivationPolicySnapshotV1()
	if err := activation.Validate(); err != nil || !activation.Enabled || activation.Plan.EffectiveTargetProtocol() != consensus.ProtocolNFTV4 ||
		activation.Plan.NetworkID != policy.NetworkID || !strings.EqualFold(activation.Plan.LegacyGenesisHex, trust.LegacyGenesisHex) ||
		!r.Blockchain.NFTAwardTrustBoundSnapshotV1() {
		return false
	}
	serial, err := policy.SerialForAwardPeriodV1(record.AwardPeriod)
	return err == nil && record.NetworkID == policy.NetworkID && record.CampaignID == policy.CampaignID && record.Serial == serial
}

func (p nftMonthlyMintHTTPRequestV1) awardV1() (blockchain.NFTMintRecordV1, blockchain.NFTAwardProofV1, error) {
	record := blockchain.NFTMintRecordV1{NetworkID: p.Record.NetworkID, CampaignID: p.Record.CampaignID, AwardPeriod: p.Record.AwardPeriod, Serial: p.Record.Serial}
	proof := blockchain.NFTAwardProofV1{Decision: consensus.NFTAwardDecisionV1{NetworkID: p.Decision.NetworkID, LegacyGenesisHex: p.Decision.LegacyGenesisHex}}
	decode := func(value string, size int) ([]byte, error) {
		if len(value) != size*2 {
			return nil, errors.New("invalid hex length")
		}
		return hex.DecodeString(value)
	}
	owner, err := decode(p.Record.RecipientPubKeyHash, 20)
	if err != nil {
		return record, proof, err
	}
	record.RecipientPubKeyHash = owner
	metadata, err := decode(p.Record.MetadataHash, 32)
	if err != nil {
		return record, proof, err
	}
	copy(record.MetadataHash[:], metadata)
	commitment, err := decode(p.Decision.RecordDigest, 32)
	if err != nil {
		return record, proof, err
	}
	copy(proof.Decision.RecordDigest[:], commitment)
	if _, err := decode(p.Decision.LegacyGenesisHex, 32); err != nil {
		return record, proof, err
	}
	if _, err := blockchain.NFTMintRecordDigestV1(record); err != nil {
		return record, proof, err
	}
	if len(p.Approvals) == 0 {
		return record, proof, errors.New("approvals required")
	}
	for _, approval := range p.Approvals {
		pub, err := decode(approval.PublicKey, 65)
		if err != nil {
			return record, proof, err
		}
		if len(approval.Signature) < 8 || len(approval.Signature) > 132 || len(approval.Signature)%2 != 0 {
			return record, proof, errors.New("invalid signature length")
		}
		sig, err := hex.DecodeString(approval.Signature)
		if err != nil {
			return record, proof, err
		}
		proof.Approvals = append(proof.Approvals, consensus.NFTAwardApprovalV1{PublicKey: pub, Signature: sig})
	}
	return record, proof, nil
}

// A receipt confirms the exact owner-bearing record and mint block on BOTH
// the current canonical chain and a freshly validated configured chain file.
// An NFT present only in RAM is not reported as persisted. Heights/tips are
// explicit snapshots; this is not a finality or cross-process durability claim.
func nftMonthlyPersistedReceiptV1(chain *blockchain.Blockchain, filename string, record blockchain.NFTMintRecordV1, proof blockchain.NFTAwardProofV1, already bool) (nftMonthlyMintHTTPReceiptV1, error) {
	var result nftMonthlyMintHTTPReceiptV1
	persisted, err := blockchain.LoadBlockchainFromFileWithLocalContextV1(filename, chain)
	if err != nil {
		return result, err
	}
	disk, err := persisted.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil {
		return result, err
	}
	current, err := chain.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil {
		return result, err
	}
	id, _ := blockchain.NFTTokenIDV1(record)
	digest, _ := blockchain.NFTMintRecordDigestV1(record)
	find := func(snapshot blockchain.NFTOwnershipSnapshotV1) (blockchain.NFTOwnershipV1, bool) {
		for _, entry := range snapshot.NFTs {
			actual, err := blockchain.NFTMintRecordDigestV1(entry.Record)
			if entry.TokenID == id && err == nil && actual == digest {
				return entry, true
			}
		}
		return blockchain.NFTOwnershipV1{}, false
	}
	owned, found := find(current)
	saved, onDisk := find(disk)
	if !found || !onDisk || owned.MintHeight != saved.MintHeight || !bytes.Equal(owned.MintBlockHash, saved.MintBlockHash) {
		return result, errors.New("canonical award is not confirmed in the configured file")
	}
	result = nftMonthlyMintHTTPReceiptV1{
		Success: true, Version: NFTMonthlyMintReceiptVersionV1, Persisted: true, AlreadyMinted: already,
		TokenID: hex.EncodeToString(id[:]), RecordDigest: hex.EncodeToString(digest[:]),
		NetworkID: record.NetworkID, LegacyGenesisHex: strings.ToLower(proof.Decision.LegacyGenesisHex),
		CampaignID: record.CampaignID, AwardPeriod: record.AwardPeriod, Serial: record.Serial,
		RecipientPubKeyHash: hex.EncodeToString(record.RecipientPubKeyHash), MetadataHash: hex.EncodeToString(record.MetadataHash[:]),
		MintHeight: owned.MintHeight, MintBlockHash: hex.EncodeToString(owned.MintBlockHash),
		ChainHeight: current.Height, ChainTipHash: hex.EncodeToString(current.TipHash),
		PersistedHeight: disk.Height, PersistedTipHash: hex.EncodeToString(disk.TipHash),
	}
	return result, nil
}

// encoding/json normally accepts duplicate object keys. Reject ambiguity and
// extra top-level values before interpreting a signed award. Depth is bounded.
func nftMonthlyUniqueJSONV1(data []byte) bool {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var visit func(int) bool
	visit = func(depth int) bool {
		if depth > 16 {
			return false
		}
		token, err := d.Token()
		if err != nil {
			return false
		}
		delimiter, container := token.(json.Delim)
		if !container {
			return true
		}
		switch delimiter {
		case '{':
			keys := make(map[string]bool)
			for d.More() {
				key, err := d.Token()
				name, ok := key.(string)
				if err != nil || !ok || keys[name] {
					return false
				}
				keys[name] = true
				if !visit(depth + 1) {
					return false
				}
			}
			end, err := d.Token()
			return err == nil && end == json.Delim('}')
		case '[':
			for d.More() {
				if !visit(depth + 1) {
					return false
				}
			}
			end, err := d.Token()
			return err == nil && end == json.Delim(']')
		default:
			return false
		}
	}
	if !visit(0) {
		return false
	}
	_, err := d.Token()
	return err == io.EOF
}
