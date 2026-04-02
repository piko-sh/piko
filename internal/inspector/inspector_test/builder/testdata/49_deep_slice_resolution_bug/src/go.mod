module testcase_49_deep_slice_resolution_bug

go 1.26.0

require piko.sh/piko v0.0.0

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.42.0 // indirect
	go.opentelemetry.io/otel/metric v1.42.0 // indirect
	go.opentelemetry.io/otel/trace v1.42.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
)

// These are transitive dependencies of piko, needed for `go mod tidy` to work.
require (
	github.com/bojanz/currency v1.4.2 // indirect
	github.com/cockroachdb/apd/v3 v3.2.3 // indirect
)

// This 'replace' directive points to your local piko project.
// It assumes this testdata directory is inside the piko project structure.
// Adjust the relative path if your test setup is different.
replace piko.sh/piko => ../../../../../../../
