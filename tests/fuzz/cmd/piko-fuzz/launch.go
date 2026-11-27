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
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/resource"
)

const (
	// commandPlan prints the launch plan for hack/fuzz/fuzz.sh.
	commandPlan = "_plan"

	// commandDoctor checks the tools and limits a campaign needs.
	commandDoctor = "doctor"

	// gibibyte is one GiB in bytes.
	gibibyte = 1 << 30

	// binaryUnitShift is the number of bits each binary size unit adds, as 1 KiB is 2^10
	// bytes.
	binaryUnitShift = 10

	// minimumHostReserve is the smallest default host memory reserve in bytes.
	minimumHostReserve = 4 * gibibyte

	// hostReserveDivisor sets the default host reserve as a fraction of total memory.
	hostReserveDivisor = 10

	// memoryHighPercent places memory.high below memory.max so the kernel reclaims first.
	memoryHighPercent = 80

	// cpuQuotaPercentPerCore is the systemd CPUQuota for one full core.
	cpuQuotaPercentPerCore = 100

	// ownedCacheMarker marks a build cache the launcher created and may evict.
	ownedCacheMarker = ".piko-fuzz-owned"
)

var (
	// byteSizePattern matches a byte size such as 512MiB, 4G or 1.5GiB.
	byteSizePattern = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*(B|[KMGT](?:i?B)?)?$`)

	// scopedCommands lists commands that require kernel limits.
	scopedCommands = []string{commandCampaign, commandResume, commandReplay, commandReduce, commandDoctor}

	// sourceCommands lists commands that require kernel limits only when requested.
	sourceCommands = []string{commandGenerate, "promote", "coverage", "inventory", commandPreview}

	// limitFlags name the options that request a resource budget.
	limitFlags = []string{"cpus", "mem", "min-free", "disk"}

	// errLaunchPathNewline reports a path the line-based plan cannot carry.
	errLaunchPathNewline = errors.New("launcher paths must not contain newlines")
)

// launchPlan specifies the supervisor's environment and requested limits.
type launchPlan struct {
	// root is the repository root.
	root string

	// state is the absolute campaign state directory.
	state string

	// enclosing is the cgroup that already limits the launcher, or empty for none.
	enclosing string

	// cpuList lists the CPUs the scope may use.
	cpuList []int

	// cpus is the CPU ceiling.
	cpus uint64

	// memory is the memory ceiling in bytes.
	memory uint64

	// minFree is the host memory reserve in bytes.
	minFree uint64

	// disk is the state-tree disk budget in bytes.
	disk uint64

	// scoped reports whether the command requires verified kernel limits.
	scoped bool
}

// plan prints launch settings as KEY=value lines.
//
// For a resource-limited command, planning may evict the launcher-owned build cache.
//
// Takes out (io.Writer) which receives the plan.
// Takes arguments ([]string) which holds the command and its arguments.
//
// Returns error which reports invalid options, failed resource planning or failed output.
func plan(ctx context.Context, out io.Writer, arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("_plan needs a command")
	}
	command := arguments[0]
	if command == "help" || slices.Contains([]string{"-h", "--help"}, command) {
		_, err := fmt.Fprintln(out, "SCOPED=0")
		return err
	}
	options, _, err := parse(command, arguments[1:])
	if err != nil {
		return err
	}
	repo, err := prepareState(&options)
	if err != nil {
		return err
	}
	launch, err := planLaunch(command, repo, &options)
	if err != nil {
		return err
	}
	if launch.scoped {
		if err := evictOwnedCache(ctx, launch); err != nil {
			return err
		}
	}
	_, err = io.WriteString(out, launch.render())
	return err
}

// planLaunch resolves the budget and scope for a command.
//
// Takes command (string) which names the campaign CLI subcommand.
// Takes repo (string) which is the repository root.
// Takes options (*flags) which holds the parsed command options.
//
// Returns launchPlan which holds the resolved budget.
// Returns error which reports invalid sizes, paths containing newlines or unreadable host
// limits.
func planLaunch(command, repo string, options *flags) (launchPlan, error) {
	if strings.ContainsRune(repo, '\n') || strings.ContainsRune(options.state, '\n') {
		return launchPlan{}, errLaunchPathNewline
	}
	launch := launchPlan{
		root: repo, state: options.state, enclosing: "", cpuList: nil, cpus: uint64(max(options.cpus, 1)),
		memory: 0, minFree: 0, disk: 0, scoped: false,
	}
	launch.scoped = slices.Contains(scopedCommands, command) ||
		(slices.Contains(sourceCommands, command) && slices.ContainsFunc(limitFlags, func(name string) bool { return options.set[name] }))
	var err error
	if launch.memory, err = parseByteSize(options.mem); err != nil {
		return launchPlan{}, fmt.Errorf("-mem: %w", err)
	}
	if launch.disk, err = parseByteSize(options.disk); err != nil {
		return launchPlan{}, fmt.Errorf("-disk: %w", err)
	}
	if launch.minFree, err = hostReserve(options.reserve); err != nil {
		return launchPlan{}, err
	}
	if !launch.scoped {
		return launch, nil
	}
	inherited, err := resource.Affinity()
	if err != nil {
		return launchPlan{}, err
	}
	launch.cpuList = inherited[:min(len(inherited), int(min(launch.cpus, math.MaxInt32)))]
	if launch.enclosing, err = resource.EnclosingLimit(); err != nil {
		return launchPlan{}, err
	}
	if strings.ContainsRune(launch.enclosing, '\n') {
		return launchPlan{}, errLaunchPathNewline
	}
	return launch, nil
}

// render writes the plan as KEY=value lines for the launcher to read.
//
// Returns string which holds one assignment per line.
func (p launchPlan) render() string {
	cpuList := make([]string, len(p.cpuList))
	for i, cpu := range p.cpuList {
		cpuList[i] = strconv.Itoa(cpu)
	}
	scoped := "0"
	if p.scoped {
		scoped = "1"
	}
	lines := []string{
		"SCOPED=" + scoped,
		"CPU_LIST=" + strings.Join(cpuList, ","),
		"CPU_QUOTA=" + strconv.Itoa(len(p.cpuList)*cpuQuotaPercentPerCore) + "%",
		"MEMORY_MAX=" + strconv.FormatUint(p.memory, 10),
		"MEMORY_HIGH=" + strconv.FormatUint(p.memory/100*memoryHighPercent, 10),
		"TASKS_MAX=" + strconv.Itoa(resource.TasksMax),
		resource.EnvPrefix + "ROOT=" + p.root,
		resource.EnvPrefix + "STATE=" + p.state,
		resource.EnvPrefix + "CPUS=" + strconv.FormatUint(p.cpus, 10),
		resource.EnvPrefix + "MEMORY_BYTES=" + strconv.FormatUint(p.memory, 10),
		resource.EnvPrefix + "MIN_FREE_BYTES=" + strconv.FormatUint(p.minFree, 10),
		resource.EnvPrefix + "DISK_BYTES=" + strconv.FormatUint(p.disk, 10),
		resource.EnvPrefix + "ENCLOSING_LIMIT=" + p.enclosing,
	}
	return strings.Join(lines, "\n") + "\n"
}

// hostReserve resolves the host memory reserve.
//
// Without an explicit size, the reserve is the larger of 4 GiB and a tenth of host
// memory.
//
// Takes requested (string) which is the -min-free value, or empty for the default.
//
// Returns uint64 which is the reserve in bytes.
// Returns error which reports an invalid size or unreadable host memory.
func hostReserve(requested string) (uint64, error) {
	if requested != "" {
		reserve, err := parseByteSize(requested)
		if err != nil {
			return 0, fmt.Errorf("-min-free: %w", err)
		}
		return reserve, nil
	}
	total, err := resource.MemInfo("MemTotal")
	if err != nil {
		return 0, err
	}
	return max(minimumHostReserve, total/hostReserveDivisor), nil
}

// parseByteSize parses a positive size using binary units.
//
// K, KB and KiB all mean 1024 bytes. Fractional bytes round down.
//
// Takes text (string) which is the size.
//
// Returns uint64 which is the size in bytes.
// Returns error which reports malformed, zero or out-of-range sizes.
func parseByteSize(text string) (uint64, error) {
	match := byteSizePattern.FindStringSubmatch(strings.TrimSpace(text))
	if match == nil {
		return 0, fmt.Errorf("invalid byte size %q, e.g. 4GiB", text)
	}
	value, ok := new(big.Rat).SetString(match[1])
	if !ok {
		return 0, fmt.Errorf("invalid byte size %q", text)
	}
	unit := strings.ToUpper(cmp.Or(match[2], "B"))[:1]
	multiplier := new(big.Int).Lsh(big.NewInt(1), uint(binaryUnitShift*strings.Index("BKMGT", unit)))
	bytes := new(big.Int).Quo(new(big.Int).Mul(value.Num(), multiplier), value.Denom())
	if bytes.Sign() <= 0 || !bytes.IsInt64() {
		return 0, fmt.Errorf("byte size %q is out of range", text)
	}
	return bytes.Uint64(), nil
}

// evictOwnedCache frees disk space by removing the launcher-owned Go build cache.
//
// The state tree must reach the disk budget minus its reserve. The cache needs the
// launcher ownership marker, and neither the cache nor its parent may be a symlink.
//
// Takes launch (launchPlan) which holds the state directory and disk budget.
//
// Returns error which reports an unreadable state tree or a failed removal.
func evictOwnedCache(ctx context.Context, launch launchPlan) error {
	cache := filepath.Join(launch.state, ".cache", "go-build")
	if _, err := os.Lstat(filepath.Join(cache, ownedCacheMarker)); err != nil {
		return nil
	}
	for _, directory := range []string{cache, filepath.Dir(cache)} {
		if info, err := os.Lstat(directory); err != nil || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
	}
	used, err := resource.DiskUsed(ctx, launch.state)
	if err != nil {
		return err
	}
	if used < launch.disk-min(resource.DiskReserve(launch.disk), launch.disk) {
		return nil
	}
	return os.RemoveAll(cache)
}
