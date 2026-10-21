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

package netutil

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// tcpListenState is the state /proc/net/tcp reports for a listening socket.
	tcpListenState = "0A"

	// procRoot is the root of the proc filesystem.
	procRoot = "/proc"

	// tcpTableMinFields is the fewest fields a /proc/net/tcp row holds up to its inode.
	tcpTableMinFields = 10

	// tcpTableStateField is the index of a /proc/net/tcp row's socket state.
	tcpTableStateField = 3

	// tcpTableInodeField is the index of a /proc/net/tcp row's socket inode.
	tcpTableInodeField = 9
)

// PortHolder describes the process listening on a TCP port.
//
// It uses /proc and only finds processes the caller may inspect. Other platforms return
// no description.
//
// Takes port (int) which is the TCP port.
//
// Returns string which describes the holder, or "" when it cannot be found.
func PortHolder(port int) string {
	inodes := listeningInodes(port)
	if len(inodes) == 0 {
		return ""
	}
	pid := socketOwner(inodes)
	if pid == 0 {
		return ""
	}
	return describeProcess(pid)
}

// listeningInodes returns the socket inodes listening on port, from /proc/net/tcp and
// /proc/net/tcp6.
//
// Takes port (int) which is the TCP port.
//
// Returns map[string]struct{} which holds the inodes, empty when none are found.
func listeningInodes(port int) map[string]struct{} {
	inodes := make(map[string]struct{})
	wantPort := fmt.Sprintf(":%04X", port)
	for _, table := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		file, err := os.Open(table) //nolint:gosec // table is one of the two fixed /proc/net paths above
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < tcpTableMinFields || fields[tcpTableStateField] != tcpListenState ||
				!strings.HasSuffix(fields[1], wantPort) {
				continue
			}
			inodes[fields[tcpTableInodeField]] = struct{}{}
		}
		_ = file.Close()
	}
	return inodes
}

// socketOwner finds a process holding one of the socket inodes open.
//
// Takes inodes (map[string]struct{}) which are the sockets to look for.
//
// Returns int which is the owning process ID, or 0 when none is visible.
func socketOwner(inodes map[string]struct{}) int {
	processes, err := os.ReadDir(procRoot)
	if err != nil {
		return 0
	}
	for _, process := range processes {
		pid, err := strconv.Atoi(process.Name())
		if err != nil {
			continue
		}
		fdDir := filepath.Join(procRoot, process.Name(), "fd")
		descriptors, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		for _, descriptor := range descriptors {
			target, err := os.Readlink(filepath.Join(fdDir, descriptor.Name()))
			if err != nil || !strings.HasPrefix(target, "socket:[") {
				continue
			}
			if _, ok := inodes[strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")]; ok {
				return pid
			}
		}
	}
	return 0
}

// describeProcess names a process by its ID and command line.
//
// Takes pid (int) which is the process ID.
//
// Returns string which is like `pid 4242 (piko dev)`.
func describeProcess(pid int) string {
	data, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "cmdline"))
	if err != nil || len(data) == 0 {
		return fmt.Sprintf("pid %d", pid)
	}
	args := strings.Split(strings.TrimRight(string(data), "\x00"), "\x00")
	args[0] = filepath.Base(args[0])
	const maxArgs = 4
	if len(args) > maxArgs {
		args = append(args[:maxArgs], "...")
	}
	return fmt.Sprintf("pid %d (%s)", pid, strings.Join(args, " "))
}
