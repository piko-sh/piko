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
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

// errComputeRetry is a sentinel error used internally to signal that a compute
// attempt should be retried due to an optimistic lock conflict.
var errComputeRetry = errors.New("compute retry")

// Compute atomically updates a cache entry using a compute function with
// optimistic locking via DynamoDB conditional expressions.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which calculates the new value based on
// the current value and whether it exists.
//
// Returns V which is the computed value, or zero value if the operation fails.
// Returns bool which indicates whether the operation succeeded.
// Returns error when the operation fails.
func (a *DynamoDBAdapter[K, V]) Compute(ctx context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache.ComputeAction)) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("dynamodb Compute exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for attempt := range a.maxComputeRetries {
		oldValue, found, currentVersion, getErr := a.getItemWithVersion(timeoutCtx, keyString)
		if getErr != nil {
			return *new(V), false, getErr
		}

		newValue, action := computeFunction(oldValue, found)

		writeErr := a.executeComputeAction(timeoutCtx, keyString, newValue, action, found, currentVersion, a.ttl)
		if writeErr != nil {
			if isConditionalCheckFailed(writeErr) {
				_, l := logger.From(ctx, log)
				l.Trace("Compute transaction failed, retrying",
					logger.String(logKeyField, keyString),
					logger.Int("attempt", attempt+1))
				continue
			}
			return *new(V), false, writeErr
		}

		switch action {
		case cache.ComputeActionSet:
			return newValue, true, nil
		case cache.ComputeActionDelete:
			return *new(V), false, nil
		default:
			if found {
				return oldValue, true, nil
			}
			return *new(V), false, nil
		}
	}

	return *new(V), false, fmt.Errorf("compute max retries exceeded (%d) for key %q", a.maxComputeRetries, keyString)
}

// ComputeIfAbsent atomically computes and stores a value only if the key is
// not present.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to check or create.
// Takes computeFunction (func() V) which generates the value if the key is
// absent.
//
// Returns V which is the existing or newly computed value.
// Returns bool which indicates whether computation occurred.
// Returns error when the operation fails.
func (a *DynamoDBAdapter[K, V]) ComputeIfAbsent(ctx context.Context, key K, computeFunction func() V) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("dynamodb ComputeIfAbsent exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	existing, found, getErr := a.GetIfPresent(timeoutCtx, key)
	if getErr != nil {
		return *new(V), false, getErr
	}
	if found {
		return existing, false, nil
	}

	newValue := computeFunction()

	valBytes, err := a.encodeValue(newValue)
	if err != nil {
		return *new(V), false, fmt.Errorf("failed to encode value: %w", err)
	}

	ttlUnix := calculateTTLUnix(a.ttl)

	item := map[string]types.AttributeValue{
		attrPK:        &types.AttributeValueMemberS{Value: keyString},
		attrSK:        &types.AttributeValueMemberS{Value: skData},
		attrValue:     &types.AttributeValueMemberB{Value: valBytes},
		attrTTLUnix:   &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", ttlUnix)},
		attrVersion:   &types.AttributeValueMemberN{Value: "1"},
		attrNamespace: &types.AttributeValueMemberS{Value: a.namespace},
	}

	_, putErr := a.client.PutItem(timeoutCtx, &dynamodb.PutItemInput{
		TableName:           aws.String(a.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(#pk)"),
		ExpressionAttributeNames: map[string]string{
			"#pk": attrPK,
		},
	})
	if putErr != nil {
		if isConditionalCheckFailed(putErr) {
			existing, found, getErr = a.GetIfPresent(timeoutCtx, key)
			if getErr != nil {
				return *new(V), false, getErr
			}
			if found {
				return existing, false, nil
			}
			return *new(V), false, nil
		}
		return *new(V), false, fmt.Errorf("dynamodb PutItem conditional failed: %w", putErr)
	}

	return newValue, true, nil
}

// ComputeIfPresent atomically updates a value only if the key exists in the
// cache.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the current value and
// returns the new value along with an action.
//
// Returns V which is the resulting value after computation.
// Returns bool which is true if the key existed and the computation succeeded.
// Returns error when the operation fails.
func (a *DynamoDBAdapter[K, V]) ComputeIfPresent(ctx context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache.ComputeAction)) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("dynamodb ComputeIfPresent exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for attempt := range a.maxComputeRetries {
		oldValue, found, currentVersion, getErr := a.getItemWithVersion(timeoutCtx, keyString)
		if getErr != nil {
			return *new(V), false, getErr
		}
		if !found {
			return *new(V), false, nil
		}

		newValue, action := computeFunction(oldValue)

		writeErr := a.executeComputeAction(timeoutCtx, keyString, newValue, action, true, currentVersion, a.ttl)
		if writeErr != nil {
			if isConditionalCheckFailed(writeErr) {
				_, l := logger.From(ctx, log)
				l.Trace("ComputeIfPresent transaction failed, retrying",
					logger.String(logKeyField, keyString),
					logger.Int("attempt", attempt+1))
				continue
			}
			return *new(V), false, writeErr
		}

		switch action {
		case cache.ComputeActionSet:
			return newValue, true, nil
		case cache.ComputeActionDelete:
			return *new(V), false, nil
		default:
			return oldValue, true, nil
		}
	}

	return *new(V), false, fmt.Errorf("compute if present max retries exceeded (%d) for key %q", a.maxComputeRetries, keyString)
}

// ComputeWithTTL atomically computes a new value with per-call TTL control.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the old value and found
// flag, returning a ComputeResult.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error when the operation fails.
func (a *DynamoDBAdapter[K, V]) ComputeWithTTL(ctx context.Context, key K, computeFunction func(oldValue V, found bool) cache.ComputeResult[V]) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("dynamodb ComputeWithTTL exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for attempt := range a.maxComputeRetries {
		value, ok, retryErr := a.attemptComputeWithTTL(ctx, timeoutCtx, keyString, attempt, computeFunction)
		if errors.Is(retryErr, errComputeRetry) {
			continue
		}
		return value, ok, retryErr
	}

	return *new(V), false, fmt.Errorf("compute with TTL max retries exceeded (%d) for key %q", a.maxComputeRetries, keyString)
}

// attemptComputeWithTTL performs a single compute-with-TTL attempt, returning
// errComputeRetry when the attempt should be retried.
//
// Takes keyString (string) which is the encoded partition key.
// Takes attempt (int) which is the current retry attempt number.
// Takes computeFunction (func(...)) which calculates the new value from the
// current state.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error when the operation fails or errComputeRetry for retries.
func (a *DynamoDBAdapter[K, V]) attemptComputeWithTTL(
	ctx context.Context,
	timeoutCtx context.Context,
	keyString string,
	attempt int,
	computeFunction func(oldValue V, found bool) cache.ComputeResult[V],
) (V, bool, error) {
	oldValue, found, currentVersion, getErr := a.getItemWithVersion(timeoutCtx, keyString)
	if getErr != nil {
		return *new(V), false, getErr
	}

	result := computeFunction(oldValue, found)

	effectiveTTL := a.ttl
	if result.TTL > 0 {
		effectiveTTL = result.TTL
	}

	writeErr := a.executeComputeAction(timeoutCtx, keyString, result.Value, result.Action, found, currentVersion, effectiveTTL)
	if writeErr != nil {
		if isConditionalCheckFailed(writeErr) {
			_, l := logger.From(ctx, log)
			l.Trace("ComputeWithTTL transaction failed, retrying",
				logger.String(logKeyField, keyString),
				logger.Int("attempt", attempt+1))
			return *new(V), false, errComputeRetry
		}
		return *new(V), false, writeErr
	}

	switch result.Action {
	case cache.ComputeActionSet:
		return result.Value, true, nil
	case cache.ComputeActionDelete:
		return *new(V), false, nil
	default:
		if found {
			return oldValue, true, nil
		}
		return *new(V), false, nil
	}
}

// getItemWithVersion retrieves an item and its version number for optimistic
// locking.
//
// Takes keyString (string) which is the encoded partition key.
//
// Returns V which is the decoded value if found.
// Returns bool which is true if the item exists and is not expired.
// Returns int64 which is the current version number (0 if not found).
// Returns error when the operation fails.
func (a *DynamoDBAdapter[K, V]) getItemWithVersion(ctx context.Context, keyString string) (V, bool, int64, error) {
	output, err := a.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(a.tableName),
		Key: map[string]types.AttributeValue{
			attrPK: &types.AttributeValueMemberS{Value: keyString},
			attrSK: &types.AttributeValueMemberS{Value: skData},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return *new(V), false, 0, fmt.Errorf("dynamodb GetItem failed: %w", err)
	}

	if output.Item == nil {
		return *new(V), false, 0, nil
	}

	if isItemExpired(output.Item) {
		return *new(V), false, 0, nil
	}

	var currentVersion int64
	if versionAttr, ok := output.Item[attrVersion].(*types.AttributeValueMemberN); ok {
		currentVersion, _ = strconv.ParseInt(versionAttr.Value, 10, 64)
	}

	valAttr, ok := output.Item[attrValue].(*types.AttributeValueMemberB)
	if !ok {
		return *new(V), false, currentVersion, nil
	}

	value, err := a.decodeValue(valAttr.Value)
	if err != nil {
		return *new(V), false, 0, fmt.Errorf("failed to decode value: %w", err)
	}

	return value, true, currentVersion, nil
}

// executeComputeAction performs the write operation for a compute function
// using conditional expressions for optimistic locking.
//
// Takes keyString (string) which is the encoded partition key.
// Takes newValue (V) which is the value to set when the action is Set.
// Takes action (cache.ComputeAction) which specifies the operation.
// Takes found (bool) which indicates whether the key currently exists.
// Takes expectedVersion (int64) which is the version for the condition check.
// Takes ttl (time.Duration) which is the TTL to apply.
//
// Returns error when the conditional write fails.
func (a *DynamoDBAdapter[K, V]) executeComputeAction(ctx context.Context, keyString string, newValue V, action cache.ComputeAction, found bool, expectedVersion int64, ttl time.Duration) error {
	switch action {
	case cache.ComputeActionSet:
		return a.executeComputeSet(ctx, keyString, newValue, found, expectedVersion, ttl)
	case cache.ComputeActionDelete:
		return a.executeComputeDelete(ctx, keyString, found, expectedVersion)
	case cache.ComputeActionNoop:
		return nil
	}
	return nil
}

// executeComputeSet performs a conditional set as part of a compute operation.
// When the key already exists it uses UpdateItem with a version check;
// otherwise it inserts a new item with attribute_not_exists.
//
// Takes keyString (string) which is the encoded partition key.
// Takes newValue (V) which is the value to store.
// Takes found (bool) which indicates whether the key currently exists.
// Takes expectedVersion (int64) which is the version for the condition check.
// Takes ttl (time.Duration) which is the TTL to apply.
//
// Returns error when encoding fails or the conditional write is rejected.
func (a *DynamoDBAdapter[K, V]) executeComputeSet(ctx context.Context, keyString string, newValue V, found bool, expectedVersion int64, ttl time.Duration) error {
	valBytes, err := a.encodeValue(newValue)
	if err != nil {
		return fmt.Errorf("failed to encode value: %w", err)
	}

	ttlUnix := calculateTTLUnix(ttl)
	newVersion := expectedVersion + 1

	if found {
		_, err = a.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: aws.String(a.tableName),
			Key: map[string]types.AttributeValue{
				attrPK: &types.AttributeValueMemberS{Value: keyString},
				attrSK: &types.AttributeValueMemberS{Value: skData},
			},
			UpdateExpression:    aws.String("SET #val = :val, #ttl = :ttl, #ver = :newVer, #ns = :ns"),
			ConditionExpression: aws.String("#ver = :expectedVer"),
			ExpressionAttributeNames: map[string]string{
				"#val": attrValue,
				"#ttl": attrTTLUnix,
				"#ver": attrVersion,
				"#ns":  attrNamespace,
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":val":         &types.AttributeValueMemberB{Value: valBytes},
				":ttl":         &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", ttlUnix)},
				":newVer":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", newVersion)},
				":expectedVer": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", expectedVersion)},
				":ns":          &types.AttributeValueMemberS{Value: a.namespace},
			},
		})
		return err
	}

	item := map[string]types.AttributeValue{
		attrPK:        &types.AttributeValueMemberS{Value: keyString},
		attrSK:        &types.AttributeValueMemberS{Value: skData},
		attrValue:     &types.AttributeValueMemberB{Value: valBytes},
		attrTTLUnix:   &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", ttlUnix)},
		attrVersion:   &types.AttributeValueMemberN{Value: "1"},
		attrNamespace: &types.AttributeValueMemberS{Value: a.namespace},
	}
	_, err = a.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(a.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(#pk)"),
		ExpressionAttributeNames: map[string]string{
			"#pk": attrPK,
		},
	})
	return err
}

// executeComputeDelete performs a conditional delete as part of a compute
// operation, using DeleteItem with a version check when the key exists.
//
// Takes keyString (string) which is the encoded partition key.
// Takes found (bool) which indicates whether the key currently exists.
// Takes expectedVersion (int64) which is the version for the condition check.
//
// Returns error when the conditional delete is rejected.
func (a *DynamoDBAdapter[K, V]) executeComputeDelete(ctx context.Context, keyString string, found bool, expectedVersion int64) error {
	if !found {
		return nil
	}
	_, err := a.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(a.tableName),
		Key: map[string]types.AttributeValue{
			attrPK: &types.AttributeValueMemberS{Value: keyString},
			attrSK: &types.AttributeValueMemberS{Value: skData},
		},
		ConditionExpression: aws.String("#ver = :expectedVer"),
		ExpressionAttributeNames: map[string]string{
			"#ver": attrVersion,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":expectedVer": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", expectedVersion)},
		},
	})
	return err
}

// isConditionalCheckFailed returns true if the error is a DynamoDB
// ConditionalCheckFailedException, indicating an optimistic lock conflict.
//
// Takes err (error) which is the error to inspect.
//
// Returns bool which is true when the error is a conditional check failure.
func isConditionalCheckFailed(err error) bool {
	var condErr *types.ConditionalCheckFailedException
	return errors.As(err, &condErr)
}
