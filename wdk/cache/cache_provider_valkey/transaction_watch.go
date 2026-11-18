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

package cache_provider_valkey

import (
	"context"
	"errors"
	"time"

	"github.com/valkey-io/valkey-go"
	"piko.sh/piko/wdk/logger"
)

const (
	// unwatchTimeout bounds the UNWATCH sent after a transaction that ended without EXEC.
	unwatchTimeout = 5 * time.Second
)

var (
	// errUnwatchTimedOut is the cause given when clearing a WATCH takes longer than
	// unwatchTimeout.
	errUnwatchTimedOut = errors.New("valkey UNWATCH exceeded its timeout")
)

// execTracker wraps a dedicated connection and records whether a transaction called
// DoMulti for its MULTI/EXEC block.
type execTracker struct {
	valkey.DedicatedClient

	// execSent records a DoMulti call, which may fail before EXEC reaches the server.
	execSent bool
}

// DoMulti forwards a transaction's MULTI/EXEC block and records the attempt.
//
// Takes multi (...valkey.Completed) which are the commands of the block.
//
// Returns []valkey.ValkeyResult which holds one result per command.
func (t *execTracker) DoMulti(ctx context.Context, multi ...valkey.Completed) []valkey.ValkeyResult {
	t.execSent = true
	return t.DedicatedClient.DoMulti(ctx, multi...)
}

// runWatchedTransaction runs one transaction on a dedicated connection.
//
// If the attempt ends without calling DoMulti, it clears WATCH so the pooled connection
// cannot abort a later transaction. A failed DoMulti call does not trigger this cleanup.
//
// Takes transaction (func(valkey.DedicatedClient) error) which runs the attempt.
//
// Returns error which is the attempt's error.
func (a *ValkeyAdapter[K, V]) runWatchedTransaction(ctx context.Context, transaction func(c valkey.DedicatedClient) error) error {
	return a.client.Dedicated(func(c valkey.DedicatedClient) error {
		tracker := &execTracker{DedicatedClient: c, execSent: false}
		err := transaction(tracker)
		if !tracker.execSent {
			releaseWatch(ctx, c)
		}
		return err
	})
}

// releaseWatch sends UNWATCH on a dedicated connection. When that fails it closes the
// connection, so the pool never hands out one that may still be watching a key.
//
// The UNWATCH keeps the values of ctx but not its cancellation, because the transaction
// may have ended by running out of time.
//
// Takes c (valkey.DedicatedClient) which is the connection to clear.
func releaseWatch(ctx context.Context, c valkey.DedicatedClient) {
	unwatchCtx, cancel := context.WithTimeoutCause(context.WithoutCancel(ctx), unwatchTimeout, errUnwatchTimedOut)
	defer cancel()
	if err := c.Do(unwatchCtx, c.B().Unwatch().Build()).Error(); err != nil {
		_, l := logger.From(ctx, log)
		l.Warn("Closing a dedicated Valkey connection whose WATCH could not be cleared", logger.Error(err))
		c.Close()
	}
}
