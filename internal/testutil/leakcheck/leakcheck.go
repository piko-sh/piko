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

package leakcheck

import (
	"testing"

	"go.uber.org/goleak"
)

var baseOptions = []goleak.Option{
	goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
}

// VerifyTestMain wraps goleak.VerifyTestMain with Piko's standard ignore rules.
// Use in simple TestMain functions that only need leak checking.
//
// Takes m (*testing.M) which is the test main instance.
// Takes extraOpts (...goleak.Option) which provides additional ignore rules.
func VerifyTestMain(m *testing.M, extraOpts ...goleak.Option) {
	goleak.VerifyTestMain(m, buildOptions(extraOpts)...)
}

// FindLeaks wraps goleak.Find with Piko's standard ignore rules.
//
// Use in complex TestMain functions that need teardown before exit, since
// VerifyTestMain calls os.Exit internally.
//
// Takes extraOpts (...goleak.Option) which provides additional ignore rules.
//
// Returns error when goroutine leaks are detected.
func FindLeaks(extraOpts ...goleak.Option) error {
	return goleak.Find(buildOptions(extraOpts)...)
}

// buildOptions combines base options with extra options into a single slice.
//
// Takes extraOpts ([]goleak.Option) which provides additional options to append.
//
// Returns []goleak.Option which contains baseOptions followed by extraOpts.
func buildOptions(extraOpts []goleak.Option) []goleak.Option {
	opts := make([]goleak.Option, 0, len(baseOptions)+len(extraOpts))
	opts = append(opts, baseOptions...)
	opts = append(opts, extraOpts...)
	return opts
}
