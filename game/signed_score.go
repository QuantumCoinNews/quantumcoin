package game

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

const ScoreEventVersion = "QC_GAME_SCORE_EVENT_V1"
const MaxScorePlayers = 10000

var scorePlayerPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type ScoreEvent struct {
	Version   string `json:"version"`
	Genesis   string `json:"genesis"`
	Network   string `json:"network"`
	Player    string `json:"player"`
	Score     int    `json:"score"`
	Sequence  uint64 `json:"sequence"`
	Expires   int64  `json:"expires"`
	Signature string `json:"signature"`
}
type scoreRecord struct {
	Sequence  uint64 `json:"sequence"`
	Score     int    `json:"score"`
	Digest    string `json:"digest"`
	Expires   int64  `json:"expires"`
	Signature string `json:"signature"`
}
type scoreStore struct {
	Version   string                 `json:"version"`
	Genesis   string                 `json:"genesis"`
	Network   string                 `json:"network"`
	Authority string                 `json:"authority"`
	Players   map[string]scoreRecord `json:"players"`
}
type SignedScoreService struct {
	mu      sync.Mutex
	path    string
	genesis string
	network string
	key     ed25519.PublicKey
	state   *GameState
	store   scoreStore
}

func ScoreSigningBytes(e ScoreEvent) ([]byte, error) {
	if e.Version != ScoreEventVersion || !scorePlayerPattern.MatchString(e.Player) || !scorePlayerPattern.MatchString(e.Network) || len(e.Genesis) != 64 || e.Score < 0 || e.Score > 1000000000 || e.Sequence == 0 || e.Sequence > 9007199254740991 {
		return nil, errors.New("invalid score event")
	}
	if _, err := hex.DecodeString(e.Genesis); err != nil {
		return nil, errors.New("invalid score scope")
	}
	return json.Marshal([]any{e.Version, e.Network, e.Genesis, e.Player, e.Score, e.Sequence, e.Expires})
}
func NewSignedScoreService(path, network, genesis, keyHex string, state *GameState) (*SignedScoreService, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != ed25519.PublicKeySize || path == "" || state == nil {
		return nil, errors.New("score authority unavailable")
	}
	service := &SignedScoreService{path: path, network: network, genesis: genesis, key: key, state: state, store: scoreStore{Version: ScoreEventVersion, Network: network, Genesis: genesis, Authority: keyHex, Players: map[string]scoreRecord{}}}
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		raw, err := io.ReadAll(io.LimitReader(f, 2<<20+1))
		if err != nil || len(raw) > 2<<20 {
			return nil, errors.New("invalid score store")
		}
		if json.Unmarshal(raw, &service.store) != nil {
			return nil, errors.New("invalid score store")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if service.store.Version != ScoreEventVersion || service.store.Network != network || service.store.Genesis != genesis || service.store.Authority != keyHex || service.store.Players == nil || len(service.store.Players) > MaxScorePlayers {
		return nil, errors.New("score store scope mismatch")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	for player, record := range service.store.Players {
		if !scorePlayerPattern.MatchString(player) || record.Score < 0 || record.Score > 1000000000 || record.Sequence == 0 || len(record.Digest) != 64 {
			return nil, errors.New("invalid persisted score")
		}
		event := ScoreEvent{Version: ScoreEventVersion, Network: network, Genesis: genesis, Player: player, Score: record.Score, Sequence: record.Sequence, Expires: record.Expires, Signature: record.Signature}
		payload, err := ScoreSigningBytes(event)
		signature, sigErr := hex.DecodeString(record.Signature)
		if err != nil || sigErr != nil || !ed25519.Verify(key, payload, signature) || record.Digest != fmt.Sprintf("%x", sha256.Sum256(payload)) {
			return nil, errors.New("persisted score authorization failed")
		}
	}
	for player, record := range service.store.Players {
		state.scores[player] = record.Score
	}
	return service, nil
}

// Apply persists absolute score and monotonic sequence before updating memory.
// Replaying the identical event is idempotent; it never adds the score twice.
func (s *SignedScoreService) Apply(e ScoreEvent, now time.Time) (bool, error) {
	payload, err := ScoreSigningBytes(e)
	if err != nil {
		return false, err
	}
	signature, err := hex.DecodeString(e.Signature)
	if err != nil || !ed25519.Verify(s.key, payload, signature) {
		return false, errors.New("invalid score authority signature")
	}
	if e.Genesis != s.genesis || e.Network != s.network || e.Expires < now.Unix() || e.Expires > now.Unix()+300 {
		return false, errors.New("score scope or expiry rejected")
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, exists := s.store.Players[e.Player]
	if exists && e.Sequence <= previous.Sequence {
		if e.Sequence == previous.Sequence && previous.Digest == digest {
			return true, nil
		}
		return false, errors.New("score event replay or sequence conflict")
	}
	if !exists && len(s.store.Players) >= MaxScorePlayers {
		return false, errors.New("score player capacity reached")
	}
	next := s.store
	next.Players = make(map[string]scoreRecord, len(s.store.Players)+1)
	for player, record := range s.store.Players {
		next.Players[player] = record
	}
	next.Players[e.Player] = scoreRecord{Sequence: e.Sequence, Score: e.Score, Digest: digest, Expires: e.Expires, Signature: e.Signature}
	raw, err := json.Marshal(next)
	if err != nil || len(raw) > 2<<20 {
		return false, errors.New("score storage capacity reached")
	}
	file, err := os.CreateTemp(filepath.Dir(s.path), ".score-commit-*")
	if err != nil {
		return false, err
	}
	temp := file.Name()
	defer func() { _ = file.Close(); _ = os.Remove(temp) }()
	if _, err = file.Write(raw); err != nil {
		return false, err
	}
	if err = file.Sync(); err != nil {
		return false, err
	}
	if err = file.Close(); err != nil {
		return false, err
	}
	if err = os.Rename(temp, s.path); err != nil {
		return false, err
	}
	s.store = next
	s.state.mu.Lock()
	s.state.scores[e.Player] = e.Score
	s.state.mu.Unlock()
	return false, nil
}
