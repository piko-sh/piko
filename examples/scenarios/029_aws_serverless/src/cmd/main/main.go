package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "testmodule/dist"

	"piko.sh/piko"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
	"piko.sh/piko/wdk/cache/cache_provider_dynamodb"
	"piko.sh/piko/wdk/crypto"
	"piko.sh/piko/wdk/crypto/crypto_provider_aws_kms"
	"piko.sh/piko/wdk/email"
	"piko.sh/piko/wdk/email/email_provider_ses"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/storage"
	"piko.sh/piko/wdk/storage/storage_provider_s3"
)

const (
	bucketName    = "content-store"
	dynamoTable   = "piko_cache"
	region        = "us-east-1"
	accessKey     = "test"
	secretKey     = "test"
	senderAddress = "noreply@example.com"
)

func main() {
	logger.AddPrettyOutput()

	ctx := context.Background()

	// Start a single LocalStack container with DynamoDB, S3, SES, and KMS.
	fmt.Println("[aws-serverless] Starting LocalStack container...")
	container, endpoint := startLocalStack(ctx)
	defer func() {
		fmt.Println("[aws-serverless] Stopping LocalStack container...")
		_ = container.Terminate(ctx)
	}()
	fmt.Printf("[aws-serverless] LocalStack ready at %s\n", endpoint)

	// Create AWS resources inside LocalStack.
	awsCfg := buildAWSConfig(ctx)
	createDynamoDBTable(ctx, awsCfg, endpoint)
	createS3Bucket(ctx, awsCfg, endpoint)
	verifySESIdentity(ctx, awsCfg, endpoint)

	// Build providers for each AWS service.
	dynamoProvider := createDynamoDBProvider(endpoint)
	s3Provider := createS3Provider(ctx, endpoint)
	sesProvider := createSESProvider(ctx, endpoint)
	kmsKeyID := createKMSKey(ctx, awsCfg, endpoint)
	kmsProvider := createKMSProvider(ctx, endpoint, kmsKeyID)

	// Create typed DynamoDB caches for registry and orchestrator backends.
	registryCache := createRegistryCache(endpoint)
	orchestratorCache := createOrchestratorCache(endpoint)

	command := piko.RunModeDev
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	port := os.Getenv("PIKO_PORT")
	if port == "" {
		port = "8080"
	}
	presignBase := fmt.Sprintf("http://localhost:%s", port)

	ssr := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),

		// DynamoDB as the general-purpose cache backend.
		piko.WithCacheProvider("dynamodb", dynamoProvider),
		piko.WithDefaultCacheProvider("dynamodb"),

		// DynamoDB-backed registry and orchestrator (replaces default otter).
		piko.WithRegistryCache(registryCache),
		piko.WithOrchestratorCache(orchestratorCache),

		// S3 for file storage.
		piko.WithStorageProvider("s3", s3Provider),
		piko.WithStoragePresignBaseURL(presignBase),

		// SES for email.
		piko.WithEmailProvider("ses", sesProvider),
		piko.WithDefaultEmailProvider("ses"),

		// KMS for encryption.
		piko.WithCryptoProvider("kms", kmsProvider),
		piko.WithDefaultCryptoProvider("kms"),

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

// buildAWSConfig creates a shared AWS config with static credentials for
// LocalStack.
func buildAWSConfig(ctx context.Context) aws.Config {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("loading AWS config: %v", err))
	}
	return awsCfg
}

// startLocalStack starts a LocalStack container with DynamoDB, S3, SES, and
// KMS services.
func startLocalStack(ctx context.Context) (testcontainers.Container, string) {
	request := testcontainers.ContainerRequest{
		Image:        "localstack/localstack:latest",
		ExposedPorts: []string{"4566/tcp"},
		Env: map[string]string{
			"SERVICES":              "dynamodb,s3,ses,kms",
			"EAGER_SERVICE_LOADING": "1",
			"DEFAULT_REGION":        region,
			"AWS_ACCESS_KEY_ID":     accessKey,
			"AWS_SECRET_ACCESS_KEY": secretKey,
		},
		WaitingFor: wait.ForLog("Ready.").
			WithStartupTimeout(180 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		if container != nil {
			_ = container.Terminate(context.Background())
		}
		panic(fmt.Sprintf("starting LocalStack: %v", err))
	}

	host, err := container.Host(ctx)
	if err != nil {
		panic(fmt.Sprintf("getting container host: %v", err))
	}

	mappedPort, err := container.MappedPort(ctx, "4566/tcp")
	if err != nil {
		panic(fmt.Sprintf("getting container port: %v", err))
	}

	return container, fmt.Sprintf("http://%s:%s", host, mappedPort.Port())
}

// createDynamoDBTable creates the DynamoDB table used for Piko's cache
// backend.
func createDynamoDBTable(ctx context.Context, awsCfg aws.Config, endpoint string) {
	client := dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(dynamoTable),
		KeySchema: []dynamodbtypes.KeySchemaElement{
			{AttributeName: aws.String("pk"), KeyType: dynamodbtypes.KeyTypeHash},
			{AttributeName: aws.String("sk"), KeyType: dynamodbtypes.KeyTypeRange},
		},
		AttributeDefinitions: []dynamodbtypes.AttributeDefinition{
			{AttributeName: aws.String("pk"), AttributeType: dynamodbtypes.ScalarAttributeTypeS},
			{AttributeName: aws.String("sk"), AttributeType: dynamodbtypes.ScalarAttributeTypeS},
		},
		BillingMode: dynamodbtypes.BillingModePayPerRequest,
	})
	if err != nil {
		panic(fmt.Sprintf("creating DynamoDB table: %v", err))
	}
	fmt.Printf("[aws-serverless] Created DynamoDB table '%s'\n", dynamoTable)
}

// createS3Bucket creates the content storage bucket in LocalStack S3.
func createS3Bucket(ctx context.Context, awsCfg aws.Config, endpoint string) {
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		panic(fmt.Sprintf("creating S3 bucket: %v", err))
	}

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
	fmt.Printf("[aws-serverless] Created S3 bucket '%s'\n", bucketName)
}

// verifySESIdentity verifies the sender email address in LocalStack SES.
// LocalStack auto-verifies identities, so this call always succeeds.
func verifySESIdentity(ctx context.Context, awsCfg aws.Config, endpoint string) {
	client := ses.NewFromConfig(awsCfg, func(o *ses.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	_, err := client.VerifyEmailIdentity(ctx, &ses.VerifyEmailIdentityInput{
		EmailAddress: aws.String(senderAddress),
	})
	if err != nil {
		panic(fmt.Sprintf("verifying SES identity: %v", err))
	}
	fmt.Printf("[aws-serverless] Verified SES identity '%s'\n", senderAddress)
}

// createDynamoDBProvider creates a DynamoDB cache provider pointing at
// LocalStack. The provider is used as Piko's general-purpose cache backend.
func createDynamoDBProvider(endpoint string) *cache_provider_dynamodb.DynamoDBProvider {
	// Use a JSON encoder as the default for all value types.
	defaultEncoder := cache_encoder_json.New[any]()
	encodingRegistry := cache.NewEncodingRegistry(defaultEncoder.(cache.AnyEncoder))

	provider, err := cache_provider_dynamodb.NewDynamoDBProvider(cache_provider_dynamodb.Config{
		TableName:              dynamoTable,
		Region:                 region,
		EndpointURL:            endpoint,
		Registry:               encodingRegistry,
		AutoCreateTable:        false,
		OperationTimeout:       10 * time.Second,
		AtomicOperationTimeout: 15 * time.Second,
		BulkOperationTimeout:   30 * time.Second,
		FlushTimeout:           60 * time.Second,
	})
	if err != nil {
		panic(fmt.Sprintf("creating DynamoDB provider: %v", err))
	}
	fmt.Println("[aws-serverless] DynamoDB cache provider ready")
	return provider
}

// createS3Provider creates an S3 storage provider pointing at LocalStack.
func createS3Provider(ctx context.Context, endpoint string) storage.ProviderPort {
	s3Provider, err := storage_provider_s3.NewS3Provider(ctx, &storage_provider_s3.Config{
		RepositoryMappings: map[string]string{
			"content": bucketName,
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
	fmt.Println("[aws-serverless] S3 storage provider ready")
	return s3Provider
}

// createKMSKey creates a KMS key in LocalStack and returns its key ID.
func createKMSKey(ctx context.Context, awsCfg aws.Config, endpoint string) string {
	client := kms.NewFromConfig(awsCfg, func(o *kms.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	output, err := client.CreateKey(ctx, &kms.CreateKeyInput{
		Description: aws.String("Piko example encryption key"),
	})
	if err != nil {
		panic(fmt.Sprintf("creating KMS key: %v", err))
	}

	keyID := aws.ToString(output.KeyMetadata.KeyId)
	fmt.Printf("[aws-serverless] Created KMS key '%s'\n", keyID)
	return keyID
}

// createKMSProvider creates an AWS KMS crypto provider pointing at LocalStack.
func createKMSProvider(ctx context.Context, endpoint string, keyID string) crypto.EncryptionProvider {
	provider, err := crypto_provider_aws_kms.NewProvider(ctx, crypto_provider_aws_kms.Config{
		KeyID:                keyID,
		Region:               region,
		EndpointURL:          endpoint,
		UseStaticCredentials: true,
	})
	if err != nil {
		panic(fmt.Sprintf("creating KMS provider: %v", err))
	}
	fmt.Println("[aws-serverless] KMS crypto provider ready")
	return provider
}

// createRegistryCache creates a typed DynamoDB cache for the registry backend.
func createRegistryCache(endpoint string) cache.Cache[string, *piko.RegistryArtefactMeta] {
	registryEncoder := cache_encoder_json.New[*piko.RegistryArtefactMeta]()
	registryRegistry := cache.NewEncodingRegistry(registryEncoder.(cache.AnyEncoder))

	registryProvider, err := cache_provider_dynamodb.NewDynamoDBProvider(cache_provider_dynamodb.Config{
		TableName:              dynamoTable,
		Region:                 region,
		EndpointURL:            endpoint,
		Registry:               registryRegistry,
		Namespace:              "registry:",
		AutoCreateTable:        false,
		OperationTimeout:       10 * time.Second,
		AtomicOperationTimeout: 15 * time.Second,
		BulkOperationTimeout:   30 * time.Second,
	})
	if err != nil {
		panic(fmt.Sprintf("creating registry DynamoDB provider: %v", err))
	}

	registryCache, err := cache_provider_dynamodb.DynamoDBProviderFactory(
		registryProvider, "registry",
		cache.Options[string, *piko.RegistryArtefactMeta]{},
	)
	if err != nil {
		panic(fmt.Sprintf("creating registry cache: %v", err))
	}
	fmt.Println("[aws-serverless] DynamoDB registry cache ready")
	return registryCache
}

// createOrchestratorCache creates a typed DynamoDB cache for the orchestrator
// backend.
func createOrchestratorCache(endpoint string) cache.Cache[string, *piko.OrchestratorTask] {
	orchestratorEncoder := cache_encoder_json.New[*piko.OrchestratorTask]()
	orchestratorRegistry := cache.NewEncodingRegistry(orchestratorEncoder.(cache.AnyEncoder))

	orchestratorProvider, err := cache_provider_dynamodb.NewDynamoDBProvider(cache_provider_dynamodb.Config{
		TableName:              dynamoTable,
		Region:                 region,
		EndpointURL:            endpoint,
		Registry:               orchestratorRegistry,
		Namespace:              "orchestrator:",
		AutoCreateTable:        false,
		OperationTimeout:       10 * time.Second,
		AtomicOperationTimeout: 15 * time.Second,
		BulkOperationTimeout:   30 * time.Second,
	})
	if err != nil {
		panic(fmt.Sprintf("creating orchestrator DynamoDB provider: %v", err))
	}

	orchestratorCache, err := cache_provider_dynamodb.DynamoDBProviderFactory(
		orchestratorProvider, "orchestrator",
		cache.Options[string, *piko.OrchestratorTask]{},
	)
	if err != nil {
		panic(fmt.Sprintf("creating orchestrator cache: %v", err))
	}
	fmt.Println("[aws-serverless] DynamoDB orchestrator cache ready")
	return orchestratorCache
}

// createSESProvider creates an SES email provider pointing at LocalStack.
func createSESProvider(ctx context.Context, endpoint string) email.ProviderPort {
	sesProvider, err := email_provider_ses.NewSESProvider(ctx, email_provider_ses.SESProviderArgs{
		Region:           region,
		FromEmail:        senderAddress,
		AWSKey:           accessKey,
		AWSSecret:        secretKey,
		AWSLocalEndpoint: endpoint,
	})
	if err != nil {
		panic(fmt.Sprintf("creating SES provider: %v", err))
	}
	fmt.Println("[aws-serverless] SES email provider ready")
	return sesProvider
}
