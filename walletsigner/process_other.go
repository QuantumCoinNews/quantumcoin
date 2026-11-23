//go:build !windows

package walletsigner

import "os/exec"

func hideSignerProcess(cmd *exec.Cmd) {}
