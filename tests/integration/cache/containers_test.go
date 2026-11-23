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

//go:build integration

package cache_integration_test

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestEnvironment(ctx context.Context) (*testEnv, error) {
	env := &testEnv{}
	var cleanups []func()

	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		env.redisAddr = addr
	} else {
		redisContainer, addr, err := startRedisContainer(ctx)
		if err != nil {
			return nil, fmt.Errorf("starting redis redisContainer: %w", err)
		}
		env.redisAddr = addr
		if redisContainer != nil {
			cleanups = append(cleanups, func() { _ = redisContainer.Terminate(context.Background()) })
		}
	}

	env.rawClient = redis.NewClient(&redis.Options{Addr: env.redisAddr})

	if addr := os.Getenv("VALKEY_ADDR"); addr != "" {
		env.valkeyAddr = addr
	} else {
		valkeyContainer, addr, err := startValkeyContainer(ctx)
		if err != nil {

			_, _ = fmt.Fprintf(os.Stderr, "WARN: could not start valkey valkeyContainer (valkey tests will be skipped): %v\n", err)
		} else {
			env.valkeyAddr = addr
			if valkeyContainer != nil {
				cleanups = append(cleanups, func() { _ = valkeyContainer.Terminate(context.Background()) })
			}
		}
	}

	if addrs := os.Getenv("REDIS_CLUSTER_ADDRS"); addrs != "" {
		env.redisClusterAddrs = strings.Split(addrs, ",")
	} else {
		redisClusterContainer, addrs, err := startRedisClusterContainer(ctx)
		if err != nil {

			_, _ = fmt.Fprintf(os.Stderr, "WARN: could not start redis cluster redisClusterContainer (cluster tests will be skipped): %v\n", err)
		} else {
			env.redisClusterAddrs = addrs
			if redisClusterContainer != nil {
				cleanups = append(cleanups, func() { _ = redisClusterContainer.Terminate(context.Background()) })
			}
		}
	}

	if len(env.redisClusterAddrs) > 0 {
		if err := waitForClusterReady(ctx, env.redisClusterAddrs, clusterReadyTimeout); err != nil {
			for _, cleanup := range slices.Backward(cleanups) {
				cleanup()
			}
			_ = env.rawClient.Close()
			return nil, fmt.Errorf("waiting for the redis cluster at %s: %w", strings.Join(env.redisClusterAddrs, ","), err)
		}
		env.rawClusterClient = redis.NewClusterClient(&redis.ClusterOptions{Addrs: env.redisClusterAddrs})
	}

	env.cleanup = func() {
		if env.rawClient != nil {
			_ = env.rawClient.Close()
		}
		if env.rawClusterClient != nil {
			_ = env.rawClusterClient.Close()
		}
		for _, cleanup := range slices.Backward(cleanups) {
			cleanup()
		}
	}

	return env, nil
}

func startRedisContainer(ctx context.Context) (testcontainers.Container, string, error) {
	request := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor: wait.ForLog("Ready to accept connections").
			WithStartupTimeout(60 * time.Second),
	}

	genericContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("creating genericContainer: %w", err)
	}

	host, err := genericContainer.Host(ctx)
	if err != nil {
		_ = genericContainer.Terminate(ctx)
		return nil, "", fmt.Errorf("getting host: %w", err)
	}

	port, err := genericContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		_ = genericContainer.Terminate(ctx)
		return nil, "", fmt.Errorf("getting port: %w", err)
	}

	addr := fmt.Sprintf("%s:%s", host, port.Port())
	return genericContainer, addr, nil
}

func startValkeyContainer(ctx context.Context) (testcontainers.Container, string, error) {
	request := testcontainers.ContainerRequest{
		Image:        "valkey/valkey:8-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor: wait.ForLog("Ready to accept connections").
			WithStartupTimeout(60 * time.Second),
	}

	genericContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("creating valkey genericContainer: %w", err)
	}

	host, err := genericContainer.Host(ctx)
	if err != nil {
		_ = genericContainer.Terminate(ctx)
		return nil, "", fmt.Errorf("getting host: %w", err)
	}

	port, err := genericContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		_ = genericContainer.Terminate(ctx)
		return nil, "", fmt.Errorf("getting port: %w", err)
	}

	addr := fmt.Sprintf("%s:%s", host, port.Port())
	return genericContainer, addr, nil
}

func startRedisClusterContainer(ctx context.Context) (testcontainers.Container, []string, error) {
	clusterPorts := []string{"7000", "7001", "7002", "7003", "7004", "7005"}

	exposedPorts := make([]string, 0, len(clusterPorts))
	for _, p := range clusterPorts {
		exposedPorts = append(exposedPorts, p+"/tcp")
	}

	hostIP := netip.MustParseAddr("127.0.0.1")
	portBindings := make(network.PortMap, len(clusterPorts))
	for _, p := range clusterPorts {
		portBindings[network.MustParsePort(p+"/tcp")] = []network.PortBinding{
			{HostIP: hostIP, HostPort: p},
		}
	}

	request := testcontainers.ContainerRequest{
		Image:        "grokzen/redis-cluster:7.0.15",
		ExposedPorts: exposedPorts,
		Env: map[string]string{
			"IP":                  "0.0.0.0",
			"INITIAL_PORT":        "7000",
			"MASTERS":             "3",
			"SLAVES_PER_MASTER":   "1",
			"SENTINEL":            "false",
			"REDIS_CLUSTER_IP":    "0.0.0.0",
			"BIND_ADDRESS":        "0.0.0.0",
			"CLUSTER_ANNOUNCE_IP": "127.0.0.1",
		},
		WaitingFor: wait.ForLog("Cluster state changed: ok").
			WithStartupTimeout(120 * time.Second),
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.PortBindings = portBindings
		},
	}

	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating cluster container: %w", err)
	}

	addrs := make([]string, 0, len(clusterPorts))
	for _, p := range clusterPorts {
		addrs = append(addrs, "127.0.0.1:"+p)
	}

	return ctr, addrs, nil
}

const (
	clusterSlotCount    = 16384
	clusterReadyTimeout = 120 * time.Second
)

type clusterNodeState struct {
	err        error
	state      string
	slotsOK    int
	knownNodes int
}

func (s clusterNodeState) ready(want int) bool {
	return s.err == nil && s.state == "ok" && s.slotsOK == clusterSlotCount && s.knownNodes >= want
}

func (s clusterNodeState) String() string {
	if s.err != nil {
		return "error: " + s.err.Error()
	}
	return fmt.Sprintf("cluster_state:%s cluster_slots_ok:%d cluster_known_nodes:%d", s.state, s.slotsOK, s.knownNodes)
}

func waitForClusterReady(ctx context.Context, addrs []string, timeout time.Duration) error {
	clients := make([]*redis.Client, len(addrs))
	for index, addr := range addrs {
		clients[index] = redis.NewClient(&redis.Options{Addr: addr, DialTimeout: 2 * time.Second, ReadTimeout: 2 * time.Second})
	}
	defer func() {
		for _, client := range clients {
			_ = client.Close()
		}
	}()

	ctx, cancel := context.WithTimeoutCause(ctx, timeout, fmt.Errorf("cluster not ready after %s", timeout))
	defer cancel()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	states := make([]clusterNodeState, len(addrs))
	for {
		allReady := true
		for index, client := range clients {
			states[index] = readClusterNodeState(ctx, client)
			allReady = allReady && states[index].ready(len(addrs))
		}
		if allReady {
			return nil
		}

		select {
		case <-ctx.Done():
			report := make([]string, len(addrs))
			for index, addr := range addrs {
				report[index] = addr + " " + states[index].String()
			}
			return fmt.Errorf("%w; node states: %s", context.Cause(ctx), strings.Join(report, "; "))
		case <-ticker.C:
		}
	}
}

func readClusterNodeState(ctx context.Context, client *redis.Client) clusterNodeState {
	info, err := client.ClusterInfo(ctx).Result()
	if err != nil {
		return clusterNodeState{err: err}
	}
	var state clusterNodeState
	for line := range strings.SplitSeq(info, "\r\n") {
		name, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		var parseErr error
		switch name {
		case "cluster_state":
			state.state = value
		case "cluster_slots_ok":
			state.slotsOK, parseErr = strconv.Atoi(value)
		case "cluster_known_nodes":
			state.knownNodes, parseErr = strconv.Atoi(value)
		}
		if parseErr != nil {
			return clusterNodeState{err: fmt.Errorf("parsing %s: %w", name, parseErr)}
		}
	}
	return state
}
