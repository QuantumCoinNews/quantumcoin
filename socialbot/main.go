package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// .env yükle
	_ = godotenv.Load()

	// TZ ayarı (opsiyonel)
	if tz := os.Getenv("TZ"); tz != "" {
		_ = os.Setenv("TZ", tz)
	}

	log.Printf("socialbot starting… (TZ=%s)", os.Getenv("TZ"))

	// Yetkiler / bağlantılar
	auth := NewAuth()

	// Bot (içerik + publish akışı)
	bot := NewSocialBot(auth)

	// Cron scheduler (UTC)
	sched := NewCronScheduler(bot)
	if err := sched.AddSpec(getEnv("CRON_1", "0 18 * * *")); err != nil {
		log.Fatal(err)
	}
	if err := sched.AddSpec(getEnv("CRON_2", "0 21 * * *")); err != nil {
		log.Fatal(err)
	}
	sched.Start()

	log.Printf("running; press Ctrl+C to stop")

	// Basit sonsuz bekleme (daemon)
	for {
		time.Sleep(24 * time.Hour)
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
