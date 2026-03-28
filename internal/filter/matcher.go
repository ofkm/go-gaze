package filter

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Config struct {
	Prefixes []string
	Globs    []string
	Exclude  func(path string, isDir bool) bool
}

type Matcher struct {
	prefixes []string
	globs    []string
	exclude  func(path string, isDir bool) bool
}

func New(cfg Config) (*Matcher, error) {
	prefixes := make([]string, 0, len(cfg.Prefixes))
	for _, prefix := range cfg.Prefixes {
		if strings.TrimSpace(prefix) == "" {
			continue
		}
		prefixes = append(prefixes, filepath.Clean(prefix))
	}

	globs := make([]string, 0, len(cfg.Globs))
	for _, glob := range cfg.Globs {
		if strings.TrimSpace(glob) == "" {
			continue
		}
		if _, err := filepath.Match(glob, "probe"); err != nil {
			return nil, fmt.Errorf("gaze: invalid exclude glob %q: %w", glob, err)
		}
		globs = append(globs, glob)
	}

	return &Matcher{
		prefixes: prefixes,
		globs:    globs,
		exclude:  cfg.Exclude,
	}, nil
}

func (m *Matcher) ShouldExclude(path string, isDir bool) bool {
	if len(m.prefixes) == 0 && len(m.globs) == 0 && m.exclude == nil {
		return false
	}

	path = filepath.Clean(path)

	for _, prefix := range m.prefixes {
		if path == prefix || strings.HasPrefix(path, prefix+string(filepath.Separator)) {
			return true
		}
	}

	base := filepath.Base(path)
	slashed := filepath.ToSlash(path)
	for _, glob := range m.globs {
		if ok, _ := filepath.Match(glob, base); ok {
			return true
		}
		if ok, _ := filepath.Match(glob, slashed); ok {
			return true
		}
	}

	return m.exclude != nil && m.exclude(path, isDir)
}
