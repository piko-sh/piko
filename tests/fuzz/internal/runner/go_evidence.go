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

package runner

import (
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

// HostEvidence counts planned, executed and verified Go-host operations.
type HostEvidence struct {
	// Planned counts modelled host operations.
	Planned uint64

	// Executed counts executed host operations.
	Executed uint64

	// Verified counts independently verified host operations.
	Verified uint64
}

// topologyTally counts the executed operations of a topology receipt.
type topologyTally struct {
	// resolution counts the reported module resolutions.
	resolution uint64

	// hosts counts the instances that served a read.
	hosts uint64

	// actions counts the instances that answered the ledger action.
	actions uint64

	// pages counts the direct ledger reads.
	pages uint64

	// callbacks counts the callback page visits.
	callbacks uint64
}

// GoHostEvidence counts real process, route and action observations.
//
// A partial receipt retains its executed counts. Verification requires its complete
// model.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
//
// Returns map[string]HostEvidence which maps Go-host operation names to planned, executed
// and verified counts.
func GoHostEvidence(c *casefile.Case, o *Observation) map[string]HostEvidence {
	var counts map[string]HostEvidence
	record := func(name string, planned, observed uint64, ok bool) {
		if counts == nil {
			counts = map[string]HostEvidence{}
		}
		value := HostEvidence{Planned: planned, Executed: observed, Verified: 0}
		if ok {
			value.Verified = observed
		}
		counts[name] = value
	}
	if c.Target == "embedded-deploy" {
		var got uint64
		if o.Deployment != nil {
			got = 1
		}
		record("binary-only-deployment", 1, got, checkDeployment(o.Deployment) == nil && o.Status == c.Expected.Status && o.ComponentAsset != "")
	}
	if model := c.Expected.Topology; model != nil {
		tally := countTopology(o.Topology)
		verified := checkTopology(model, o.Topology) == nil
		record("local-module-resolution", topologyInstanceCount, tally.resolution, verified)
		record("fresh-service-instance", topologyInstanceCount, tally.hosts, verified)
		record("typed-native-action", topologyInstanceCount, tally.actions, verified)
		record("service-route", topologyInstanceCount*topologyReadCount, tally.pages, verified)
		if model.GuestCallbacks {
			record("retained-callback-route", topologyInstanceCount*topologyReadCount, tally.callbacks, verified)
		}
	}
	return counts
}

// countTopology counts the executed topology operations in a receipt.
//
// Takes topology (*GoTopologyReceipt) which holds the receipt, if any.
//
// Returns topologyTally which holds the executed counts.
func countTopology(topology *GoTopologyReceipt) topologyTally {
	var tally topologyTally
	if topology == nil {
		return tally
	}
	if topology.Workspace != "" {
		tally.resolution++
	}
	if topology.Module != "" {
		tally.resolution++
	}
	for index := range topology.Instances {
		instance := &topology.Instances[index]
		if len(instance.Reads) > 0 {
			tally.hosts++
		}
		if instance.ActionStatus > 0 {
			tally.actions++
		}
		tally.pages += uint64(len(instance.Reads))
		tally.callbacks += uint64(len(instance.Callbacks))
	}
	return tally
}
