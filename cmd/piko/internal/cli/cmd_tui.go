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

package cli

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"piko.sh/piko/cmd/piko/internal/tui"
	"piko.sh/piko/internal/logger/logger_domain"
)

// runTUICmd launches the interactive terminal UI.
//
// Takes ctx (context.Context) which controls the lifetime of the TUI
// session.
// Takes cc (*CommandContext) which provides CLI options including
// the monitoring endpoint.
//
// Returns error when the TUI fails to initialise or encounters a
// fatal error during execution.
func runTUICmd(ctx context.Context, cc *CommandContext, _ []string) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ctx, l := logger_domain.From(ctx, log)

	tuiConfig, err := tui.LoadConfig("")
	if err != nil {
		l.Warn("Could not load config file, using defaults")
	}

	tuiOpts := []tui.Option{
		tui.WithConfig(tuiConfig),
	}

	if cc.Opts.Endpoint != defaultEndpoint {
		tuiOpts = append(tuiOpts, tui.WithMonitoringEndpoint(cc.Opts.Endpoint))
	}

	t, err := tui.New(tuiOpts...)
	if err != nil {
		return fmt.Errorf("initialising TUI: %w", err)
	}
	defer func() { _ = t.Close() }()

	return t.Run(ctx)
}
