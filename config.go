package gaze

import (
	"errors"
	"log/slog"

	"go.ofkm.dev/gaze/types"
)

// ErrWatcherClosed is returned when an operation is attempted after Close.
var ErrWatcherClosed = errors.New("gaze: watcher closed")

func defaultConfig() types.Config {
	return types.Config{
		Logger:        slog.Default(),
		Ops:           types.AllOps,
		QueueCapacity: 1024,
	}
}

func resolveConfig(override types.Config) types.Config {
	cfg := defaultConfig()
	cfg.Recursion = override.Recursion
	cfg.ExcludeGlobs = override.ExcludeGlobs
	cfg.ExcludePrefixes = override.ExcludePrefixes
	cfg.Exclude = override.Exclude
	cfg.OnEvent = override.OnEvent
	cfg.OnError = override.OnError
	cfg.Logger = override.Logger
	cfg.Ops = override.Ops
	cfg.QueueCapacity = override.QueueCapacity
	cfg.FollowSymlinks = override.FollowSymlinks
	if cfg.Ops == 0 {
		cfg.Ops = types.AllOps
	} else {
		cfg.Ops |= types.OpOverflow
	}
	if cfg.QueueCapacity <= 0 {
		cfg.QueueCapacity = defaultConfig().QueueCapacity
	}
	if cfg.Logger == nil {
		cfg.Logger = defaultConfig().Logger
	}

	return cfg
}

func recursiveEnabled(cfg types.Config, defaultValue bool) bool {
	switch cfg.Recursion {
	case types.RecursionEnabled:
		return true
	case types.RecursionDisabled:
		return false
	case types.RecursionDefault:
		return defaultValue
	default:
		return defaultValue
	}
}
