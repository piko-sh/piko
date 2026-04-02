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

package browser_provider_chromedp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

// FindAvailablePort finds an available TCP port on localhost.
//
// Returns int which is the port number that was found to be available.
// Returns error when the TCP address cannot be resolved or the port cannot
// be opened.
func FindAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("resolving TCP address: %w", err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("listening on TCP: %w", err)
	}
	defer func() { _ = l.Close() }()

	return l.Addr().(*net.TCPAddr).Port, nil
}

// WaitForServerReady waits for a server to become ready by attempting TCP
// connections.
//
// Takes port (int) which specifies the TCP port to connect to.
// Takes timeout (time.Duration) which sets the maximum time to wait.
//
// Returns error when the server does not become ready within the timeout.
func WaitForServerReady(port int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeoutCause(context.Background(), timeout, fmt.Errorf("browser server startup exceeded %s timeout", timeout))
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for server on port %d to become ready", port)
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 50*time.Millisecond)
			if err == nil {
				_ = conn.Close()
				return nil
			}
		}
	}
}

// WaitForURL waits for a URL to become reachable.
//
// Returns error when called, as this function is not implemented.
func WaitForURL(_ string, _ time.Duration) error {
	return errors.New("WaitForURL not implemented, use WaitForServerReady with port")
}
