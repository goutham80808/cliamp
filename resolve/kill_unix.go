//go:build !windows

package resolve

import "syscall"

// killProcessTree terminates a process and all of its children on Unix
// by sending SIGKILL to the process group. The negative PID targets the
// entire group, which requires the child to have been started with
// SysProcAttr{Setpgid: true}.
func killProcessTree(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}
