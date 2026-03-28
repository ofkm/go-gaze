//go:build !linux && !darwin && !windows

package backend

import "fmt"

func New(Config) (Watcher, error) {
	return nil, fmt.Errorf("filewatch: unsupported platform")
}
