//go:build !windows && !unix

package wallet

import (
	"errors"
	"os"
)

func openWalletRead(path string) (*os.File, error) { return os.Open(path) }

// Browser recovery uses an encrypted worker vault, never this native file store.
func lockWalletFile(string) (func(), error) {
	return nil, errors.New("native wallet storage is unavailable on this platform")
}
func replaceWalletFile(string, string) error {
	return errors.New("native wallet storage is unavailable on this platform")
}
