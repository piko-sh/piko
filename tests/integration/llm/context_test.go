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

package llm_integration_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDo_ContextTimeout_ReturnsError(t *testing.T) {
	service, _ := createZoltaiService(t)

	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err := service.NewCompletion().
		User("Tell me a fortune").
		Do(ctx)

	assert.ErrorIs(t, err, context.Canceled)
}

func TestToolLoop_ContextTimeout_ReturnsError(t *testing.T) {
	service, _ := createZoltaiService(t)

	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err := service.NewCompletion().
		User("What is the weather?").
		ToolFunc("get_weather", "Get weather", nil, func(_ context.Context, _ string) (string, error) {
			return "sunny", nil
		}).
		Do(ctx)

	assert.ErrorIs(t, err, context.Canceled)
}
