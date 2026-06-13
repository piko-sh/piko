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

// This file contains worker service related container methods.

import (
	"context"
	"errors"
	"fmt"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/shutdown"
	worker_querier_postgres "piko.sh/piko/internal/worker/worker_dal/querier_postgres"
	worker_querier_sqlite "piko.sh/piko/internal/worker/worker_dal/querier_sqlite"
	"piko.sh/piko/internal/worker/worker_domain"
	"piko.sh/piko/wdk/clock"
)

// GetWorkerService returns the worker service, creating it on first call. When no worker
// database is registered via WithDatabase(DatabaseNameWorker, ...), it defaults to the
// in-memory otter store obtained from the shared persistence provider, mirroring the
// registry and orchestrator subsystems.
//
// Returns worker_domain.Service which runs and drains the job pool.
// Returns error when the store or service cannot be created.
func (c *Container) GetWorkerService() (worker_domain.Service, error) {
	c.workerOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.workerServiceOverride != nil {
			l.Internal("Using provided WorkerService override.")
			c.workerService = c.workerServiceOverride
			return
		}
		c.workerService, c.workerErr = c.buildWorkerService(c.workerOptions...)
		if c.workerErr == nil {
			c.registerWorkerShutdownHandlers()
		}
	})
	return c.workerService, c.workerErr
}

// buildWorkerService builds the default worker service: it resolves the store behind the
// same otter/sqlite/postgres switch-off the other subsystems use, threads the resolved
// clock through the store and the wake notifier, and wires the notifier into the service.
//
// Takes opts (...worker_domain.ServiceOption) which customise the service (clock, config,
// queues, concurrency, and so on).
//
// Returns worker_domain.Service which is ready to Start.
// Returns error when the store cannot be created.
func (c *Container) buildWorkerService(opts ...worker_domain.ServiceOption) (worker_domain.Service, error) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default WorkerService...")

	clk := worker_domain.ResolveClock(opts...)

	store, err := c.createWorkerStore(clk)
	if err != nil {
		return nil, fmt.Errorf("creating worker store: %w", err)
	}

	notifier := worker_domain.NewInProcessNotifier(worker_domain.WithNotifierClock(clk))
	c.workerNotifier = notifier

	serviceOpts := make([]worker_domain.ServiceOption, 0, len(opts)+1)
	serviceOpts = append(serviceOpts, opts...)
	serviceOpts = append(serviceOpts, worker_domain.WithNotifier(notifier))

	return worker_domain.NewService(store, serviceOpts...), nil
}

// createWorkerStore resolves the worker Store behind the composition-root switch-off: a
// database registered under DatabaseNameWorker selects the querier-backed DAL, otherwise
// the default in-memory otter store from the shared persistence provider is used. The
// bootstrap never registers the worker database itself - the user's WithDatabase does.
//
// Takes clk (clock.Clock) which the SQL-backed stores stamp timestamps from.
//
// Returns worker_domain.Store which is the resolved store.
// Returns error when the store cannot be created.
func (c *Container) createWorkerStore(clk clock.Clock) (worker_domain.Store, error) {
	if c.dbRegistrations != nil {
		if _, registered := c.dbRegistrations[DatabaseNameWorker]; registered {
			return c.createQuerierWorkerDAL(clk)
		}
	}

	return c.createProviderWorkerDAL()
}

// createQuerierWorkerDAL creates a worker Store from a querier-managed database
// connection registered via AddDatabase(DatabaseNameWorker, ...).
//
// Takes clk (clock.Clock) which the store stamps timestamps from.
//
// Returns worker_domain.Store which is the querier-backed store.
// Returns error when the database connection cannot be obtained.
func (c *Container) createQuerierWorkerDAL(clk clock.Clock) (worker_domain.Store, error) {
	database, driver, err := c.resolveQuerierDatabase(DatabaseNameWorker, "worker")
	if err != nil {
		return nil, err
	}

	if isPostgresDriver(driver) {
		return worker_querier_postgres.New(database, clk), nil
	}

	return worker_querier_sqlite.New(database, clk), nil
}

// createProviderWorkerDAL creates a worker Store from the default otter in-memory backend
// via the shared persistence provider.
//
// Returns worker_domain.Store which is the otter-backed store.
// Returns error when the otter DAL cannot be created or does not implement the Store
// interface.
func (c *Container) createProviderWorkerDAL() (worker_domain.Store, error) {
	dalAny, err := c.createOtterWorkerDAL()
	if err != nil {
		return nil, fmt.Errorf("failed to create otter worker DAL: %w", err)
	}

	store, ok := dalAny.(worker_domain.Store)
	if !ok {
		return nil, errors.New("otter worker DAL does not implement worker_domain.Store")
	}

	return store, nil
}

// registerWorkerShutdownHandlers registers the worker service for graceful drain on
// application shutdown.
func (c *Container) registerWorkerShutdownHandlers() {
	shutdown.Register(c.GetAppContext(), "WorkerService", func(_ context.Context) error {
		return c.workerService.Shutdown(c.appCtx)
	})
}
