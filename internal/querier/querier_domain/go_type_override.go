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

package querier_domain

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// standardLibraryLookalikes maps a single segment package path that resolves to the Go
// standard library onto the third party import path a go_type override almost certainly
// meant.
var standardLibraryLookalikes = map[string]string{
	"uuid": "github.com/google/uuid",
}

// parseGoTypeOverride splits a go_type directive value into its package path and type
// name.
//
// Takes raw (string) which is the directive value as written.
//
// Returns *querier_dto.GoType which is the parsed override, or nil when raw is malformed.
// Returns string which is a problem description when the override parses but is very
// likely to be wrong, and is empty otherwise.
func parseGoTypeOverride(raw string) (*querier_dto.GoType, string) {
	packagePath, typeName, found := strings.CutLast(raw, ".")

	if !found {
		return &querier_dto.GoType{Name: raw}, ""
	}

	if packagePath == "" || typeName == "" {
		return nil, ""
	}

	goType := &querier_dto.GoType{
		Package: packagePath,
		Name:    typeName,
	}

	return goType, describeStandardLibraryLookalike(goType.Package)
}

// describeStandardLibraryLookalike reports why a package path is likely to be a mistake.
//
// Takes packagePath (string) which is the parsed import path.
//
// Returns string which explains the problem, or an empty string when the path looks fine.
func describeStandardLibraryLookalike(packagePath string) string {
	intended, isLookalike := standardLibraryLookalikes[packagePath]
	if !isLookalike {
		return ""
	}

	return fmt.Sprintf(
		"%q now resolves to the Go standard library package %q, whose type implements neither "+
			"sql.Scanner nor driver.Valuer, so every row would fail to scan at runtime. Write the "+
			"full import path %q instead",
		packagePath, packagePath, intended,
	)
}
