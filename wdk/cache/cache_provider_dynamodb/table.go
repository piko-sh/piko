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

package cache_provider_dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/wdk/logger"
)

const (
	// logTableField is the structured logging key for table names.
	logTableField = "table"

	// attrPK is the partition key attribute name.
	attrPK = "pk"

	// attrSK is the sort key attribute name.
	attrSK = "sk"

	// attrValue is the attribute name for the encoded value bytes.
	attrValue = "val"

	// attrTTLUnix is the attribute name for the DynamoDB native TTL field,
	// stored as Unix epoch seconds.
	attrTTLUnix = "ttl_unix"

	// attrVersion is the attribute name for the monotonic version used in
	// optimistic locking.
	attrVersion = "version"

	// attrNamespace is the attribute name for the namespace string, used by
	// the GSI for namespace-scoped queries.
	attrNamespace = "ns"

	// skData is the sort key value for cache data items.
	skData = "#DATA"

	// gsiNamespaceName is the name of the global secondary index keyed on
	// namespace.
	gsiNamespaceName = "gsi_ns"

	// defaultTableName is the default DynamoDB table name when none is
	// configured.
	defaultTableName = "piko_cache"

	// tableActiveCheckInterval is the delay between DescribeTable polls when
	// waiting for a newly created table to become active.
	tableActiveCheckInterval = 500 * time.Millisecond

	// tableActiveMaxAttempts is the maximum number of polls before giving up.
	tableActiveMaxAttempts = 60

	// gsiFieldPrefix is the prefix for search field GSI names.
	gsiFieldPrefix = "gsi_sf_"
)

// createTable creates the DynamoDB table with the required key schema, GSI,
// and TTL configuration.
//
// Takes client (*dynamodb.Client) which is the DynamoDB client.
// Takes tableName (string) which is the table to create.
// Takes billingMode (types.BillingMode) which sets the billing mode.
// Takes readCapacity (int64) which sets provisioned RCU (only for PROVISIONED).
// Takes writeCapacity (int64) which sets provisioned WCU (only for PROVISIONED).
//
// Returns error when the table creation fails.
func createTable(ctx context.Context, client *dynamodb.Client, tableName string, billingMode types.BillingMode, readCapacity int64, writeCapacity int64) error {
	_, l := logger.From(ctx, log)

	input := &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String(attrPK), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String(attrSK), KeyType: types.KeyTypeRange},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String(attrPK), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String(attrSK), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String(attrNamespace), AttributeType: types.ScalarAttributeTypeS},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String(gsiNamespaceName),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String(attrNamespace), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeKeysOnly,
				},
			},
		},
	}

	if billingMode == types.BillingModeProvisioned {
		input.BillingMode = types.BillingModeProvisioned
		input.ProvisionedThroughput = &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(readCapacity),
			WriteCapacityUnits: aws.Int64(writeCapacity),
		}
		input.GlobalSecondaryIndexes[0].ProvisionedThroughput = &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(readCapacity),
			WriteCapacityUnits: aws.Int64(writeCapacity),
		}
	} else {
		input.BillingMode = types.BillingModePayPerRequest
	}

	if _, err := client.CreateTable(ctx, input); err != nil {
		var resourceInUseErr *types.ResourceInUseException
		if errors.As(err, &resourceInUseErr) {
			l.Internal("DynamoDB table already being created by another caller",
				logger.String(logTableField, tableName))
			return nil
		}
		return fmt.Errorf("failed to create DynamoDB table %q: %w", tableName, err)
	}

	l.Internal("Created DynamoDB table", logger.String(logTableField, tableName))
	return nil
}

// enableTTL enables the DynamoDB native TTL on the ttl_unix attribute.
//
// Takes client (*dynamodb.Client) which is the DynamoDB client.
// Takes tableName (string) which is the target table.
//
// Returns error when the TTL update fails.
func enableTTL(ctx context.Context, client *dynamodb.Client, tableName string) error {
	_, l := logger.From(ctx, log)

	_, err := client.UpdateTimeToLive(ctx, &dynamodb.UpdateTimeToLiveInput{
		TableName: aws.String(tableName),
		TimeToLiveSpecification: &types.TimeToLiveSpecification{
			Enabled:       aws.Bool(true),
			AttributeName: aws.String(attrTTLUnix),
		},
	})
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "already enabled") || strings.Contains(errMsg, "TimeToLive is already enabled") {
			l.Internal("TTL already enabled on table",
				logger.String(logTableField, tableName))
			return nil
		}
		return fmt.Errorf("failed to enable TTL on table %q: %w", tableName, err)
	}

	l.Internal("Enabled TTL on DynamoDB table",
		logger.String(logTableField, tableName),
		logger.String("attribute", attrTTLUnix))
	return nil
}

// gsiSpec holds the data needed to create a single GSI via UpdateTable.
type gsiSpec struct {
	// update is the GSI creation action for the UpdateTable call.
	update types.GlobalSecondaryIndexUpdate

	// attrDef is the attribute definition for the GSI sort key.
	attrDef types.AttributeDefinition
}

// ensureFieldGSIs creates GSIs for TAG and sortable NUMERIC fields in the
// SearchSchema, enabling efficient range queries that read only matching items.
//
// Takes client (*dynamodb.Client) which is the DynamoDB client.
// Takes tableName (string) which is the target table.
// Takes billingMode (types.BillingMode) which sets the billing mode for new
// GSIs.
// Takes readCapacity (int64) which sets provisioned RCU for PROVISIONED mode.
// Takes writeCapacity (int64) which sets provisioned WCU for PROVISIONED mode.
// Takes schema (*cache_dto.SearchSchema) which defines the searchable fields.
//
// Returns map[string]string which maps field names to their GSI names.
// Returns error when GSI creation fails.
func ensureFieldGSIs(
	ctx context.Context,
	client *dynamodb.Client,
	tableName string,
	billingMode types.BillingMode,
	readCapacity int64,
	writeCapacity int64,
	schema *cache_dto.SearchSchema,
) (map[string]string, error) {
	existingGSIs, err := describeTableGSIs(ctx, client, tableName)
	if err != nil {
		return nil, err
	}

	gsiFields := make(map[string]string)
	specs := collectGSISpecs(schema, existingGSIs, gsiFields, billingMode, readCapacity, writeCapacity)

	if len(specs) == 0 {
		return gsiFields, nil
	}

	if err := applyGSISpecs(ctx, client, tableName, specs); err != nil {
		return gsiFields, err
	}

	return gsiFields, nil
}

// collectGSISpecs builds the list of GSIs to create, populating gsiFields
// with both existing and new GSI mappings.
//
// Takes schema (*cache_dto.SearchSchema) which defines the searchable fields.
// Takes existingGSIs (map[string]bool) which tracks GSIs already on the table.
// Takes gsiFields (map[string]string) which collects field-to-GSI mappings.
// Takes billingMode (types.BillingMode) which sets the billing mode for new
// GSIs.
// Takes readCapacity (int64) which sets provisioned RCU for PROVISIONED mode.
// Takes writeCapacity (int64) which sets provisioned WCU for PROVISIONED mode.
//
// Returns []gsiSpec which contains the GSI creation specifications.
func collectGSISpecs(
	schema *cache_dto.SearchSchema,
	existingGSIs map[string]bool,
	gsiFields map[string]string,
	billingMode types.BillingMode,
	readCapacity int64,
	writeCapacity int64,
) []gsiSpec {
	var specs []gsiSpec

	for _, field := range schema.Fields {
		if !isGSIEligible(field) {
			continue
		}

		gsiName := gsiFieldPrefix + field.Name
		attrName := searchFieldPrefix + field.Name
		gsiFields[field.Name] = gsiName

		if existingGSIs[gsiName] {
			continue
		}

		attrType := types.ScalarAttributeTypeS
		if field.Type == cache_dto.FieldTypeNumeric {
			attrType = types.ScalarAttributeTypeN
		}

		spec := gsiSpec{
			update: types.GlobalSecondaryIndexUpdate{
				Create: &types.CreateGlobalSecondaryIndexAction{
					IndexName: aws.String(gsiName),
					KeySchema: []types.KeySchemaElement{
						{AttributeName: aws.String(attrNamespace), KeyType: types.KeyTypeHash},
						{AttributeName: aws.String(attrName), KeyType: types.KeyTypeRange},
					},
					Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
				},
			},
			attrDef: types.AttributeDefinition{
				AttributeName: aws.String(attrName),
				AttributeType: attrType,
			},
		}

		if billingMode == types.BillingModeProvisioned {
			spec.update.Create.ProvisionedThroughput = &types.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(readCapacity),
				WriteCapacityUnits: aws.Int64(writeCapacity),
			}
		}

		specs = append(specs, spec)
	}

	return specs
}

// applyGSISpecs creates each GSI one at a time (DynamoDB allows only one GSI
// per UpdateTable call) and waits for the table to become active after each.
//
// Takes client (*dynamodb.Client) which is the DynamoDB client.
// Takes tableName (string) which is the target table.
// Takes specs ([]gsiSpec) which are the GSI specifications to apply.
//
// Returns error when a GSI creation or table activation fails.
func applyGSISpecs(ctx context.Context, client *dynamodb.Client, tableName string, specs []gsiSpec) error {
	_, l := logger.From(ctx, log)

	for _, spec := range specs {
		_, err := client.UpdateTable(ctx, &dynamodb.UpdateTableInput{
			TableName:                   aws.String(tableName),
			GlobalSecondaryIndexUpdates: []types.GlobalSecondaryIndexUpdate{spec.update},
			AttributeDefinitions:        []types.AttributeDefinition{spec.attrDef},
		})
		if err != nil {
			var resourceInUseErr *types.ResourceInUseException
			if errors.As(err, &resourceInUseErr) {
				l.Internal("GSI already being created",
					logger.String("gsi", *spec.update.Create.IndexName))
				continue
			}
			return fmt.Errorf("failed to create GSI %q: %w", *spec.update.Create.IndexName, err)
		}

		l.Internal("Created field GSI",
			logger.String("gsi", *spec.update.Create.IndexName),
			logger.String(logTableField, tableName))

		if err := waitForTableActive(ctx, client, tableName); err != nil {
			return fmt.Errorf("table did not become active after GSI creation: %w", err)
		}
	}

	return nil
}

// isGSIEligible returns true if the field type benefits from a GSI, covering
// TAG fields for equality lookups and sortable NUMERIC fields for range queries.
//
// Takes field (cache_dto.FieldSchema) which describes the field to check.
//
// Returns bool which is true when the field should have a GSI.
func isGSIEligible(field cache_dto.FieldSchema) bool {
	if field.Type == cache_dto.FieldTypeTag {
		return true
	}
	if field.Type == cache_dto.FieldTypeNumeric && field.Sortable {
		return true
	}
	return false
}

// describeTableGSIs returns a set of existing GSI names on the table.
//
// Takes client (*dynamodb.Client) which is the DynamoDB client.
// Takes tableName (string) which is the table to describe.
//
// Returns map[string]bool which contains the existing GSI names.
// Returns error when the DescribeTable call fails.
func describeTableGSIs(ctx context.Context, client *dynamodb.Client, tableName string) (map[string]bool, error) {
	output, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe table for GSI check: %w", err)
	}

	result := make(map[string]bool)
	for _, gsi := range output.Table.GlobalSecondaryIndexes {
		if gsi.IndexName != nil {
			result[*gsi.IndexName] = true
		}
	}
	return result, nil
}

// ensureTableExists checks whether the DynamoDB table exists. If not and
// autoCreate is true, it creates the table and waits for it to become active.
//
// Takes client (*dynamodb.Client) which is the DynamoDB client.
// Takes config (Config) which provides table creation settings.
//
// Returns error when the table does not exist and cannot be created.
func ensureTableExists(ctx context.Context, client *dynamodb.Client, config Config) error {
	_, l := logger.From(ctx, log)

	_, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(config.TableName),
	})
	if err == nil {
		l.Internal("DynamoDB table already exists",
			logger.String(logTableField, config.TableName))
		return nil
	}

	var notFoundErr *types.ResourceNotFoundException
	if !errors.As(err, &notFoundErr) {
		return fmt.Errorf("failed to describe DynamoDB table %q: %w", config.TableName, err)
	}

	if !config.AutoCreateTable {
		return fmt.Errorf("DynamoDB table %q does not exist and AutoCreateTable is false", config.TableName)
	}

	if err := createTable(ctx, client, config.TableName, config.BillingMode, config.ReadCapacityUnits, config.WriteCapacityUnits); err != nil {
		return err
	}

	if err := waitForTableActive(ctx, client, config.TableName); err != nil {
		return err
	}

	return enableTTL(ctx, client, config.TableName)
}

// waitForTableActive polls DescribeTable until the table status is ACTIVE.
//
// Takes client (*dynamodb.Client) which is the DynamoDB client.
// Takes tableName (string) which identifies the table to wait for.
//
// Returns error when the table does not become active within the allowed
// polling window.
func waitForTableActive(ctx context.Context, client *dynamodb.Client, tableName string) error {
	for range tableActiveMaxAttempts {
		output, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(tableName),
		})
		if err != nil {
			return fmt.Errorf("failed to describe table %q while waiting: %w", tableName, err)
		}
		if output.Table.TableStatus == types.TableStatusActive {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(tableActiveCheckInterval):
		}
	}

	return fmt.Errorf("table %q did not become active after %d attempts", tableName, tableActiveMaxAttempts)
}
