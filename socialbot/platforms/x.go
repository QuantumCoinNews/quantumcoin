package platforms

import "fmt"

func PostXStatus(status string, mediaPath string) error {
	fmt.Printf("X (Twitter)'a gönderiliyor: %s %s\n", status, mediaPath)
	// X API ile paylaşım (mock)
	return nil
}
