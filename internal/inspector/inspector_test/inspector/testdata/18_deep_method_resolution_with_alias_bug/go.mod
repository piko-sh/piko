module testcase_18_deep_method_resolution_with_alias_bug

go 1.27.0

require piko.sh/piko v0.0.0

require (
	github.com/bojanz/currency v1.4.4 // indirect
	github.com/cockroachdb/apd/v3 v3.2.3 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	piko.sh/json v0.1.1 // indirect
)

replace piko.sh/piko => ../../../../../../
