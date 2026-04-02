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

package daemon_adapters

import (
	"context"
	"os/signal"
	"syscall"

	"piko.sh/piko/internal/daemon/daemon_domain"
)

// osSignalNotifier implements SignalNotifier for production use.
// It listens for SIGINT and SIGTERM signals using signal.NotifyContext.
type osSignalNotifier struct{}

var _ daemon_domain.SignalNotifier = (*osSignalNotifier)(nil)

// NotifyContext returns a context that is cancelled when SIGINT or SIGTERM
// is received.
//
// Returns context.Context which is cancelled upon receiving a signal.
// Returns context.CancelFunc which can be called to release resources.
func (*osSignalNotifier) NotifyContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
}

// newOSSignalNotifier creates a new OS signal notifier.
//
// Returns daemon_domain.SignalNotifier which sends OS signals to listeners.
func newOSSignalNotifier() daemon_domain.SignalNotifier {
	return &osSignalNotifier{}
}
