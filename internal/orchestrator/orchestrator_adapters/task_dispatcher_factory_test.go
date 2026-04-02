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

package orchestrator_adapters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func TestEventBusTypeName_AllCases(t *testing.T) {
	t.Parallel()

	t.Run("nil event bus returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "nil", eventBusTypeName(nil))
	})

	t.Run("watermillEventBus returns watermillEventBus", func(t *testing.T) {
		t.Parallel()
		bus := &watermillEventBus{}
		assert.Equal(t, "watermillEventBus", eventBusTypeName(bus))
	})

	t.Run("unknown event bus returns unknown", func(t *testing.T) {
		t.Parallel()
		bus := &simpleEventBus{}
		assert.Equal(t, "unknown", eventBusTypeName(bus))
	})
}

func TestCreateTaskDispatcher_NilEventBus(t *testing.T) {
	t.Parallel()

	config := orchestrator_domain.DefaultDispatcherConfig()
	dispatcher := CreateTaskDispatcher(context.Background(), config, nil, nil)
	assert.Nil(t, dispatcher)
}
