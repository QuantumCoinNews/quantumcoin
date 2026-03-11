module socialbot

go 1.22

// Not:
// - Bu proje Go 1.22 ile test edilmiştir.
// - CI/ekip senkronu için gerekirse aşağıdaki satır açılabilir:
// toolchain go1.22.x

require (
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/joho/godotenv v1.5.1
	github.com/robfig/cron/v3 v3.0.1
	gopkg.in/yaml.v3 v3.0.1
)
