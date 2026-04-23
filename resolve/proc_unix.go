//go:build !windows

package resolve

import (
	"os/exec"
	"syscall"
)

// setProcAttr configures the child process to start in its own process group
// so that killProcessTree can target the entire group via negative PID.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
