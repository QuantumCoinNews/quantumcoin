package consolefx

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSpinKeepsConcurrentLogLinesIntact(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "console")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		spinFor(file, 50*time.Millisecond)
	}()
	for i := 0; i < 20; i++ {
		if _, err := fmt.Fprintf(file, "log %d\n", i); err != nil {
			t.Error(err)
		}
	}
	<-done

	raw, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	progress, nextLog := 0, 0
	for _, line := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		switch {
		case line == "| Waiting for the next mining cycle":
			progress++
		case line == fmt.Sprintf("log %d", nextLog):
			nextLog++
		default:
			t.Fatalf("interrupted console line %q", line)
		}
	}
	if progress != 1 || nextLog != 20 {
		t.Fatalf("progress=%d logs=%d", progress, nextLog)
	}
}
