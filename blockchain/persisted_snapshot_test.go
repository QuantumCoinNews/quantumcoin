package blockchain

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestChainSnapshotProcessWriter(t *testing.T) {
	source := os.Getenv("QC_SNAPSHOT_PROCESS_SOURCE")
	if source == "" {
		return
	}
	err := persistChainFileV1(source, func(w io.Writer) error {
		if _, err := w.Write(bytes.Repeat([]byte("committed-generation"), 8192)); err != nil {
			return err
		}
		if err := os.WriteFile(source+".entered", []byte("ready"), 0600); err != nil {
			return err
		}
		deadline := time.Now().Add(20 * time.Second)
		for {
			if _, err := os.Stat(source + ".release"); err == nil {
				break
			}
			if time.Now().After(deadline) {
				return os.ErrDeadlineExceeded
			}
			time.Sleep(10 * time.Millisecond)
		}
		_, err := w.Write([]byte("-complete"))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestChainSnapshotCoordinatesWithAnotherProcessCommit(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "chain.dat")
	target := filepath.Join(root, "snapshot.dat")
	if err := os.WriteFile(source, []byte("old-complete-generation"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestChainSnapshotProcessWriter$", "-test.timeout=30s")
	cmd.Env = append(os.Environ(), "QC_SNAPSHOT_PROCESS_SOURCE="+source)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(source + ".entered"); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("writer did not acquire persistence lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	done := make(chan error, 1)
	go func() { done <- SnapshotPersistedChainFileV1(source, target) }()
	select {
	case err := <-done:
		t.Fatal("snapshot bypassed active commit lock", err)
	case <-time.After(150 * time.Millisecond):
	}
	if err := os.WriteFile(source+".release", []byte("release"), 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("snapshot did not finish")
	}
	want := append(bytes.Repeat([]byte("committed-generation"), 8192), []byte("-complete")...)
	got, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatal("snapshot did not capture a complete coordinated commit", err)
	}
	if err := SnapshotPersistedChainFileV1(source, target); err == nil {
		t.Fatal("snapshot overwrote target")
	}
}
