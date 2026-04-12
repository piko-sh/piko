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
	"os"

	_ "testmodule/dist"

	"piko.sh/piko"
	ga4 "piko.sh/piko/wdk/analytics/analytics_collector_ga4"
	"piko.sh/piko/wdk/analytics/analytics_collector_stdout"
	"piko.sh/piko/wdk/logger"
)

func main() {
	command := piko.RunModeDev
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	logger.AddPrettyOutput()

	ssr := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),

		// CSP: extend Piko defaults to allow GA4 scripts and connect endpoints.
		piko.WithCSP(func(b *piko.CSPBuilder) {
			b.WithPikoDefaults().
				ScriptSrc(piko.CSPSelf, piko.CSPHost("https://www.googletagmanager.com")).
				ConnectSrc(piko.CSPSelf,
					piko.CSPHost("https://www.google-analytics.com"),
					piko.CSPHost("https://*.google-analytics.com"),
					piko.CSPHost("https://*.analytics.google.com"),
					piko.CSPHost("https://*.googletagmanager.com"),
				)
		}),

		piko.WithFrontendModule(piko.ModuleAnalytics, piko.AnalyticsConfig{
			TrackingIDs: []string{"G-DEBUGDEMO1"},
			DebugMode:   true,
		}),
		piko.WithBackendAnalytics(
			analytics_collector_stdout.NewCollector(),
			ga4.NewCollector("G-DEBUGDEMO1", "demo-api-secret", ga4.WithDebug(true)),
		),
		piko.WithDevWidget(),
		piko.WithDevHotreload(),
		piko.WithMonitoring(),
	)
	if err := ssr.Run(command); err != nil {
		panic(err)
	}
}
