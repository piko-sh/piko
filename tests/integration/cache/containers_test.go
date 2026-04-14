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
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
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
		env.rawClusterClient = redis.NewClusterClient(&redis.ClusterOptions{Addrs: env.redisClusterAddrs})
	}

	if endpoint := os.Getenv("DYNAMODB_ENDPOINT"); endpoint != "" {
		env.dynamoDBEndpoint = endpoint
	} else {
		localstackContainer, endpoint, err := startLocalStackContainer(ctx)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "WARN: could not start localstack container (dynamodb tests will be skipped): %v\n", err)
		} else {
			env.dynamoDBEndpoint = endpoint
			if localstackContainer != nil {
				cleanups = append(cleanups, func() { _ = localstackContainer.Terminate(context.Background()) })
			}
		}
	}

	if addr := os.Getenv("FIRESTORE_EMULATOR_HOST"); addr != "" {
		env.firestoreAddr = addr
	} else {
		firestoreContainer, addr, err := startFirestoreEmulatorContainer(ctx)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "WARN: could not start firestore emulator (firestore tests will be skipped): %v\n", err)
		} else {
			env.firestoreAddr = addr
			if firestoreContainer != nil {
				cleanups = append(cleanups, func() { _ = firestoreContainer.Terminate(context.Background()) })
			}
		}
	}

	env.cleanup = func() {
		if env.rawClient != nil {
			_ = env.rawClient.Close()
		}
		if env.rawClusterClient != nil {
			_ = env.rawClusterClient.Close()
		}
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
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

	portBindings := make(nat.PortMap, len(clusterPorts))
	for _, p := range clusterPorts {
		portBindings[nat.Port(p+"/tcp")] = []nat.PortBinding{
			{HostIP: "127.0.0.1", HostPort: p},
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

	time.Sleep(2 * time.Second)

	return ctr, addrs, nil
}

func startLocalStackContainer(ctx context.Context) (testcontainers.Container, string, error) {
	request := testcontainers.ContainerRequest{
		Image:        "localstack/localstack:latest",
		ExposedPorts: []string{"4566/tcp"},
		Env: map[string]string{
			"SERVICES":              "dynamodb",
			"EAGER_SERVICE_LOADING": "1",
		},
		WaitingFor: wait.ForHTTP("/_localstack/health").
			WithPort("4566/tcp").
			WithStatusCodeMatcher(func(status int) bool { return status == 200 }).
			WithStartupTimeout(120 * time.Second),
	}

	genericContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("creating localstack container: %w", err)
	}

	host, err := genericContainer.Host(ctx)
	if err != nil {
		_ = genericContainer.Terminate(ctx)
		return nil, "", fmt.Errorf("getting host: %w", err)
	}

	port, err := genericContainer.MappedPort(ctx, "4566/tcp")
	if err != nil {
		_ = genericContainer.Terminate(ctx)
		return nil, "", fmt.Errorf("getting port: %w", err)
	}

	endpoint := fmt.Sprintf("http://%s:%s", host, port.Port())
	return genericContainer, endpoint, nil
}

func startFirestoreEmulatorContainer(ctx context.Context) (testcontainers.Container, string, error) {
	request := testcontainers.ContainerRequest{
		Image:        "gcr.io/google.com/cloudsdktool/google-cloud-cli:emulators",
		ExposedPorts: []string{"8080/tcp"},
		Cmd:          []string{"gcloud", "beta", "emulators", "firestore", "start", "--host-port=0.0.0.0:8080"},
		WaitingFor: wait.ForLog("Dev App Server is now running").
			WithStartupTimeout(120 * time.Second),
	}

	genericContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("creating firestore emulator container: %w", err)
	}

	host, err := genericContainer.Host(ctx)
	if err != nil {
		_ = genericContainer.Terminate(ctx)
		return nil, "", fmt.Errorf("getting host: %w", err)
	}

	port, err := genericContainer.MappedPort(ctx, "8080/tcp")
	if err != nil {
		_ = genericContainer.Terminate(ctx)
		return nil, "", fmt.Errorf("getting port: %w", err)
	}

	addr := fmt.Sprintf("%s:%s", host, port.Port())
	return genericContainer, addr, nil
}
