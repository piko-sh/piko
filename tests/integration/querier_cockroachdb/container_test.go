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

package querier_cockroachdb_test

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	cockroachContainer   testcontainers.Container
	testConnectionString string
)

func startCockroachDBContainer(ctx context.Context) (testcontainers.Container, string, error) {
	request := testcontainers.ContainerRequest{
		Image:        "cockroachdb/cockroach:latest-v25.1",
		ExposedPorts: []string{"26257/tcp"},
		Cmd:          []string{"start-single-node", "--insecure"},
		WaitingFor: wait.ForLog("CockroachDB node starting").
			WithStartupTimeout(120 * time.Second),
	}

	genericContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("creating cockroachdb container: %w", err)
	}

	host, err := genericContainer.Host(ctx)
	if err != nil {
		_ = genericContainer.Terminate(ctx)
		return nil, "", fmt.Errorf("getting host: %w", err)
	}

	port, err := genericContainer.MappedPort(ctx, "26257/tcp")
	if err != nil {
		_ = genericContainer.Terminate(ctx)
		return nil, "", fmt.Errorf("getting port: %w", err)
	}

	connectionString := fmt.Sprintf("postgresql://root@%s:%s/defaultdb?sslmode=disable", host, port.Port())
	return genericContainer, connectionString, nil
}
