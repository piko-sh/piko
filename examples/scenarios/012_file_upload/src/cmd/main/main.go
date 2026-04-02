package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "testmodule/dist"

	"piko.sh/piko"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/storage/storage_provider_s3"
)

const (
	bucketName = "uploads"
	region     = "us-east-1"
	accessKey  = "test"
	secretKey  = "test"
)

func main() {
	logger.AddPrettyOutput()

	ctx := context.Background()

	// Start a LocalStack container with S3 support.
	fmt.Println("[file-upload] Starting LocalStack container...")
	container, endpoint := startLocalStack(ctx)
	defer func() {
		fmt.Println("[file-upload] Stopping LocalStack container...")
		_ = container.Terminate(ctx)
	}()
	fmt.Printf("[file-upload] LocalStack ready at %s\n", endpoint)

	// Create the uploads bucket.
	createBucket(ctx, endpoint)

	// Create an S3 storage provider pointing at LocalStack.
	s3Provider, err := storage_provider_s3.NewS3Provider(ctx, &storage_provider_s3.Config{
		RepositoryMappings: map[string]string{
			"uploads": bucketName,
		},
		Region:          region,
		AccessKey:       accessKey,
		SecretKey:       secretKey,
		EndpointURL:     endpoint,
		UsePathStyle:    true,
		DisableChecksum: true,
	})
	if err != nil {
		panic(fmt.Sprintf("creating S3 provider: %v", err))
	}

	// Allow uploads up to 50 MB (default is 1 MB).
	if os.Getenv("PIKO_ACTION_MAX_BODY_BYTES") == "" {
		_ = os.Setenv("PIKO_ACTION_MAX_BODY_BYTES", "52428800")
	}

	command := piko.RunModeDev
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	// Build presign base URL so browser can PUT to presigned URLs.
	port := os.Getenv("PIKO_PORT")
	if port == "" {
		port = "8080"
	}
	presignBase := fmt.Sprintf("http://localhost:%s", port)

	ssr := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),
		piko.WithStorageProvider("s3", s3Provider),
		piko.WithStoragePresignBaseURL(presignBase),
		piko.WithCSP(func(b *piko.CSPBuilder) {
			b.WithPikoDefaults().
				ConnectSrc(piko.CSPSelf, piko.CSPHost(endpoint))
		}),
		piko.WithDevWidget(),
		piko.WithDevHotreload(),
		piko.WithMonitoring(),
	)
	if err := ssr.Run(command); err != nil {
		panic(err)
	}
}

// startLocalStack starts a LocalStack container with S3.
func startLocalStack(ctx context.Context) (testcontainers.Container, string) {
	request := testcontainers.ContainerRequest{
		Image:        "localstack/localstack:latest",
		ExposedPorts: []string{"4566/tcp"},
		Env: map[string]string{
			"SERVICES":              "s3",
			"DEFAULT_REGION":        region,
			"AWS_ACCESS_KEY_ID":     accessKey,
			"AWS_SECRET_ACCESS_KEY": secretKey,
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
		panic(fmt.Sprintf("starting LocalStack: %v", err))
	}

	host, err := container.Host(ctx)
	if err != nil {
		panic(fmt.Sprintf("getting container host: %v", err))
	}

	port, err := container.MappedPort(ctx, "4566/tcp")
	if err != nil {
		panic(fmt.Sprintf("getting container port: %v", err))
	}

	return container, fmt.Sprintf("http://%s:%s", host, port.Port())
}

// createBucket creates the uploads bucket in LocalStack S3.
func createBucket(ctx context.Context, endpoint string) {
	awsConfig, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("loading AWS config: %v", err))
	}

	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		panic(fmt.Sprintf("creating bucket: %v", err))
	}

	// Allow cross-origin PUT requests so the browser can upload directly
	// to LocalStack via presigned URLs.
	_, err = client.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: aws.String(bucketName),
		CORSConfiguration: &s3types.CORSConfiguration{
			CORSRules: []s3types.CORSRule{{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "PUT", "POST", "HEAD"},
				AllowedHeaders: []string{"*"},
				ExposeHeaders:  []string{"ETag"},
				MaxAgeSeconds:  aws.Int32(3600),
			}},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("setting bucket CORS: %v", err))
	}

	fmt.Printf("[file-upload] Created bucket '%s'\n", bucketName)
}
