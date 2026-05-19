package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"socialbot/core"
)

func main() {
	_ = godotenv.Load()

	if tz := os.Getenv("TZ"); tz != "" {
		_ = os.Setenv("TZ", tz)
	}

	log.Printf("socialbot starting... (TZ=%s)", os.Getenv("TZ"))

	auth := core.NewAuth()
	bot := NewSocialBot(auth)

	sched := NewCronScheduler(bot)
	if err := sched.AddSpec(getEnv("CRON_1", "0 18 * * *")); err != nil {
		log.Fatal(err)
	}
	if err := sched.AddSpec(getEnv("CRON_2", "0 21 * * *")); err != nil {
		log.Fatal(err)
	}
	sched.Start()

	log.Printf("running; press Ctrl+C to stop")

	// Wait for SIGINT/SIGTERM
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	// Give logs a moment to flush
	time.Sleep(200 * time.Millisecond)
	log.Printf("bye")
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
