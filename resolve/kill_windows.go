//go:build windows

package resolve

import (
	"os/exec"
	"strconv"
)

// killProcessTree terminates a process and all of its children on Windows
// using taskkill. This is needed because exec.CommandContext only kills the
// direct process, leaving child processes (e.g. ffmpeg spawned by yt-dlp)
// running and holding file handles open.
func killProcessTree(pid int) error {
	return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run()
}
