package blockchain

import (
	"io"
	"os"
)

// SnapshotPersistedChainFileV1 captures one completed disk commit while holding
// the same in-process and OS lock as mining, NFT commits, autosave and shutdown.
// It never saves a potentially newer RAM state or changes the live chain.
func SnapshotPersistedChainFileV1(source, destination string) error {
	abs, key, err := canonicalChainFilenameV1(source)
	if err != nil {
		return err
	}
	unlock := lockChainFilenameV1(key)
	defer unlock()
	unlockOS, err := lockChainFileOSV1(abs, key)
	if err != nil {
		return err
	}
	defer unlockOS()
	in, err := openChainFileOSV1(abs)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err == nil {
		err = out.Sync()
	}
	cerr := out.Close()
	if err == nil {
		err = cerr
	}
	return err
}
