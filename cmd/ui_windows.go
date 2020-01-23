// +build windows

package cmd

import (
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

// GetTermSize returns the current terminal window size on Windows,
// but is a no-op on all other platforms.
func GetTermSize(fd, termWidth int) (width, height int, err error) {
	return terminal.GetSize(fd)
}

// NotifyWindowResize listens for SIGWINCH (terminal window size changes)
// on *nix platforms, and is a no-op on Windows.
func NotifyWindowResize() <-chan os.Signal {
	return make(<-chan os.Signal, 1)
}