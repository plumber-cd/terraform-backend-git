package pid

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

var pidFile = os.TempDir() + "/.terraform-backend-git.pid"

func readPid() (int, error) {
	piddata, err := ioutil.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return -1, nil
		}
		return -1, err
	}

	pid, err := strconv.Atoi(string(piddata))
	if err != nil {
		return -1, err
	}

	return pid, nil
}

func pidRunning() (int, error) {
	pid, err := readPid()
	if err != nil {
		return -1, err
	}
	if pid <= 0 {
		return -1, nil
	}

	running, err := processRunning(pid)
	if err != nil || !running {
		return -1, err
	}

	return pid, nil
}

func LockPidFile() error {
	pid, err := pidRunning()
	if err != nil {
		return err
	}

	if pid > 0 {
		return fmt.Errorf("another pid already running: %d", pid)
	}

	return ioutil.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0664)
}

func StopPidFile() error {
	pid, err := pidRunning()
	if err != nil {
		log.Fatal(err)
	}

	if pid <= 0 {
		return nil
	}

	if err := processKill(pid); err != nil {
		log.Fatal(err)
	}

	if err := os.Remove(pidFile); err != nil {
		log.Fatal(err)
	}

	return nil
}
