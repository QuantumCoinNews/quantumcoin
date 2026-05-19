package main

import (
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

type CronScheduler struct {
	bot     *SocialBot
	cron    *cron.Cron
	started bool
}

func NewCronScheduler(bot *SocialBot) *CronScheduler {
	// Default parser = 5-field cron (min hour dom month dow) which matches "0 18 * * *"
	c := cron.New(cron.WithLocation(time.UTC))
	return &CronScheduler{
		bot:  bot,
		cron: c,
	}
}

func (s *CronScheduler) AddSpec(spec string) error {
	if s == nil || s.cron == nil {
		return nil
	}
	_, err := s.cron.AddFunc(spec, func() {
		log.Printf("⏰ cron trigger (UTC): %s", time.Now().UTC().Format(time.RFC3339))
		if s.bot != nil {
			s.bot.DoAutoShare()
		}
	})
	if err == nil {
		log.Printf("cron registered: %q (UTC)", spec)
	}
	return err
}

func (s *CronScheduler) Start() {
	if s == nil || s.cron == nil {
		return
	}
	if s.started {
		return
	}
	s.started = true
	log.Printf("scheduler started (UTC)")
	s.cron.Start()
}

func (s *CronScheduler) Stop() {
	if s == nil || s.cron == nil {
		return
	}
	if !s.started {
		return
	}
	s.started = false
	log.Printf("scheduler stopping…")
	ctx := s.cron.Stop()

	// wait briefly for running job(s) to finish
	select {
	case <-ctx.Done():
		log.Printf("scheduler stopped")
	case <-time.After(2 * time.Second):
		log.Printf("scheduler stop timeout (jobs may still be running)")
	}
}
