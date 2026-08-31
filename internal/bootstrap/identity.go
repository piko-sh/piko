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
	"sync"
	"time"

	"github.com/google/uuid"

	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// defaultServiceName is the service name reported when neither an explicit override nor
	// a platform-supplied name is present.
	defaultServiceName = "piko"

	// defaultServiceVersion is the version reported when the binary carries no build info -
	// which is the normal case for `go run`.
	defaultServiceVersion = "dev"
)

var (
	// identityEnvKeys lists, in precedence order, the environment variables that carry a
	// platform-assigned replica identifier.
	identityEnvKeys = []string{
		"POD_NAME",
		"CONTAINER_APP_REPLICA_NAME",
		"AWS_LAMBDA_LOG_STREAM_NAME",
	}

	// serviceNameEnvKeys lists, in precedence order, the environment variables that name the
	// running service.
	serviceNameEnvKeys = []string{
		"PIKO_SERVICE_NAME",
		"K_SERVICE",
		"AWS_LAMBDA_FUNCTION_NAME",
		"CONTAINER_APP_NAME",
	}

	// cachedIdentity resolves the process identity exactly once.
	cachedIdentity = sync.OnceValue(resolveIdentity)

	// processStart is captured when the package loads, which is process start.
	processStart = time.Now()
)

// DeploymentIdentity describes the running process: which build it is, where it runs, and
// how to distinguish it from a sibling replica.
type DeploymentIdentity struct {
	// StartedAt is when this process started, captured when the package loaded rather than
	// when the identity was first resolved, so it does not depend on when a caller asks.
	StartedAt time.Time

	// InstanceID identifies this process among sibling replicas of the same service, stable
	// for the process lifetime.
	InstanceID string

	// Hostname is os.Hostname(), empty when it cannot be resolved.
	Hostname string

	// ServiceName is the deployed service's name, defaulting to "piko".
	ServiceName string

	// ServiceVersion is the running build's version: the PIKO_SERVICE_VERSION override, else
	// the main module version from build info, else "dev".
	ServiceVersion string

	// Environment is the deployment environment ("production", "staging", ...) from
	// PIKO_ENVIRONMENT, empty when unset.
	Environment string

	// Region is the cloud region, empty off-cloud.
	Region string

	// NodeName is the cluster node hosting the process, empty off-cluster.
	NodeName string

	// Runtime labels the detected platform ("kubernetes", "aws-lambda", "cloud-run",
	// "azure-container-apps", "aws-ecs"), empty when none matched.
	Runtime string

	// PID is the operating-system process identifier.
	PID int
}

// Identity returns this process's DeploymentIdentity, resolving it on first call.
//
// Returns DeploymentIdentity which describes the running process.
func Identity() DeploymentIdentity { return cachedIdentity() }

// resolveIdentity performs the one-time detection behind Identity.
//
// Returns DeploymentIdentity which describes the running process.
func resolveIdentity() DeploymentIdentity {
	hostname, _ := os.Hostname()

	return DeploymentIdentity{
		StartedAt:      processStart,
		InstanceID:     resolveInstanceID(hostname),
		Hostname:       hostname,
		ServiceName:    firstEnv(serviceNameEnvKeys, defaultServiceName),
		ServiceVersion: resolveServiceVersion(),
		Environment:    os.Getenv("PIKO_ENVIRONMENT"),
		Region:         os.Getenv("AWS_REGION"),
		NodeName:       os.Getenv("NODE_NAME"),
		Runtime:        logger_domain.RuntimeLabel(),
		PID:            os.Getpid(),
	}
}

// resolveInstanceID returns the platform-assigned replica identifier, falling back to the
// hostname and then to a generated UUID when no platform supplies one.
//
// Takes hostname (string) which is the machine name, used when no platform variable is
// set.
//
// Returns string which identifies this process among its siblings.
func resolveInstanceID(hostname string) string {
	if id := firstEnv(identityEnvKeys, ""); id != "" {
		return id
	}

	if hostname != "" {
		return hostname
	}

	return uuid.NewString()
}

// resolveServiceVersion returns the running build's version.
//
// Returns string which is the version, "dev" when the binary carries no build info.
func resolveServiceVersion() string {
	return logger_domain.ServiceVersion(defaultServiceVersion)
}

// firstEnv returns the value of the first environment variable in keys that is set and
// non-empty.
//
// Takes keys ([]string) which are the variable names to try, in precedence order.
// Takes fallback (string) which is returned when none of them is set.
//
// Returns string which is the resolved value.
func firstEnv(keys []string, fallback string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return fallback
}
