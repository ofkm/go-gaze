package gaze

import (
	"errors"
	"log/slog"
)

// ErrWatcherClosed is returned by [Watcher.Add] and [Watcher.Remove] when they
// are called after [Watcher.Close]. Test for it with errors.Is.
var ErrWatcherClosed = errors.New("gaze: watcher closed")

func defaultConfig() Config {
	return Config{
		Logger:        slog.Default(),
		Ops:           AllOps,
		QueueCapacity: 1024,
	}
}

func resolveConfig(override Config) Config {
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
		cfg.Ops = AllOps
	} else {
		cfg.Ops |= OpOverflow
	}
	if cfg.QueueCapacity <= 0 {
		cfg.QueueCapacity = defaultConfig().QueueCapacity
	}
	if cfg.Logger == nil {
		cfg.Logger = defaultConfig().Logger
	}

	return cfg
}

func recursiveEnabled(cfg Config, defaultValue bool) bool {
	switch cfg.Recursion {
	case RecursionEnabled:
		return true
	case RecursionDisabled:
		return false
	case RecursionDefault:
		return defaultValue
	default:
		return defaultValue
	}
}
