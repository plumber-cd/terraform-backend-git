//go:build windows
// +build windows

package server

func startReloadHandler() {
	// No-op: Windows doesn't support POSIX-style reload signals.
}
