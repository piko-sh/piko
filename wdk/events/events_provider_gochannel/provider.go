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

package events_provider_gochannel

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"piko.sh/piko/internal/events/events_domain"
	"piko.sh/piko/internal/goroutine"
	watermilllogger "piko.sh/piko/internal/logger/logger_adapters/integrations/watermill"
	"piko.sh/piko/wdk/logger"
)

const (
	// defaultOutputChannelBuffer is the default buffer size for subscriber output
	// channels.
	defaultOutputChannelBuffer = 8192

	// defaultCloseTimeoutSeconds is the default timeout in seconds for closing the
	// router.
	defaultCloseTimeoutSeconds = 30
)

// Config contains configuration options for the GoChannel provider.
type Config struct {
	// OutputChannelBuffer is the buffer size for subscriber output channels.
	// Larger buffers reduce blocking but use more memory; default is 1024.
	OutputChannelBuffer int64

	// BlockPublishUntilSubscriberAck controls back-pressure behaviour.
	//
	// When true (default), Publish blocks until all subscribers have acknowledged.
	// This guarantees no message loss but may reduce throughput. When false,
	// messages are delivered asynchronously and may be dropped if subscriber
	// channels are full.
	//
	// Unlike the old MemoryEventBus which silently dropped events, the default
	// here blocks to guarantee delivery.
	BlockPublishUntilSubscriberAck bool

	// Persistent controls whether messages are persisted until
	// consumed, where true causes messages published before
	// subscriptions exist to be delivered when a subscription is
	// created via fan-out to all subscribers (default: false).
	Persistent bool

	// RouterConfig holds settings for the Watermill message router.
	RouterConfig events_domain.RouterConfig
}

var _ events_domain.Provider = (*GoChannelProvider)(nil)

// GoChannelProvider provides in-memory message passing using Watermill's
// GoChannel pub/sub. It implements the events_domain.Provider interface.
type GoChannelProvider struct {
	// logger adapts logging for the Watermill router.
	logger watermill.LoggerAdapter

	// pikoLogger handles internal logging for provider state changes and errors.
	pikoLogger logger.Logger

	// ctx controls the lifetime of the router goroutine.
	ctx context.Context

	// pubsub is the GoChannel instance that implements both Publisher and
	// Subscriber.
	pubsub *gochannel.GoChannel

	// router is the Watermill message router that handles pub/sub events.
	router *message.Router

	// cancel stops the router context when Close is called.
	cancel context.CancelCauseFunc

	// config holds the GoChannel provider settings.
	config Config

	// runningMutex guards access to the running field.
	runningMutex sync.RWMutex

	// running indicates whether the router is currently active.
	running bool
}

// NewGoChannelProvider creates a new GoChannel provider with the given
// configuration. Call Start() to set up the router before use.
//
// Takes config (Config) which specifies the channel buffer size, persistence,
// and acknowledgement behaviour.
//
// Returns *GoChannelProvider which is ready to use after calling Start().
// Returns error when the provider cannot be created.
func NewGoChannelProvider(config Config) (*GoChannelProvider, error) {
	pikoLogger := logger.GetLogger("piko/wdk/events/events_provider_gochannel")

	wmLogger := watermilllogger.NewAdapter(pikoLogger)

	pubsub := gochannel.NewGoChannel(gochannel.Config{
		OutputChannelBuffer:            config.OutputChannelBuffer,
		Persistent:                     config.Persistent,
		BlockPublishUntilSubscriberAck: config.BlockPublishUntilSubscriberAck,
	}, wmLogger)

	return &GoChannelProvider{
		config:     config,
		pubsub:     pubsub,
		logger:     wmLogger,
		pikoLogger: pikoLogger,
	}, nil
}

// Start initialises the Watermill router and starts processing messages.
// The provided context controls the router's lifecycle.
//
// Returns error when router initialisation fails.
//
// Safe for concurrent use. Spawns a goroutine for the router which runs until
// Stop is called.
func (p *GoChannelProvider) Start(ctx context.Context) error {
	startTime := time.Now()

	p.runningMutex.Lock()
	defer p.runningMutex.Unlock()

	if p.running {
		p.pikoLogger.Internal("GoChannel provider already running")
		return nil
	}

	p.pikoLogger.Internal("Starting GoChannel provider",
		logger.Int64("outputChannelBuffer", p.config.OutputChannelBuffer),
		logger.Bool("blockPublishUntilSubscriberAck", p.config.BlockPublishUntilSubscriberAck),
		logger.Bool("persistent", p.config.Persistent))

	if err := p.initialiseRouter(ctx); err != nil {
		return fmt.Errorf("initialising GoChannel router: %w", err)
	}

	p.ctx, p.cancel = context.WithCancelCause(ctx)
	p.startRouterGoroutine()
	<-p.router.Running()
	p.running = true

	p.recordStartMetrics(ctx, startTime)
	return nil
}

// Router returns the Watermill Router for adding handlers.
//
// Returns *message.Router which is the router instance for registering
// message handlers.
func (p *GoChannelProvider) Router() *message.Router {
	return p.router
}

// Publisher returns the Watermill Publisher for sending messages.
// For GoChannel, the pub/sub instance implements both Publisher and Subscriber.
//
// Returns message.Publisher which is the publisher for sending messages.
func (p *GoChannelProvider) Publisher() message.Publisher {
	return p.pubsub
}

// Subscriber returns the Watermill Subscriber for subscribing to topics.
// For GoChannel, the pub/sub instance implements both Publisher and Subscriber.
//
// Returns message.Subscriber which provides topic subscription capabilities.
func (p *GoChannelProvider) Subscriber() message.Subscriber {
	return p.pubsub
}

// Running reports whether the router has been started and is active.
//
// Returns bool which indicates whether the router is currently active.
//
// Safe for concurrent use.
func (p *GoChannelProvider) Running() bool {
	p.runningMutex.RLock()
	defer p.runningMutex.RUnlock()
	return p.running
}

// Close shuts down the provider and releases its resources.
//
// Returns error when the provider cannot be closed cleanly.
//
// Safe for concurrent use. Uses a mutex to guard against multiple close calls.
func (p *GoChannelProvider) Close() error {
	ctx := context.Background()
	startTime := time.Now()

	p.runningMutex.Lock()
	if !p.running {
		p.runningMutex.Unlock()
		p.pikoLogger.Internal("GoChannel provider already closed")
		return nil
	}
	p.running = false
	p.runningMutex.Unlock()

	p.pikoLogger.Internal("Closing GoChannel provider")

	if p.cancel != nil {
		p.cancel(errors.New("goChannel event provider closed"))
	}

	if p.router != nil {
		if err := p.router.Close(); err != nil {
			p.pikoLogger.Error("Error closing Watermill router", logger.Error(err))
			providerCloseErrorCount.Add(ctx, 1)
		}
	}

	if err := p.pubsub.Close(); err != nil {
		p.pikoLogger.Error("Error closing GoChannel pub/sub", logger.Error(err))
		providerCloseErrorCount.Add(ctx, 1)
	}

	duration := time.Since(startTime)
	providerCloseDuration.Record(ctx, float64(duration.Milliseconds()))
	providerCloseCount.Add(ctx, 1)

	p.pikoLogger.Internal("GoChannel provider closed",
		logger.Int64("durationMs", duration.Milliseconds()))

	return nil
}

// initialiseRouter creates and sets up the Watermill router.
//
// Returns error when the router cannot be created.
func (p *GoChannelProvider) initialiseRouter(ctx context.Context) error {
	routerConfig := message.RouterConfig{
		CloseTimeout: time.Duration(p.config.RouterConfig.CloseTimeout) * time.Second,
	}

	router, err := message.NewRouter(routerConfig, p.logger)
	if err != nil {
		p.pikoLogger.Error("Failed to create Watermill router", logger.Error(err))
		providerStartErrorCount.Add(ctx, 1)
		return fmt.Errorf("creating watermill router: %w", err)
	}

	p.router = router
	return nil
}

// startRouterGoroutine starts the router in a background goroutine.
//
// The spawned goroutine runs until the context is cancelled or the router
// stops. It updates the running state when the router exits.
//
// Concurrency: Safe for concurrent use. The spawned goroutine runs until the
// context is cancelled.
func (p *GoChannelProvider) startRouterGoroutine() {
	go func() {
		defer goroutine.RecoverPanic(p.ctx, "events.goChannelRouter")

		p.pikoLogger.Internal("Watermill router starting")
		if err := p.router.Run(p.ctx); err != nil {
			if p.ctx.Err() == nil {
				p.pikoLogger.Error("Watermill router exited with error", logger.Error(err))
			}
		}
		p.pikoLogger.Internal("Watermill router stopped")

		p.runningMutex.Lock()
		p.running = false
		p.runningMutex.Unlock()
	}()
}

// recordStartMetrics records metrics for provider startup.
//
// Takes startTime (time.Time) which is the time when startup began.
func (p *GoChannelProvider) recordStartMetrics(ctx context.Context, startTime time.Time) {
	duration := time.Since(startTime)
	providerStartDuration.Record(ctx, float64(duration.Milliseconds()))
	providerStartCount.Add(ctx, 1)

	p.pikoLogger.Internal("GoChannel provider started",
		logger.Int64("durationMs", duration.Milliseconds()))
}

// DefaultConfig returns sensible defaults for production use.
//
// BlockPublishUntilSubscriberAck is disabled to allow fire-and-forget
// publishing. This supports high-throughput event streams without blocking.
// Large output buffers and idempotent event handlers (via singleflight)
// prevent message loss.
//
// Returns Config which contains the default settings for the event system.
func DefaultConfig() Config {
	return Config{
		OutputChannelBuffer:            defaultOutputChannelBuffer,
		BlockPublishUntilSubscriberAck: false,
		Persistent:                     false,
		RouterConfig: events_domain.RouterConfig{
			CloseTimeout: defaultCloseTimeoutSeconds,
		},
	}
}
