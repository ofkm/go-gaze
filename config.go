package filewatch

import (
	"errors"
	"log/slog"
)

var ErrWatcherClosed = errors.New("filewatch: watcher closed")

type Configure func(*Config)

type Config struct {
	Recursive       bool
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

func DefaultConfig() Config {
	return defaultConfig()
}

func defaultConfig() Config {
	return Config{
		Recursive:     true,
		Logger:        slog.Default(),
		Ops:           allOps,
		QueueCapacity: 1024,
	}
}

func applyConfig(cfg *Config, configure []Configure) {
	for _, fn := range configure {
		if fn != nil {
			fn(cfg)
		}
	}
	if cfg.Ops == 0 {
		cfg.Ops = allOps
	} else {
		cfg.Ops |= OpOverflow
	}
	if cfg.QueueCapacity <= 0 {
		cfg.QueueCapacity = defaultConfig().QueueCapacity
	}
}
