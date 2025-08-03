package platforms

import "fmt"

func PostTikTokVideo(caption, filePath string) error {
	fmt.Printf("TikTok'a video gönderiliyor: %s %s\n", caption, filePath)
	// TikTok API ile paylaşım (mock)
	return nil
}
