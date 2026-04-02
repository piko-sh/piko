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

package piko

import (
	"context"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

// LifecycleComponent defines the contract for components that require
// lifecycle management.
//
// Components implementing this interface have their OnStart method called
// during server startup (before accepting HTTP traffic) and their OnStop
// method called during graceful shutdown.
//
// Lifecycle components are executed in the order they are registered. During
// shutdown, they are stopped in reverse order (last registered, first
// stopped).
//
// Example implementation:
//
//	type MyComponent struct {
//	    db *sql.DB
//	}
//
//	func (c *MyComponent) OnStart(ctx context.Context) error {
//	    var err error
//	    c.db, err = sql.Open("postgres", connString)
//	    if err != nil {
//	        return fmt.Errorf("failed to connect to database: %w", err)
//	    }
//	    return nil
//	}
//
//	func (c *MyComponent) OnStop(ctx context.Context) error {
//	    if c.db != nil {
//	        return c.db.Close()
//	    }
//	    return nil
//	}
//
//	func (c *MyComponent) Name() string {
//	    return "MyDatabaseComponent"
//	}
type LifecycleComponent interface {
	// OnStart runs when the server starts up.
	// Use this to set up resources, open connections, run migrations, and so on.
	//
	// Returns error to stop the server from starting.
	//
	// The context has a timeout. If OnStart does not finish in time, the server
	// will fail to start.
	OnStart(ctx context.Context) error

	// OnStop is called when the server is shutting down gracefully.
	// Use this to close connections, flush caches, save state, etc.
	//
	// The context will have a timeout applied. If OnStop does not complete
	// within the timeout, the server will forcefully terminate.
	//
	// Return an error to log the issue, but shutdown will continue.
	OnStop(ctx context.Context) error

	// Name returns a readable name for this component, used for logging and
	// identification.
	//
	// Returns string which is the component name.
	Name() string
}

// LifecycleStartTimeout is an optional interface that lifecycle components can
// implement to override the default 30-second startup timeout. Components
// performing long-running initialisation (such as document ingestion or
// database migrations) should implement this to request more time.
//
// Example:
//
//	func (c *MyComponent) StartTimeout() time.Duration {
//	    return 5 * time.Minute
//	}
type LifecycleStartTimeout interface {
	// StartTimeout returns the timeout duration for starting the server.
	//
	// Returns time.Duration which is the maximum time allowed for startup.
	StartTimeout() time.Duration
}

// LifecycleHealthProbe provides health monitoring for components. It is an
// alias to the internal healthprobe_domain.Probe interface, allowing external
// packages to implement health probes without importing internal packages.
//
// Probes are called periodically by the health monitoring system and exposed
// via HTTP endpoints. Components can implement both LifecycleComponent and
// this interface to get both lifecycle management and health monitoring.
//
// Example implementation:
//
//	func (c *MyComponent) Check(ctx context.Context, checkType piko.HealthCheckType) piko.HealthStatus {
//	    // For liveness: check if component is alive (not deadlocked)
//	    if checkType == piko.HealthCheckLiveness {
//	        return piko.HealthStatus{
//	            Name:  c.Name(),
//	            State: piko.HealthStateHealthy,
//	            Timestamp: time.Now(),
//	        }
//	    }
//	    // For readiness: check if component is ready to serve traffic
//	    if err := c.db.PingContext(ctx); err != nil {
//	        return piko.HealthStatus{
//	            Name:    c.Name(),
//	            State:   piko.HealthStateUnhealthy,
//	            Message: fmt.Sprintf("database unreachable: %v", err),
//	            Timestamp: time.Now(),
//	        }
//	    }
//	    return piko.HealthStatus{
//	        Name:  c.Name(),
//	        State: piko.HealthStateHealthy,
//	        Timestamp: time.Now(),
//	    }
//	}
type LifecycleHealthProbe interface {
	healthprobe_domain.Probe
}

// LifecycleWithHealth combines lifecycle management and health probing for
// components that need both managed startup/shutdown and runtime health checks.
//
// When you register a component that implements LifecycleWithHealth:
//  1. OnStart is called during server startup.
//  2. The component is automatically registered as a health probe.
//  3. Check is called periodically for /health, /live, and /ready endpoints.
//  4. OnStop is called during server shutdown.
type LifecycleWithHealth interface {
	LifecycleComponent

	LifecycleHealthProbe
}

// lifecycleProbeAdapter adapts a LifecycleComponent to the internal Probe
// interface. If the component implements LifecycleHealthProbe, its Check method
// is used directly; otherwise a basic "component exists" check is provided.
type lifecycleProbeAdapter struct {
	// component is the lifecycle component being adapted as a health probe.
	component LifecycleComponent

	// probe is the health probe for the component; nil if the component does not
	// implement LifecycleHealthProbe.
	probe LifecycleHealthProbe
}

// Name returns the probe's display name, implementing
// healthprobe_domain.Probe.
//
// Returns string which is the name of the underlying component.
func (a *lifecycleProbeAdapter) Name() string {
	return a.component.Name()
}

// Check implements healthprobe_domain.Probe by returning the health status of
// the adapted component.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies whether to
// perform a liveness or readiness check.
//
// Returns healthprobe_dto.Status which indicates the component health state.
func (a *lifecycleProbeAdapter) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	if a.probe != nil {
		return a.probe.Check(ctx, checkType)
	}

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return healthprobe_dto.Status{
			Name:      a.component.Name(),
			State:     healthprobe_dto.StateHealthy,
			Message:   "Component is running (no custom health check provided)",
			Timestamp: time.Now(),
			Duration:  "0s",
		}
	}

	return healthprobe_dto.Status{
		Name:      a.component.Name(),
		State:     healthprobe_dto.StateDegraded,
		Message:   "Component does not provide readiness check",
		Timestamp: time.Now(),
		Duration:  "0s",
	}
}
