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

package inspector

import (
	"fmt"
	"strconv"

	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

// rateLimiterPercentageFactor scales a 0..1 ratio into a percentage
// for the "Allow Rate" row.
const rateLimiterPercentageFactor = 100

// BuildRateLimiterDetailSections converts a rate limiter status
// response into the shared section/row shape consumed by the CLI
// Printer and the TUI detail panel renderer.
//
// The output contains two sections: the configuration row group and
// the counter row group. The "Allow Rate" row renders as "-" when no
// checks have been recorded yet.
//
// Takes response (*pb.GetRateLimiterStatusResponse) which contains the
// rate limiter status returned by the monitoring API.
//
// Returns []DetailSection which contains exactly two sections - "Rate
// Limiter" configuration and "Counters" totals.
func BuildRateLimiterDetailSections(response *pb.GetRateLimiterStatusResponse) []DetailSection {
	allowedDeniedRatio := "-"
	total := response.GetTotalAllowed() + response.GetTotalDenied()
	if total > 0 {
		pct := float64(response.GetTotalAllowed()) / float64(total) * rateLimiterPercentageFactor
		allowedDeniedRatio = fmt.Sprintf("%.1f%% allowed", pct)
	}

	return []DetailSection{
		{
			Heading: "Rate Limiter",
			Rows: []DetailRow{
				{Label: "Token Bucket Store", Value: response.GetTokenBucketStore()},
				{Label: "Counter Store", Value: response.GetCounterStore()},
				{Label: "Fail Policy", Value: response.GetFailPolicy()},
				{Label: "Key Prefix", Value: response.GetKeyPrefix()},
			},
		},
		{
			Heading: "Counters",
			Rows: []DetailRow{
				{Label: "Total Checks", Value: strconv.FormatInt(response.GetTotalChecks(), 10)},
				{Label: "Total Allowed", Value: strconv.FormatInt(response.GetTotalAllowed(), 10)},
				{Label: "Total Denied", Value: strconv.FormatInt(response.GetTotalDenied(), 10)},
				{Label: "Total Errors", Value: strconv.FormatInt(response.GetTotalErrors(), 10)},
				{Label: "Allow Rate", Value: allowedDeniedRatio},
			},
		},
	}
}
