// Package discovery holds information about discovered cmd
package discovery

import (
	"github.com/spf13/cobra"
)

var root *cobra.Command

// backends is a list of backend types available via cmd wrapper (such as 'git')
var backends = make([]*cobra.Command, 0)

// wrappers is a list of wrapper commands to add as child to each backend from above
var wrappers = make([]*cobra.Command, 0)

func RegisterRoot(r *cobra.Command) {
	root = r

	for _, backend := range backends {
		root.AddCommand(backend)
	}
}

func Root() *cobra.Command {
	return root
}

func RegisterBackend(backend *cobra.Command) {
	backends = append(backends, backend)

	for _, wrapper := range wrappers {
		backend.AddCommand(wrapper)
	}

	if root != nil {
		root.AddCommand(backend)
	}
}

func RegisterWrapper(wrapper *cobra.Command) {
	wrappers = append(wrappers, wrapper)

	for _, backend := range backends {
		backend.AddCommand(wrapper)
	}
}
