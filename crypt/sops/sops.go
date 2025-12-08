package sops

import (
	"log"

	sops "github.com/getsops/sops/v3"
)

type Config interface {
	IsActivated() bool
	KeyGroup() (sops.KeyGroup, error)
}

var Configs = make(map[string]Config)

func GetActivatedKeyGroups() ([]sops.KeyGroup, error) {
	keyGroups := make([]sops.KeyGroup, 0)

	for provider, config := range Configs {
		if config.IsActivated() {
			log.Printf("Activating %q encryption provider", provider)
			kg, err := config.KeyGroup()
			if err != nil {
				return nil, err
			}
			keyGroups = append(keyGroups, kg)
		}
	}

	return keyGroups, nil
}
