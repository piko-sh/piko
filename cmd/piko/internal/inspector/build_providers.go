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
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

// BuildProviderDetailSections converts a DescribeProvider gRPC response
// into the shared section/row shape consumed by the CLI Printer and the
// TUI detail panel renderer.
//
// The protobuf already carries titled sections containing key/value entries;
// this builder re-shapes them as DetailSection values without altering field
// order or labels.
//
// Takes response (*pb.DescribeProviderResponse) which contains the
// provider detail payload returned by the monitoring API.
//
// Returns []DetailSection which contains one DetailSection per
// protobuf section, with rows mirroring the entry order on the wire.
func BuildProviderDetailSections(response *pb.DescribeProviderResponse) []DetailSection {
	pbSections := response.GetSections()
	sections := make([]DetailSection, 0, len(pbSections))

	for _, s := range pbSections {
		rows := make([]DetailRow, 0, len(s.GetEntries()))
		for _, e := range s.GetEntries() {
			rows = append(rows, DetailRow{
				Label: e.GetKey(),
				Value: e.GetValue(),
			})
		}
		sections = append(sections, DetailSection{
			Heading: s.GetTitle(),
			Rows:    rows,
		})
	}

	return sections
}
