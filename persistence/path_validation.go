package persistence

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var domainSegmentV1 = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func safeDomainSegmentV1(s string) bool {
	if !domainSegmentV1.MatchString(s) {
		return false
	}
	switch s {
	case "con", "prn", "aux", "nul":
		return false
	}
	return !(len(s) == 4 && (strings.HasPrefix(s, "com") || strings.HasPrefix(s, "lpt")) && s[3] >= '1' && s[3] <= '9')
}

// Reject existing symlink/junction ancestors before resolving inventory paths.
// Callers must still protect their directories from concurrent replacement.
func rejectPathLinksV1(path string) error {
	p, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for {
		info, err := os.Lstat(p)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			return fmt.Errorf("linked persistence path rejected: %s", p)
		}
		parent := filepath.Dir(p)
		if parent == p {
			return nil
		}
		p = parent
	}
}
