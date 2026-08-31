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

package bootstrap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	identityEnvVars = []string{
		"POD_NAME", "CONTAINER_APP_REPLICA_NAME", "AWS_LAMBDA_LOG_STREAM_NAME",
		"PIKO_SERVICE_NAME", "K_SERVICE", "AWS_LAMBDA_FUNCTION_NAME", "CONTAINER_APP_NAME",
		"PIKO_SERVICE_VERSION", "PIKO_ENVIRONMENT", "AWS_REGION", "NODE_NAME",
	}
)

func clearIdentityEnv(t *testing.T) {
	t.Helper()
	for _, key := range identityEnvVars {
		t.Setenv(key, "")
		_ = os.Unsetenv(key)
	}
}

func TestResolveInstanceID_PrefersPlatformReplicaID(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "kubernetes pod name wins",
			env:  map[string]string{"POD_NAME": "web-7c1f", "CONTAINER_APP_REPLICA_NAME": "ignored"},
			want: "web-7c1f",
		},
		{
			name: "azure replica name when there is no pod",
			env:  map[string]string{"CONTAINER_APP_REPLICA_NAME": "app--rev-abc"},
			want: "app--rev-abc",
		},
		{
			name: "lambda log stream identifies the execution environment",
			env:  map[string]string{"AWS_LAMBDA_LOG_STREAM_NAME": "2026/08/19/[$LATEST]ff01"},
			want: "2026/08/19/[$LATEST]ff01",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearIdentityEnv(t)
			for key, value := range tc.env {
				t.Setenv(key, value)
			}

			require.Equal(t, tc.want, resolveInstanceID("test-host"))
		})
	}
}

func TestResolveInstanceID_PrefersTheMachineName(t *testing.T) {
	clearIdentityEnv(t)

	require.Equal(t, "worker-3", resolveInstanceID("worker-3"))
}

func TestResolveInstanceID_GeneratesWhenNothingIdentifiesTheHost(t *testing.T) {
	clearIdentityEnv(t)

	first := resolveInstanceID("")
	require.NotEmpty(t, first, "a sink keys node identity on the instance id")

	require.NotEqual(t, first, resolveInstanceID(""),
		"with nothing to derive from, each call generates a fresh id")
}

func TestIdentity_StartedAtIsProcessStart(t *testing.T) {
	assert.Equal(t, processStart, Identity().StartedAt)
	assert.False(t, Identity().StartedAt.IsZero())
}

func TestIdentity_IsStableAcrossCalls(t *testing.T) {
	first, second := Identity(), Identity()

	require.Equal(t, first, second, "Identity is resolved once per process")
	assert.NotEmpty(t, first.InstanceID)
	assert.Equal(t, os.Getpid(), first.PID)
	assert.False(t, first.StartedAt.IsZero())
}

func TestResolveServiceVersion_PrefersExplicitOverride(t *testing.T) {
	clearIdentityEnv(t)
	t.Setenv("PIKO_SERVICE_VERSION", "1.8.4")

	require.Equal(t, "1.8.4", resolveServiceVersion())
}

func TestResolveServiceVersion_NeverEmpty(t *testing.T) {
	clearIdentityEnv(t)

	require.NotEmpty(t, resolveServiceVersion(), "the version falls back rather than returning empty")
}

func TestFirstEnv_PrecedenceAndFallback(t *testing.T) {
	clearIdentityEnv(t)

	require.Equal(t, defaultServiceName, firstEnv(serviceNameEnvKeys, defaultServiceName))

	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "checkout")
	require.Equal(t, "checkout", firstEnv(serviceNameEnvKeys, defaultServiceName))

	t.Setenv("PIKO_SERVICE_NAME", "billing")
	require.Equal(t, "billing", firstEnv(serviceNameEnvKeys, defaultServiceName),
		"PIKO_SERVICE_NAME outranks a platform name")
}

func TestFirstEnv_TreatsEmptyAsUnset(t *testing.T) {
	clearIdentityEnv(t)
	t.Setenv("PIKO_SERVICE_NAME", "")
	t.Setenv("K_SERVICE", "renderer")

	require.Equal(t, "renderer", firstEnv(serviceNameEnvKeys, defaultServiceName),
		"an exported-but-empty variable counts as unset")
}
