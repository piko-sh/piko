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

package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseByteSizeUsesBinaryUnits(t *testing.T) {
	for text, want := range map[string]uint64{
		"512":    512,
		"4K":     4 << 10,
		"4KB":    4 << 10,
		"12GiB":  12 << 30,
		"1.5GiB": 3 << 29,
		"2 t":    2 << 40,
		"64gib":  64 << 30,
		"0.5KiB": 512,
		" 3MiB ": 3 << 20,
	} {
		got, err := parseByteSize(text)
		require.NoError(t, err, text)
		assert.Equal(t, want, got, text)
	}
	for _, text := range []string{"", "0", "-1GiB", "4XB", "GiB", "99999999999T", "0.0001B"} {
		_, err := parseByteSize(text)
		assert.Error(t, err, text)
	}
}

func TestPlanLaunchScopesCampaignsAndBudgetedSourceCommands(t *testing.T) {
	options := flags{state: "/state", cpus: 1, mem: "1GiB", disk: "2GiB", reserve: "1GiB", set: map[string]bool{}}

	launch, err := planLaunch(commandStatus, "/repo", &options)
	require.NoError(t, err)
	assert.False(t, launch.scoped, "status must run without a scope")

	launch, err = planLaunch(commandGenerate, "/repo", &options)
	require.NoError(t, err)
	assert.False(t, launch.scoped, "gen without limits must run without a scope")

	options.set["mem"] = true
	launch, err = planLaunch(commandGenerate, "/repo", &options)
	require.NoError(t, err)
	assert.True(t, launch.scoped, "gen with -mem must run in a scope")

	launch, err = planLaunch(commandCampaign, "/repo", &options)
	require.NoError(t, err)
	assert.True(t, launch.scoped)
	assert.Len(t, launch.cpuList, 1)
	assert.Equal(t, uint64(1<<30), launch.memory)
	assert.Equal(t, uint64(2<<30), launch.disk)
	assert.Equal(t, uint64(1<<30), launch.minFree)
}

func TestPlanLaunchRejectsInvalidSizesAndNewlines(t *testing.T) {
	options := flags{state: "/state", cpus: 1, mem: "lots", disk: "2GiB", reserve: "1GiB", set: map[string]bool{}}
	_, err := planLaunch(commandCampaign, "/repo", &options)
	require.ErrorContains(t, err, "-mem")

	options.mem = "1GiB"
	options.state = "/state\nSCOPED=0"
	_, err = planLaunch(commandCampaign, "/repo", &options)
	require.ErrorIs(t, err, errLaunchPathNewline)
}

func TestRenderWritesEveryLauncherVariable(t *testing.T) {
	launch := launchPlan{
		root: "/repo", state: "/state", cpuList: []int{0, 2}, cpus: 2,
		memory: 10 << 30, minFree: 4 << 30, disk: 20 << 30, scoped: true,
	}

	lines := strings.Split(strings.TrimSpace(launch.render()), "\n")

	assert.Contains(t, lines, "SCOPED=1")
	assert.Contains(t, lines, "CPU_LIST=0,2")
	assert.Contains(t, lines, "CPU_QUOTA=200%")
	assert.Contains(t, lines, "MEMORY_MAX=10737418240")
	assert.Contains(t, lines, "MEMORY_HIGH=8589934560")
	assert.Contains(t, lines, "PIKO_FUZZ_STATE=/state")
	assert.Contains(t, lines, "PIKO_FUZZ_ENCLOSING_LIMIT=")
}

func TestPlanReportsHelpUnscopedAndRejectsBadOptions(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, plan(t.Context(), &out, []string{"help"}))
	assert.Equal(t, "SCOPED=0\n", out.String())

	require.Error(t, plan(t.Context(), &out, nil))
	require.Error(t, plan(t.Context(), &out, []string{commandCampaign, "-mode", "nope"}))
}
