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

package cache_provider_redis_cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClusterOptionsCarryEveryRoutingSetting(t *testing.T) {
	config := Config{
		Addrs:          []string{"node-a:7000", "node-b:7001"},
		Password:       "secret",
		MaxRedirects:   5,
		ReadOnly:       true,
		RouteByLatency: true,
		RouteRandomly:  true,
	}

	options := clusterOptions(&config)

	assert.Equal(t, config.Addrs, options.Addrs)
	assert.Equal(t, "secret", options.Password)
	assert.Equal(t, 5, options.MaxRedirects)
	assert.True(t, options.ReadOnly)
	assert.True(t, options.RouteByLatency)
	assert.True(t, options.RouteRandomly)
}
