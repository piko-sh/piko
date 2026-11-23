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

package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"runtime/coverage"
	"strconv"

	_ "example.org/pikofuzzcase/dist"
	"piko.sh/piko"
	"piko.sh/piko/components"
	"piko.sh/piko/wdk/email/email_provider_disk"
)

var fixtureOptions []piko.Option
var workflowDatabase func() (piko.Option, func(), error)
var collectionParser func() piko.Option

func main() {
	port, err := strconv.Atoi(os.Getenv("PIKO_FUZZ_PORT"))
	if err != nil {
		panic(err)
	}
	opts := []piko.Option{piko.WithPort(port), piko.WithBindAddress("127.0.0.1"),
		piko.WithHealthEnabled(false), piko.WithAutoNextPort(false), piko.WithCSSReset(piko.WithCSSResetComplete())}
	if outbox := os.Getenv("PIKO_FUZZ_CHECKOUT_OUTBOX"); outbox != "" {
		provider, err := email_provider_disk.NewDiskProvider(context.Background(), email_provider_disk.DiskProviderArgs{OutboxPath: outbox})
		if err != nil {
			panic(err)
		}
		opts = append(opts, piko.WithEmailProvider("checkout-disk", provider), piko.WithDefaultEmailProvider("checkout-disk"))
	}
	if os.Getenv("PIKO_FUZZ_SHIPPED") == "true" {
		for _, def := range append(components.All(), components.Dev()...) {
			switch def.TagName {
			case "piko-counter", "piko-card", "example-greeting", "m2-data-table-pagination", "m3e-switch", "piko-dev-widget":
				opts = append(opts, piko.WithComponents(def))
			}
		}
	}
	if os.Getenv("PIKO_FUZZ_PARTIAL_COMPONENTS") != "" {
		opts = append(opts, piko.WithWebsiteConfig(piko.WebsiteConfig{I18n: piko.I18nConfig{DefaultLocale: "en", Locales: []string{"en", "fr"}, Strategy: "prefix_except_default"}}))
	}
	if collectionParser != nil {
		opts = append(opts, collectionParser())
	}
	if workflowDatabase != nil {
		opt, closeDB, err := workflowDatabase()
		if err != nil {
			panic(err)
		}
		defer closeDB()
		opts = append(opts, opt)
	}
	opts = append(opts, fixtureOptions...)
	server := piko.New(opts...)
	server.AppRouter.Get("/__piko_fuzz_ready", func(w http.ResponseWriter, r *http.Request) {
		secret := os.Getenv("PIKO_FUZZ_COVERAGE_SECRET")
		authorised := secret != "" && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Piko-Fuzz-Coverage")), []byte(secret)) == 1
		if dir := os.Getenv("PIKO_FUZZ_GO_COVERAGE"); dir != "" && authorised {
			if err := coverage.WriteMetaDir(dir); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if err := coverage.WriteCountersDir(dir); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(os.Getenv("PIKO_FUZZ_TOKEN")))
	})
	if err := server.Run(piko.RunModeProd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
