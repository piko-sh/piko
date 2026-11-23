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

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

const (
	// maxCorpusBytes is the largest accepted Go fuzz corpus entry.
	maxCorpusBytes = 1 << 20

	// minimumByteLiteral is the length of the shortest quoted byte-slice literal and its
	// closing parenthesis.
	minimumByteLiteral = 3
)

var (
	// errCorpusTooLarge reports a corpus entry above the read limit.
	errCorpusTooLarge = errors.New("corpus entry exceeds 1 MiB")

	// errCorpusFormat reports a corpus entry that is not a single byte-slice argument.
	errCorpusFormat = errors.New("expected single-argument FuzzProject Go corpus")
)

// readProjectCorpus decodes a standard Go corpus containing one byte-slice argument.
//
// Takes reader (io.Reader) which supplies the encoded corpus bytes.
//
// Returns []byte which holds the decoded deterministic project choice recording.
// Returns error which reports an oversized or malformed Go byte-slice corpus, or nil on
// success.
func readProjectCorpus(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxCorpusBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxCorpusBytes {
		return nil, errCorpusTooLarge
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 2 || lines[0] != "go test fuzz v1" {
		return nil, errCorpusFormat
	}
	literal, ok := strings.CutPrefix(lines[1], "[]byte(")
	if !ok || !strings.HasSuffix(literal, ")") || len(literal) < minimumByteLiteral || literal[0] != '"' {
		return nil, fmt.Errorf("%w: expected []byte string literal", errCorpusFormat)
	}
	raw, err := strconv.Unquote(strings.TrimSuffix(literal, ")"))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errCorpusFormat, err)
	}
	return []byte(raw), nil
}

// promote saves a native Go fuzz input as a replayable project case.
//
// The corpus path is named by the operator on the command line, so it is opened as given.
// The read is bounded by maxCorpusBytes.
//
// Takes path (string) which is the native Go FuzzProject corpus file.
// Takes out (string) which is the promoted case's output directory.
//
// Returns error which reports a corpus read, case generation or destination-write
// failure, or nil on success.
func promote(path, out string) (err error) {
	if path == "" || out == "" {
		return errors.New("promote needs a FuzzProject corpus file and -out directory")
	}
	file, err := os.Open(path) //nolint:gosec // G304, G703: the operator names this corpus file on the command line.
	if err != nil {
		return fmt.Errorf("open corpus %s: %w", path, err)
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()
	tape, err := readProjectCorpus(file)
	if err != nil {
		return err
	}
	promoted := generate.NativeProject(tape)
	if writeErr := casefile.Write(out, &promoted); writeErr != nil {
		existing, readErr := casefile.Read(out)
		if readErr != nil || existing.Hash() != promoted.Hash() || existing.Target != promoted.Target || existing.Index != promoted.Index || existing.Seed != promoted.Seed {
			return writeErr
		}
	}
	fmt.Println(out, promoted.Hash())
	return nil
}
