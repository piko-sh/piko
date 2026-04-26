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

// BuildDLQDetailSections turns a DispatcherSummary slice into the
// shared section/row shape used by the CLI Printer and the TUI detail
// pane. Filter narrows the output to dispatchers whose Type matches.
//
// Takes summaries ([]*pb.DispatcherSummary) which are the dispatcher
// summaries returned by the monitoring API.
// Takes filter (string) which restricts output to dispatchers whose
// Type matches the filter; an empty filter passes everything through.
//
// Returns []DetailSection which contains one DetailSection per
// matching dispatcher, in the same order they appeared in summaries.
func BuildDLQDetailSections(summaries []*pb.DispatcherSummary, filter string) []DetailSection {
	sections := make([]DetailSection, 0)
	for _, s := range summaries {
		if !matchesFilter(s.GetType(), filter) {
			continue
		}
		sections = append(sections, DetailSection{
			Heading: fmt.Sprintf("Dispatcher %s", s.GetType()),
			Rows: []DetailRow{
				{Label: "Type", Value: s.GetType()},
				{Label: "Queued", Value: strconv.Itoa(int(s.GetQueuedItems()))},
				{Label: "Retry Queue", Value: strconv.Itoa(int(s.GetRetryQueueSize()))},
				{Label: "Dead Letter", Value: strconv.Itoa(int(s.GetDeadLetterCount()))},
				{Label: "Total Processed", Value: strconv.FormatInt(s.GetTotalProcessed(), 10)},
				{Label: "Total Successful", Value: strconv.FormatInt(s.GetTotalSuccessful(), 10)},
				{Label: "Total Failed", Value: strconv.FormatInt(s.GetTotalFailed(), 10)},
				{Label: "Total Retries", Value: strconv.FormatInt(s.GetTotalRetries(), 10)},
				{Label: "Uptime", Value: FormatMilliseconds(s.GetUptimeMs())},
			},
		})
	}
	return sections
}
