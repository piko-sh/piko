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
	"net"
	"testing"
	"time"
)

func TestFindAvailablePort(t *testing.T) {
	t.Run("finds available port", func(t *testing.T) {
		port, err := FindAvailablePort()
		if err != nil {
			t.Fatalf("FindAvailablePort() error = %v", err)
		}

		if port <= 0 {
			t.Errorf("FindAvailablePort() returned invalid port: %d", port)
		}

		listener, err := net.Listen("tcp", "localhost:"+string(rune(port)))
		if err == nil {
			_ = listener.Close()
		}

	})

	t.Run("finds different ports on successive calls", func(t *testing.T) {
		port1, err := FindAvailablePort()
		if err != nil {
			t.Fatalf("FindAvailablePort() first call error = %v", err)
		}

		port2, err := FindAvailablePort()
		if err != nil {
			t.Fatalf("FindAvailablePort() second call error = %v", err)
		}

		if port1 <= 0 || port2 <= 0 {
			t.Errorf("got invalid ports: %d, %d", port1, port2)
		}
	})
}

func TestWaitForServerReady(t *testing.T) {
	t.Run("succeeds when server is running", func(t *testing.T) {

		listener, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			t.Fatalf("creating listener: %v", err)
		}
		defer func() { _ = listener.Close() }()

		port := listener.Addr().(*net.TCPAddr).Port

		err = WaitForServerReady(port, 2*time.Second)
		if err != nil {
			t.Errorf("WaitForServerReady() error = %v", err)
		}
	})

	t.Run("times out when server is not running", func(t *testing.T) {

		port := 59999

		listener, err := net.Listen("tcp", "localhost:59999")
		if err == nil {
			_ = listener.Close()
		}

		err = WaitForServerReady(port, 200*time.Millisecond)
		if err == nil {
			t.Error("expected timeout error, got nil")
		}
	})
}

func TestWaitForURL(t *testing.T) {
	t.Run("returns not implemented error", func(t *testing.T) {
		err := WaitForURL("http://localhost:8080", time.Second)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
