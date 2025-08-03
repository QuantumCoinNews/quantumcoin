package platforms

import "fmt"

func PostInstagramMedia(caption, filePath string) error {
	fmt.Printf("Instagram'a gönderi paylaşılıyor: %s %s\n", caption, filePath)
	// Instagram API ile paylaşım (mock)
	return nil
}
