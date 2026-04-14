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

package events_provider_sqs

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-aws/sqs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	amazonsqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/wdk/events"
	"piko.sh/piko/wdk/logger"
)

var _ events.Provider = (*SQSProvider)(nil)

const (
	// defaultTimeoutSeconds is the default timeout in seconds for various SQS
	// operations.
	defaultTimeoutSeconds = 30

	// defaultRegion is the default AWS region for SQS.
	defaultRegion = "us-east-1"
)

// Config contains configuration options for the AWS SQS provider.
type Config struct {
	// Region is the AWS region for SQS (e.g., "us-east-1").
	Region string

	// EndpointURL overrides the SQS endpoint for LocalStack.
	EndpointURL string

	// AccessKey is the AWS access key ID for static credentials. Optional;
	// when empty, the default AWS credential chain is used.
	AccessKey string

	// SecretKey is the AWS secret access key for static credentials. Optional;
	// when empty, the default AWS credential chain is used.
	SecretKey string

	// QueueAutoCreate creates SQS queues automatically if they do not exist.
	// Default: true.
	QueueAutoCreate bool

	// CloseTimeout is how long to wait for clean shutdown. Default 30s.
	CloseTimeout time.Duration

	// RouterConfig holds Watermill router settings.
	RouterConfig events.RouterConfig
}

// SQSProvider implements the events.Provider interface using AWS SQS
// for managed message passing.
type SQSProvider struct {
	// publisher handles sending messages to SQS queues.
	publisher message.Publisher

	// subscriber handles message consumption from SQS using Watermill.
	subscriber message.Subscriber

	// logger provides logging for the router, publisher, and subscriber.
	logger watermill.LoggerAdapter

	// ctx controls the lifetime of the router goroutine.
	ctx context.Context

	// router is the Watermill message router that manages SQS subscriptions.
	router *message.Router

	// cancel stops the router goroutine when Close is called.
	cancel context.CancelCauseFunc

	// config holds the SQS connection settings such as region and endpoint.
	config Config

	// runningMutex guards access to the running field.
	runningMutex sync.RWMutex

	// running indicates whether the SQS provider is currently active.
	running bool
}

// NewSQSProvider creates a new AWS SQS provider with the given configuration.
// Call Start to set up the connection and router before use.
//
// Takes config (Config) which specifies the AWS SQS connection settings.
//
// Returns *SQSProvider which is the configured provider ready for starting.
// Returns error when the configuration is not valid.
func NewSQSProvider(config Config) (*SQSProvider, error) {
	wmLogger := events.NewWatermillLoggerAdapter(log)

	return &SQSProvider{
		config: config,
		logger: wmLogger,
	}, nil
}

// Start initialises the AWS SQS connection, creates publisher/subscriber,
// and starts the Watermill router.
//
// Returns error when connection, pub/sub initialisation, or router setup fails.
//
// Safe for concurrent use. Returns nil immediately if already running. Spawns
// a background goroutine for the router that runs until Close is called.
func (p *SQSProvider) Start(ctx context.Context) error {
	ctx, l := logger.From(ctx, log)

	startTime := time.Now()

	p.runningMutex.Lock()
	defer p.runningMutex.Unlock()

	if p.running {
		l.Internal("SQS provider already running")
		return nil
	}

	p.logStarting(ctx)
	providerConnectionAttempts.Add(ctx, 1)

	awsCfg, err := p.loadAWSConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading AWS config: %w", err)
	}

	if err := p.initialisePubSub(ctx, &awsCfg); err != nil {
		return fmt.Errorf("initialising SQS pub/sub: %w", err)
	}

	if err := p.initialiseRouter(ctx); err != nil {
		return fmt.Errorf("initialising SQS router: %w", err)
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
func (p *SQSProvider) Router() *message.Router {
	return p.router
}

// Publisher returns the Watermill Publisher for publishing messages.
//
// Returns message.Publisher which handles message publishing operations.
func (p *SQSProvider) Publisher() message.Publisher {
	return p.publisher
}

// Subscriber returns the Watermill Subscriber for subscribing to topics.
//
// Returns message.Subscriber which handles message consumption from SQS.
func (p *SQSProvider) Subscriber() message.Subscriber {
	return p.subscriber
}

// Running returns true if the router has been started and is running.
//
// Returns bool which indicates whether the router is currently active.
//
// Safe for concurrent use.
func (p *SQSProvider) Running() bool {
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
func (p *SQSProvider) Close() error {
	ctx := context.Background()
	_, l := logger.From(ctx, log)
	startTime := time.Now()

	p.runningMutex.Lock()
	if !p.running {
		p.runningMutex.Unlock()
		l.Internal("SQS provider already closed")
		return nil
	}
	p.running = false
	p.runningMutex.Unlock()

	l.Internal("Closing SQS provider")

	if p.cancel != nil {
		p.cancel(errors.New("SQS event provider closed"))
	}

	if p.router != nil {
		if err := p.router.Close(); err != nil {
			l.Error("Error closing Watermill router", logger.Error(err))
			providerCloseErrorCount.Add(ctx, 1)
		}
	}

	if p.subscriber != nil {
		if err := p.subscriber.Close(); err != nil {
			l.Error("Error closing SQS subscriber", logger.Error(err))
			providerCloseErrorCount.Add(ctx, 1)
		}
	}

	if p.publisher != nil {
		if err := p.publisher.Close(); err != nil {
			l.Error("Error closing SQS publisher", logger.Error(err))
			providerCloseErrorCount.Add(ctx, 1)
		}
	}

	duration := time.Since(startTime)
	providerCloseDuration.Record(ctx, float64(duration.Milliseconds()))
	providerCloseCount.Add(ctx, 1)

	l.Internal("SQS provider closed",
		logger.Int64("durationMs", duration.Milliseconds()))

	return nil
}

// logStarting logs that the provider is starting with its settings.
func (p *SQSProvider) logStarting(ctx context.Context) {
	_, l := logger.From(ctx, log)
	l.Internal("Starting SQS provider",
		logger.String("region", p.config.Region),
		logger.String("endpointURL", p.config.EndpointURL),
		logger.Bool("queueAutoCreate", p.config.QueueAutoCreate))
}

// loadAWSConfig builds the AWS SDK configuration from the provider settings.
//
// Returns aws.Config which is the loaded AWS configuration.
// Returns error when the configuration cannot be loaded.
func (p *SQSProvider) loadAWSConfig(ctx context.Context) (aws.Config, error) {
	_, l := logger.From(ctx, log)

	var loadOpts []func(*awsconfig.LoadOptions) error
	if p.config.Region != "" {
		loadOpts = append(loadOpts, awsconfig.WithRegion(p.config.Region))
	}
	if p.config.EndpointURL != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cmp.Or(p.config.AccessKey, "test"),
				cmp.Or(p.config.SecretKey, "test"),
				"",
			),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		l.Error("Failed to load AWS config", logger.Error(err))
		providerStartErrorCount.Add(ctx, 1)
		providerConnectionErrors.Add(ctx, 1)
		return aws.Config{}, fmt.Errorf("loading AWS default config: %w", err)
	}

	return awsCfg, nil
}

// initialisePubSub creates the publisher and subscriber for the SQS connection.
//
// Takes awsCfg (*aws.Config) which provides the AWS SDK configuration with
// region and credentials.
//
// Returns error when the publisher or subscriber cannot be created.
func (p *SQSProvider) initialisePubSub(ctx context.Context, awsCfg *aws.Config) error {
	_, l := logger.From(ctx, log)

	var sqsOptFns []func(*amazonsqs.Options)
	if p.config.EndpointURL != "" {
		sqsOptFns = append(sqsOptFns, func(o *amazonsqs.Options) {
			o.BaseEndpoint = aws.String(p.config.EndpointURL)
		})
	}

	publisher, err := sqs.NewPublisher(sqs.PublisherConfig{
		AWSConfig: *awsCfg,
		OptFns:    sqsOptFns,
		Marshaler: sqs.DefaultMarshalerUnmarshaler{},
	}, p.logger)
	if err != nil {
		l.Error("Failed to create SQS publisher", logger.Error(err))
		providerStartErrorCount.Add(ctx, 1)
		return fmt.Errorf("creating SQS publisher: %w", err)
	}
	p.publisher = publisher

	subscriber, err := sqs.NewSubscriber(sqs.SubscriberConfig{
		AWSConfig:   *awsCfg,
		OptFns:      sqsOptFns,
		Unmarshaler: sqs.DefaultMarshalerUnmarshaler{},
	}, p.logger)
	if err != nil {
		_ = p.publisher.Close()
		l.Error("Failed to create SQS subscriber", logger.Error(err))
		providerStartErrorCount.Add(ctx, 1)
		return fmt.Errorf("creating SQS subscriber: %w", err)
	}
	p.subscriber = subscriber

	return nil
}

// initialiseRouter creates and sets up the Watermill router.
//
// Returns error when router creation fails.
func (p *SQSProvider) initialiseRouter(ctx context.Context) error {
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
func (p *SQSProvider) startRouterGoroutine(ctx context.Context) {
	_, l := logger.From(ctx, log)
	go func() {
		defer goroutine.RecoverPanic(p.ctx, "events.sqsRouter")

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
func (p *SQSProvider) recordStartMetrics(ctx context.Context, startTime time.Time) {
	ctx, l := logger.From(ctx, log)

	duration := time.Since(startTime)
	providerStartDuration.Record(ctx, float64(duration.Milliseconds()))
	providerStartCount.Add(ctx, 1)

	l.Internal("SQS provider started",
		logger.Int64("durationMs", duration.Milliseconds()),
		logger.String("region", p.config.Region))
}

// DefaultConfig returns settings suited for production use with AWS SQS.
// Queue auto-creation is enabled for simple deployment.
//
// Returns Config which contains the default settings ready for use.
func DefaultConfig() Config {
	return Config{
		Region:          defaultRegion,
		QueueAutoCreate: true,
		CloseTimeout:    defaultTimeoutSeconds * time.Second,
		RouterConfig: events.RouterConfig{
			CloseTimeout: defaultTimeoutSeconds,
		},
	}
}
