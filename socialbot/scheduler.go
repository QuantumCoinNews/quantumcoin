package main

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

type CronScheduler struct {
	bot  *SocialBot
	cron *cron.Cron
}

func NewCronScheduler(bot *SocialBot) *CronScheduler {
	return &CronScheduler{
		bot:  bot,
		cron: cron.New(cron.WithLocation(time.UTC)),
	}
}

func (s *CronScheduler) AddSpec(spec string) error {
	_, err := s.cron.AddFunc(spec, func() {
		fmt.Println("⏰ Tetikleme:", time.Now().UTC().Format(time.RFC3339))
		s.bot.DoAutoShare()
	})
	return err
}

func (s *CronScheduler) Start() {
	fmt.Println("Scheduler başlatıldı.")
	s.cron.Start()
}
