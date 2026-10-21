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
	"io/fs"
	"os"
	"strings"
)

type source interface {
	fs.FS
}

type shelf struct {
	files source
}

var nativeShelf = shelf{files: os.DirFS("pkg/library/books")}

var shelvesByName = map[string]source{"books": os.DirFS("pkg/library/books")}

func NativeSummary() string {
	matches, err := fs.Glob(nativeShelf.files, "*.txt")
	if err != nil {
		return "glob error: " + err.Error()
	}
	data, err := fs.ReadFile(shelvesByName["books"], "first.txt")
	if err != nil {
		return "read error: " + err.Error()
	}
	var walked []string
	err = fs.WalkDir(nativeShelf.files, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		walked = append(walked, name)
		return nil
	})
	if err != nil {
		return "walk error: " + err.Error()
	}
	return strings.Join(matches, ",") + " | " + strings.TrimSpace(string(data)) + " | " + strings.Join(walked, ",")
}
