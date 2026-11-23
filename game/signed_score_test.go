package game

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func scoreFixture(t *testing.T) (*SignedScoreService, ScoreEvent, ed25519.PrivateKey, string) {
	t.Helper()
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	path := filepath.Join(t.TempDir(), "scores.json")
	s, err := NewSignedScoreService(path, "mainnet", strings.Repeat("ab", 32), hex.EncodeToString(pub), NewGameState())
	if err != nil {
		t.Fatal(err)
	}
	e := ScoreEvent{Version: ScoreEventVersion, Network: "mainnet", Genesis: strings.Repeat("ab", 32), Player: "tg_123", Score: 200, Sequence: 1, Expires: time.Now().Unix() + 120}
	return s, e, key, path
}
func signScore(t *testing.T, e ScoreEvent, key ed25519.PrivateKey) ScoreEvent {
	t.Helper()
	b, err := ScoreSigningBytes(e)
	if err != nil {
		t.Fatal(err)
	}
	e.Signature = hex.EncodeToString(ed25519.Sign(key, b))
	return e
}
func TestSignedScoresConcurrentReplayPersistsOneAbsoluteScore(t *testing.T) {
	s, e, key, path := scoreFixture(t)
	e = signScore(t, e, key)
	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.Apply(e, time.Now()); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if s.state.GetScore(e.Player) != 200 {
		t.Fatal("score inflation")
	}
	restored, err := NewSignedScoreService(path, e.Network, e.Genesis, hex.EncodeToString(key.Public().(ed25519.PublicKey)), NewGameState())
	if err != nil {
		t.Fatal(err)
	}
	if replay, err := restored.Apply(e, time.Now()); err != nil || !replay || restored.state.GetScore(e.Player) != 200 {
		t.Fatal(replay, err)
	}
	e.Sequence = 2
	e.Score = 201
	e = signScore(t, e, key)
	if _, err := restored.Apply(e, time.Now()); err != nil {
		t.Fatal(err)
	}
	if restored.state.GetScore(e.Player) != 201 {
		t.Fatal("absolute update became delta")
	}
}
func TestSignedScoresRejectForgeryConflictsAndTamperedStorage(t *testing.T) {
	s, e, key, path := scoreFixture(t)
	signed := signScore(t, e, key)
	if _, err := s.Apply(signed, time.Now()); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*ScoreEvent){func(e *ScoreEvent) { e.Score++ }, func(e *ScoreEvent) { e.Sequence = 0 }, func(e *ScoreEvent) { e.Expires = 1 }, func(e *ScoreEvent) { e.Network = "testnet" }} {
		altered := signed
		mutate(&altered)
		if _, err := s.Apply(altered, time.Now()); err == nil {
			t.Fatal("forged event accepted")
		}
	}
	e.Score = 999
	e = signScore(t, e, key)
	if _, err := s.Apply(e, time.Now()); err == nil {
		t.Fatal("sequence reused for another score")
	}
	raw, _ := os.ReadFile(path)
	var store scoreStore
	_ = json.Unmarshal(raw, &store)
	r := store.Players[e.Player]
	r.Score = 999
	store.Players[e.Player] = r
	tampered, _ := json.Marshal(store)
	_ = os.WriteFile(path, tampered, 0600)
	state := NewGameState()
	if _, err := NewSignedScoreService(path, e.Network, e.Genesis, hex.EncodeToString(key.Public().(ed25519.PublicKey)), state); err == nil || state.GetScore(e.Player) != 0 {
		t.Fatal("tampered storage trusted or partially loaded")
	}
}
func TestSignedScoreStorageFailureDoesNotPublishOrConsume(t *testing.T) {
	s, e, key, path := scoreFixture(t)
	e = signScore(t, e, key)
	s.path = filepath.Join(filepath.Dir(path), "missing", "score.json")
	if _, err := s.Apply(e, time.Now()); err == nil || s.state.GetScore(e.Player) != 0 {
		t.Fatal("failed persistence published score")
	}
	s.path = path
	if _, err := s.Apply(e, time.Now()); err != nil {
		t.Fatal("retry lost event", err)
	}
	before, _ := os.ReadFile(path)
	if _, err := s.Apply(e, time.Now()); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("replay rewrote state")
	}
}

func TestSignedScoreAcceptsPythonAuthorityVector(t *testing.T) {
	raw, err := os.ReadFile("testdata/score_event.json")
	if err != nil {
		t.Fatal(err)
	}
	var vector struct {
		Event           ScoreEvent `json:"event"`
		Authority       string     `json:"authority"`
		SigningBytesHex string     `json:"signingBytesHex"`
	}
	if err := json.Unmarshal(raw, &vector); err != nil {
		t.Fatal(err)
	}
	payload, err := ScoreSigningBytes(vector.Event)
	if err != nil || hex.EncodeToString(payload) != vector.SigningBytesHex {
		t.Fatal("cross-language canonical payload differs")
	}
	service, err := NewSignedScoreService(filepath.Join(t.TempDir(), "score.json"), vector.Event.Network, vector.Event.Genesis, vector.Authority, NewGameState())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Apply(vector.Event, time.Unix(1700000000, 0)); err != nil {
		t.Fatal("Python signature rejected", err)
	}
}
