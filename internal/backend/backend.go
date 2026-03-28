package backend

type Op uint32

const (
	OpCreate Op = 1 << iota
	OpWrite
	OpRemove
	OpRename
	OpChmod
	OpOverflow
)

func (o Op) Has(other Op) bool {
	return o&other != 0
}

type Event struct {
	Path    string
	OldPath string
	Op      Op
	IsDir   bool
}

type Target struct {
	Path      string
	WatchPath string
	IsDir     bool
	Recursive bool
}

type Config struct {
	BufferSize     int
	FollowSymlinks bool
	ShouldExclude  func(path string, isDir bool) bool
}

type Watcher interface {
	Add(Target) error
	Remove(path string) error
	Events() <-chan Event
	Errors() <-chan error
	Close() error
}
