package core

import (
	"log"
	"os"
)

func InitLogging() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if AppConfig != nil && AppConfig.AppEnv == "prod" {
		log.SetOutput(os.Stdout)
	}
}
