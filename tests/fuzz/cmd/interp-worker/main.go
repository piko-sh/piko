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

// Package main runs the isolated interpreted application host for a fuzz case.
package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime/coverage"
	"slices"
	"strconv"

	"piko.sh/piko"
	"piko.sh/piko/components"
	"piko.sh/piko/wdk/email/email_provider_disk"
	"piko.sh/piko/wdk/interp/interp_provider_pipit"
	"piko.sh/piko/wdk/markdown/markdown_provider_goldmark"
)

const (
	// envReadyToken names the public readiness identity the host echoes to any caller.
	envReadyToken = "PIKO_FUZZ_TOKEN"

	// envCoverageSecret names the separate secret that authorises a coverage flush.
	envCoverageSecret = "PIKO_FUZZ_COVERAGE_SECRET"

	// envGoCoverage names the directory that receives flushed Go coverage.
	envGoCoverage = "PIKO_FUZZ_GO_COVERAGE"

	// coverageHeader carries the coverage secret on a flush request.
	coverageHeader = "X-Piko-Fuzz-Coverage"
)

var (
	// extraSymbols holds case-specific native symbol exports.
	extraSymbols interp_provider_pipit.SymbolExports

	// fixtureOptions holds generated provider configuration for this host.
	fixtureOptions []piko.Option

	// workflowDatabase opens the workflow database and supplies its cleanup.
	workflowDatabase func() (piko.Option, func(), error)

	// errMissingShippedComponent reports a selected tag absent from the shipped catalogue.
	errMissingShippedComponent = errors.New("shipped component missing from catalogue")

	// shippedTags lists the components the interpreted host can register. Generated projects
	// compile this host outside the fuzz module and cannot import shippedhost.
	shippedTags = []string{"piko-counter", "piko-card", "example-greeting", "m2-data-table-pagination", "m3e-switch", "piko-dev-widget"}

	// serveHost runs the configured host until it shuts down.
	serveHost = func(server *piko.SSRServer) error {
		return server.Run(piko.RunModeDevInterpreted)
	}
)

// run starts the configured interpreted application host.
//
// Returns error which reports interpreted-server setup or serving failure.
func run(ctx context.Context) error {
	port, err := strconv.Atoi(os.Getenv("PIKO_FUZZ_PORT"))
	if err != nil {
		return fmt.Errorf("reading PIKO_FUZZ_PORT: %w", err)
	}
	options := []piko.Option{piko.WithBindAddress("127.0.0.1"), piko.WithPort(port), piko.WithAutoNextPort(false), piko.WithCSSReset(piko.WithCSSResetComplete()), piko.WithHealthEnabled(false)}
	if outbox := os.Getenv("PIKO_FUZZ_CHECKOUT_OUTBOX"); outbox != "" {
		var outboxArguments email_provider_disk.DiskProviderArgs
		outboxArguments.OutboxPath = outbox
		provider, err := email_provider_disk.NewDiskProvider(ctx, outboxArguments)
		if err != nil {
			return fmt.Errorf("opening checkout outbox: %w", err)
		}
		options = append(options, piko.WithEmailProvider("checkout-disk", provider), piko.WithDefaultEmailProvider("checkout-disk"))
	}
	if os.Getenv("PIKO_FUZZ_SHIPPED") == "true" {
		shipped, err := shippedOptions()
		if err != nil {
			return err
		}
		options = append(options, shipped...)
	}
	if os.Getenv("PIKO_FUZZ_PARTIAL_COMPONENTS") != "" {
		var website piko.WebsiteConfig
		website.I18n = piko.I18nConfig{DefaultLocale: "en", Locales: []string{"en", "fr"}, Strategy: "prefix_except_default"}
		options = append(options, piko.WithWebsiteConfig(website))
	}
	if os.Getenv("PIKO_FUZZ_COLLECTIONS") == "true" {
		options = append(options, piko.WithMarkdownParser(markdown_provider_goldmark.NewParser()))
	}
	if workflowDatabase != nil {
		option, closeDatabase, err := workflowDatabase()
		if err != nil {
			return fmt.Errorf("opening workflow database: %w", err)
		}
		defer closeDatabase()
		options = append(options, option)
	}
	options = append(options, fixtureOptions...)
	server := piko.New(options...)
	if os.Getenv("PIKO_FUZZ_CATALOG_WATCH") == "true" {
		var public piko.PublicConfig
		public.WatchMode = true
		server.Configure(public)
	}
	defer server.Close()
	server.WithInterpreterProvider(interp_provider_pipit.NewProvider())
	server.WithSymbols(extraSymbols)
	server.AppRouter.Get("/__piko_fuzz_ready", ready)
	return serveHost(server)
}

// ready answers the harness readiness probe and, when authorised, flushes Go coverage.
//
// The readiness body carries only the public readiness identity. A flush needs the
// separate coverage secret, which the host never writes to a response, so page script or
// another local process that reads the readiness route cannot trigger one.
//
// Takes writer (http.ResponseWriter) which receives the readiness response.
// Takes request (*http.Request) which is the harness probe.
func ready(writer http.ResponseWriter, request *http.Request) {
	if directory := os.Getenv(envGoCoverage); directory != "" && coverageAuthorised(request) {
		if err := coverage.WriteMetaDir(directory); err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := coverage.WriteCountersDir(directory); err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	if _, err := writer.Write([]byte(os.Getenv(envReadyToken))); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "writing readiness token:", err)
	}
}

// coverageAuthorised reports whether a request carries the configured coverage secret.
//
// Takes request (*http.Request) which is the harness probe.
//
// Returns bool which is true only when a non-empty secret is configured and matches.
func coverageAuthorised(request *http.Request) bool {
	secret := os.Getenv(envCoverageSecret)
	if secret == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(request.Header.Get(coverageHeader)), []byte(secret)) == 1
}

// shippedOptions returns the host options that register the selected shipped components.
//
// Returns []piko.Option which registers each selected component in a fixed order.
// Returns error which wraps errMissingShippedComponent for every absent tag, or nil on
// success.
func shippedOptions() ([]piko.Option, error) {
	catalogue := slices.Concat(components.All(), components.Dev())
	options := make([]piko.Option, 0, len(shippedTags))
	var missing []error
	for _, tag := range shippedTags {
		index := slices.IndexFunc(catalogue, func(definition piko.ComponentDefinition) bool { return definition.TagName == tag })
		if index < 0 {
			missing = append(missing, fmt.Errorf("%w: %s", errMissingShippedComponent, tag))
			continue
		}
		options = append(options, piko.WithComponents(catalogue[index]))
	}
	if err := errors.Join(missing...); err != nil {
		return nil, err
	}
	return options, nil
}

// main starts the isolated interpreted application host.
func main() {
	if err := run(context.Background()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
