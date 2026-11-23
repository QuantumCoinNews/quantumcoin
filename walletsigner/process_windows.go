package walletsigner

import (
	"os/exec"
	"syscall"
)

func hideSignerProcess(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} }
