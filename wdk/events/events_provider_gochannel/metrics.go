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

package events_provider_gochannel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	// publicationMeter creates metrics for the GoChannel publication queues.
	publicationMeter = otel.Meter("piko/events/gochannel")

	// publicationMessages counts queued and delivering messages across publishers.
	publicationMessages = publicationGauge("events.gochannel.messages", "Queued and delivering publications")

	// publicationBytes tracks estimated retained publication bytes across publishers.
	publicationBytes = publicationGauge("events.gochannel.bytes", "Retained publication byte budget including bookkeeping")

	// publicationTopics counts active topic delivery workers across publishers.
	publicationTopics = publicationGauge("events.gochannel.topics", "Active per-topic delivery workers")

	// publicationRejected counts batches rejected by publication admission limits.
	publicationRejected = func() metric.Int64Counter {
		counter, err := publicationMeter.Int64Counter("events.gochannel.rejected", metric.WithDescription("Publications rejected by admission limits"))
		if err != nil {
			otel.Handle(err)
		}
		return counter
	}()
)

// publicationGauge creates an up-down counter and reports setup errors to OpenTelemetry.
//
// Takes name (string) which identifies the metric.
// Takes description (string) which describes the measured resource.
//
// Returns metric.Int64UpDownCounter which is the instrument returned by the meter.
func publicationGauge(name, description string) metric.Int64UpDownCounter {
	counter, err := publicationMeter.Int64UpDownCounter(name, metric.WithDescription(description))
	if err != nil {
		otel.Handle(err)
	}
	return counter
}

// publicationChange records changes to retained publication resources.
//
// Takes messages (int) which is the signed change in queued and delivering messages.
// Takes bytes (int64) which is the signed change in estimated retained bytes.
// Takes topics (int) which is the signed change in active topic workers.
func publicationChange(messages int, bytes int64, topics int) {
	ctx := context.Background()
	if messages != 0 {
		publicationMessages.Add(ctx, int64(messages))
	}
	if bytes != 0 {
		publicationBytes.Add(ctx, bytes)
	}
	if topics != 0 {
		publicationTopics.Add(ctx, int64(topics))
	}
}
