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

package monitoring_domain

import (
	"bufio"
	"cmp"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safeconv"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// ResourceCategoryFile is the category for regular resources.
	ResourceCategoryFile = "file"

	// ResourceCategoryTCP is the category for TCP socket resources.
	ResourceCategoryTCP = "tcp"

	// ResourceCategoryUDP is the category for UDP socket resources.
	ResourceCategoryUDP = "udp"

	// ResourceCategoryUnix is the category for Unix domain sockets.
	ResourceCategoryUnix = "unix"

	// ResourceCategoryPipe is the category label for pipe resources.
	ResourceCategoryPipe = "pipe"

	// ResourceCategorySocket is the category for network socket file
	// descriptors.
	ResourceCategorySocket = "socket"

	// ResourceCategoryOther is the category for resources that do not
	// match other categories, such as anonymous inodes.
	ResourceCategoryOther = "other"

	// resourceScanInterval is the time between background resource scans.
	resourceScanInterval = 2 * time.Second

	// procSelfFileDescriptor is the path to the directory listing open file
	// descriptors for the current process.
	procSelfFileDescriptor = "/proc/self/fd"

	// procSelfNet is the path to the network information for the current process.
	procSelfNet = "/proc/self/net"
)

// resourceInfo holds details about a single resource.
type resourceInfo struct {
	// FirstSeen is when the resource was first observed.
	FirstSeen time.Time

	// Category is the classification of the resource (e.g. socket, pipe).
	Category string

	// Target is the destination path or resource this resource points to.
	Target string

	// Number int // Number is the resource number.
	Number int
}

// ResourceCollector collects information about open resources.
// It implements ResourceProvider.
type ResourceCollector struct {
	// sandboxFactory creates sandboxes when needed. When non-nil, this factory
	// is used instead of safedisk.NewSandbox.
	sandboxFactory safedisk.Factory

	// clock provides time operations for testing; defaults to real time.
	clock clock.Clock

	// firstSeen tracks when each resource was first observed.
	firstSeen map[int]time.Time

	// stopCh signals the background scan goroutine to stop.
	stopCh chan struct{}

	// mu guards concurrent access to the collector's data.
	mu sync.RWMutex

	// stopped indicates whether the collector has been stopped.
	stopped bool
}

// ResourceCollectorOption is a function type that configures a
// ResourceCollector.
type ResourceCollectorOption func(*ResourceCollector)

// NewResourceCollector creates a new resource collector.
//
// Takes opts (...ResourceCollectorOption) which configures the collector
// behaviour.
//
// Returns *ResourceCollector which is ready to collect resource
// information.
func NewResourceCollector(opts ...ResourceCollectorOption) *ResourceCollector {
	c := &ResourceCollector{
		clock:     nil,
		firstSeen: make(map[int]time.Time),
		stopCh:    make(chan struct{}),
		mu:        sync.RWMutex{},
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.clock == nil {
		c.clock = clock.RealClock()
	}

	return c
}

// Start begins periodic scanning of resources to keep firstSeen
// timestamps accurate.
//
// Spawns a background goroutine that runs until the context is cancelled
// or Stop is called.
func (c *ResourceCollector) Start(ctx context.Context) {
	c.scanFirstSeen()

	go c.loop(ctx)
}

// Stop stops the background scan goroutine.
//
// Safe for concurrent use. Calling Stop multiple times is safe; only the first
// call closes the stop channel.
func (c *ResourceCollector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.stopped {
		close(c.stopCh)
		c.stopped = true
	}
}

var (
	// socketPattern matches socket:[inode] format.
	socketPattern = regexp.MustCompile(`^socket:\[(\d+)\]$`)

	// pipePattern matches pipe:[inode] format.
	pipePattern = regexp.MustCompile(`^pipe:\[(\d+)\]$`)

	// resourceCategoryOrder defines the order in which resource categories are
	// returned.
	resourceCategoryOrder = []string{
		ResourceCategoryFile,
		ResourceCategoryTCP,
		ResourceCategoryUDP,
		ResourceCategoryUnix,
		ResourceCategoryPipe,
		ResourceCategorySocket,
		ResourceCategoryOther,
	}

	_ ResourceProvider = (*ResourceCollector)(nil)
)

// GetResources returns current resource information in domain
// format. Implements ResourceProvider.
//
// Returns ResourceData which contains categorised resource
// information with totals and timestamp.
//
// Safe for concurrent use; protected by a mutex.
func (c *ResourceCollector) GetResources() ResourceData {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.clock.Now()
	entries, err := os.ReadDir(procSelfFileDescriptor)
	if err != nil {
		return ResourceData{
			Categories:  []ResourceCategory{},
			Total:       0,
			TimestampMs: now.UnixMilli(),
		}
	}

	resourcesByCategory, currentResources := c.collectResourceInfo(entries, now)
	c.cleanupStaleFirstSeen(currentResources)
	categories, total := c.buildCategoriesResponse(resourcesByCategory, now)

	return ResourceData{
		Categories:  categories,
		Total:       safeconv.IntToInt32(total),
		TimestampMs: now.UnixMilli(),
	}
}

// loop runs the periodic firstSeen scan until stopped.
//
// Runs until the context is cancelled or Stop is called.
func (c *ResourceCollector) loop(ctx context.Context) {
	ticker := c.clock.NewTicker(resourceScanInterval)
	defer ticker.Stop()
	defer goroutine.RecoverPanic(ctx, "monitoring.resourceCollectorLoop")

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C():
			c.scanFirstSeen()
		}
	}
}

// scanFirstSeen performs a lightweight scan of /proc/self/fd to record when
// resources are first observed. Unlike GetResources, this does
// not resolve symlinks or categorise descriptors.
//
// Safe for concurrent use; protected by the collector's mutex.
func (c *ResourceCollector) scanFirstSeen() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.clock.Now()

	entries, err := os.ReadDir(procSelfFileDescriptor)
	if err != nil {
		return
	}

	currentFDs := make(map[int]struct{}, len(entries))

	for _, entry := range entries {
		number, err := strconv.Atoi(entry.Name())
		if err != nil || number <= 2 {
			continue
		}

		currentFDs[number] = struct{}{}

		if _, ok := c.firstSeen[number]; !ok {
			c.firstSeen[number] = now
		}
	}

	c.cleanupStaleFirstSeen(currentFDs)
}

// collectResourceInfo reads resource entries and categorises them.
//
// Takes entries ([]os.DirEntry) which contains the directory entries to
// process.
// Takes now (time.Time) which is the current time for first-seen tracking.
//
// Returns map[string][]resourceInfo which maps category names to their
// resource details.
// Returns map[int]struct{} which contains the set of currently active
// resource numbers.
func (c *ResourceCollector) collectResourceInfo(entries []os.DirEntry, now time.Time) (map[string][]resourceInfo, map[int]struct{}) {
	resourcesByCategory := make(map[string][]resourceInfo)
	currentResources := make(map[int]struct{})

	for _, entry := range entries {
		info := c.processResourceEntry(entry, now)
		if info == nil {
			continue
		}

		currentResources[info.Number] = struct{}{}
		resourcesByCategory[info.Category] = append(resourcesByCategory[info.Category], *info)
	}

	return resourcesByCategory, currentResources
}

// processResourceEntry processes a single resource directory entry.
//
// Takes entry (os.DirEntry) which is the directory entry to process.
// Takes now (time.Time) which is the current time for tracking first seen.
//
// Returns *resourceInfo which contains the descriptor details, or nil if the
// entry
// is invalid, is a standard stream (stdin/stdout/stderr), or cannot be read.
func (c *ResourceCollector) processResourceEntry(entry os.DirEntry, now time.Time) *resourceInfo {
	number, err := strconv.Atoi(entry.Name())
	if err != nil {
		return nil
	}

	if number <= 2 {
		return nil
	}

	target, err := os.Readlink(filepath.Join(procSelfFileDescriptor, entry.Name()))
	if err != nil {
		return nil
	}

	firstSeen, ok := c.firstSeen[number]
	if !ok {
		firstSeen = now
		c.firstSeen[number] = firstSeen
	}

	return &resourceInfo{
		Number:    number,
		Category:  c.categoriseResource(target),
		Target:    target,
		FirstSeen: firstSeen,
	}
}

// cleanupStaleFirstSeen removes entries for resources that no longer
// exist.
//
// Takes currentResources (map[int]struct{}) which contains the set of
// currently active resource numbers to check against.
func (c *ResourceCollector) cleanupStaleFirstSeen(currentResources map[int]struct{}) {
	for number := range c.firstSeen {
		if _, exists := currentResources[number]; !exists {
			delete(c.firstSeen, number)
		}
	}
}

// buildCategoriesResponse builds the categories response in consistent order.
//
// Takes resourcesByCategory (map[string][]resourceInfo) which maps category
// names to their resource information.
// Takes now (time.Time) which is the current time for age calculations.
//
// Returns []ResourceCategory which contains the sorted categories with
// their resources.
// Returns int which is the total count of resources across all
// categories.
func (*ResourceCollector) buildCategoriesResponse(resourcesByCategory map[string][]resourceInfo, now time.Time) ([]ResourceCategory, int) {
	categories := make([]ResourceCategory, 0, len(resourceCategoryOrder))
	total := 0

	for _, cat := range resourceCategoryOrder {
		resources := resourcesByCategory[cat]
		if len(resources) == 0 {
			continue
		}

		sortResourcesByAge(resources)

		domainFDs := make([]ResourceInfo, len(resources))
		for i, fd := range resources {
			domainFDs[i] = ResourceInfo{
				FD:          safeconv.IntToInt32(fd.Number),
				Category:    fd.Category,
				Target:      fd.Target,
				FirstSeenMs: fd.FirstSeen.UnixMilli(),
				AgeMs:       now.Sub(fd.FirstSeen).Milliseconds(),
			}
		}

		categories = append(categories, ResourceCategory{
			Category:  cat,
			Count:     safeconv.IntToInt32(len(resources)),
			Resources: domainFDs,
		})
		total += len(resources)
	}

	return categories, total
}

// categoriseResource determines the category of a resource based
// on its target.
//
// Takes target (string) which is the resource target path to categorise.
//
// Returns string which is the category constant for the resource type.
func (c *ResourceCollector) categoriseResource(target string) string {
	if pipePattern.MatchString(target) {
		return ResourceCategoryPipe
	}

	if socketPattern.MatchString(target) {
		return cmp.Or(c.lookupSocketType(target), ResourceCategorySocket)
	}

	if strings.HasPrefix(target, "anon_inode:") {
		return ResourceCategoryOther
	}

	if strings.HasPrefix(target, "/") {
		return ResourceCategoryFile
	}

	return ResourceCategoryOther
}

// lookupSocketType attempts to determine if a socket is TCP, UDP, or Unix.
//
// Takes target (string) which is the socket path to analyse.
//
// Returns string which is the socket category, or empty if not determined.
func (c *ResourceCollector) lookupSocketType(target string) string {
	matches := socketPattern.FindStringSubmatch(target)
	if len(matches) < 2 {
		return ""
	}

	inode := matches[1]

	if c.inodeInNetFile("tcp", inode) || c.inodeInNetFile("tcp6", inode) {
		return ResourceCategoryTCP
	}

	if c.inodeInNetFile("udp", inode) || c.inodeInNetFile("udp6", inode) {
		return ResourceCategoryUDP
	}

	if c.inodeInNetFile("unix", inode) {
		return ResourceCategoryUnix
	}

	return ""
}

// inodeInNetFile checks if an inode appears in a /proc/self/net/* file.
//
// Takes filename (string) which is the path relative to /proc/self/net
// (e.g., "tcp", "tcp6").
// Takes inode (string) which is the inode number to search for.
//
// Returns bool which is true if the inode is found in the file.
func (c *ResourceCollector) inodeInNetFile(filename, inode string) bool {
	var sandbox safedisk.Sandbox
	var err error
	if c.sandboxFactory != nil {
		sandbox, err = c.sandboxFactory.Create("proc-net", procSelfNet, safedisk.ModeReadOnly)
	} else {
		sandbox, err = safedisk.NewSandbox(procSelfNet, safedisk.ModeReadOnly)
	}
	if err != nil {
		return false
	}
	defer func() { _ = sandbox.Close() }()

	file, err := sandbox.Open(filename)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return false
	}

	for scanner.Scan() {
		for field := range strings.FieldsSeq(scanner.Text()) {
			if field == inode {
				return true
			}
		}
	}

	return false
}

// WithResourceCollectorClock sets the clock for the ResourceCollector.
//
// Takes clk (clock.Clock) which provides the time source for the collector.
//
// Returns ResourceCollectorOption which configures the collector's clock.
func WithResourceCollectorClock(clk clock.Clock) ResourceCollectorOption {
	return func(c *ResourceCollector) {
		c.clock = clk
	}
}

// WithResourceCollectorSandboxFactory sets a factory for creating sandboxes
// when reading /proc/self/net files.
//
// Takes factory (safedisk.Factory) which creates sandboxes for network file
// access.
//
// Returns ResourceCollectorOption which configures the collector's factory.
func WithResourceCollectorSandboxFactory(factory safedisk.Factory) ResourceCollectorOption {
	return func(c *ResourceCollector) {
		c.sandboxFactory = factory
	}
}

// sortResourcesByAge sorts resources by age (oldest first) using
// insertion sort for small slices.
//
// Takes resources ([]resourceInfo) which is the slice to sort in place.
func sortResourcesByAge(resources []resourceInfo) {
	for i := 1; i < len(resources); i++ {
		for j := i; j > 0 && resources[j].FirstSeen.Before(resources[j-1].FirstSeen); j-- {
			resources[j], resources[j-1] = resources[j-1], resources[j]
		}
	}
}
