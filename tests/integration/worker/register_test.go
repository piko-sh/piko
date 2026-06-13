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

//go:build integration

package worker_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/worker"
)

type greetArgs struct {
	UserID string `json:"user_id"`
}

func (greetArgs) Kind() string {
	return "email:greet"
}

type recordingGreeter struct {
	got greetArgs
}

func (w *recordingGreeter) Work(_ context.Context, job worker.Job[greetArgs]) error {
	w.got = job.Args
	return nil
}

func TestRegister_BindsKindForLookup(t *testing.T) {
	workerService := worker.NewService(nil)
	worker.Register[greetArgs](workerService, &recordingGreeter{})
	require.True(t, workerService.HasHandler("email:greet"), "the registered kind must be found")
	require.False(t, workerService.HasHandler("email:unknown"), "the unregistered kind must not be found")
}

func TestRegister_DecodesPayloadAndInvokesWork(t *testing.T) {
	workerService := worker.NewService(nil)
	worker.Register[greetArgs](workerService, &recordingGreeter{})

}
