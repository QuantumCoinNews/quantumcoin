package socialbot

import (
	"fmt"
	"time"
)

func Start(bot *SocialBot) {
	fmt.Println("Scheduler başlatıldı.")
	go func() {
		for {
			bot.DoAutoShare()
			// Her gün bir kez çalışması için 24 saat bekler
			time.Sleep(24 * time.Hour)
		}
	}()
}
