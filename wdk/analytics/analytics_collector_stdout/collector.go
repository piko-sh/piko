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

package analytics_collector_stdout

import (
	"context"
	"time"

	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/analytics"
)

const (
	// collectorName identifies this collector in logs and metrics.
	collectorName = "stdout"

	// maxFields is the maximum number of log fields an event can
	// produce (4 required + 10 optional + headroom for properties).
	maxFields = 30
)

var log = logger_domain.GetLogger("piko/wdk/analytics/analytics_collector_stdout")

// Collector logs analytics events to the structured logger. It does
// not buffer or batch — each event is logged immediately in Collect.
type Collector struct{}

// NewCollector creates an analytics collector that prints events to
// the structured logger at INFO level.
//
// Returns analytics.Collector which logs events to stdout.
func NewCollector() analytics.Collector {
	return &Collector{}
}

// Start is a no-op for the stdout collector.
func (*Collector) Start(_ context.Context) {}

// Collect logs the event immediately.
//
// Takes event (*analytics_dto.Event) which carries the event data.
//
// Returns error which is always nil.
func (*Collector) Collect(ctx context.Context, event *analytics_dto.Event) error {
	ctx, l := logger_domain.From(ctx, log)

	fields := make([]logger_domain.Attr, 0, maxFields)
	fields = appendCoreFields(fields, event)
	fields = appendOptionalFields(fields, event)
	fields = appendPropertyFields(fields, event)

	l.Info("Analytics event", fields...)
	return nil
}

// Flush is a no-op for the stdout collector.
//
// Returns error which is always nil.
func (*Collector) Flush(_ context.Context) error { return nil }

// Close is a no-op for the stdout collector.
//
// Returns error which is always nil.
func (*Collector) Close(_ context.Context) error { return nil }

// Name returns the collector name.
//
// Returns string which identifies this collector.
func (*Collector) Name() string { return collectorName }

// appendCoreFields adds the fields that are always present.
//
// Takes fields ([]logger_domain.Attr) which is the target slice.
// Takes event (*analytics_dto.Event) which provides the data.
//
// Returns []logger_domain.Attr which is the extended slice.
func appendCoreFields(fields []logger_domain.Attr, event *analytics_dto.Event) []logger_domain.Attr {
	return append(fields,
		logger_domain.String("type", event.Type.String()),
		logger_domain.String("path", event.Path),
		logger_domain.String("method", event.Method),
		logger_domain.Int("status_code", event.StatusCode),
	)
}

// appendOptionalFields adds non-empty string fields and computed
// values.
//
// Takes fields ([]logger_domain.Attr) which is the target slice.
// Takes event (*analytics_dto.Event) which provides the data.
//
// Returns []logger_domain.Attr which is the extended slice.
func appendOptionalFields(fields []logger_domain.Attr, event *analytics_dto.Event) []logger_domain.Attr {
	if event.Hostname != "" {
		fields = append(fields, logger_domain.String("hostname", event.Hostname))
	}
	if event.URL != "" {
		fields = append(fields, logger_domain.String("url", event.URL))
	}
	if event.EventName != "" {
		fields = append(fields, logger_domain.String("event_name", event.EventName))
	}
	if event.ActionName != "" {
		fields = append(fields, logger_domain.String("action_name", event.ActionName))
	}
	if event.UserID != "" {
		fields = append(fields, logger_domain.String("user_id", event.UserID))
	}
	if event.ClientIP != "" {
		fields = append(fields, logger_domain.String("client_ip", event.ClientIP))
	}
	if event.Referrer != "" {
		fields = append(fields, logger_domain.String("referrer", event.Referrer))
	}
	if event.Duration > 0 {
		fields = append(fields, logger_domain.String("duration", event.Duration.Round(time.Millisecond).String()))
	}
	if event.Revenue != nil {
		fields = append(fields, logger_domain.String("revenue", event.Revenue.DefaultFormat()))
	}
	return fields
}

// appendPropertyFields adds custom properties as individual log
// fields with a "prop." prefix.
//
// Takes fields ([]logger_domain.Attr) which is the target slice.
// Takes event (*analytics_dto.Event) which provides the data.
//
// Returns []logger_domain.Attr which is the extended slice.
func appendPropertyFields(fields []logger_domain.Attr, event *analytics_dto.Event) []logger_domain.Attr {
	for key, value := range event.Properties {
		fields = append(fields, logger_domain.String("prop."+key, value))
	}
	return fields
}
