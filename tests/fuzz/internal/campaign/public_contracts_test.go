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
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/tests/fuzz/internal/publiccontracts"
)

func TestPublicContractReceiptCountsObservedWork(t *testing.T) {
	var metrics ModeMetrics
	for _, check := range []CheckMetric{
		{Status: "failed", PublicContract: &publiccontracts.Receipt{Contract: "example/demo", SourceSHA256: "first", HTTPStatusVerified: true}},
		{Status: "pass", PublicContract: &publiccontracts.Receipt{Contract: "example/demo", SourceSHA256: "second", HTTPStatusVerified: true, BrowserVisited: true, Steps: 4, Assertions: 2}},
	} {
		metrics.observe("package", &check)
	}
	assert.Equal(t, PublicContractCounts{Attempts: 2, Passed: 1, Steps: 4, Assertions: 2, HTTPStatusVerified: 2, BrowserVisited: 1, LastSourceSHA256: "second"}, metrics.PublicContracts["example/demo"])
}
