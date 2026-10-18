// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package library

import (
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"
)

type memInfo struct {
	name string
	size int64
	dir  bool
}

func (i memInfo) Name() string { return i.name }

func (i memInfo) Size() int64 { return i.size }

func (i memInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0o555
	}
	return 0o444
}

func (memInfo) ModTime() time.Time { return time.Time{} }

func (i memInfo) IsDir() bool { return i.dir }

func (memInfo) Sys() any { return nil }

type memEntry struct {
	info memInfo
}

func (e memEntry) Name() string { return e.info.name }

func (e memEntry) IsDir() bool { return e.info.dir }

func (e memEntry) Type() fs.FileMode { return e.info.Mode().Type() }

func (e memEntry) Info() (fs.FileInfo, error) { return e.info, nil }

type memFile struct {
	info    memInfo
	reader  *strings.Reader
	entries []fs.DirEntry
	offset  int
}

func (f *memFile) Stat() (fs.FileInfo, error) { return f.info, nil }

func (f *memFile) Read(buffer []byte) (int, error) {
	if f.info.dir {
		return 0, &fs.PathError{Op: "read", Path: f.info.name, Err: fs.ErrInvalid}
	}
	return f.reader.Read(buffer)
}

func (*memFile) Close() error { return nil }

func (f *memFile) ReadDir(count int) ([]fs.DirEntry, error) {
	rest := f.entries[f.offset:]
	if count > 0 && count < len(rest) {
		rest = rest[:count]
	}
	f.offset += len(rest)
	if count > 0 && len(rest) == 0 {
		return nil, io.EOF
	}
	return rest, nil
}

type memFS map[string]string

func (m memFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	if data, ok := m[name]; ok {
		return &memFile{info: memInfo{name: path.Base(name), size: int64(len(data))}, reader: strings.NewReader(data)}, nil
	}
	prefix := name + "/"
	if name == "." {
		prefix = ""
	}
	seen := map[string]bool{}
	var entries []fs.DirEntry
	for file := range m {
		if !strings.HasPrefix(file, prefix) {
			continue
		}
		child, _, nested := strings.Cut(strings.TrimPrefix(file, prefix), "/")
		if seen[child] {
			continue
		}
		seen[child] = true
		entries = append(entries, memEntry{info: memInfo{name: child, dir: nested}})
	}
	if len(entries) == 0 {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	sort.Slice(entries, func(a, b int) bool { return entries[a].Name() < entries[b].Name() })
	return &memFile{info: memInfo{name: path.Base(name), dir: true}, entries: entries}, nil
}

var archive = memFS{
	"index.md":       "the archive index",
	"notes/one.md":   "first note",
	"notes/two.md":   "second note",
	"notes/skip.txt": "not markdown",
}

func InterpretedSummary() string {
	matches, err := fs.Glob(archive, "notes/*.md")
	if err != nil {
		return "glob error: " + err.Error()
	}
	data, err := fs.ReadFile(archive, "index.md")
	if err != nil {
		return "read error: " + err.Error()
	}
	var walked []string
	err = fs.WalkDir(archive, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		walked = append(walked, name)
		return nil
	})
	if err != nil {
		return "walk error: " + err.Error()
	}
	return strings.Join(matches, ",") + " | " + string(data) + " | " + strings.Join(walked, ",")
}
