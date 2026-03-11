package content

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ContentItem — tüm platformların kullandığı TEK içerik modeli
type ContentItem struct {
	ID      string   `yaml:"id"`
	Title   string   `yaml:"title"`
	Caption string   `yaml:"caption"`
	Text    string   `yaml:"text"`
	Tags    []string `yaml:"tags"`

	ImagePath string   `yaml:"image"`
	VideoPath string   `yaml:"video"`
	Album     []string `yaml:"album"`

	YTLink string `yaml:"yt"`
	XLink  string `yaml:"x"`
	IGLink string `yaml:"ig"`
}

// PickToday -> içerik kuyruğundan sıradaki dosyayı seçer
func PickToday() (ContentItem, error) {
	qdir := getEnv("CONTENT_QUEUE_DIR", "content/queue")
	adir := getEnv("CONTENT_ARCHIVE_DIR", "content/posted")

	// Queue klasörü var mı?
	if st, err := os.Stat(qdir); err != nil || !st.IsDir() {
		return ContentItem{}, fmt.Errorf("content queue dir not found: %s", qdir)
	}

	files, err := filepath.Glob(filepath.Join(qdir, "*.yaml"))
	if err != nil {
		return ContentItem{}, fmt.Errorf("glob failed: %w", err)
	}
	if len(files) == 0 {
		return ContentItem{}, errors.New("content queue is empty")
	}

	// deterministik seçim (alfabetik = en eski önce)
	sort.Strings(files)

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
		item.ID = strings.TrimSuffix(filepath.Base(fp), ".yaml")
	}

	// Minimum gönderilebilir içerik kontrolü
	if strings.TrimSpace(item.Text) == "" &&
		strings.TrimSpace(item.Caption) == "" &&
		strings.TrimSpace(item.ImagePath) == "" &&
		strings.TrimSpace(item.VideoPath) == "" &&
		len(item.Album) == 0 {
		return ContentItem{}, fmt.Errorf("content item %q has no postable fields (text/caption/image/video/album)", item.ID)
	}

	// Arşiv klasörünü garanti et
	if err := os.MkdirAll(adir, 0o755); err != nil {
		return ContentItem{}, fmt.Errorf("mkdir archive dir failed: %w", err)
	}

	// Gönderildikten sonra arşive taşı (aynı isimle çakışmasın)
	dst := filepath.Join(adir, filepath.Base(fp))
	dst = dst + fmt.Sprintf(".posted.%s", time.Now().UTC().Format("20060102-1504"))

	if err := os.Rename(fp, dst); err != nil {
		// rename fail olursa tekrar post riskine girmeyelim: burada hata döndürüyoruz
		return ContentItem{}, fmt.Errorf("archive move failed (%s -> %s): %w", fp, dst, err)
	}

	return item, nil
}

// =========================================================
// ENV HELPERS (local to content package)
// =========================================================

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func getEnvInt(k string, d int) int {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
		return n
	}
	return d
}

func getEnvBool(k string, d bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
	if v == "" {
		return d
	}
	switch v {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return d
	}
}
