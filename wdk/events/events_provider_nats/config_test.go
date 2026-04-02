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

package events_provider_nats

import (
	"testing"
	"time"

	nc "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()

	assert.Equal(t, nc.DefaultURL, config.URL)
	assert.Equal(t, "piko-events", config.ClusterID)
	assert.Equal(t, "piko", config.QueueGroupPrefix)
	assert.Equal(t, 1, config.SubscribersCount)
	assert.Equal(t, 30*time.Second, config.AckWaitTimeout)
	assert.Equal(t, 30*time.Second, config.CloseTimeout)

	assert.False(t, config.JetStream.Disabled)
	assert.True(t, config.JetStream.AutoProvision)
	assert.False(t, config.JetStream.TrackMessageID)
	assert.False(t, config.JetStream.AckAsync)
	assert.Equal(t, "piko", config.JetStream.DurablePrefix)

	assert.Equal(t, int64(30), config.RouterConfig.CloseTimeout)
}
