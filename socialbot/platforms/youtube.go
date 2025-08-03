package platforms

import "fmt"

func PostYouTubeVideo(title, description, filePath string) error {
	fmt.Printf("YouTube'a video gönderiliyor: %s | %s %s\n", title, description, filePath)
	// YouTube API ile paylaşım (mock)
	return nil
}
