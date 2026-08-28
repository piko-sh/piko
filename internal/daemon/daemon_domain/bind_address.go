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

package daemon_domain

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

var (
	// errBindAddressHasPort reports a bind address that already carries a port.
	errBindAddressHasPort = errors.New("bind address must not include a port")
)

// normaliseBindAddress prepares a configured bind address for net.JoinHostPort.
//
// Takes address (string) which is the configured bind address.
//
// Returns string which is the address with any surrounding brackets removed.
func normaliseBindAddress(address string) string {
	trimmed := strings.TrimSpace(address)

	if len(trimmed) > 1 && strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		return trimmed[1 : len(trimmed)-1]
	}

	return trimmed
}

// validateBindAddress rejects a bind address that carries a port.
//
// Takes address (string) which is the normalised bind address.
// Takes portOptionName (string) which names the option that sets the port.
//
// Returns error when the address carries a port.
func validateBindAddress(address, portOptionName string) error {
	if address == "" {
		return nil
	}

	if net.ParseIP(address) != nil {
		return nil
	}

	if _, _, err := net.SplitHostPort(address); err == nil {
		return fmt.Errorf(
			"%w: %q sets the port with %s instead", errBindAddressHasPort, address, portOptionName,
		)
	}

	return nil
}
