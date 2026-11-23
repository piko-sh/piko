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

package campaign

import (
	"piko.sh/safeconv"
)

// PublicContractCounts aggregates completed public-consumer checks.
type PublicContractCounts struct {
	// LastSourceSHA256 identifies the latest checked contract source.
	LastSourceSHA256 string

	// Attempts counts attempted checks.
	Attempts uint64

	// Passed counts passed checks.
	Passed uint64

	// Steps counts completed contract steps.
	Steps uint64

	// Assertions counts completed assertions.
	Assertions uint64

	// HTTPStatusVerified counts verified HTTP statuses.
	HTTPStatusVerified uint64

	// BrowserVisited counts browser visits.
	BrowserVisited uint64
}

// observePublicContract accumulates completed public-consumer assertions.
//
// Takes s (*ModeMetrics) which receives accumulated mode metrics.
// Takes check (*CheckMetric) which holds one mode check's measured outcome.
func observePublicContract(s *ModeMetrics, check *CheckMetric) {
	receipt := check.PublicContract
	if s.PublicContracts == nil {
		s.PublicContracts = map[string]PublicContractCounts{}
	}
	counts := s.PublicContracts[receipt.Contract]
	counts.Attempts++
	counts.LastSourceSHA256 = receipt.SourceSHA256
	if check.Status == "pass" {
		counts.Passed++
	}
	if receipt.HTTPStatusVerified {
		counts.HTTPStatusVerified++
	}
	if receipt.BrowserVisited {
		counts.BrowserVisited++
	}
	counts.Steps += safeconv.IntToUint64(receipt.Steps)
	counts.Assertions += safeconv.IntToUint64(receipt.Assertions)
	s.PublicContracts[receipt.Contract] = counts
}
