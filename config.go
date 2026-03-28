package gaze

import (
	"errors"
	"log/slog"
)

var ErrWatcherClosed = errors.New("gaze: watcher closed")

type RecursionMode uint8

const (
	RecursionDefault RecursionMode = iota
	RecursionDisabled
	RecursionEnabled
)

type Config struct {
	Recursion       RecursionMode
	ExcludeGlobs    []string
	ExcludePrefixes []string
	Exclude         func(PathInfo) bool
	OnEvent         func(Event)
	OnError         func(error)
	Logger          *slog.Logger
	Ops             Op
	QueueCapacity   int
	FollowSymlinks  bool
}

func defaultConfig() Config {
	return Config{
		Logger:        slog.Default(),
		Ops:           allOps,
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
		cfg.Ops = allOps
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

func (cfg Config) recursiveEnabled(defaultValue bool) bool {
	switch cfg.Recursion {
	case RecursionEnabled:
		return true
	case RecursionDisabled:
		return false
	default:
		return defaultValue
	}
}
