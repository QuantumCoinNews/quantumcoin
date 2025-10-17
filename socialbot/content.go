package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

type ContentItem struct {
	ID        string   `yaml:"id"`
	Title     string   `yaml:"title"`
	Caption   string   `yaml:"caption"`
	Text      string   `yaml:"text"`
	Tags      []string `yaml:"tags"`
	ImagePath string   `yaml:"image"`
	VideoPath string   `yaml:"video"`
	Album     []string `yaml:"album"`
	YTLink    string   `yaml:"yt"`
	XLink     string   `yaml:"x"`
	IGLink    string   `yaml:"ig"`
}

// PickToday -> içerik kuyruğundan sıradaki dosyayı seçer
func PickToday() (ContentItem, error) {
	qdir := getEnv("CONTENT_QUEUE_DIR", "content/queue")

	files, _ := filepath.Glob(filepath.Join(qdir, "*.yaml"))
	if len(files) == 0 {
		return ContentItem{}, errors.New("içerik kuyruğu boş")
	}
	sort.Strings(files) // deterministik seçim (alfabetik/eskiden yeniye)

	fp := files[0]
	var item ContentItem
	b, err := os.ReadFile(fp)
	if err != nil {
		return item, err
	}
	if err := yaml.Unmarshal(b, &item); err != nil {
		return item, err
	}
	if item.ID == "" {
		item.ID = filepath.Base(fp)
	}

	// Gönderildikten sonra arşive taşı
	posted := fp + fmt.Sprintf(".posted.%s", time.Now().UTC().Format("20060102-1504"))
	_ = os.Rename(fp, posted)

	return item, nil
}
