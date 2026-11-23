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
	"piko.sh/fuzzkit/tape"
	"piko.sh/safeconv"
)

const (
	// byteRange is the number of distinct byte values.
	byteRange = 256
)

// drawInt draws a bounded choice as an int.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
// Takes limit (uint64) which is the exclusive upper bound of the draw.
//
// Returns int which is the drawn value below limit.
func drawInt(choices *tape.Tape, limit uint64) int {
	return safeconv.Uint64ToInt(choices.Uint(limit))
}

// drawInt64 draws a bounded choice as an int64.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
// Takes limit (uint64) which is the exclusive upper bound of the draw.
//
// Returns int64 which is the drawn value below limit.
func drawInt64(choices *tape.Tape, limit uint64) int64 {
	return safeconv.Uint64ToInt64(choices.Uint(limit))
}

// drawByte draws one byte-sized choice.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns uint8 which is the drawn value.
func drawByte(choices *tape.Tape) uint8 {
	return safeconv.Uint64ToUint8(choices.Uint(byteRange))
}

// pick draws one of the supplied options.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
// Takes options ([]T) which lists one or more candidates.
//
// Returns T which is the drawn candidate.
func pick[T any](choices *tape.Tape, options []T) T {
	return options[choices.Uint(safeconv.IntToUint64(len(options)))]
}
