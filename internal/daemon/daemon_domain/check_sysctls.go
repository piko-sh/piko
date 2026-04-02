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
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// keySysctl is the logging field key for sysctl parameter names.
	keySysctl = "sysctl"

	// keyKey is the logging field name for sysctl parameter keys.
	keyKey = "key"

	// msgCouldNotReadSysctl is the warning message logged when a sysctl value
	// cannot be read.
	msgCouldNotReadSysctl = "Could not read sysctl value."
)

var (
	// recommendedSysctlInts defines integer-based kernel settings and their
	// recommended minimum values.
	recommendedSysctlInts = map[string]int{
		// Maximum number of connections queued for acceptance. Low values can
		// lead to connection drops during traffic bursts. 65535 is a common
		// high-performance value for servers expecting high connection rates.
		"net.core.somaxconn": 65535,
		// Allows the reuse of sockets in TIME_WAIT state for new connections when it's
		// safe from a protocol perspective. Essential for services with many
		// short-lived connections to avoid ephemeral port exhaustion. A value of 1
		// enables it.
		"net.ipv4.tcp_tw_reuse": 1,
		// Reduces the time the kernel holds sockets in FIN_WAIT2 after a connection is
		// closed. A lower value (e.g., 15-30s) allows for faster resource reclamation.
		"net.ipv4.tcp_fin_timeout": 30,
	}

	// recommendedSysctlRanges defines range-based kernel settings.
	recommendedSysctlRanges = map[string][2]int{
		// The range of ephemeral ports for outgoing connections. A wider range
		// prevents port exhaustion. We check if the available range is at least 32768
		// ports.
		"net.ipv4.ip_local_port_range": {32768, 65535},
	}

	// recommendedRlimits defines resource limits and their recommended minimums.
	recommendedRlimits = map[string]uint64{
		// Maximum number of open file descriptors. Each network connection is a file
		// descriptor. This is a critical setting for handling 10k+ connections.
		"ulimit-n": 65536,
	}
)

// checkHostConfiguration checks kernel settings and resource limits to ensure
// the host is ready for high-performance network tasks.
//
// When the environment is not production, skips checks and logs a message.
//
// When the system is not Linux, skips checks and logs a message.
func checkHostConfiguration(ctx context.Context) {
	ctx, span, l := log.Span(ctx, "CheckHostConfiguration")
	defer span.End()

	if os.Getenv("PIKO_ENV") != "production" {
		l.Internal("Development environment detected, production advisories silenced.", logger_domain.String("env", os.Getenv("PIKO_ENV")))
		return
	}

	if runtime.GOOS != "linux" {
		l.Internal("Skipping host configuration checks on non-Linux OS.", logger_domain.String("os", runtime.GOOS))
		return
	}

	l.Internal("Checking host configuration for optimal performance...")
	checkAllSysctls(ctx)
	checkAllRlimits(ctx)
	l.Internal("Host configuration check is complete.")
}

// checkAllSysctls checks all sysctl settings against their expected values.
func checkAllSysctls(ctx context.Context) {
	for key, expectedValue := range recommendedSysctlInts {
		checkIntSysctl(ctx, key, expectedValue)
	}
	for key, expectedRange := range recommendedSysctlRanges {
		checkRangeSysctl(ctx, key, expectedRange)
	}
}

// checkAllRlimits checks all resource limits and logs warnings for any that
// are below the suggested values.
func checkAllRlimits(ctx context.Context) {
	for key, expectedValue := range recommendedRlimits {
		if key == "ulimit-n" {
			checkRlimitNofile(ctx, expectedValue)
		}
	}
}

// checkIntSysctl reads and checks a single integer sysctl value.
//
// Takes key (string) which specifies the sysctl key in dotted form.
// Takes expectedValue (int) which sets the minimum acceptable value.
func checkIntSysctl(ctx context.Context, key string, expectedValue int) {
	ctx, l := logger_domain.From(ctx, log)
	root, err := os.OpenRoot("/proc/sys")
	if err != nil {
		l.Warn("Could not open /proc/sys for sysctl access.", logger_domain.Error(err))
		return
	}
	defer func() { _ = root.Close() }()

	relPath := strings.ReplaceAll(key, ".", "/")
	file, err := root.Open(relPath)
	if err != nil {
		l.Warn(msgCouldNotReadSysctl,
			logger_domain.String(keyKey, key),
			logger_domain.String("path", "/proc/sys/"+relPath),
			logger_domain.Error(err))
		return
	}

	content, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil {
		l.Warn(msgCouldNotReadSysctl,
			logger_domain.String(keyKey, key),
			logger_domain.Error(err))
		return
	}

	valueString := strings.TrimSpace(string(content))
	currentValue, err := strconv.Atoi(valueString)
	if err != nil {
		l.Warn("Could not parse sysctl value.",
			logger_domain.String(keyKey, key),
			logger_domain.String("value", valueString))
		return
	}

	if currentValue < expectedValue {
		l.Warn("[PERFORMANCE WARNING] Suboptimal kernel setting detected. Consider increasing this value for production workloads.",
			logger_domain.String(keySysctl, key),
			logger_domain.Int("current_value", currentValue),
			logger_domain.Int("recommended_minimum", expectedValue),
			logger_domain.String("command", fmt.Sprintf("sudo sysctl -w %s=%d", key, expectedValue)))
	} else {
		l.Internal("[OK] Kernel setting meets recommendation.",
			logger_domain.String(keySysctl, key),
			logger_domain.Int("current_value", currentValue))
	}
}

// checkRangeSysctl reads a sysctl value and checks it falls within a range.
//
// Takes key (string) which specifies the sysctl key in dot notation.
// Takes expectedRange ([2]int) which defines the minimum and maximum values.
func checkRangeSysctl(ctx context.Context, key string, expectedRange [2]int) {
	ctx, l := logger_domain.From(ctx, log)
	root, err := os.OpenRoot("/proc/sys")
	if err != nil {
		l.Warn("Could not open /proc/sys for sysctl access.", logger_domain.Error(err))
		return
	}
	defer func() { _ = root.Close() }()

	relPath := strings.ReplaceAll(key, ".", "/")
	file, err := root.Open(relPath)
	if err != nil {
		l.Warn(msgCouldNotReadSysctl, logger_domain.String(keyKey, key), logger_domain.Error(err))
		return
	}

	content, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil {
		l.Warn(msgCouldNotReadSysctl, logger_domain.String(keyKey, key), logger_domain.Error(err))
		return
	}

	parts := strings.Fields(string(content))
	if len(parts) != 2 {
		l.Warn("Could not parse sysctl range value.", logger_domain.String(keyKey, key), logger_domain.String("value", string(content)))
		return
	}

	minVal, errMin := strconv.Atoi(parts[0])
	maxVal, errMax := strconv.Atoi(parts[1])
	if errMin != nil || errMax != nil {
		l.Warn("Could not parse sysctl range components.", logger_domain.String(keyKey, key), logger_domain.String("value", string(content)))
		return
	}

	availablePorts := maxVal - minVal
	expectedPorts := expectedRange[1] - expectedRange[0]

	if availablePorts < expectedPorts {
		l.Warn("[PERFORMANCE WARNING] Ephemeral port range is narrow. This can lead to port exhaustion under a high load.",
			logger_domain.String(keySysctl, key),
			logger_domain.String("current_range", fmt.Sprintf("%d-%d (%d ports)", minVal, maxVal, availablePorts)),
			logger_domain.String("recommended_range", fmt.Sprintf("%d-%d (%d ports)", expectedRange[0], expectedRange[1], expectedPorts)),
			logger_domain.String("command", fmt.Sprintf("sudo sysctl -w %s='%d %d'", key, expectedRange[0], expectedRange[1])))
	} else {
		l.Internal("[OK] Kernel setting meets recommendation.",
			logger_domain.String(keySysctl, key),
			logger_domain.String("current_range", fmt.Sprintf("%d-%d", minVal, maxVal)))
	}
}

// checkRlimitNofile reads and checks the open file descriptor limit.
//
// Takes expectedValue (uint64) which is the smallest allowed limit.
func checkRlimitNofile(ctx context.Context, expectedValue uint64) {
	ctx, l := logger_domain.From(ctx, log)
	var rlimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit)
	if err != nil {
		l.Warn("Could not read resource limit for open files (ulimit -n).", logger_domain.Error(err))
		return
	}

	currentLimit := rlimit.Cur
	if currentLimit < expectedValue {
		l.Warn("[PERFORMANCE WARNING] The open file descriptor limit (ulimit -n) is low. This can cause 'too many open files' errors under high connection load.",
			logger_domain.Uint64("current_limit", currentLimit),
			logger_domain.Uint64("recommended_minimum", expectedValue),
			logger_domain.String("command", fmt.Sprintf("ulimit -n %d", expectedValue)))
	} else {
		l.Internal("[OK] Resource limit for open files meets recommendation.",
			logger_domain.Uint64("current_limit", currentLimit))
	}
}
