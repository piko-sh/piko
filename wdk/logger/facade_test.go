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

package logger

import (
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/apitest"
)

func TestLoggerFacadeAPI(t *testing.T) {

	surface := apitest.Surface{

		"Attr": Attr{},

		"Logger": (*Logger)(nil),

		"GetLogger":       GetLogger,
		"GetShutdownFunc": GetShutdownFunc,
		"ResetLogger":     ResetLogger,
		"AddPrettyOutput": AddPrettyOutput,
		"AddJSONOutput":   AddJSONOutput,
		"AddFileOutput":   AddFileOutput,

		"String":   String,
		"Strings":  Strings,
		"Int":      Int,
		"Int64":    Int64,
		"Uint64":   Uint64,
		"Float64":  Float64,
		"Bool":     Bool,
		"Time":     Time,
		"Duration": Duration,
		"Error":    Error,
		"Field":    Field,

		"LevelTrace":  LevelTrace,
		"LevelDebug":  LevelDebug,
		"LevelInfo":   LevelInfo,
		"LevelNotice": LevelNotice,
		"LevelWarn":   LevelWarn,
		"LevelError":  LevelError,
	}

	apitest.Check(t, surface, filepath.Join("facade_test.golden.yaml"))
}
