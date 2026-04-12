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

//go:build go1.26

package interp_domain

import (
	"runtime"
	"strings"
)

var (
	// supportedGoVersionPrefixes lists the verified runtime.Version prefixes.
	//
	// The unsafe Value layout in reflect_value_unsafe.go has been verified against each
	// prefix here. When a new Go release ships, run the layout tests
	// (TestReflectValueLayout, TestUnsafeNewAtParity) on it; if they pass, add its prefix
	// here so the runtime gate accepts it. Without this list the gate would either reject
	// every subsequent release (forcing immediate code edits) or accept untested ones
	// (silently risking memory unsafety against changed reflect internals).
	//
	//nolint:gochecknoglobals // intentional one-time runtime invariant table
	supportedGoVersionPrefixes = [...]string{
		"go1.26",
		"go1.27",
		"go1.28",
		"go1.29",
	}
)

func init() { //nolint:gochecknoinits // intentional one-time runtime invariant
	version := runtime.Version()
	for _, prefix := range supportedGoVersionPrefixes {
		if strings.HasPrefix(version, prefix) {
			return
		}
	}
	panic("piko interp_domain requires Go 1.26+ runtime; running on " + version +
		"; extend supportedGoVersionPrefixes after running TestReflectValueLayout " +
		"and TestUnsafeNewAtParity against this release")
}
