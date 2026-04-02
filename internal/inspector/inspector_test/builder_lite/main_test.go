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

package builder_lite_test

import (
	"context"
	"os"
	"testing"

	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/testutil/leakcheck"
)

var testStdlibData *inspector_dto.TypeData

var MinimalStdlibPackages = []string{
	"time",
	"context",
	"fmt",
	"strings",
	"net/http",
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error
	testStdlibData, err = inspector_domain.GenerateStdlibTypeDataWithPackages(
		ctx,
		MinimalStdlibPackages,
		nil,
	)
	if err != nil {

		_, _ = os.Stderr.WriteString("Failed to generate test stdlib: " + err.Error() + "\n")
		os.Exit(1)
	}

	code := m.Run()
	if code == 0 {
		if err := leakcheck.FindLeaks(); err != nil {
			_, _ = os.Stderr.WriteString("goleak: " + err.Error() + "\n")
			os.Exit(1)
		}
	}
	os.Exit(code)
}
