package i18n

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
)

var messages map[string]map[string]string

// init varsayılan dil dosyalarını yükler
func init() {
	messages = make(map[string]map[string]string)

	loadLanguage("en")
	loadLanguage("tr")
}

// loadLanguage belirtilen dili dosyadan yükler
func loadLanguage(lang string) {
	path := filepath.Join("i18n", fmt.Sprintf("%s.json", lang))
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("[i18n] %s.json yüklenemedi: %v", lang, err)
		return
	}
	var langMessages map[string]string
	err = json.Unmarshal(data, &langMessages)
	if err != nil {
		log.Printf("[i18n] %s.json parse hatası: %v", lang, err)
		return
	}
	messages[lang] = langMessages
}

// T: çeviri fonksiyonu. Anahtar ve dil verildiğinde çeviriyi döndürür.
func T(lang, key string) string {
	if langMap, ok := messages[lang]; ok {
		if msg, ok := langMap[key]; ok {
			return msg
		}
	}
	return fmt.Sprintf("[%s:%s]", lang, key) // bulunamadıysa
}
