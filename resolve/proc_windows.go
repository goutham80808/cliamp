//go:build windows

package resolve

import "os/exec"

// setProcAttr is a no-op on Windows. Process tree cleanup is handled by
// taskkill /T which targets the child and all descendants directly.
func setProcAttr(_ *exec.Cmd) {}
