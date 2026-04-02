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

package netutil

import (
	"errors"
	"net"
	"strings"
)

// IsPortInUseError checks whether an error indicates that a network port is
// already in use.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true if the error wraps a *net.OpError whose inner
// error message contains "address already in use".
func IsPortInUseError(err error) bool {
	opErr, ok := errors.AsType[*net.OpError](err)
	if !ok {
		return false
	}

	return strings.Contains(opErr.Err.Error(), "address already in use")
}
