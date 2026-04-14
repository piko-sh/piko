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

package events_provider_gcp_pubsub

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-googlecloud/v2/pkg/googlecloud"
	"github.com/ThreeDotsLabs/watermill/message"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/wdk/events"
	"piko.sh/piko/wdk/logger"
)

var _ events.Provider = (*GCPPubSubProvider)(nil)

const (
	// defaultTimeoutSeconds is the default timeout in seconds for various
	// GCP Pub/Sub operations.
	defaultTimeoutSeconds = 30

	// defaultSubscriptionPrefix is the default prefix prepended to topic names
	// when generating subscription names.
	defaultSubscriptionPrefix = "piko"
)

// Config contains configuration options for the GCP Pub/Sub provider.
type Config struct {
	// ProjectID is the Google Cloud project ID. Required.
	ProjectID string

	// SubscriptionPrefix is prepended to topic names to form subscription
	// names. Default: "piko".
	SubscriptionPrefix string

	// EmulatorHost overrides the Pub/Sub endpoint for local emulator testing.
	// When set, the PUBSUB_EMULATOR_HOST env var is configured automatically
	// (scoped to client creation, then restored).
	EmulatorHost string

	// AutoCreateTopics creates topics automatically if they don't exist.
	// Default: true.
	AutoCreateTopics bool

	// AutoCreateSubscriptions creates subscriptions automatically.
	// Default: true.
	AutoCreateSubscriptions bool

	// CloseTimeout is how long to wait for clean shutdown. Default 30s.
	CloseTimeout time.Duration

	// RouterConfig holds Watermill router settings.
	RouterConfig events.RouterConfig
}

// GCPPubSubProvider implements the events.Provider interface using Google
// Cloud Pub/Sub for message passing.
type GCPPubSubProvider struct {
	// publisher handles sending messages to Pub/Sub topics.
	publisher message.Publisher

	// subscriber handles message consumption from Pub/Sub using Watermill.
	subscriber message.Subscriber

	// logger provides logging for the router, publisher, and subscriber.
	logger watermill.LoggerAdapter

	// ctx controls the lifetime of the router goroutine.
	ctx context.Context

	// router is the Watermill message router that manages Pub/Sub subscriptions.
	router *message.Router

	// cancel stops the router goroutine when Close is called.
	cancel context.CancelCauseFunc

	// config holds the GCP Pub/Sub settings such as project ID and subscription
	// prefix.
	config Config

	// runningMutex guards access to the running field.
	runningMutex sync.RWMutex

	// running indicates whether the GCP Pub/Sub provider is currently active.
	running bool
}

// NewGCPPubSubProvider creates a new GCP Pub/Sub provider with the given
// configuration. Call Start to initialise the connection and router before use.
//
// Takes config (Config) which specifies the GCP Pub/Sub connection settings.
//
// Returns *GCPPubSubProvider which is the configured provider ready for starting.
// Returns error when the configuration is not valid.
func NewGCPPubSubProvider(config Config) (*GCPPubSubProvider, error) {
	if config.ProjectID == "" {
		return nil, errors.New("GCP Pub/Sub project ID is required")
	}

	wmLogger := events.NewWatermillLoggerAdapter(log)

	return &GCPPubSubProvider{
		config: config,
		logger: wmLogger,
	}, nil
}

// Start initialises the GCP Pub/Sub publisher/subscriber and starts the
// Watermill router.
//
// Returns error when pub/sub initialisation or router setup fails.
//
// Safe for concurrent use. Returns nil immediately if already running. Spawns
// a background goroutine for the router that runs until Close is called.
func (p *GCPPubSubProvider) Start(ctx context.Context) error {
	ctx, l := logger.From(ctx, log)

	startTime := time.Now()

	p.runningMutex.Lock()
	defer p.runningMutex.Unlock()

	if p.running {
		l.Internal("GCP Pub/Sub provider already running")
		return nil
	}

	p.logStarting(ctx)
	providerConnectionAttempts.Add(ctx, 1)

	if err := p.initialisePubSub(ctx); err != nil {
		return fmt.Errorf("initialising GCP Pub/Sub pub/sub: %w", err)
	}

	if err := p.initialiseRouter(ctx); err != nil {
		return fmt.Errorf("initialising GCP Pub/Sub router: %w", err)
	}

	p.ctx, p.cancel = context.WithCancelCause(ctx)
	p.startRouterGoroutine(ctx)
	<-p.router.Running()
	p.running = true

	p.recordStartMetrics(ctx, startTime)
	return nil
}

// Router returns the Watermill Router for adding handlers.
//
// Returns *message.Router which manages message routing and handler execution.
func (p *GCPPubSubProvider) Router() *message.Router {
	return p.router
}

// Publisher returns the Watermill Publisher for publishing messages.
//
// Returns message.Publisher which handles message publishing operations.
func (p *GCPPubSubProvider) Publisher() message.Publisher {
	return p.publisher
}

// Subscriber returns the Watermill Subscriber for subscribing to topics.
//
// Returns message.Subscriber which handles message consumption from GCP
// Pub/Sub.
func (p *GCPPubSubProvider) Subscriber() message.Subscriber {
	return p.subscriber
}

// Running returns true if the router has been started and is running.
//
// Returns bool which indicates whether the router is currently active.
//
// Safe for concurrent use.
func (p *GCPPubSubProvider) Running() bool {
	p.runningMutex.RLock()
	defer p.runningMutex.RUnlock()
	return p.running
}

// Close shuts down the provider and releases all resources.
//
// Returns error when the shutdown fails, though the current version always
// returns nil.
//
// Safe for concurrent use. Logs errors for individual component failures but
// does not return them.
func (p *GCPPubSubProvider) Close() error {
	ctx := context.Background()
	_, l := logger.From(ctx, log)
	startTime := time.Now()

	p.runningMutex.Lock()
	if !p.running {
		p.runningMutex.Unlock()
		l.Internal("GCP Pub/Sub provider already closed")
		return nil
	}
	p.running = false
	p.runningMutex.Unlock()

	l.Internal("Closing GCP Pub/Sub provider")

	if p.cancel != nil {
		p.cancel(errors.New("GCP Pub/Sub event provider closed"))
	}

	if p.router != nil {
		if err := p.router.Close(); err != nil {
			l.Error("Error closing Watermill router", logger.Error(err))
			providerCloseErrorCount.Add(ctx, 1)
		}
	}

	if p.subscriber != nil {
		if err := p.subscriber.Close(); err != nil {
			l.Error("Error closing GCP Pub/Sub subscriber", logger.Error(err))
			providerCloseErrorCount.Add(ctx, 1)
		}
	}

	if p.publisher != nil {
		if err := p.publisher.Close(); err != nil {
			l.Error("Error closing GCP Pub/Sub publisher", logger.Error(err))
			providerCloseErrorCount.Add(ctx, 1)
		}
	}

	duration := time.Since(startTime)
	providerCloseDuration.Record(ctx, float64(duration.Milliseconds()))
	providerCloseCount.Add(ctx, 1)

	l.Internal("GCP Pub/Sub provider closed",
		logger.Int64("durationMs", duration.Milliseconds()))

	return nil
}

// logStarting logs that the provider is starting with its settings.
func (p *GCPPubSubProvider) logStarting(ctx context.Context) {
	_, l := logger.From(ctx, log)
	l.Internal("Starting GCP Pub/Sub provider",
		logger.String("projectID", p.config.ProjectID),
		logger.String("subscriptionPrefix", p.config.SubscriptionPrefix),
		logger.Bool("autoCreateTopics", p.config.AutoCreateTopics),
		logger.Bool("autoCreateSubscriptions", p.config.AutoCreateSubscriptions))
}

// initialisePubSub creates the publisher and subscriber for GCP Pub/Sub.
//
// Returns error when the publisher or subscriber cannot be created.
func (p *GCPPubSubProvider) initialisePubSub(ctx context.Context) error {
	ctx, l := logger.From(ctx, log)

	if p.config.EmulatorHost != "" {
		previousHost := os.Getenv("PUBSUB_EMULATOR_HOST")
		if err := os.Setenv("PUBSUB_EMULATOR_HOST", p.config.EmulatorHost); err != nil {
			return fmt.Errorf("failed to set PUBSUB_EMULATOR_HOST: %w", err)
		}
		defer func() {
			if previousHost == "" {
				_ = os.Unsetenv("PUBSUB_EMULATOR_HOST")
			} else {
				_ = os.Setenv("PUBSUB_EMULATOR_HOST", previousHost)
			}
		}()
	}

	publisher, err := p.createPublisher()
	if err != nil {
		l.Error("Failed to create GCP Pub/Sub publisher", logger.Error(err))
		providerStartErrorCount.Add(ctx, 1)
		return fmt.Errorf("creating GCP Pub/Sub publisher: %w", err)
	}
	p.publisher = publisher

	subscriber, err := p.createSubscriber()
	if err != nil {
		_ = p.publisher.Close()
		l.Error("Failed to create GCP Pub/Sub subscriber", logger.Error(err))
		providerStartErrorCount.Add(ctx, 1)
		return fmt.Errorf("creating GCP Pub/Sub subscriber: %w", err)
	}
	p.subscriber = subscriber

	return nil
}

// initialiseRouter creates and sets up the Watermill router.
//
// Returns error when router creation fails.
func (p *GCPPubSubProvider) initialiseRouter(ctx context.Context) error {
	_, l := logger.From(ctx, log)

	routerConfig := message.RouterConfig{
		CloseTimeout: time.Duration(p.config.RouterConfig.CloseTimeout) * time.Second,
	}

	router, err := message.NewRouter(routerConfig, p.logger)
	if err != nil {
		_ = p.subscriber.Close()
		_ = p.publisher.Close()
		l.Error("Failed to create Watermill router", logger.Error(err))
		providerStartErrorCount.Add(ctx, 1)
		return fmt.Errorf("creating watermill router: %w", err)
	}
	p.router = router
	return nil
}

// startRouterGoroutine starts the router in a background goroutine.
//
// The goroutine runs until the router stops or returns an error. It updates
// the running state when the router stops.
//
// Safe for concurrent use. Uses a mutex to update the running state.
func (p *GCPPubSubProvider) startRouterGoroutine(ctx context.Context) {
	_, l := logger.From(ctx, log)
	go func() {
		defer goroutine.RecoverPanic(p.ctx, "events.gcpPubSubRouter")

		l.Internal("Watermill router starting")
		if err := p.router.Run(p.ctx); err != nil {
			if p.ctx.Err() == nil {
				l.Error("Watermill router exited with error", logger.Error(err))
			}
		}
		l.Internal("Watermill router stopped")

		p.runningMutex.Lock()
		p.running = false
		p.runningMutex.Unlock()
	}()
}

// recordStartMetrics records metrics for provider startup.
//
// Takes startTime (time.Time) which marks when the startup began.
func (p *GCPPubSubProvider) recordStartMetrics(ctx context.Context, startTime time.Time) {
	ctx, l := logger.From(ctx, log)

	duration := time.Since(startTime)
	providerStartDuration.Record(ctx, float64(duration.Milliseconds()))
	providerStartCount.Add(ctx, 1)

	l.Internal("GCP Pub/Sub provider started",
		logger.Int64("durationMs", duration.Milliseconds()),
		logger.String("projectID", p.config.ProjectID))
}

// createPublisher creates a GCP Pub/Sub publisher using the provider settings.
//
// Returns message.Publisher which is the configured publisher ready for use.
// Returns error when the publisher cannot be created.
func (p *GCPPubSubProvider) createPublisher() (message.Publisher, error) {
	publisherConfig := googlecloud.PublisherConfig{
		ProjectID:                 p.config.ProjectID,
		DoNotCreateTopicIfMissing: !p.config.AutoCreateTopics,
	}

	return googlecloud.NewPublisher(publisherConfig, p.logger)
}

// createSubscriber creates a GCP Pub/Sub subscriber based on the provider
// settings.
//
// Returns message.Subscriber which handles incoming messages from GCP Pub/Sub.
// Returns error when the subscriber cannot be created.
func (p *GCPPubSubProvider) createSubscriber() (message.Subscriber, error) {
	subscriberConfig := googlecloud.SubscriberConfig{
		ProjectID:                        p.config.ProjectID,
		GenerateSubscriptionName:         generateSubscriptionName(p.config.SubscriptionPrefix),
		DoNotCreateSubscriptionIfMissing: !p.config.AutoCreateSubscriptions,
		DoNotCreateTopicIfMissing:        !p.config.AutoCreateTopics,
	}

	return googlecloud.NewSubscriber(subscriberConfig, p.logger)
}

// generateSubscriptionName returns a function that generates subscription names
// by prepending the given prefix to the topic name.
//
// Takes prefix (string) which is prepended to topic names with an underscore
// separator.
//
// Returns googlecloud.SubscriptionNameFn which generates "{prefix}_{topic}"
// names.
func generateSubscriptionName(prefix string) googlecloud.SubscriptionNameFn {
	return func(topic string) string {
		return prefix + "_" + topic
	}
}

// DefaultConfig returns settings suited for production use with Google Cloud
// Pub/Sub. Auto-creation of topics and subscriptions is enabled for simple
// deployment.
//
// Returns Config which contains the default settings ready for use.
func DefaultConfig() Config {
	return Config{
		SubscriptionPrefix:      defaultSubscriptionPrefix,
		AutoCreateTopics:        true,
		AutoCreateSubscriptions: true,
		CloseTimeout:            defaultTimeoutSeconds * time.Second,
		RouterConfig: events.RouterConfig{
			CloseTimeout: defaultTimeoutSeconds,
		},
	}
}
