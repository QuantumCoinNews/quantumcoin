package miner

import (
	"time"
)

type Schedule struct {
	StartHour int
	EndHour   int
}

func ShouldMineNow(s Schedule) bool {
	now := time.Now()
	hour := now.Hour()
	return hour >= s.StartHour && hour < s.EndHour
}
