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
	// baseGlobs are matched against the path's base name only; pathGlobs are
	// matched against the slash-separated path only. Splitting at construction
	// halves filepath.Match calls in the ShouldExclude hot path.
	baseGlobs []string
	pathGlobs []string
	exclude   func(path string, isDir bool) bool
}

func New(cfg Config) (*Matcher, error) {
	prefixes := make([]string, 0, len(cfg.Prefixes))
	for _, prefix := range cfg.Prefixes {
		if strings.TrimSpace(prefix) == "" {
			continue
		}
		prefixes = append(prefixes, filepath.Clean(prefix))
	}

	var baseGlobs, pathGlobs []string
	for _, glob := range cfg.Globs {
		if strings.TrimSpace(glob) == "" {
			continue
		}
		if _, err := filepath.Match(glob, "probe"); err != nil {
			return nil, fmt.Errorf("gaze: invalid exclude glob %q: %w", glob, err)
		}
		if strings.ContainsRune(glob, '/') || strings.ContainsRune(glob, filepath.Separator) {
			// A pattern with a separator can never match a bare base name.
			pathGlobs = append(pathGlobs, glob)
			continue
		}
		baseGlobs = append(baseGlobs, glob)
		if filepath.Separator != '/' {
			// On Windows, '/' is an ordinary character to filepath.Match, so a
			// separator-free pattern can still match a slashed path. Keep
			// matching it both ways to preserve semantics.
			pathGlobs = append(pathGlobs, glob)
		}
	}

	return &Matcher{
		prefixes:  prefixes,
		baseGlobs: baseGlobs,
		pathGlobs: pathGlobs,
		exclude:   cfg.Exclude,
	}, nil
}

func (m *Matcher) ShouldExclude(path string, isDir bool) bool {
	if len(m.prefixes) == 0 && len(m.baseGlobs) == 0 && len(m.pathGlobs) == 0 && m.exclude == nil {
		return false
	}

	path = cleanFast(path)

	for _, prefix := range m.prefixes {
		if path == prefix || len(path) > len(prefix) && path[len(prefix)] == filepath.Separator && path[:len(prefix)] == prefix {
			return true
		}
	}

	if len(m.baseGlobs) > 0 {
		base := baseFast(path)
		for _, glob := range m.baseGlobs {
			if ok, _ := filepath.Match(glob, base); ok {
				return true
			}
		}
	}
	if len(m.pathGlobs) > 0 {
		slashed := toSlashFast(path)
		for _, glob := range m.pathGlobs {
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
	for i := range len(path) {
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
