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

package caller

import (
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// decimalBase is the base for decimal number formatting.
const decimalBase = 10

// frameInfo holds details about a program counter location.
type frameInfo struct {
	// name is the function name from the stack frame.
	name string

	// file is the source file path where the frame is located.
	file string

	// frame is the cached "\tfile:line" formatted string.
	frame string

	// line is the line number in the source file.
	line int
}

// frameCache maps program counters to their resolved frame information.
// Using sync.Map because the access pattern is read-heavy (most PCs are
// resolved once and read many times) and concurrent.
var frameCache sync.Map

// NameFileLine returns the function name, file path, and line number for this
// program counter.
//
// Results are cached to avoid repeated lookups. The file path is shortened to
// show only the useful part (for example, "internal/logger/logger.go" rather
// than the full path).
//
// Returns name (string) which is the fully qualified function name.
// Returns file (string) which is the shortened file path.
// Returns line (int) which is the line number within the file.
//
// Returns empty strings and 0 if the PC is 0 or cannot be resolved.
func (pc PC) NameFileLine() (name, file string, line int) {
	if pc == 0 {
		return "", "", 0
	}

	if cached, ok := frameCache.Load(pc); ok {
		if info, valid := cached.(frameInfo); valid {
			return info.name, info.file, info.line
		}
	}

	name, file, line = pc.resolveFrame()

	if file != "" && name != "" {
		file = cropFilename(file, name)
	}

	frame := ""
	if file != "" {
		frame = formatFrame(file, line)
	}

	frameCache.Store(pc, frameInfo{
		name:  name,
		file:  file,
		line:  line,
		frame: frame,
	})

	return name, file, line
}

// FormattedFrame returns a pre-formatted stack frame string in the format
// "\tfile:line". This is optimised for stack trace output.
//
// Returns string which is the formatted frame, or empty if the PC is 0 or the
// file cannot be resolved. Results are cached, so after warmup this returns
// zero allocations.
func (pc PC) FormattedFrame() string {
	if pc == 0 {
		return ""
	}

	if cached, ok := frameCache.Load(pc); ok {
		if info, valid := cached.(frameInfo); valid {
			return info.frame
		}
	}

	pc.NameFileLine()

	if cached, ok := frameCache.Load(pc); ok {
		if info, valid := cached.(frameInfo); valid {
			return info.frame
		}
	}

	return ""
}

// ResetFrameCache clears the frame cache. Intended for test isolation.
func ResetFrameCache() {
	frameCache.Range(func(key, _ any) bool {
		frameCache.Delete(key)
		return true
	})
}

// formatFrame builds a stack frame string in the format "\tfile:line".
// Uses a stack-allocated buffer to avoid fmt.Sprintf allocation overhead.
//
// Takes file (string) which is the source file path.
// Takes line (int) which is the line number in the file.
//
// Returns string which is the formatted stack frame.
func formatFrame(file string, line int) string {
	var buffer [280]byte
	b := buffer[:0]
	b = append(b, '\t')
	b = append(b, file...)
	b = append(b, ':')
	b = strconv.AppendInt(b, int64(line), decimalBase)
	return string(b)
}

// cropFilename extracts the relevant portion of a file path based on the
// function name. This produces paths like
// "internal/logger/logger_domain/logger.go" rather than full absolute paths.
//
// Takes file (string) which is the full file path.
// Takes functionName (string) which is the fully qualified function name.
//
// Returns string which is the cropped file path.
func cropFilename(file, functionName string) string {
	lastSlash := strings.LastIndexByte(functionName, '/')
	if lastSlash == -1 {
		return filepath.Base(file)
	}

	dotAfterSlash := strings.IndexByte(functionName[lastSlash+1:], '.')
	if dotAfterSlash == -1 {
		return filepath.Base(file)
	}

	packagePath := functionName[:lastSlash+1+dotAfterSlash]

	for packagePath != "" {
		if index := strings.LastIndex(file, packagePath); index != -1 {
			return file[index:]
		}

		slashIndex := strings.IndexByte(packagePath, '/')
		if slashIndex == -1 {
			break
		}
		packagePath = packagePath[slashIndex+1:]
	}

	return filepath.Base(file)
}
