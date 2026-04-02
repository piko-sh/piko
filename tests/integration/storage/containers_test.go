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

package storage_integration_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testBucketPrimary   = "test-storage-primary"
	testBucketSecondary = "test-storage-secondary"
	testRepoPrimary     = "primary"
	testRepoSecondary   = "secondary"
	testRegion          = "us-east-1"
	testAccessKey       = "test"
	testSecretKey       = "test"
)

type testEnv struct {
	container   testcontainers.Container
	endpointURL string
	s3Client    *s3.Client
	cleanup     func()
}

func setupTestEnvironment(ctx context.Context) (*testEnv, error) {

	if endpoint := os.Getenv("LOCALSTACK_ENDPOINT"); endpoint != "" {
		s3Client, err := newDirectS3Client(ctx, endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to create S3 client for existing endpoint: %w", err)
		}

		if err := createTestBuckets(ctx, s3Client); err != nil {
			return nil, err
		}

		return &testEnv{
			endpointURL: endpoint,
			s3Client:    s3Client,
			cleanup:     nil,
		}, nil
	}

	request := testcontainers.ContainerRequest{
		Image:        "localstack/localstack:latest",
		ExposedPorts: []string{"4566/tcp"},
		Env: map[string]string{
			"SERVICES":              "s3",
			"DEFAULT_REGION":        testRegion,
			"AWS_ACCESS_KEY_ID":     testAccessKey,
			"AWS_SECRET_ACCESS_KEY": testSecretKey,
		},
		WaitingFor: wait.ForHTTP("/_localstack/health").
			WithPort("4566/tcp").
			WithStatusCodeMatcher(func(status int) bool {
				return status == http.StatusOK
			}).
			WithStartupTimeout(120 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start LocalStack container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "4566/tcp")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	endpointURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	s3Client, err := newDirectS3Client(ctx, endpointURL)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	if err := createTestBuckets(ctx, s3Client); err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	return &testEnv{
		container:   container,
		endpointURL: endpointURL,
		s3Client:    s3Client,
		cleanup: func() {
			_ = container.Terminate(context.Background())
		},
	}, nil
}

func newDirectS3Client(ctx context.Context, endpointURL string) (*s3.Client, error) {
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpointURL)
		o.UsePathStyle = true
	}), nil
}

func createTestBuckets(ctx context.Context, client *s3.Client) error {
	for _, bucket := range []string{testBucketPrimary, testBucketSecondary} {
		_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucket),
		})
		if err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
		}
	}
	return nil
}
