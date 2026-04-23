//go:build !windows

package resolve

import (
	"fmt"
	"syscall"
)

// killProcessTree terminates a process and all of its children on Unix
// by sending SIGKILL to the process group. The negative PID targets the
// entire group, which requires the child to have been started with
// SysProcAttr{Setpgid: true}.
func killProcessTree(pid int) error {
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		return fmt.Errorf("kill process group -%d: %w", pid, err)
	}
	return nil
}
