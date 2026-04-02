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
		if _, ok := recommendedSysctlInts[key]; !ok {
			t.Errorf("recommendedSysctlInts missing key: %s", key)
		}
	}
}

func TestRecommendedSysctlInts_SomaxconnValue(t *testing.T) {
	t.Parallel()

	expected := 65535
	if recommendedSysctlInts["net.core.somaxconn"] != expected {
		t.Errorf("net.core.somaxconn = %d, want %d",
			recommendedSysctlInts["net.core.somaxconn"], expected)
	}
}

func TestRecommendedSysctlInts_TcpTwReuseValue(t *testing.T) {
	t.Parallel()

	expected := 1
	if recommendedSysctlInts["net.ipv4.tcp_tw_reuse"] != expected {
		t.Errorf("net.ipv4.tcp_tw_reuse = %d, want %d",
			recommendedSysctlInts["net.ipv4.tcp_tw_reuse"], expected)
	}
}

func TestRecommendedSysctlInts_TcpFinTimeoutValue(t *testing.T) {
	t.Parallel()

	expected := 30
	if recommendedSysctlInts["net.ipv4.tcp_fin_timeout"] != expected {
		t.Errorf("net.ipv4.tcp_fin_timeout = %d, want %d",
			recommendedSysctlInts["net.ipv4.tcp_fin_timeout"], expected)
	}
}

func TestRecommendedSysctlRanges_ContainsPortRange(t *testing.T) {
	t.Parallel()

	key := "net.ipv4.ip_local_port_range"
	if _, ok := recommendedSysctlRanges[key]; !ok {
		t.Errorf("recommendedSysctlRanges missing key: %s", key)
	}
}

func TestRecommendedSysctlRanges_PortRangeValues(t *testing.T) {
	t.Parallel()

	key := "net.ipv4.ip_local_port_range"
	expected := [2]int{32768, 65535}

	actual, ok := recommendedSysctlRanges[key]
	if !ok {
		t.Fatalf("recommendedSysctlRanges missing key: %s", key)
	}

	if actual[0] != expected[0] || actual[1] != expected[1] {
		t.Errorf("%s = [%d, %d], want [%d, %d]",
			key, actual[0], actual[1], expected[0], expected[1])
	}
}

func TestRecommendedRlimits_ContainsUlimitN(t *testing.T) {
	t.Parallel()

	if _, ok := recommendedRlimits["ulimit-n"]; !ok {
		t.Error("recommendedRlimits missing key: ulimit-n")
	}
}

func TestRecommendedRlimits_UlimitNValue(t *testing.T) {
	t.Parallel()

	expected := uint64(65536)
	if recommendedRlimits["ulimit-n"] != expected {
		t.Errorf("ulimit-n = %d, want %d",
			recommendedRlimits["ulimit-n"], expected)
	}
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

	if keySysctl != "sysctl" {
		t.Errorf("keySysctl = %q, want %q", keySysctl, "sysctl")
	}

	if keyKey != "key" {
		t.Errorf("keyKey = %q, want %q", keyKey, "key")
	}

	if msgCouldNotReadSysctl != "Could not read sysctl value." {
		t.Errorf("msgCouldNotReadSysctl = %q, want %q",
			msgCouldNotReadSysctl, "Could not read sysctl value.")
	}
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
