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

//go:build linux

package daemon_domain

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckHostConfiguration_DevelopmentEnvironment(t *testing.T) {
	t.Setenv("PIKO_ENV", "development")

	ctx := context.Background()
	checkHostConfiguration(ctx)

}

func TestCheckHostConfiguration_EmptyEnvironment(t *testing.T) {
	t.Setenv("PIKO_ENV", "")

	ctx := context.Background()
	checkHostConfiguration(ctx)

}

func TestCheckHostConfiguration_ProductionEnvironmentNonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("Skipping non-Linux test on Linux")
	}

	t.Setenv("PIKO_ENV", "production")

	ctx := context.Background()
	checkHostConfiguration(ctx)

}

func TestRecommendedSysctlInts_ContainsExpectedKeys(t *testing.T) {
	t.Parallel()

	expectedKeys := []string{
		"net.core.somaxconn",
		"net.ipv4.tcp_tw_reuse",
		"net.ipv4.tcp_fin_timeout",
	}

	for _, key := range expectedKeys {
		assert.Containsf(t, recommendedSysctlInts, key, "recommendedSysctlInts missing key: %s", key)
	}
}

func TestRecommendedSysctlInts_SomaxconnValue(t *testing.T) {
	t.Parallel()

	expected := 65535
	assert.Equal(t, expected, recommendedSysctlInts["net.core.somaxconn"], "net.core.somaxconn")
}

func TestRecommendedSysctlInts_TcpTwReuseValue(t *testing.T) {
	t.Parallel()

	expected := 1
	assert.Equal(t, expected, recommendedSysctlInts["net.ipv4.tcp_tw_reuse"], "net.ipv4.tcp_tw_reuse")
}

func TestRecommendedSysctlInts_TcpFinTimeoutValue(t *testing.T) {
	t.Parallel()

	expected := 30
	assert.Equal(t, expected, recommendedSysctlInts["net.ipv4.tcp_fin_timeout"], "net.ipv4.tcp_fin_timeout")
}

func TestRecommendedSysctlRanges_ContainsPortRange(t *testing.T) {
	t.Parallel()

	key := "net.ipv4.ip_local_port_range"
	assert.Containsf(t, recommendedSysctlRanges, key, "recommendedSysctlRanges missing key: %s", key)
}

func TestRecommendedSysctlRanges_PortRangeValues(t *testing.T) {
	t.Parallel()

	key := "net.ipv4.ip_local_port_range"
	expected := [2]int{32768, 65535}

	actual, ok := recommendedSysctlRanges[key]
	require.Truef(t, ok, "recommendedSysctlRanges missing key: %s", key)

	assert.Equalf(t, expected, actual, "%s port range values", key)
}

func TestRecommendedRlimits_ContainsUlimitN(t *testing.T) {
	t.Parallel()

	assert.Contains(t, recommendedRlimits, "ulimit-n", "recommendedRlimits missing key: ulimit-n")
}

func TestRecommendedRlimits_UlimitNValue(t *testing.T) {
	t.Parallel()

	expected := uint64(65536)
	assert.Equal(t, expected, recommendedRlimits["ulimit-n"], "ulimit-n")
}

func TestCheckAllSysctls_LinuxOnly(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-only test on non-Linux OS")
	}

	t.Parallel()

	ctx := context.Background()

	checkAllSysctls(ctx)
}

func TestCheckAllRlimits_LinuxOnly(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-only test on non-Linux OS")
	}

	t.Parallel()

	ctx := context.Background()

	checkAllRlimits(ctx)
}

func TestCheckHostConfiguration_LinuxProduction(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-only test on non-Linux OS")
	}

	t.Setenv("PIKO_ENV", "production")

	ctx := context.Background()
	checkHostConfiguration(ctx)

}

func TestKeyConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "sysctl", keySysctl, "keySysctl")

	assert.Equal(t, "key", keyKey, "keyKey")

	assert.Equal(t, "Could not read sysctl value.", msgCouldNotReadSysctl, "msgCouldNotReadSysctl")
}

func TestCheckHostConfiguration_WithTestEnvironment(t *testing.T) {
	t.Setenv("PIKO_ENV", "test")

	ctx := context.Background()
	checkHostConfiguration(ctx)

}

func TestCheckHostConfiguration_WithStagingEnvironment(t *testing.T) {
	t.Setenv("PIKO_ENV", "staging")

	ctx := context.Background()
	checkHostConfiguration(ctx)

}
