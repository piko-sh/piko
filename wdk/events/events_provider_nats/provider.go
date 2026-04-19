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
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/wdk/events"
	"piko.sh/piko/wdk/logger"
)

var _ events.Provider = (*NATSProvider)(nil)

const (
	// defaultTimeoutSeconds is the default timeout in seconds for various NATS
	// operations.
	defaultTimeoutSeconds = 30
)

// Config contains configuration options for the NATS JetStream provider.
type Config struct {
	// URL is the NATS server connection address.
	// Default: "nats://localhost:4222".
	URL string

	// ClusterID identifies this client in logs and monitoring.
	// Default: "piko-events".
	ClusterID string

	// QueueGroupPrefix is prepended to topic names to form queue group names,
	// enabling competing consumer patterns where multiple subscribers share work
	// on the same topic. Default: "piko".
	QueueGroupPrefix string

	// NATSOptions provides extra NATS connection options.
	// These are added after the default options.
	NATSOptions []nc.Option

	// JetStream holds settings for NATS JetStream messaging.
	JetStream JetStreamConfig

	// SubscribersCount is the number of concurrent subscribers
	// per topic, where higher values increase throughput but
	// use more resources (default: 1).
	SubscribersCount int

	// AckWaitTimeout is how long to wait for message acknowledgement before
	// redelivery. Default is 30 seconds.
	AckWaitTimeout time.Duration

	// CloseTimeout is how long to wait for a clean shutdown. Default is 30
	// seconds.
	CloseTimeout time.Duration

	// RouterConfig holds settings for the Watermill message router.
	RouterConfig events.RouterConfig
}

// JetStreamConfig contains JetStream-specific configuration options.
type JetStreamConfig struct {
	// DurablePrefix is prepended to subscription names to create
	// durable subscriptions that survive client disconnections
	// and server restarts (default: "piko").
	DurablePrefix string

	// SubscribeOptions holds extra options passed to JetStream when subscribing.
	SubscribeOptions []nc.SubOpt

	// PublishOptions specifies extra settings for JetStream publish operations.
	PublishOptions []nc.PubOpt

	// Disabled disables JetStream and uses core NATS instead, which provides
	// at-most-once delivery without persistence; default is false.
	Disabled bool

	// AutoProvision automatically creates streams if they do not
	// exist, which is useful for development but should be
	// disabled in production where streams are pre-created with
	// specific configurations (default: true).
	AutoProvision bool

	// TrackMessageID enables message ID tracking for exactly-once delivery.
	// This has a performance cost but prevents duplicates; default is false.
	TrackMessageID bool

	// AckAsync enables asynchronous acknowledgement. When true, Ack() returns
	// immediately without waiting for server confirmation, improving performance
	// at the cost of delivery guarantees; default is false.
	AckAsync bool
}

// NATSProvider implements the events.Provider interface using NATS
// JetStream for message passing with storage.
type NATSProvider struct {
	// publisher handles sending messages to NATS topics.
	publisher message.Publisher

	// subscriber handles message consumption from NATS using Watermill.
	subscriber message.Subscriber

	// logger provides logging for the router, publisher, and subscriber.
	logger watermill.LoggerAdapter

	// ctx controls the lifetime of the router goroutine.
	ctx context.Context

	// conn holds the NATS connection for publishing and subscribing.
	conn *nc.Conn

	// router is the Watermill message router that manages NATS subscriptions.
	router *message.Router

	// cancel stops the router goroutine when Close is called.
	cancel context.CancelCauseFunc

	// config holds the NATS connection settings such as URL and cluster ID.
	config Config

	// runningMutex guards access to the running field.
	runningMutex sync.RWMutex

	// running indicates whether the NATS provider is currently active.
	running bool
}

// NewNATSProvider creates a new NATS JetStream provider with the given
// configuration. Call Start to set up the connection and router before use.
//
// Takes config (Config) which specifies the NATS connection and JetStream
// settings.
//
// Returns *NATSProvider which is the configured provider ready for starting.
// Returns error when the configuration is not valid.
func NewNATSProvider(config Config) (*NATSProvider, error) {
	wmLogger := events.NewWatermillLoggerAdapter(log)

	return &NATSProvider{
		config: config,
		logger: wmLogger,
	}, nil
}

// Start initialises the NATS connection, creates publisher/subscriber,
// and starts the Watermill router.
//
// Returns error when connection, pub/sub initialisation, or router setup fails.
//
// Safe for concurrent use. Returns nil immediately if already running. Spawns
// a background goroutine for the router that runs until Stop is called.
func (p *NATSProvider) Start(ctx context.Context) error {
	ctx, l := logger.From(ctx, log)

	startTime := time.Now()

	p.runningMutex.Lock()
	defer p.runningMutex.Unlock()

	if p.running {
		l.Internal("NATS provider already running")
		return nil
	}

	p.logStarting(ctx)
	providerConnectionAttempts.Add(ctx, 1)

	if err := p.connect(ctx); err != nil {
		return fmt.Errorf("connecting to NATS: %w", err)
	}

	if err := p.initialisePubSub(ctx); err != nil {
		return fmt.Errorf("initialising NATS pub/sub: %w", err)
	}

	if err := p.initialiseRouter(ctx); err != nil {
		return fmt.Errorf("initialising NATS router: %w", err)
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
func (p *NATSProvider) Router() *message.Router {
	return p.router
}

// Publisher returns the Watermill Publisher for publishing messages.
//
// Returns message.Publisher which handles message publishing operations.
func (p *NATSProvider) Publisher() message.Publisher {
	return p.publisher
}

// Subscriber returns the Watermill Subscriber for subscribing to topics.
//
// Returns message.Subscriber which handles message consumption from NATS.
func (p *NATSProvider) Subscriber() message.Subscriber {
	return p.subscriber
}

// Running reports whether the router has been started and is active.
//
// Returns bool which indicates whether the router is currently active.
//
// Safe for concurrent use.
func (p *NATSProvider) Running() bool {
	p.runningMutex.RLock()
	defer p.runningMutex.RUnlock()
	return p.running
}

// Connection returns the underlying NATS connection.
// This can be used for advanced operations or health checks.
//
// Returns *nc.Conn which is the active NATS connection.
func (p *NATSProvider) Connection() *nc.Conn {
	return p.conn
}

// Close shuts down the provider and releases all resources.
//
// Returns error when the shutdown fails, though the current version always
// returns nil.
//
// Safe for concurrent use. Logs errors for individual component failures but
// does not return them.
func (p *NATSProvider) Close() error {
	ctx := context.Background()
	_, l := logger.From(ctx, log)
	startTime := time.Now()

	p.runningMutex.Lock()
	if !p.running {
		p.runningMutex.Unlock()
		l.Internal("NATS provider already closed")
		return nil
	}
	p.running = false
	p.runningMutex.Unlock()

	l.Internal("Closing NATS provider")

	if p.cancel != nil {
		p.cancel(errors.New("NATS event provider closed"))
	}

	if p.router != nil {
		if err := p.router.Close(); err != nil {
			l.Error("Error closing Watermill router", logger.Error(err))
			providerCloseErrorCount.Add(ctx, 1)
		}
	}

	if p.subscriber != nil {
		if err := p.subscriber.Close(); err != nil {
			l.Error("Error closing NATS subscriber", logger.Error(err))
			providerCloseErrorCount.Add(ctx, 1)
		}
	}

	if p.publisher != nil {
		if err := p.publisher.Close(); err != nil {
			l.Error("Error closing NATS publisher", logger.Error(err))
			providerCloseErrorCount.Add(ctx, 1)
		}
	}

	if p.conn != nil {
		p.conn.Close()
	}

	duration := time.Since(startTime)
	providerCloseDuration.Record(ctx, float64(duration.Milliseconds()))
	providerCloseCount.Add(ctx, 1)

	l.Internal("NATS provider closed",
		logger.Int64("durationMs", duration.Milliseconds()))

	return nil
}

// logStarting logs that the provider is starting with its settings.
func (p *NATSProvider) logStarting(ctx context.Context) {
	_, l := logger.From(ctx, log)
	l.Internal("Starting NATS provider",
		logger.String("url", p.config.URL),
		logger.String("clusterID", p.config.ClusterID),
		logger.Bool("jetStreamEnabled", !p.config.JetStream.Disabled))
}

// connect establishes a connection to the NATS server.
//
// Returns error when the connection fails.
func (p *NATSProvider) connect(ctx context.Context) error {
	ctx, l := logger.From(ctx, log)

	natsOpts := p.buildNATSOptions(ctx)

	conn, err := nc.Connect(p.config.URL, natsOpts...)
	if err != nil {
		l.Error("Failed to connect to NATS", logger.Error(err))
		providerStartErrorCount.Add(ctx, 1)
		providerConnectionErrors.Add(ctx, 1)
		return fmt.Errorf("connecting to NATS at %s: %w", p.config.URL, err)
	}
	p.conn = conn

	l.Internal("Connected to NATS",
		logger.String("connectedURL", conn.ConnectedUrl()),
		logger.String("serverID", conn.ConnectedServerId()))
	return nil
}

// initialisePubSub creates the publisher and subscriber for the NATS connection.
//
// Returns error when the publisher or subscriber cannot be created.
func (p *NATSProvider) initialisePubSub(ctx context.Context) error {
	ctx, l := logger.From(ctx, log)

	publisher, err := p.createPublisher()
	if err != nil {
		p.conn.Close()
		l.Error("Failed to create NATS publisher", logger.Error(err))
		providerStartErrorCount.Add(ctx, 1)
		return fmt.Errorf("creating NATS publisher: %w", err)
	}
	p.publisher = publisher

	subscriber, err := p.createSubscriber()
	if err != nil {
		_ = p.publisher.Close()
		p.conn.Close()
		l.Error("Failed to create NATS subscriber", logger.Error(err))
		providerStartErrorCount.Add(ctx, 1)
		return fmt.Errorf("creating NATS subscriber: %w", err)
	}
	p.subscriber = subscriber

	return nil
}

// initialiseRouter creates and sets up the Watermill router.
//
// Returns error when router creation fails.
func (p *NATSProvider) initialiseRouter(ctx context.Context) error {
	_, l := logger.From(ctx, log)

	routerConfig := message.RouterConfig{
		CloseTimeout: time.Duration(p.config.RouterConfig.CloseTimeout) * time.Second,
	}

	router, err := message.NewRouter(routerConfig, p.logger)
	if err != nil {
		_ = p.subscriber.Close()
		_ = p.publisher.Close()
		p.conn.Close()
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
func (p *NATSProvider) startRouterGoroutine(ctx context.Context) {
	_, l := logger.From(ctx, log)
	go func() {
		defer goroutine.RecoverPanic(p.ctx, "events.natsRouter")

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
func (p *NATSProvider) recordStartMetrics(ctx context.Context, startTime time.Time) {
	ctx, l := logger.From(ctx, log)

	duration := time.Since(startTime)
	providerStartDuration.Record(ctx, float64(duration.Milliseconds()))
	providerStartCount.Add(ctx, 1)

	l.Internal("NATS provider started",
		logger.Int64("durationMs", duration.Milliseconds()),
		logger.String("connectedURL", p.conn.ConnectedUrl()))
}

// buildNATSOptions builds the NATS connection options.
//
// Returns []nc.Option which contains the connection options, including
// handlers for reconnection, disconnection, and errors.
func (p *NATSProvider) buildNATSOptions(ctx context.Context) []nc.Option {
	ctx, l := logger.From(ctx, log)

	opts := make([]nc.Option, 0, 6+len(p.config.NATSOptions))
	opts = append(opts,
		nc.Name(p.config.ClusterID),
		nc.ReconnectWait(2*time.Second),
		nc.MaxReconnects(-1),
		nc.ReconnectHandler(func(conn *nc.Conn) {
			l.Internal("Reconnected to NATS",
				logger.String("url", conn.ConnectedUrl()))
			providerReconnections.Add(ctx, 1)
		}),
		nc.DisconnectErrHandler(func(_ *nc.Conn, err error) {
			if err != nil {
				l.Warn("Disconnected from NATS",
					logger.Error(err))
				providerConnectionErrors.Add(ctx, 1)
			}
		}),
		nc.ErrorHandler(func(_ *nc.Conn, sub *nc.Subscription, err error) {
			l.Error("NATS error",
				logger.Error(err),
				logger.String("subject", sub.Subject))
			providerConnectionErrors.Add(ctx, 1)
		}),
	)

	opts = append(opts, p.config.NATSOptions...)

	return opts
}

// createPublisher creates a NATS publisher using the provider settings.
//
// Returns message.Publisher which is the configured publisher ready for use.
// Returns error when the publisher cannot be created.
func (p *NATSProvider) createPublisher() (message.Publisher, error) {
	marshaler := &nats.NATSMarshaler{}

	pubConfig := nats.PublisherConfig{
		URL:       p.config.URL,
		Marshaler: marshaler,
		NatsOptions: []nc.Option{
			nc.Name(p.config.ClusterID + "-publisher"),
		},
	}

	if !p.config.JetStream.Disabled {
		pubConfig.JetStream = nats.JetStreamConfig{
			Disabled:       false,
			AutoProvision:  p.config.JetStream.AutoProvision,
			TrackMsgId:     p.config.JetStream.TrackMessageID, //nolint:revive // external library naming
			PublishOptions: p.config.JetStream.PublishOptions,
		}
	} else {
		pubConfig.JetStream = nats.JetStreamConfig{
			Disabled: true,
		}
	}

	return nats.NewPublisher(pubConfig, p.logger)
}

// createSubscriber creates a NATS subscriber based on the provider settings.
//
// Returns message.Subscriber which handles incoming messages from NATS.
// Returns error when the subscriber cannot be created.
func (p *NATSProvider) createSubscriber() (message.Subscriber, error) {
	unmarshaler := &nats.NATSMarshaler{}

	subConfig := nats.SubscriberConfig{
		URL:              p.config.URL,
		QueueGroupPrefix: p.config.QueueGroupPrefix,
		SubscribersCount: p.config.SubscribersCount,
		CloseTimeout:     p.config.CloseTimeout,
		AckWaitTimeout:   p.config.AckWaitTimeout,
		Unmarshaler:      unmarshaler,
		NatsOptions: []nc.Option{
			nc.Name(p.config.ClusterID + "-subscriber"),
		},
	}

	if !p.config.JetStream.Disabled {
		subConfig.JetStream = nats.JetStreamConfig{
			Disabled:         false,
			AutoProvision:    p.config.JetStream.AutoProvision,
			DurablePrefix:    p.config.JetStream.DurablePrefix,
			AckAsync:         p.config.JetStream.AckAsync,
			SubscribeOptions: p.config.JetStream.SubscribeOptions,
		}
	} else {
		subConfig.JetStream = nats.JetStreamConfig{
			Disabled: true,
		}
	}

	return nats.NewSubscriber(subConfig, p.logger)
}

// DefaultConfig returns settings suited for production use with reliable
// message delivery. JetStream is enabled with auto-provisioning for simple
// deployment.
//
// Returns Config which contains the default settings ready for use.
func DefaultConfig() Config {
	return Config{
		URL:              nc.DefaultURL,
		ClusterID:        "piko-events",
		QueueGroupPrefix: "piko",
		SubscribersCount: 1,
		AckWaitTimeout:   defaultTimeoutSeconds * time.Second,
		CloseTimeout:     defaultTimeoutSeconds * time.Second,
		JetStream: JetStreamConfig{
			Disabled:       false,
			AutoProvision:  true,
			TrackMessageID: false,
			AckAsync:       false,
			DurablePrefix:  "piko",
		},
		RouterConfig: events.RouterConfig{
			CloseTimeout: defaultTimeoutSeconds,
		},
	}
}
