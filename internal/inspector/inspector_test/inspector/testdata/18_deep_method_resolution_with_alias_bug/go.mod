module testcase_18_deep_method_resolution_with_alias_bug

go 1.26.0

require piko.sh/piko v0.0.0

require (
	github.com/bojanz/currency v1.4.4 // indirect
	github.com/cockroachdb/apd/v3 v3.2.3 // indirect
)

replace piko.sh/piko => ../../../../../../
