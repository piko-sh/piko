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

package bootstrap

import (
	"fmt"
	"os"

	"piko.sh/piko/wdk/safedisk"
)

// createSystemTempSandbox creates a sandboxed file system scoped to the system
// temp folder. This is used for reading and writing secret files safely.
//
// Takes name (string) which labels the sandbox for diagnostics.
//
// Returns safedisk.Sandbox which provides sandboxed access to the temp folder.
// Returns error when the sandbox factory cannot be created or started.
func createSystemTempSandbox(name string) (safedisk.Sandbox, error) {
	tempDir := os.TempDir()

	factory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		CWD:          tempDir,
		AllowedPaths: []string{tempDir},
		Enabled:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("creating %s sandbox factory: %w", name, err)
	}

	return factory.Create(name, tempDir, safedisk.ModeReadWrite)
}
