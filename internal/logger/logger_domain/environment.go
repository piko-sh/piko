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

package logger_domain

import (
	"log/slog"
	"os"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// envMapping defines how a single environment variable maps to slog and OTEL
// attributes. When the env var is present and non-empty, both attributes are
// emitted.
type envMapping struct {
	// otelFunction builds an OTEL resource attribute from the env var value.
	otelFunction func(string) attribute.KeyValue

	// envKey is the environment variable name to read.
	envKey string

	// slogKey is the structured log field key.
	slogKey string
}

// runtimeEnvProbe detects which runtime environment the process is in based
// on the presence of a sentinel environment variable. The first match wins.
type runtimeEnvProbe struct {
	// envKey is the sentinel environment variable whose presence indicates
	// this runtime.
	envKey string

	// label is the value emitted for the runtime.environment attribute.
	label string
}

var (
	// envMappings is the table of environment variables to detect.
	envMappings = []envMapping{
		// Kubernetes (Downward API - user configures in pod spec).
		{envKey: "POD_NAME", slogKey: "k8s.pod.name", otelFunction: semconv.K8SPodName},
		{envKey: "POD_NAMESPACE", slogKey: "k8s.namespace.name", otelFunction: semconv.K8SNamespaceName},
		{envKey: "POD_IP", slogKey: "k8s.pod.ip", otelFunction: func(v string) attribute.KeyValue { return attribute.String("k8s.pod.ip", v) }},
		{envKey: "NODE_NAME", slogKey: "k8s.node.name", otelFunction: semconv.K8SNodeName},
		// AWS Lambda (auto-injected by the runtime).
		{envKey: "AWS_LAMBDA_FUNCTION_NAME", slogKey: "faas.name", otelFunction: semconv.FaaSName},
		{envKey: "AWS_LAMBDA_FUNCTION_VERSION", slogKey: "faas.version", otelFunction: semconv.FaaSVersion},
		{envKey: "AWS_REGION", slogKey: "cloud.region", otelFunction: semconv.CloudRegion},
		// Google Cloud Run (auto-injected by the platform).
		{envKey: "K_SERVICE", slogKey: "faas.name", otelFunction: semconv.FaaSName},
		{envKey: "K_REVISION", slogKey: "faas.version", otelFunction: semconv.FaaSVersion},
		{envKey: "GOOGLE_CLOUD_PROJECT", slogKey: "cloud.account.id", otelFunction: semconv.CloudAccountID},
		// Azure Container Apps (auto-injected by the platform).
		{envKey: "CONTAINER_APP_NAME", slogKey: "azure.container_app.name", otelFunction: func(v string) attribute.KeyValue { return attribute.String("azure.container_app.name", v) }},
		{envKey: "CONTAINER_APP_REVISION", slogKey: "azure.container_app.revision", otelFunction: func(v string) attribute.KeyValue { return attribute.String("azure.container_app.revision", v) }},
		{envKey: "CONTAINER_APP_REPLICA_NAME", slogKey: "azure.container_app.replica", otelFunction: func(v string) attribute.KeyValue { return attribute.String("azure.container_app.replica", v) }},
		// Piko-specific / generic overrides (applied last).
		{envKey: "PIKO_ENVIRONMENT", slogKey: "deployment.environment.name", otelFunction: semconv.DeploymentEnvironmentName},
		{envKey: "PIKO_SERVICE_NAME", slogKey: "service.name", otelFunction: semconv.ServiceName},
		{envKey: "PIKO_SERVICE_VERSION", slogKey: "service.version", otelFunction: semconv.ServiceVersion},
	}

	// runtimeEnvProbes is checked in order; the first matching sentinel env var
	// determines the runtime.environment label. Detection is presence-only: the
	// value of the variable is not used.
	runtimeEnvProbes = []runtimeEnvProbe{
		{envKey: "POD_NAME", label: "kubernetes"},
		{envKey: "AWS_LAMBDA_FUNCTION_NAME", label: "aws-lambda"},
		{envKey: "K_SERVICE", label: "cloud-run"},
		{envKey: "CONTAINER_APP_NAME", label: "azure-container-apps"},
		{envKey: "ECS_CONTAINER_METADATA_URI_V4", label: "aws-ecs"},
	}

	// cachedDetection stores the one-time detection result. Both
	// EnvironmentSlogAttrs and EnvironmentOtelAttrs read from this cache.
	cachedDetection = sync.OnceValues(detectEnvironment)
)

// EnvironmentSlogAttrs returns the detected environment attributes for slog
// logger enrichment via Logger.With().
//
// Returns []slog.Attr which contains the detected environment attributes.
func EnvironmentSlogAttrs() []slog.Attr {
	attrs, _ := cachedDetection()
	return attrs
}

// EnvironmentOtelAttrs returns the detected environment attributes for OTEL
// resource enrichment.
//
// Returns []attribute.KeyValue which contains the detected OTEL attributes.
func EnvironmentOtelAttrs() []attribute.KeyValue {
	_, otelAttrs := cachedDetection()
	return otelAttrs
}

// EnvironmentOverridesServiceName reports whether the environment provides a
// service name override via PIKO_SERVICE_NAME.
//
// Returns bool which is true when PIKO_SERVICE_NAME is set.
func EnvironmentOverridesServiceName() bool {
	_, ok := os.LookupEnv("PIKO_SERVICE_NAME")
	return ok
}

// EnvironmentOverridesServiceVersion reports whether the environment provides
// a service version override via PIKO_SERVICE_VERSION.
//
// Returns bool which is true when PIKO_SERVICE_VERSION is set.
func EnvironmentOverridesServiceVersion() bool {
	_, ok := os.LookupEnv("PIKO_SERVICE_VERSION")
	return ok
}

// SlogAttrsToAny converts a slice of slog.Attr to []any for use with
// slog.Logger.With().
//
// Takes attrs ([]slog.Attr) which contains the attributes to convert.
//
// Returns []any which contains each attribute as an any value.
func SlogAttrsToAny(attrs []slog.Attr) []any {
	result := make([]any, len(attrs))
	for i, a := range attrs {
		result[i] = a
	}
	return result
}

// detectEnvironment reads well-known environment variables and returns slog
// and OTEL resource attributes for any that are present and non-empty.
//
// Returns []slog.Attr which contains the slog attributes for detected vars.
// Returns []attribute.KeyValue which contains the OTEL attributes.
func detectEnvironment() ([]slog.Attr, []attribute.KeyValue) {
	var slogAttrs []slog.Attr
	var otelAttrs []attribute.KeyValue

	for _, m := range envMappings {
		v, ok := os.LookupEnv(m.envKey)
		if !ok || v == "" {
			continue
		}
		slogAttrs = append(slogAttrs, slog.String(m.slogKey, v))
		otelAttrs = append(otelAttrs, m.otelFunction(v))
	}

	for _, probe := range runtimeEnvProbes {
		if v, ok := os.LookupEnv(probe.envKey); ok && v != "" {
			slogAttrs = append(slogAttrs, slog.String(KeyRuntimeEnvironment, probe.label))
			otelAttrs = append(otelAttrs, attribute.String(KeyRuntimeEnvironment, probe.label))
			break
		}
	}

	return slogAttrs, otelAttrs
}
