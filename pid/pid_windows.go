//go:build windows
// +build windows

package pid

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func processRunning(pid int) (bool, error) {
	process, err := os.FindProcess(pid)
	if err != nil {
		if err.(*os.SyscallError).Err == windows.ERROR_INVALID_PARAMETER {
			return false, nil
		}
		return false, err
	}

	if err := process.Signal(syscall.Signal(0)); err != nil && err != syscall.EWINDOWS {
		return false, err
	}

	return true, nil
}

func processKill(pid int) error {
	if err := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(pid)).Run(); err != nil {
		return err
	}

	return nil
}
