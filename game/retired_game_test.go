package game

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestUnsignedDeltasCannotMutateProjection(t *testing.T) {
	s := NewGameState()
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			delta := 1000
			if i%2 == 0 {
				delta = -1000
			}
			if !errors.Is(HandleTelegramScore(s, "tg_123", delta), ErrLegacyGameDisabled) {
				t.Error("unsigned adapter accepted delta")
			}
			if !errors.Is(s.AddScore("tg_123", delta), ErrLegacyGameDisabled) {
				t.Error("unsigned mutation accepted")
			}
		}(i)
	}
	wg.Wait()
	if s.GetScore("tg_123") != 0 || len(GetTopPlayers(s, 10)) != 0 {
		t.Fatal("unsigned mutation changed cache")
	}
}
func TestMilestoneCallsNeverPretendToPayOrConsumeAClaim(t *testing.T) {
	service, event, key, _ := scoreFixture(t)
	event.Score = 5000
	event = signScore(t, event, key)
	if _, err := service.Apply(event, time.Now()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		if !errors.Is(EvaluateAndReward(service.state, event.Player), ErrLegacyGameDisabled) {
			t.Fatal("unsupported milestone reported success")
		}
		if !errors.Is(GiveReward(event.Player, "GoldMilestone", 20), ErrLegacyGameDisabled) {
			t.Fatal("fake payment reported success")
		}
	}
	if service.state.GetScore(event.Player) != 5000 {
		t.Fatal("milestone changed projection")
	}
}
func TestProjectionLeaderboardTiesHaveStableOrderAndBoundedResult(t *testing.T) {
	service, event, key, _ := scoreFixture(t)
	for _, player := range []string{"tg_z", "tg_a", "tg_m"} {
		event.Player = player
		event = signScore(t, event, key)
		if _, err := service.Apply(event, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	list := GetTopPlayers(service.state, 2)
	want := []PlayerScore{{Player: "tg_a", Score: 200}, {Player: "tg_m", Score: 200}}
	if !reflect.DeepEqual(list, want) {
		t.Fatal("unstable tie order", list)
	}
	if len(GetTopPlayers(service.state, -1)) != 0 {
		t.Fatal("invalid limit accepted")
	}
}
