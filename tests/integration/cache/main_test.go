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

package cache_integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
	"go.uber.org/goleak"
	"piko.sh/piko/internal/testutil/leakcheck"
)

type testEnv struct {
	redisAddr         string
	redisClusterAddrs []string
	valkeyAddr        string
	rawClient         *redis.Client
	rawClusterClient  *redis.ClusterClient
	cleanup           func()
}

var globalEnv *testEnv

func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error
	globalEnv, err = setupTestEnvironment(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to setup test environment: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	if globalEnv.cleanup != nil {
		globalEnv.cleanup()
	}

	if code == 0 {
		if err := leakcheck.FindLeaks(

			goleak.IgnoreAnyFunction("github.com/valkey-io/valkey-go.(*call).LazyDo.func1"),
		); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "goleak: %v\n", err)
			os.Exit(1)
		}
	}
	os.Exit(code)
}
