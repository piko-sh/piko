module testcase_49_deep_slice_resolution_bug

go 1.27.0

require piko.sh/piko v0.0.0

// These are transitive dependencies of piko, needed for `go mod tidy` to work.
require (
	github.com/bojanz/currency v1.4.4 // indirect
	github.com/cockroachdb/apd/v3 v3.2.3 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	piko.sh/json v0.1.1 // indirect
)

// This 'replace' directive points to your local piko project.
// It assumes this testdata directory is inside the piko project structure.
// Adjust the relative path if your test setup is different.
replace piko.sh/piko => ../../../../../../../
