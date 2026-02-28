//go:build !windows
// +build !windows

package server

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/plumber-cd/terraform-backend-git/backend"
)

func startReloadHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)

	go func() {
		for range ch {
			log.Println("reload signal received")
			for name, client := range backend.KnownStorageTypes {
				reloadable, ok := client.(interface{ Reload() error })
				if !ok {
					continue
				}
				if err := reloadable.Reload(); err != nil {
					log.Printf("reload %s: %v", name, err)
				}
			}
		}
	}()
}
