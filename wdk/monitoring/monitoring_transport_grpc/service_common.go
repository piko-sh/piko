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

package monitoring_transport_grpc

import (
	"context"
	"fmt"
	"time"

	"piko.sh/piko/wdk/clock"
)

// runWatchLoop executes a periodic update loop for gRPC watch streams.
// It handles ticker management, context cancellation, and error reporting.
//
// Takes intervalMs (int64) which is the requested polling interval.
// Takes sendUpdate (func(...)) which fetches and sends the actual data.
// Takes modeName (string) which describes the update type for error messages.
// Takes clk (clock.Clock) which provides time operations; nil defaults to the
// real system clock.
//
// Returns error when the context is cancelled or an update fails.
func runWatchLoop(ctx context.Context, intervalMs int64, sendUpdate func() error, modeName string, clk clock.Clock) error {
	if clk == nil {
		clk = clock.RealClock()
	}

	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < minWatchIntervalMs*time.Millisecond {
		interval = 1 * time.Second
	}

	ticker := clk.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C():
			if err := sendUpdate(); err != nil {
				return fmt.Errorf("sending %s update: %w", modeName, err)
			}
		}
	}
}
