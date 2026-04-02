module testcase_37_multi_package_name_collision

go 1.25.1

// Require three different external libraries that provide a 'uuid' package.
require (
	github.com/gofrs/uuid/v5 v5.4.0
	github.com/google/uuid v1.6.0
	modernc.org/libc v1.70.0
)
