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

package orchestrator_pipeline_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal/dalcore"
	orchestrator_querier_postgres "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_postgres"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	clockpkg "piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_postgres"
	"piko.sh/piko/wdk/events/events_provider_nats"
)

const clusterExecutorName = "cluster.work"

type cluster struct {
	t        *testing.T
	ctx      context.Context
	natsURL  string
	dsn      string
	config   orchestrator_domain.DispatcherConfig
	executor func(replica string) orchestrator_domain.TaskExecutor
	replicas []*clusterReplica
}

type clusterReplica struct {
	name       string
	database   *sql.DB
	store      orchestrator_domain.TaskStore
	provider   *events_provider_nats.NATSProvider
	bus        orchestrator_domain.EventBus
	dispatcher orchestrator_domain.TaskDispatcher
	cancel     context.CancelCauseFunc
	stopped    chan struct{}
}

type executorFunc func(ctx context.Context, payload map[string]any) (map[string]any, error)

func (f executorFunc) Execute(ctx context.Context, payload map[string]any) (map[string]any, error) {
	return f(ctx, payload)
}

func newCluster(t *testing.T, executor func(replica string) orchestrator_domain.TaskExecutor) *cluster {
	t.Helper()
	if testing.Short() {
		t.Skip("starts NATS and Postgres containers")
	}
	ctx, cancel := context.WithTimeoutCause(context.Background(), 3*time.Minute, errors.New("cluster test exceeded 3 minutes"))
	t.Cleanup(cancel)

	natsContainer, natsURL := startNATSContainer(ctx, t)
	t.Cleanup(func() { _ = natsContainer.Terminate(context.Background()) })
	dsn := startPostgresContainer(ctx, t)

	admin, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	migrator, err := db.NewMigrator(db_engine_postgres.Postgres(), admin, orchestrator_querier_postgres.Migrations, "migrations")
	require.NoError(t, err)
	_, err = migrator.Up(ctx)
	require.NoError(t, err, "migrating the orchestrator schema")
	require.NoError(t, admin.Close())

	return &cluster{
		t:       t,
		ctx:     ctx,
		natsURL: natsURL,
		dsn:     dsn,
		config: orchestrator_domain.DispatcherConfig{
			DefaultTimeout:          time.Minute,
			DefaultMaxRetries:       3,
			RecoveryInterval:        300 * time.Millisecond,
			StaleTaskThreshold:      time.Second,
			HeartbeatInterval:       200 * time.Millisecond,
			SweepInterval:           500 * time.Millisecond,
			WatermillHighHandlers:   2,
			WatermillNormalHandlers: 2,
			WatermillLowHandlers:    1,
			Clock:                   clockpkg.RealClock(),
			Transport:               orchestrator_domain.TransportBus,
		},
		executor: executor,
	}
}

func startPostgresContainer(ctx context.Context, t *testing.T) string {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:17-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env:          map[string]string{"POSTGRES_PASSWORD": "piko", "POSTGRES_DB": "piko"},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(time.Minute),
		},
		Started: true,
	})
	require.NoError(t, err, "starting Postgres")
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	return fmt.Sprintf("postgres://postgres:piko@%s:%s/piko?sslmode=disable", host, port.Port())
}

func (c *cluster) addReplica(name string) *clusterReplica {
	c.t.Helper()
	ctx, cancel := context.WithCancelCause(c.ctx)
	replica := &clusterReplica{name: name, cancel: cancel, stopped: make(chan struct{})}

	database, err := sql.Open("pgx", c.dsn)
	require.NoError(c.t, err)
	replica.database = database
	replica.store = orchestrator_querier_postgres.New(database, dalcore.WithClock(clockpkg.RealClock()))

	natsConfig := events_provider_nats.DefaultConfig()
	natsConfig.URL = c.natsURL
	natsConfig.ClusterID = "replica-" + name
	natsConfig.QueueGroupPrefix = "cluster"
	natsConfig.JetStream.Disabled = true
	natsConfig.SubscribersCount = 1
	natsConfig.AckWaitTimeout = 10 * time.Second
	natsConfig.CloseTimeout = 2 * time.Second
	provider, err := events_provider_nats.NewNATSProvider(natsConfig)
	require.NoError(c.t, err)
	require.NoError(c.t, provider.Start(ctx))
	replica.provider = provider
	replica.bus = orchestrator_adapters.NewWatermillEventBus(
		provider.Publisher(), provider.Subscriber(), provider.Router(),
		orchestrator_adapters.WithDistributedTransport(),
		orchestrator_adapters.WithBroadcastSubscriber(provider.BroadcastSubscriber()),
	)

	replica.dispatcher = orchestrator_adapters.CreateTaskDispatcher(context.Background(), c.config, replica.bus, replica.store)
	replica.dispatcher.RegisterExecutor(context.Background(), clusterExecutorName, c.executor(name))
	go func() {
		defer close(replica.stopped)
		_ = replica.dispatcher.Start(ctx)
	}()
	c.replicas = append(c.replicas, replica)

	c.t.Cleanup(func() {
		replica.cancel(errors.New("test finished"))
		<-replica.stopped
		_ = provider.Close()
		_ = database.Close()
	})
	time.Sleep(300 * time.Millisecond)
	return replica
}

func (r *clusterReplica) crash() {
	_ = r.database.Close()
	_ = r.provider.Close()
}

func (r *clusterReplica) shutdown() {
	r.cancel(errors.New("SIGTERM"))
	<-r.stopped
}

func (r *clusterReplica) dispatch(t *testing.T, key string) (*orchestrator_domain.Task, error) {
	t.Helper()
	task := orchestrator_domain.NewTask(clusterExecutorName, map[string]any{"key": key})
	task.DeduplicationKey = key
	err := r.dispatcher.Dispatch(t.Context(), task)
	return task, err
}

func (r *clusterReplica) storedTask(t *testing.T, id string) *orchestrator_domain.Task {
	t.Helper()
	tasks, err := r.store.GetTasksByID(t.Context(), []string{id})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	return tasks[0]
}

func TestCluster_NoArtefactBuiltTwice(t *testing.T) {
	const keys = 60
	var (
		mu   sync.Mutex
		runs = map[string]int{}
		gate = make(chan struct{})
	)
	c := newCluster(t, func(string) orchestrator_domain.TaskExecutor {
		return executorFunc(func(ctx context.Context, payload map[string]any) (map[string]any, error) {
			select {
			case <-gate:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			key, _ := payload["key"].(string)
			mu.Lock()
			runs[key]++
			mu.Unlock()
			return nil, nil
		})
	})
	a, b := c.addReplica("a"), c.addReplica("b")

	var dispatching sync.WaitGroup
	for _, replica := range []*clusterReplica{a, b} {
		dispatching.Go(func() {
			for i := range keys {
				_, err := replica.dispatch(t, fmt.Sprintf("artefact-%d", i))
				if err != nil && !errors.Is(err, orchestrator_domain.ErrDuplicateTask) {
					t.Errorf("dispatch on %s: %v", replica.name, err)
				}
			}
		})
	}
	dispatching.Wait()
	close(gate)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(runs) == keys
	}, time.Minute, 50*time.Millisecond, "every artefact is built")
	time.Sleep(time.Second)
	mu.Lock()
	defer mu.Unlock()
	for key, count := range runs {
		assert.Equal(t, 1, count, "%s built once across both replicas", key)
	}
}

func TestCluster_PeerFinishesTaskOfCrashedReplica(t *testing.T) {
	var (
		started     = make(chan struct{})
		startedOnce sync.Once
		finishedOnB atomic.Int32
	)
	c := newCluster(t, func(replica string) orchestrator_domain.TaskExecutor {
		return executorFunc(func(ctx context.Context, _ map[string]any) (map[string]any, error) {
			if replica == "a" {
				startedOnce.Do(func() { close(started) })
				<-ctx.Done()
				return nil, ctx.Err()
			}
			finishedOnB.Add(1)
			return nil, nil
		})
	})
	a := c.addReplica("a")
	task, err := a.dispatch(t, "artefact-crash")
	require.NoError(t, err)
	select {
	case <-started:
	case <-time.After(30 * time.Second):
		t.Fatal("replica a never started the task")
	}

	a.crash()
	b := c.addReplica("b")

	require.Eventually(t, func() bool {
		return b.storedTask(t, task.ID).Status == orchestrator_domain.StatusComplete
	}, time.Minute, 100*time.Millisecond, "the surviving replica recovers and finishes the crashed replica's task")
	stored := b.storedTask(t, task.ID)
	assert.Equal(t, 2, stored.Attempt, "the crashed attempt and the peer's each count once")
	assert.Equal(t, int32(1), finishedOnB.Load(), "the peer runs it once")
}

func TestCluster_ShutdownHandsRunningTaskToPeer(t *testing.T) {
	var (
		firstReplica atomic.Value
		started      = make(chan struct{})
		calls        atomic.Int32
		finishedBy   atomic.Value
	)
	c := newCluster(t, func(replica string) orchestrator_domain.TaskExecutor {
		return executorFunc(func(ctx context.Context, _ map[string]any) (map[string]any, error) {
			if calls.Add(1) == 1 {
				firstReplica.Store(replica)
				close(started)
				<-ctx.Done()
				return nil, ctx.Err()
			}
			finishedBy.Store(replica)
			return nil, nil
		})
	})
	replicas := map[string]*clusterReplica{"a": c.addReplica("a"), "b": c.addReplica("b")}
	task, err := replicas["a"].dispatch(t, "artefact-shutdown")
	require.NoError(t, err)
	select {
	case <-started:
	case <-time.After(30 * time.Second):
		t.Fatal("no replica started the task")
	}

	stopping := firstReplica.Load().(string)
	survivor := map[string]string{"a": "b", "b": "a"}[stopping]
	replicas[stopping].shutdown()

	require.Eventually(t, func() bool {
		return replicas[survivor].storedTask(t, task.ID).Status == orchestrator_domain.StatusComplete
	}, time.Minute, 100*time.Millisecond, "the task a stopping replica released is finished by its peer")
	assert.Equal(t, survivor, finishedBy.Load(), "the peer finishes it")
}

func TestCluster_ArtefactChangesReachEveryReplica(t *testing.T) {
	c := newCluster(t, func(string) orchestrator_domain.TaskExecutor {
		return executorFunc(func(context.Context, map[string]any) (map[string]any, error) { return nil, nil })
	})
	a, b := c.addReplica("a"), c.addReplica("b")

	var received [2]atomic.Int32
	for i, replica := range []*clusterReplica{a, b} {
		require.NoError(t, orchestrator_domain.SubscribeBroadcast(t.Context(), replica.bus, "artefact.updated",
			func(context.Context, orchestrator_domain.Event) error {
				received[i].Add(1)
				return nil
			}))
	}
	time.Sleep(300 * time.Millisecond)

	const events = 5
	for range events {
		require.NoError(t, a.bus.Publish(t.Context(), "artefact.updated", orchestrator_domain.Event{
			Type:    "artefact.updated",
			Payload: map[string]any{"artefactID": "lib/hero.jpg"},
		}))
	}
	require.Eventually(t, func() bool {
		return received[0].Load() == events && received[1].Load() == events
	}, 10*time.Second, 20*time.Millisecond, "every replica sees every artefact change, so none keeps a stale cache")
}
