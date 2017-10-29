// Package filetree provides fast lookup of items arranged in POSIX-style
// file hierarchies.
package filetree

import (
	"os"
	"path"
	"time"
)

// A Tree is the root of a file hierarchy. It must be created
// by a call to New.
type Tree struct {
	index map[string]Entry
}

// An Entry represents a single item in a file hierarchy.
// The Children member is only valid until the next call
// to Put. *Entry satisfies the os.FileInfo interface.
type Entry struct {
	// The absolute path of this item
	FullName string

	// Contains all entries in this directory
	Children []Entry

	// Arbitrary value associated with this path. For directories,
	// this is nil.
	Value interface{}
}

func (e *Entry) Name() string {
	return path.Base(e.FullName)
}

func (e *Entry) Size() int64 {
	// This does not need to be exact
	const sizeOfEntry = 512
	return sizeOfEntry * int64(len(e.Children))
}

func (e *Entry) Mode() os.FileMode {
	if len(e.Children) > 0 {
		return os.ModeDir | 0555
	}
	return 0444
}

func (e *Entry) ModTime() time.Time {
	return time.Time{}
}

func (e *Entry) IsDir() bool {
	return e.Mode().IsDir()
}

func (e *Entry) Sys() interface{} {
	return nil
}

// New creates a new Tree with zero children.
func New() *Tree {
	return &Tree{index: make(map[string]Entry)}
}

func normalize(filename string) string {
	return path.Clean("/" + filename)
}

// Put adds a new entry in the file hierarchy. Name must be
// a POSIX-style path name, relative to the root of the tree. If
// any directories in the path are missing, they are created as
// needed. Put is not safe for concurrent use.
func (tree *Tree) Put(name string, value interface{}) {
	name = normalize(name)
	tree.index[name] = Entry{FullName: name, Value: value}

	lastPath := name
	for dir, _ := path.Split(name); len(dir) > 0; dir, _ = path.Split(dir) {
		dir = dir[:len(dir)-1]
		child := tree.index[lastPath]
		parent := tree.index[dir]
		parent.FullName = dir
		parent.Children = append(parent.Children, child)
		tree.index[dir] = parent
		lastPath = dir
	}
}

// Get retrieves the item present at the path given by name. The
// returned Entry is valid if and only if the second return value is true.
func (tree *Tree) Get(name string) (Entry, bool) {
	entry, ok := tree.index[normalize(name)]
	return entry, ok
}

// LongestPrefix retrieves the item in the tree whose path is the
// longest prefix of name. The returned Entry is valid if and only if the
// second return value is true.
func (tree *Tree) LongestPrefix(name string) (Entry, bool) {
	// NOTE(droyo) this lookup scales with the length of the name,
	// rather than the number of entries in the tree. Considering the
	// use case for this package (a path router), a hybrid approach
	// may be better; if len(name) > N, and len(tree.index) < M, loop
	// over tree.index and do a prefix match against name.
	for dir := normalize(name); dir != ""; dir, _ = path.Split(dir) {
		dir = dir[:len(dir)-1]
		if entry, ok := tree.index[dir]; ok {
			return entry, true
		}
	}
	return Entry{}, false
}