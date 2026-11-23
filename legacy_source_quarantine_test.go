package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistoricalReviewSnapshotCannotReturnAsActiveGoSource(t *testing.T) {
	if _, err := os.Stat("_wm_review"); os.IsNotExist(err) {
		return
	}
	err := filepath.WalkDir("_wm_review", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") {
			t.Errorf("historical snapshot restored as Go source: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
