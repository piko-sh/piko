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

//go:build unix

package llm_provider_ollama

import (
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedOllamaCommandSignals(t *testing.T) {
	t.Parallel()

	require.NoError(t, killManagedOllamaCommand(nil))
	require.NoError(t, interruptManagedOllamaCommand(&exec.Cmd{}))

	executable, err := os.Executable()
	require.NoError(t, err)

	command := exec.Command(executable, "-test.run=^$")
	configureManagedOllamaCommand(command)
	require.NotNil(t, command.SysProcAttr)
	assert.True(t, command.SysProcAttr.Setpgid)
	require.NoError(t, command.Run())

	assert.ErrorIs(t, killManagedOllamaCommand(command), os.ErrProcessDone)
	assert.ErrorIs(t, interruptManagedOllamaCommand(command), os.ErrProcessDone)
	assert.ErrorIs(t, signalManagedOllamaGroup(command, syscall.SIGTERM, syscall.SIGTERM), os.ErrProcessDone)
}
