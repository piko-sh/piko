module testcase_22_gomod_replace

go 1.25.1

require github.com/google/uuid v1.6.0

replace github.com/google/uuid => ./local/uuid
