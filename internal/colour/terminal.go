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

package colour

import (
	"os"
	"sync/atomic"
)

// colourEnabled holds the global flag that controls whether ANSI colour output is active.
var colourEnabled atomic.Bool

func init() {
	enableVirtualTerminalProcessing()

	if envIsSet("NO_COLOUR") || envIsSet("NO_COLOR") {
		return
	}

	if envIsSet("FORCE_COLOUR") || envIsSet("FORCE_COLOR") {
		colourEnabled.Store(true)
		return
	}

	if os.Getenv("TERM") == "dumb" {
		return
	}

	colourEnabled.Store(isTerminal(os.Stdout.Fd()))
}

// Enabled reports whether colour output is currently active. Safe for
// concurrent use by multiple goroutines.
//
// Returns bool which is true when colour escape sequences should be emitted.
func Enabled() bool {
	return colourEnabled.Load()
}

// SetEnabled overrides the auto-detected colour state. Use in tests or when
// a configuration flag controls colour output.
//
// Takes value (bool) which enables colour when true and disables it when
// false.
func SetEnabled(value bool) {
	colourEnabled.Store(value)
}

// envIsSet reports whether the environment variable with the
// given key is set.
//
// Takes key (string) which specifies the environment variable name.
//
// Returns bool which is true when the variable is present.
func envIsSet(key string) bool {
	_, present := os.LookupEnv(key)
	return present
}
