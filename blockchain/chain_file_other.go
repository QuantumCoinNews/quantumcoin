//go:build !windows && !unix

package blockchain

import (
	"fmt"
	"os"
)

func lockChainFileOSV1(_, _ string) (func(), error) {
	return nil, fmt.Errorf("chain persistence locking is unsupported on this platform")
}
func replaceChainFileOSV1(_, _ string) error {
	return fmt.Errorf("atomic chain replacement is unsupported on this platform")
}

func openChainFileOSV1(filename string) (*os.File, error) { return os.Open(filename) }
