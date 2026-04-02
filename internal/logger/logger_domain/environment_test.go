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

//go:build !bench

package logger_domain_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestDetectEnvironment_NoEnvVars(t *testing.T) {
	t.Parallel()

	slogAttrs, otelAttrs := logger_domain.DetectEnvironment()

	assert.Empty(t, slogAttrs, "should produce no slog attrs when no env vars are set")
	assert.Empty(t, otelAttrs, "should produce no OTEL attrs when no env vars are set")
}

func TestDetectEnvironment_Kubernetes(t *testing.T) {
	t.Setenv("POD_NAME", "web-abc123")
	t.Setenv("POD_NAMESPACE", "production")
	t.Setenv("POD_IP", "10.0.1.42")
	t.Setenv("NODE_NAME", "node-pool-1")

	slogAttrs, otelAttrs := logger_domain.DetectEnvironment()

	slogMap := slogAttrsToMap(slogAttrs)
	assert.Equal(t, "web-abc123", slogMap["k8s.pod.name"])
	assert.Equal(t, "production", slogMap["k8s.namespace.name"])
	assert.Equal(t, "10.0.1.42", slogMap["k8s.pod.ip"])
	assert.Equal(t, "node-pool-1", slogMap["k8s.node.name"])
	assert.Equal(t, "kubernetes", slogMap["runtime.environment"])

	otelMap := otelAttrsToMap(otelAttrs)
	assert.Equal(t, "web-abc123", otelMap["k8s.pod.name"])
	assert.Equal(t, "production", otelMap["k8s.namespace.name"])
	assert.Equal(t, "10.0.1.42", otelMap["k8s.pod.ip"])
	assert.Equal(t, "node-pool-1", otelMap["k8s.node.name"])
	assert.Equal(t, "kubernetes", otelMap["runtime.environment"])
}

func TestDetectEnvironment_AWSLambda(t *testing.T) {
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "my-handler")
	t.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "$LATEST")
	t.Setenv("AWS_REGION", "eu-west-1")

	slogAttrs, otelAttrs := logger_domain.DetectEnvironment()

	slogMap := slogAttrsToMap(slogAttrs)
	assert.Equal(t, "my-handler", slogMap["faas.name"])
	assert.Equal(t, "$LATEST", slogMap["faas.version"])
	assert.Equal(t, "eu-west-1", slogMap["cloud.region"])
	assert.Equal(t, "aws-lambda", slogMap["runtime.environment"])

	otelMap := otelAttrsToMap(otelAttrs)
	assert.Equal(t, "my-handler", otelMap["faas.name"])
	assert.Equal(t, "$LATEST", otelMap["faas.version"])
	assert.Equal(t, "eu-west-1", otelMap["cloud.region"])
	assert.Equal(t, "aws-lambda", otelMap["runtime.environment"])
}

func TestDetectEnvironment_CloudRun(t *testing.T) {
	t.Setenv("K_SERVICE", "api-service")
	t.Setenv("K_REVISION", "api-service-00042")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "my-project-123")

	slogAttrs, otelAttrs := logger_domain.DetectEnvironment()

	slogMap := slogAttrsToMap(slogAttrs)
	assert.Equal(t, "api-service", slogMap["faas.name"])
	assert.Equal(t, "api-service-00042", slogMap["faas.version"])
	assert.Equal(t, "my-project-123", slogMap["cloud.account.id"])
	assert.Equal(t, "cloud-run", slogMap["runtime.environment"])

	otelMap := otelAttrsToMap(otelAttrs)
	assert.Equal(t, "api-service", otelMap["faas.name"])
	assert.Equal(t, "api-service-00042", otelMap["faas.version"])
	assert.Equal(t, "my-project-123", otelMap["cloud.account.id"])
	assert.Equal(t, "cloud-run", otelMap["runtime.environment"])
}

func TestDetectEnvironment_AzureContainerApps(t *testing.T) {
	t.Setenv("CONTAINER_APP_NAME", "my-app")
	t.Setenv("CONTAINER_APP_REVISION", "my-app--rev1")
	t.Setenv("CONTAINER_APP_REPLICA_NAME", "my-app--rev1-abc123")

	slogAttrs, otelAttrs := logger_domain.DetectEnvironment()

	slogMap := slogAttrsToMap(slogAttrs)
	assert.Equal(t, "my-app", slogMap["azure.container_app.name"])
	assert.Equal(t, "my-app--rev1", slogMap["azure.container_app.revision"])
	assert.Equal(t, "my-app--rev1-abc123", slogMap["azure.container_app.replica"])
	assert.Equal(t, "azure-container-apps", slogMap["runtime.environment"])

	otelMap := otelAttrsToMap(otelAttrs)
	assert.Equal(t, "my-app", otelMap["azure.container_app.name"])
	assert.Equal(t, "my-app--rev1", otelMap["azure.container_app.revision"])
	assert.Equal(t, "my-app--rev1-abc123", otelMap["azure.container_app.replica"])
	assert.Equal(t, "azure-container-apps", otelMap["runtime.environment"])
}

func TestDetectEnvironment_AWSECS(t *testing.T) {
	t.Setenv("ECS_CONTAINER_METADATA_URI_V4", "http://169.254.170.2/v4/abc123")

	slogAttrs, otelAttrs := logger_domain.DetectEnvironment()

	slogMap := slogAttrsToMap(slogAttrs)
	assert.Equal(t, "aws-ecs", slogMap["runtime.environment"])

	assert.Len(t, slogAttrs, 1)

	otelMap := otelAttrsToMap(otelAttrs)
	assert.Equal(t, "aws-ecs", otelMap["runtime.environment"])
	assert.Len(t, otelAttrs, 1)
}

func TestDetectEnvironment_PikoOverrides(t *testing.T) {
	t.Setenv("PIKO_ENVIRONMENT", "staging")
	t.Setenv("PIKO_SERVICE_NAME", "piko-api")
	t.Setenv("PIKO_SERVICE_VERSION", "2.1.0")

	slogAttrs, otelAttrs := logger_domain.DetectEnvironment()

	slogMap := slogAttrsToMap(slogAttrs)
	assert.Equal(t, "staging", slogMap["deployment.environment.name"])
	assert.Equal(t, "piko-api", slogMap["service.name"])
	assert.Equal(t, "2.1.0", slogMap["service.version"])

	otelMap := otelAttrsToMap(otelAttrs)
	assert.Equal(t, "staging", otelMap["deployment.environment.name"])
	assert.Equal(t, "piko-api", otelMap["service.name"])
	assert.Equal(t, "2.1.0", otelMap["service.version"])
}

func TestDetectEnvironment_PartialEnvVars(t *testing.T) {
	t.Setenv("POD_NAME", "web-abc123")

	slogAttrs, otelAttrs := logger_domain.DetectEnvironment()

	slogMap := slogAttrsToMap(slogAttrs)
	assert.Equal(t, "web-abc123", slogMap["k8s.pod.name"])
	assert.Equal(t, "kubernetes", slogMap["runtime.environment"])
	assert.NotContains(t, slogMap, "k8s.namespace.name")
	assert.NotContains(t, slogMap, "k8s.pod.ip")
	assert.NotContains(t, slogMap, "k8s.node.name")

	assert.Len(t, slogAttrs, 2)
	assert.Len(t, otelAttrs, 2)
}

func TestDetectEnvironment_EmptyValueIgnored(t *testing.T) {
	t.Setenv("POD_NAME", "")

	slogAttrs, otelAttrs := logger_domain.DetectEnvironment()

	assert.Empty(t, slogAttrs)
	assert.Empty(t, otelAttrs)
}

func TestEnvironmentOverridesServiceName(t *testing.T) {
	t.Run("not set", func(t *testing.T) {
		assert.False(t, logger_domain.EnvironmentOverridesServiceName())
	})

	t.Run("set", func(t *testing.T) {
		t.Setenv("PIKO_SERVICE_NAME", "custom-name")
		assert.True(t, logger_domain.EnvironmentOverridesServiceName())
	})

	t.Run("set to empty", func(t *testing.T) {
		t.Setenv("PIKO_SERVICE_NAME", "")
		assert.True(t, logger_domain.EnvironmentOverridesServiceName(),
			"presence check should return true even when value is empty")
	})
}

func TestEnvironmentOverridesServiceVersion(t *testing.T) {
	t.Run("not set", func(t *testing.T) {
		assert.False(t, logger_domain.EnvironmentOverridesServiceVersion())
	})

	t.Run("set", func(t *testing.T) {
		t.Setenv("PIKO_SERVICE_VERSION", "1.0.0")
		assert.True(t, logger_domain.EnvironmentOverridesServiceVersion())
	})
}

func TestDetectEnvironment_RuntimeEnvironmentLabel(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		expected string
	}{
		{name: "kubernetes", envKey: "POD_NAME", envValue: "pod-1", expected: "kubernetes"},
		{name: "aws-lambda", envKey: "AWS_LAMBDA_FUNCTION_NAME", envValue: "fn", expected: "aws-lambda"},
		{name: "cloud-run", envKey: "K_SERVICE", envValue: "service", expected: "cloud-run"},
		{name: "azure-container-apps", envKey: "CONTAINER_APP_NAME", envValue: "app", expected: "azure-container-apps"},
		{name: "aws-ecs", envKey: "ECS_CONTAINER_METADATA_URI_V4", envValue: "http://169.254.170.2/v4/x", expected: "aws-ecs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envValue)

			slogAttrs, _ := logger_domain.DetectEnvironment()
			slogMap := slogAttrsToMap(slogAttrs)

			require.Contains(t, slogMap, "runtime.environment")
			assert.Equal(t, tt.expected, slogMap["runtime.environment"])
		})
	}
}

func TestDetectEnvironment_FirstRuntimeEnvironmentWins(t *testing.T) {

	t.Setenv("POD_NAME", "pod-1")
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "fn")

	slogAttrs, _ := logger_domain.DetectEnvironment()
	slogMap := slogAttrsToMap(slogAttrs)

	assert.Equal(t, "kubernetes", slogMap["runtime.environment"],
		"kubernetes probe should match before aws-lambda")
}

func TestSlogAttrsToAny(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		result := logger_domain.SlogAttrsToAny(nil)
		assert.Empty(t, result)
	})

	t.Run("converts attrs", func(t *testing.T) {
		t.Parallel()
		attrs := []slog.Attr{
			slog.String("a", "1"),
			slog.String("b", "2"),
		}
		result := logger_domain.SlogAttrsToAny(attrs)
		require.Len(t, result, 2)
		assert.Equal(t, attrs[0], result[0])
		assert.Equal(t, attrs[1], result[1])
	})
}

func slogAttrsToMap(attrs []slog.Attr) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[a.Key] = a.Value.String()
	}
	return m
}

func otelAttrsToMap(attrs []attribute.KeyValue) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, kv := range attrs {
		m[string(kv.Key)] = kv.Value.Emit()
	}
	return m
}
