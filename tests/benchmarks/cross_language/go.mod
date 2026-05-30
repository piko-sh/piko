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

// Module cross_language is the cross-language benchmark suite that compares
// piko against CPython, PyPy, and native Go. Lives in its own module so
// the testcontainers-go dependency tree does not bleed into the root
// piko.sh/piko module.

module piko.sh/piko/tests/benchmarks/cross_language

go 1.26.0

require (
	github.com/d5/tengo/v2 v2.17.0
	github.com/moby/moby/api v1.54.1
	github.com/mvm-sh/mvm v0.3.0
	github.com/open2b/scriggo v0.61.1
	github.com/stretchr/testify v1.11.1
	github.com/testcontainers/testcontainers-go v0.42.0
	github.com/traefik/yaegi v0.16.1
	piko.sh/piko v0.0.0
)
