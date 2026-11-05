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
	"database/sql"
	"fmt"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/worker/worker_dal/dalcore"
	worker_otter "piko.sh/piko/internal/worker/worker_dal/otter"
	worker_querier_postgres "piko.sh/piko/internal/worker/worker_dal/querier_postgres"
	worker_querier_sqlite "piko.sh/piko/internal/worker/worker_dal/querier_sqlite"
	"piko.sh/piko/internal/worker/worker_domain"
	"piko.sh/piko/wdk/clock"
)

// GetWorkerService returns the worker service, creating it on first call. When no worker
// database is registered via WithDatabase(DatabaseNameWorker, ...), it defaults to a fresh
// in-memory otter store.
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
// a fresh in-memory otter store is used. Build-time containers always use the in-memory
// store. The bootstrap never registers the worker database itself - the user's
// WithDatabase does.
//
// Takes clk (clock.Clock) which the stores stamp timestamps from.
//
// Returns worker_domain.Store which is the resolved store.
// Returns error when the store cannot be created.
func (c *Container) createWorkerStore(clk clock.Clock) (worker_domain.Store, error) {
	if c.isBuildTime {
		return createInMemoryWorkerDAL(clk)
	}

	if c.dbRegistrations != nil {
		if _, registered := c.dbRegistrations[DatabaseNameWorker]; registered {
			return c.createQuerierWorkerDAL(clk)
		}
	}

	return createInMemoryWorkerDAL(clk)
}

// createQuerierWorkerDAL creates a worker Store from a querier-managed database
// connection registered via AddDatabase(DatabaseNameWorker, ...).
//
// Takes clk (clock.Clock) which the store stamps timestamps from.
//
// Returns worker_domain.Store which is the querier-backed store.
// Returns error when the database connection cannot be obtained or its schema is
// outdated.
func (c *Container) createQuerierWorkerDAL(clk clock.Clock) (worker_domain.Store, error) {
	database, observed, driver, err := c.resolveQuerierDatabase(DatabaseNameWorker, "worker")
	if err != nil {
		return nil, err
	}

	if err := checkWorkerSchema(database, isPostgresDriver(driver)); err != nil {
		return nil, err
	}

	if isPostgresDriver(driver) {
		return worker_querier_postgres.NewObserved(database, observed, dalcore.WithClock(clk)), nil
	}

	return worker_querier_sqlite.NewObserved(database, observed, dalcore.WithClock(clk)), nil
}

// checkWorkerSchema verifies the worker database has every migration applied, so an
// outdated schema fails at startup with its remedy rather than on the first job.
//
// Takes database (*sql.DB) which is the worker database.
// Takes postgres (bool) which selects the Postgres schema over SQLite's.
//
// Returns error which wraps worker_domain.ErrSchemaOutdated when a migration is missing.
func checkWorkerSchema(database *sql.DB, postgres bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), schemaCheckTimeout)
	defer cancel()
	if postgres {
		return worker_querier_postgres.CheckSchema(ctx, database)
	}
	return worker_querier_sqlite.CheckSchema(ctx, database)
}

// createInMemoryWorkerDAL creates a fresh in-memory otter worker Store. The store has no
// WAL persistence, so it never writes to the persistence directory and is safe in
// build-time containers and single-binary deploys.
//
// Takes clk (clock.Clock) which the store stamps timestamps from.
//
// Returns worker_domain.Store which is the in-memory store.
// Returns error when the otter DAL cannot be created.
func createInMemoryWorkerDAL(clk clock.Clock) (worker_domain.Store, error) {
	store, err := worker_otter.NewOtterDAL(worker_otter.Config{}, worker_otter.WithClock(clk))
	if err != nil {
		return nil, fmt.Errorf("failed to create otter worker DAL: %w", err)
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
