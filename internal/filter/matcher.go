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

	path = cleanFast(path)

	for _, prefix := range m.prefixes {
		if path == prefix || len(path) > len(prefix) && path[len(prefix)] == filepath.Separator && path[:len(prefix)] == prefix {
			return true
		}
	}

	if len(m.globs) > 0 {
		base := baseFast(path)
		slashed := toSlashFast(path)
		for _, glob := range m.globs {
			if ok, _ := filepath.Match(glob, base); ok {
				return true
			}
			if ok, _ := filepath.Match(glob, slashed); ok {
				return true
			}
		}
	}

	return m.exclude != nil && m.exclude(path, isDir)
}

// cleanFast returns path unchanged if it is already clean (absolute, no
// double separators, no dot segments). Falls back to filepath.Clean.
func cleanFast(path string) string {
	if len(path) == 0 {
		return filepath.Clean(path)
	}
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '.' && (i == 0 || path[i-1] == filepath.Separator) {
			if i+1 == len(path) || path[i+1] == filepath.Separator || path[i+1] == '.' {
				return filepath.Clean(path)
			}
		}
		if c == filepath.Separator && i+1 < len(path) && path[i+1] == filepath.Separator {
			return filepath.Clean(path)
		}
	}
	return path
}

// baseFast returns the last element of path without allocating.
func baseFast(path string) string {
	if path == "" {
		return "."
	}
	// Strip trailing separator
	for len(path) > 0 && path[len(path)-1] == filepath.Separator {
		path = path[:len(path)-1]
	}
	if path == "" {
		return string(filepath.Separator)
	}
	idx := strings.LastIndexByte(path, filepath.Separator)
	if idx >= 0 {
		path = path[idx+1:]
	}
	return path
}
