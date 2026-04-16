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

package config_domain

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/maypok86/otter/v2"
)

// Resolver defines how to look up values for placeholders in configuration.
// It implements config.Resolver and can get values from sources such as
// environment variables, files, or secret management systems.
type Resolver interface {
	// GetPrefix returns the prefix string for this rule.
	GetPrefix() string

	// Resolve converts the given value to its final form.
	//
	// Takes value (string) which is the input to convert.
	//
	// Returns string which is the converted result.
	// Returns error when the conversion fails.
	Resolve(ctx context.Context, value string) (string, error)
}

// BatchResolver extends Resolver with batch resolution for better performance
// when resolving many values from the same source at once.
type BatchResolver interface {
	Resolver

	// ResolveBatch resolves multiple values in a single operation.
	//
	// Takes values ([]string) which contains the values to resolve.
	//
	// Returns map[string]string which maps each input value to its resolved form.
	// Returns error when resolution fails.
	ResolveBatch(ctx context.Context, values []string) (map[string]string, error)
}

// resolutionJob represents a piece of work for a resolver.
// Fields are ordered for memory efficiency per the fieldalignment linter.
type resolutionJob struct {
	// keyPath is the dotted path to this field, used in error messages.
	keyPath string

	// prefix is the placeholder prefix (e.g. "env:") that shows the source type.
	prefix string

	// lookupKey is the value to look up after the prefix is removed.
	lookupKey string

	// field points to the struct field metadata; nil when the job is reset.
	field *reflect.StructField

	// fieldValue holds the reflected value of the struct field to be resolved.
	fieldValue reflect.Value
}

// Reset clears the job's fields so it can be reused from the object pool.
func (j *resolutionJob) Reset() {
	j.fieldValue = reflect.Value{}
	j.field = nil
	j.keyPath = ""
	j.prefix = ""
	j.lookupKey = ""
}

// resolutionResult holds the outcome of resolving a single template variable.
type resolutionResult struct {
	// err holds any error that occurred during resolution.
	err error

	// job points to the resolution job that produced this result.
	job *resolutionJob

	// resolvedValue holds the value returned by the resolver for this job.
	resolvedValue string
}

// Reset clears the result's fields so it can be reused from the object pool.
func (r *resolutionResult) Reset() {
	r.job = nil
	r.resolvedValue = ""
	r.err = nil
}

var (
	// jobPool reuses resolutionJob instances to reduce allocation pressure during
	// placeholder resolution.
	jobPool = sync.Pool{
		New: func() any { return new(resolutionJob) },
	}

	// resultPool reuses resolutionResult instances to reduce allocation pressure
	// during placeholder resolution.
	resultPool = sync.Pool{
		New: func() any { return new(resolutionResult) },
	}
)

// resolutionOperation holds the state and logic for a single placeholder
// resolution pass.
type resolutionOperation struct {
	// groupedJobs maps provider prefixes to their resolution jobs.
	groupedJobs map[string][]*resolutionJob

	// resultsChan receives results from concurrent resolver jobs.
	resultsChan chan *resolutionResult

	// loadCtx holds the context for the current load operation.
	loadCtx *LoadContext

	// loader references the parent Loader for resolver lookup and caching.
	loader *Loader

	// errors collects failures during resolution to report after all jobs finish.
	errors []error

	// wg coordinates goroutines running resolution jobs.
	wg sync.WaitGroup

	// totalJobs is the total number of placeholder resolution jobs found.
	totalJobs int
}

// resolvePlaceholders is the main entry point for the resolver pass.
// It handles placeholder resolution in three phases: discovery, execution,
// and result application.
//
// Takes ptr (any) which points to the struct containing placeholder fields.
// Takes ctx (*LoadContext) which provides the loading context and settings.
//
// Returns error when discovery fails, the context is cancelled, or any
// resolution job fails.
func (l *Loader) resolvePlaceholders(ptr any, ctx *LoadContext) error {
	if len(l.resolverMap) == 0 {
		return nil
	}
	if err := ctx.Context.Err(); err != nil {
		return fmt.Errorf("checking context before resolving placeholders: %w", err)
	}

	operation := &resolutionOperation{
		groupedJobs: make(map[string][]*resolutionJob),
		resultsChan: nil,
		loadCtx:     ctx,
		loader:      l,
		errors:      nil,
		wg:          sync.WaitGroup{},
		totalJobs:   0,
	}

	if err := operation.discoverJobs(ptr); err != nil {
		return fmt.Errorf("discovering resolver jobs: %w", err)
	}
	if operation.totalJobs == 0 {
		return nil
	}

	operation.executeJobs()

	return operation.applyResults()
}

// discoverJobs walks the struct and fills the operation's job list.
//
// Takes ptr (any) which is the struct pointer to walk for placeholder fields.
//
// Returns error when the struct walk fails.
func (op *resolutionOperation) discoverJobs(ptr any) error {
	discoveryProcessor := func(field *reflect.StructField, value reflect.Value, _, keyPath string) error {
		if value.Kind() != reflect.String || !value.CanSet() {
			return nil
		}

		originalValue := value.String()
		for prefix := range op.loader.resolverMap {
			if strings.HasPrefix(originalValue, prefix) {
				job, ok := jobPool.Get().(*resolutionJob)
				if !ok {
					job = new(resolutionJob)
				}
				job.fieldValue = value
				job.field = field
				job.keyPath = keyPath
				job.prefix = prefix
				job.lookupKey = strings.TrimPrefix(originalValue, prefix)

				op.groupedJobs[prefix] = append(op.groupedJobs[prefix], job)
				op.totalJobs++
				break
			}
		}
		return nil
	}

	state := &walkState{
		processor: discoveryProcessor,
		ctx:       nil,
		keyPrefix: "",
		source:    "",
	}
	if err := op.loader.walk(reflect.ValueOf(ptr), state); err != nil {
		return fmt.Errorf("resolver discovery walk failed: %w", err)
	}
	return nil
}

// executeJobs starts a goroutine for each resolver type and waits for all to
// finish.
//
// Safe for concurrent use. Spawns one goroutine per resolver prefix, plus a
// cleanup goroutine that closes the results channel when all work finishes.
func (op *resolutionOperation) executeJobs() {
	op.resultsChan = make(chan *resolutionResult, op.totalJobs)

	for prefix, jobs := range op.groupedJobs {
		op.wg.Go(func() {
			op.dispatchResolver(prefix, jobs)
		})
	}

	go func() {
		op.wg.Wait()
		close(op.resultsChan)
	}()
}

// dispatchResolver intelligently chooses between batch and single resolution
// methods.
//
// Takes prefix (string) which identifies the resolver to dispatch to.
// Takes jobs ([]*resolutionJob) which contains the jobs to be resolved.
func (op *resolutionOperation) dispatchResolver(prefix string, jobs []*resolutionJob) {
	resolver := op.loader.resolverMap[prefix]

	if batchResolver, ok := resolver.(BatchResolver); ok {
		op.runBatchResolver(batchResolver, prefix, jobs)
	} else {
		op.runSingleResolver(resolver, prefix, jobs)
	}
}

// applyResults reads all results from the channel, sets the struct fields,
// and gathers any errors.
//
// Returns error when any resolution or field setting fails.
func (op *resolutionOperation) applyResults() error {
	for result := range op.resultsChan {
		if result.err != nil {
			err := fmt.Errorf("field %q (placeholder %q): %w", result.job.keyPath, result.job.prefix+result.job.lookupKey, result.err)
			op.errors = append(op.errors, err)
		} else if err := setField(result.job.fieldValue, result.resolvedValue, result.job.field.Tag); err != nil {
			err := fmt.Errorf("field %q: failed to set resolved value: %w", result.job.keyPath, err)
			op.errors = append(op.errors, err)
		} else {
			sourceName := fmt.Sprintf("%s:%s", sourceResolver, strings.TrimSuffix(result.job.prefix, ":"))
			op.loadCtx.FieldSources[result.job.keyPath] = sourceName
		}
		jobPool.Put(result.job)
		resultPool.Put(result)
	}

	if len(op.errors) > 0 {
		return errors.Join(op.errors...)
	}
	return nil
}

// runBatchResolver resolves multiple placeholder values using a
// BatchResolver in a single batch call, deduplicating lookup keys
// before fetching.
//
// Takes resolver (BatchResolver) which performs the batch lookup.
// Takes jobs ([]*resolutionJob) which are the placeholder jobs to
// resolve.
func (op *resolutionOperation) runBatchResolver(resolver BatchResolver, _ string, jobs []*resolutionJob) {
	uniqueKeys := make(map[string]struct{})
	keysToFetch := make([]string, 0, len(jobs))
	for _, job := range jobs {
		if _, exists := uniqueKeys[job.lookupKey]; !exists {
			uniqueKeys[job.lookupKey] = struct{}{}
			keysToFetch = append(keysToFetch, job.lookupKey)
		}
	}

	batchResult, err := op.loader.breaker.Execute(func() (any, error) {
		return resolver.ResolveBatch(op.loadCtx.Context, keysToFetch)
	})

	var batchMap map[string]string
	if err == nil && batchResult != nil {
		if typed, ok := batchResult.(map[string]string); ok {
			batchMap = typed
		} else {
			err = fmt.Errorf("unexpected batch result type: %T", batchResult)
		}
	}

	for _, job := range jobs {
		result, ok := resultPool.Get().(*resolutionResult)
		if !ok {
			result = new(resolutionResult)
		}
		result.job = job
		if err != nil {
			result.err = err
		} else if resolvedValue, found := batchMap[job.lookupKey]; found {
			result.resolvedValue = resolvedValue
		} else {
			result.err = fmt.Errorf("key %q not found in batch result", job.lookupKey)
		}
		op.resultsChan <- result
	}
}

// runSingleResolver runs a resolver for a batch of jobs at the same time.
//
// Takes resolver (Resolver) which looks up values.
// Takes prefix (string) which specifies the key prefix for resolution.
// Takes jobs ([]*resolutionJob) which contains the jobs to process.
func (op *resolutionOperation) runSingleResolver(resolver Resolver, prefix string, jobs []*resolutionJob) {
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Go(func() {
			resolvedValue, err := op.resolveSingleValue(resolver, prefix, job.lookupKey)
			op.sendResult(job, resolvedValue, err)
		})
	}
	wg.Wait()
}

// resolveSingleValue resolves a single value using either direct resolution or
// cached resolution.
//
// Takes resolver (Resolver) which performs the actual resolution.
// Takes prefix (string) which is the resolver prefix for cache key
// construction.
// Takes lookupKey (string) which is the key to resolve.
//
// Returns string which is the resolved value.
// Returns error when resolution fails or the result type is unexpected.
func (op *resolutionOperation) resolveSingleValue(resolver Resolver, prefix, lookupKey string) (string, error) {
	if op.loader.resolverCache == nil {
		return op.resolveWithoutCache(resolver, lookupKey)
	}
	return op.resolveWithCache(resolver, prefix, lookupKey)
}

// resolveWithoutCache performs direct resolution without caching.
//
// Takes resolver (Resolver) which handles the actual resolution logic.
// Takes lookupKey (string) which identifies the value to resolve.
//
// Returns string which is the resolved value, or empty if not found.
// Returns error when the circuit breaker fails or the result type is invalid.
func (op *resolutionOperation) resolveWithoutCache(resolver Resolver, lookupKey string) (string, error) {
	cbResult, err := op.loader.breaker.Execute(func() (any, error) {
		return resolver.Resolve(op.loadCtx.Context, lookupKey)
	})
	if err != nil {
		return "", fmt.Errorf("resolving %q without cache: %w", lookupKey, err)
	}
	if cbResult == nil {
		return "", nil
	}
	resolved, ok := cbResult.(string)
	if !ok {
		return "", fmt.Errorf("unexpected resolver result type: %T", cbResult)
	}
	return resolved, nil
}

// resolveWithCache performs resolution using the cache.
//
// Takes resolver (Resolver) which performs the actual resolution lookup.
// Takes prefix (string) which is prepended to form the cache key.
// Takes lookupKey (string) which identifies the item to resolve.
//
// Returns string which is the resolved value from cache or resolver.
// Returns error when the circuit breaker fails or the resolver returns an
// unexpected type.
func (op *resolutionOperation) resolveWithCache(resolver Resolver, prefix, lookupKey string) (string, error) {
	cacheKey := prefix + lookupKey

	loader := otter.LoaderFunc[string, string](
		func(ctx context.Context, _ string) (string, error) {
			cbResult, cbErr := op.loader.breaker.Execute(func() (any, error) {
				return resolver.Resolve(ctx, lookupKey)
			})
			if cbErr != nil {
				return "", cbErr
			}
			if cbResult == nil {
				return "", nil
			}
			resolved, ok := cbResult.(string)
			if !ok {
				return "", fmt.Errorf("unexpected resolver result type: %T", cbResult)
			}
			return resolved, nil
		},
	)

	return op.loader.resolverCache.Get(op.loadCtx.Context, cacheKey, loader)
}

// sendResult sends a resolution result to the results channel.
//
// Takes job (*resolutionJob) which is the job being completed.
// Takes resolvedValue (string) which is the resolved path value.
// Takes err (error) which is any error that occurred during resolution.
func (op *resolutionOperation) sendResult(job *resolutionJob, resolvedValue string, err error) {
	result, ok := resultPool.Get().(*resolutionResult)
	if !ok {
		result = new(resolutionResult)
	}
	result.job = job
	result.err = err
	result.resolvedValue = resolvedValue
	op.resultsChan <- result
}

// clearResolutionPools is a no-op provided for API consistency. sync.Pool
// cannot be cleared since objects are lazily garbage collected, and pool
// objects are stateless between operations as Reset is called before reuse.
func clearResolutionPools() {
}
