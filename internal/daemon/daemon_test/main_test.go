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

// Package daemon_test provides exhaustive integration tests for the daemon HTTP surface layer.
// These tests focus on HTTP contract testing (status codes, headers, body patterns) without
// testing the underlying pk compilation or generation pipeline.
//
// All tests are designed to run in short mode (go test -short) and use in-memory mocks
// for fast execution.
package daemon_test

import (
	"flag"
	"testing"

	"piko.sh/piko/internal/testutil/leakcheck"
)

func TestMain(m *testing.M) {
	flag.Parse()
	leakcheck.VerifyTestMain(m)
}
