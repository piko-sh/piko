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
	"strings"

	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

// BuildHealthDetailSections turns the liveness and readiness probes on
// a GetHealth response into the shared section/row shape used by the
// CLI Printer and the TUI detail pane. Filter narrows the output to a
// single named probe.
//
// The State row carries IsStatus=true so renderers can colourise it as
// a status value; the raw state string flows through unchanged.
//
// Takes response (*pb.GetHealthResponse) which is the health payload
// returned by the monitoring API.
// Takes filter (string) which restricts output to the probe whose name
// matches; an empty filter passes both probes through.
//
// Returns []DetailSection which contains zero, one, or two sections
// depending on whether each probe is present and matches the filter.
func BuildHealthDetailSections(response *pb.GetHealthResponse, filter string) []DetailSection {
	probes := []struct {
		status *pb.HealthStatus
		name   string
	}{
		{name: "Liveness", status: response.GetLiveness()},
		{name: "Readiness", status: response.GetReadiness()},
	}

	sections := make([]DetailSection, 0, len(probes))
	for _, probe := range probes {
		if !matchesFilter(probe.name, filter) {
			continue
		}
		if probe.status == nil {
			continue
		}
		sections = append(sections, healthStatusSection(probe.name, probe.status))
	}
	return sections
}

// healthStatusSection builds the section for a single probe, including
// dependency sub-sections.
//
// Takes name (string) which is the section heading.
// Takes status (*pb.HealthStatus) which is the probe payload.
//
// Returns DetailSection which contains the labelled rows and any
// dependency sub-sections.
func healthStatusSection(name string, status *pb.HealthStatus) DetailSection {
	section := DetailSection{
		Heading: name,
		Rows: []DetailRow{
			{Label: "State", Value: status.GetState(), IsStatus: true},
			{Label: "Message", Value: status.GetMessage()},
			{Label: "Duration", Value: status.GetDuration()},
			{Label: "Timestamp", Value: FormatUnixSeconds(status.GetTimestampMs())},
			{Label: "Dependencies", Value: formatHealthReady(status)},
		},
	}

	for _, dependency := range status.GetDependencies() {
		section.SubSections = append(section.SubSections, healthDependencySection(dependency))
	}
	return section
}

// healthDependencySection builds a sub-section for a single
// dependency. The Message row is appended only when a non-empty
// message is present.
//
// Takes dependency (*pb.HealthStatus) which is the dependency payload.
//
// Returns DetailSection which contains the dependency rows.
func healthDependencySection(dependency *pb.HealthStatus) DetailSection {
	section := DetailSection{
		Heading: dependency.GetName(),
		Rows: []DetailRow{
			{Label: "State", Value: dependency.GetState(), IsStatus: true},
			{Label: "Duration", Value: dependency.GetDuration()},
		},
	}
	if dependency.GetMessage() != "" {
		section.Rows = append(section.Rows, DetailRow{Label: "Message", Value: dependency.GetMessage()})
	}
	return section
}

// formatHealthReady counts healthy dependencies on a probe and returns "x/y"
// for display in the Dependencies row. When the probe has no dependencies,
// returns "-" so the row reads cleanly.
//
// Takes status (*pb.HealthStatus) which contains the dependency list.
//
// Returns string which is the ready count in "x/y" form, or "-" when
// there are no dependencies.
func formatHealthReady(status *pb.HealthStatus) string {
	deps := status.GetDependencies()
	if len(deps) == 0 {
		return "-"
	}
	healthy := 0
	for _, d := range deps {
		if strings.EqualFold(d.GetState(), "healthy") {
			healthy++
		}
	}
	return fmt.Sprintf("%d/%d", healthy, len(deps))
}
