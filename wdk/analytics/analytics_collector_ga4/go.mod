module piko.sh/piko/wdk/analytics/analytics_collector_ga4

go 1.26.0

require (
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/metric v1.43.0
	piko.sh/piko v0.0.0
)

require (
	github.com/bojanz/currency v1.4.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/apd/v3 v3.2.3 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/sony/gobreaker/v2 v2.4.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.uber.org/goleak v1.3.0 // indirect
)

replace piko.sh/piko => ../../../
