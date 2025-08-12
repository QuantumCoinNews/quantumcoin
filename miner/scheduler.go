package miner

import "time"

type Schedule struct {
	StartHour int
	EndHour   int
}

func ShouldMineNow(s Schedule) bool {
	h := time.Now().Hour()
	return h >= s.StartHour && h < s.EndHour
}
