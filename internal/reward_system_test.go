package internal

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestQMA0050RelativeBonusPathCreatesParent(t *testing.T) {
	t.Setenv("QC_NODE_DIR", t.TempDir())
	SetBonusFile("nested/bonus.log")
	t.Cleanup(func() {
		SetBonusFile("")
	})

	path := currentBonusFilePath()
	want := filepath.Join(os.Getenv("QC_NODE_DIR"), "nested", "bonus.log")

	if path != want {
		t.Fatalf("bonus path = %q, want %q", path, want)
	}

	if !GiveBonus("QCTestAddress", "Yearly", 1, "test", "") {
		t.Fatal("GiveBonus returned false")
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("bonus file was not created: %v", err)
	}
}

func TestQMA0050WriteFailureIsFailClosed(t *testing.T) {
	SetBonusFile(t.TempDir())
	t.Cleanup(func() {
		SetBonusFile("")
	})

	if GiveBonus("QCTestAddress", "Yearly", 1, "test", "") {
		t.Fatal("GiveBonus returned true for an unwritable directory path")
	}
}

func TestQMA0050ConcurrentSetAndGiveBonus(t *testing.T) {
	root := t.TempDir()
	SetBonusFile("")
	t.Cleanup(func() {
		SetBonusFile("")
	})

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			path := filepath.Join(
				root,
				"bonus",
				string(rune('a'+(i%5))),
				"bonus.log",
			)

			SetBonusFile(path)
			_ = GiveBonus(
				"QCTestAddress",
				"Yearly",
				1,
				"concurrency",
				"",
			)
		}(i)
	}

	wg.Wait()
}
