package socialbot

import "fmt"

func GetAPIKey(platform string) string {
	fmt.Printf("%s için API anahtarı alınıyor.\n", platform)
	// Gerçek sistemde .env veya config dosyasından okumalısın!
	return "API_KEY"
}
