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

//go:build integration

package examples_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, relPath)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = srcFile.Close() }()
		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer func() { _ = destFile.Close() }()
		_, err = io.Copy(destFile, srcFile)
		return err
	})
}

func findPikoProjectRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(directory, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			content, readErr := os.ReadFile(goModPath)

			if readErr == nil && strings.Contains(string(content), "module piko.sh/piko\n") {
				return directory, nil
			}
		}

		parentDir := filepath.Dir(directory)
		if parentDir == directory {
			return "", errors.New("could not find piko project root")
		}
		directory = parentDir
	}
}
