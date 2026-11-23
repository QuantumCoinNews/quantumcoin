package offline

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestChunkedBackupAuthenticatesFrameOrderAndEmptyLogs(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	key := bytes.Repeat([]byte{7}, 32)
	for _, size := range []int{0, chunkSize, 3*chunkSize + 17} {
		source := filepath.Join(root, "input")
		sealed := filepath.Join(root, "sealed")
		opened := filepath.Join(root, "opened")
		mustWrite(t, source, bytes.Repeat([]byte{42}, size))
		if err := cryptArtifact(ctx, source, sealed, "log-artifact", key, true); err != nil {
			t.Fatal(err)
		}
		if err := cryptArtifact(ctx, sealed, opened, "log-artifact", key, false); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(opened)
		if err != nil || len(got) != size || !bytes.Equal(got, bytes.Repeat([]byte{42}, size)) {
			t.Fatal("chunked restore differs", err)
		}
		if size > 2*chunkSize {
			ciphertext, _ := os.ReadFile(sealed)
			frameSize := 4 + chunkSize + 16
			reordered := append([]byte{}, ciphertext[:8]...)
			reordered = append(reordered, ciphertext[8+frameSize:8+2*frameSize]...)
			reordered = append(reordered, ciphertext[8:8+frameSize]...)
			reordered = append(reordered, ciphertext[8+2*frameSize:]...)
			bad := filepath.Join(root, "reordered")
			mustWrite(t, bad, reordered)
			if err := cryptArtifact(ctx, bad, filepath.Join(root, "rejected"), "log-artifact", key, false); err == nil {
				t.Fatal("frame reordering accepted")
			}
		}
		for _, path := range []string{source, sealed, opened} {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		}
	}
}
