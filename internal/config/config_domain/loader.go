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
	"os"
	"reflect"
	"slices"
	"time"

	"github.com/maypok86/otter/v2"
	"github.com/sony/gobreaker/v2"
)

const (
	// sourceDefault is the source name for values set from defaults.
	sourceDefault = "default"

	// sourceFile is the prefix used for file-based loading sources.
	sourceFile = "file"

	// sourceResolver is the prefix used to show where field values come from.
	sourceResolver = "resolver"

	// sourceEnv identifies environment variables as the configuration source.
	sourceEnv = "env"

	// sourceFlag is the source name for values set by command-line flags.
	sourceFlag = "flag"

	// defaultDelimiter is the comma used to split slice and map values.
	defaultDelimiter = ","

	// defaultSeparator is the character used to split key-value pairs in maps.
	defaultSeparator = ":"

	// defaultResolverCacheTTL is the default time to live for resolver cache
	// entries.
	defaultResolverCacheTTL = 15 * time.Second

	// defaultResolverCacheSize is the maximum number of entries the resolver cache
	// can hold.
	defaultResolverCacheSize = 1000

	// circuitBreakerTimeout is the duration the circuit stays open before
	// allowing a test request through.
	circuitBreakerTimeout = 30 * time.Second

	// circuitBreakerBucketPeriod is the duration of each measurement bucket
	// for tracking failure counts.
	circuitBreakerBucketPeriod = 10 * time.Second

	// circuitBreakerConsecutiveFailures is the number of consecutive failures
	// required to trip the circuit breaker.
	circuitBreakerConsecutiveFailures = 5
)

const (
	// PassDefaults applies default values from struct tags.
	PassDefaults Pass = iota

	// PassFiles loads and merges configuration from files.
	PassFiles

	// PassDotEnv applies overrides from environment files.
	PassDotEnv

	// PassEnv applies settings from environment variables.
	PassEnv

	// PassFlags applies settings from command-line flags.
	PassFlags

	// PassResolvers processes placeholder strings (e.g., for secrets).
	PassResolvers

	// PassProgrammaticOverrides applies programmatic overrides that take highest
	// precedence, overwriting any values set by earlier passes. This ensures that
	// values set via WithXxx functional options always win over YAML/env/flag
	// values.
	PassProgrammaticOverrides

	// PassValidation validates the final populated struct.
	PassValidation

	// PassProgrammatic applies default values provided programmatically.
	PassProgrammatic
)

// Pass represents a stage in the configuration loading process.
// It implements fmt.Stringer.
type Pass int

// passToString maps Pass constants to their string representations.
// This is safer than a simple array lookup.
var passToString = map[Pass]string{
	PassDefaults:              "defaults",
	PassFiles:                 "files",
	PassDotEnv:                "dotenv",
	PassEnv:                   "env",
	PassFlags:                 "flags",
	PassResolvers:             "resolvers",
	PassProgrammaticOverrides: "programmatic_overrides",
	PassValidation:            "validation",
	PassProgrammatic:          "programmatic",
}

// String returns the string representation of the Pass value, implementing
// the fmt.Stringer interface for better logging.
//
// When the Pass value is unknown, it returns a formatted string like
// "Pass(99)" instead of panicking.
//
// Returns string which is the human-readable name of the pass.
func (p Pass) String() string {
	if s, ok := passToString[p]; ok {
		return s
	}
	return fmt.Sprintf("Pass(%d)", p)
}

// FileReader provides an interface for reading files, used to allow testing
// with mock file systems. Implements config_domain.FileReader.
type FileReader interface {
	// ReadFile reads the contents of the file at the given path.
	//
	// Takes path (string) which specifies the file to read.
	//
	// Returns []byte which contains the file contents.
	// Returns error when the file cannot be read.
	ReadFile(path string) ([]byte, error)
}

// osFileReader implements the FileReader interface using os.ReadFile.
type osFileReader struct{}

// ReadFile reads the file at the given path using os.ReadFile.
// Paths come from settings or environment, not user input.
//
// Takes path (string) which specifies the file path to read.
//
// Returns []byte which contains the file contents.
// Returns error when the file cannot be read.
func (osFileReader) ReadFile(path string) ([]byte, error) {
	//nolint:gosec // trusted config paths
	return os.ReadFile(path)
}

// StructValidator defines the minimal interface for struct validation.
// It is satisfied by the playground validator from the
// validation_provider_playground WDK module.
type StructValidator interface {
	// Struct validates a struct's exposed fields based on validation tags.
	//
	// Takes s (any) which is the struct to validate.
	//
	// Returns error when any field fails its validation constraint.
	Struct(s any) error
}

// LoaderOptions configures the behaviour of a Loader.
type LoaderOptions struct {
	// FileReader is the file reader to use, defaulting to osFileReader
	// (os.ReadFile) when nil, primarily for testing purposes.
	FileReader FileReader

	// ProgrammaticDefaults is a struct with default values to merge into the
	// target. These have the lowest precedence and are overwritten by all other
	// sources.
	ProgrammaticDefaults any

	// ProgrammaticOverrides contains values that take the highest precedence over
	// all other configuration sources. Non-zero fields overwrite values from
	// files, environment variables, flags, or resolvers, ensuring functional
	// options (e.g., WithXxx) always win.
	ProgrammaticOverrides any

	// Validator is the validation instance to use. If nil, the validation
	// pass is skipped entirely.
	Validator StructValidator

	// ResolverRegistry specifies which resolver registry to use; nil uses the
	// global singleton. Used mainly for testing when UseGlobalResolvers is true.
	ResolverRegistry *ResolverRegistry

	// ResolverCacheTTL sets how long resolved values are kept in the cache.
	// Zero or negative values disable caching.
	ResolverCacheTTL *time.Duration

	// FlagCoordinator is the flag coordinator to use; nil uses the global
	// singleton. This is primarily for testing purposes.
	FlagCoordinator *FlagCoordinator

	// EnvPrefix is an optional prefix added to all environment variable names.
	EnvPrefix string

	// FlagPrefix is the prefix for flag registration with the global coordinator.
	// Flags are registered as "prefix.flagName" (e.g., "app.dbUrl"); empty means
	// no prefix.
	FlagPrefix string

	// Resolvers is a list of custom resolvers that handle placeholder values.
	Resolvers []Resolver

	// PassOrder sets the order in which passes run. If nil or
	// empty, uses the default order: [Defaults, Programmatic,
	// Files, DotEnv, Env, Flags, Resolvers,
	// ProgrammaticOverrides, Validation].
	PassOrder []Pass

	// FilePaths specifies the file paths to load and merge, where later files
	// override earlier ones. Supports .json, .yaml and .yml extensions.
	FilePaths []string

	// UseGlobalResolvers, when true, adds all resolvers from the global registry
	// to the Resolvers field.
	UseGlobalResolvers bool

	// StrictFile enables strict mode for config files. When true, loading returns
	// an error if a file contains fields that do not exist in the target struct.
	StrictFile bool
}

// LoadContext holds the results of a loading operation, including debugging
// information.
type LoadContext struct {
	// FieldSources maps each field path to the source that set its value.
	// For example, "Server.Port" might map to "env:SERVER_PORT".
	FieldSources map[string]string

	// Target is the struct pointer after it has been filled in and checked.
	Target any

	// Context is the context for the load operation, passed to resolvers.
	Context context.Context
}

// Loader manages the process of loading and resolving configuration values.
type Loader struct {
	// resolverMap maps prefix strings to their Resolver for placeholder
	// resolution.
	resolverMap map[string]Resolver

	// resolverCache stores resolved values by key to avoid repeated lookups;
	// nil disables caching.
	resolverCache *otter.Cache[string, string]

	// validator checks struct field values against validation rules.
	// When nil, the validation pass is skipped.
	validator StructValidator

	// fileReader reads file contents when loading settings.
	fileReader FileReader

	// flagCoordinator manages flag registration and parsing for config structs.
	flagCoordinator *FlagCoordinator

	// resolverRegistry provides access to resolvers that are registered globally.
	resolverRegistry *ResolverRegistry

	// breaker wraps resolver calls to prevent failures from spreading.
	breaker *gobreaker.CircuitBreaker[any]

	// opts holds the configuration settings for this loader.
	opts LoaderOptions
}

// LoaderOption is a function that sets up a Loader.
type LoaderOption func(*Loader)

// NewLoader creates a new configuration loader with the given options.
//
// Takes opts (LoaderOptions) which sets the base settings for the loader,
// including validator, cache TTL, and component overrides.
// Takes options (...LoaderOption) which provides optional settings to change
// loader behaviour after creation.
//
// Returns *Loader which is ready to load and validate configuration files.
func NewLoader(opts LoaderOptions, options ...LoaderOption) *Loader {
	l := &Loader{
		opts:             opts,
		validator:        opts.Validator,
		resolverMap:      make(map[string]Resolver),
		resolverCache:    createResolverCache(opts.ResolverCacheTTL),
		fileReader:       getFileReader(opts.FileReader),
		flagCoordinator:  getFlagCoordinator(opts.FlagCoordinator),
		resolverRegistry: getResolverRegistry(opts.ResolverRegistry),
		breaker:          newResolverCircuitBreaker(),
	}

	for _, option := range options {
		option(l)
	}

	l.buildResolverMap()
	return l
}

// Close releases resources held by the loader.
func (l *Loader) Close() {
	if l.resolverCache != nil {
		l.resolverCache.StopAllGoroutines()
	}
}

// Load executes the full configuration loading and validation process.
// It proceeds in distinct passes, with an order of precedence that can be
// configured.
//
// Takes ptr (any) which must be a pointer to a struct that will receive the
// loaded configuration values.
//
// Returns *LoadContext which contains metadata about the loading process,
// including which source provided each field value.
// Returns error when ptr is not a pointer to a struct, flag registration
// fails, or any loading pass fails.
func (l *Loader) Load(ctx context.Context, ptr any) (*LoadContext, error) {
	if v := reflect.ValueOf(ptr); v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a pointer to a struct, got %T", ptr)
	}

	if l.shouldUseFlagCoordinator() {
		if err := l.flagCoordinator.RegisterStruct(ptr, l.opts.FlagPrefix, l); err != nil {
			return nil, fmt.Errorf("failed to register flags: %w", err)
		}
	}

	loadCtx := &LoadContext{
		FieldSources: make(map[string]string),
		Target:       ptr,
		Context:      ctx,
	}

	passes, err := l.buildPasses()
	if err != nil {
		return nil, fmt.Errorf("building pass order: %w", err)
	}

	for i, pass := range passes {
		if err := pass.operation(ptr, loadCtx); err != nil {
			return nil, fmt.Errorf("pass %d (%s) failed: %w", i+1, pass.name, err)
		}
	}

	return loadCtx, nil
}

// shouldUseFlagCoordinator determines if the loader should use the flag
// coordinator for flag parsing.
//
// Returns bool which is true when PassFlags is in the pass order or the order
// is empty (defaulting to include PassFlags).
func (l *Loader) shouldUseFlagCoordinator() bool {
	order := l.opts.PassOrder
	if len(order) == 0 {
		return true
	}

	return slices.Contains(order, PassFlags)
}

// buildResolverMap fills the resolver map using the options and the global
// registry.
func (l *Loader) buildResolverMap() {
	if l.opts.UseGlobalResolvers {
		globalResolvers := l.resolverRegistry.GetAll()
		l.opts.Resolvers = append(globalResolvers, l.opts.Resolvers...)
	}
	for _, r := range l.opts.Resolvers {
		if prefix := r.GetPrefix(); prefix != "" {
			l.resolverMap[prefix] = r
		}
	}
}

// passDefinition describes a single step in the configuration loading process.
type passDefinition struct {
	// operation is the function to run for this pass.
	operation func(any, *LoadContext) error

	// name identifies the pass for error messages.
	name string
}

// buildPasses creates the list of passes to run based on LoaderOptions.
//
// Returns []passDefinition which contains the passes in their run order.
// Returns error when PassOrder contains an unknown pass name.
func (l *Loader) buildPasses() ([]passDefinition, error) {
	passMap := map[Pass]passDefinition{
		PassDefaults:              {operation: l.applyDefaults, name: PassDefaults.String()},
		PassFiles:                 {operation: l.loadFiles, name: PassFiles.String()},
		PassDotEnv:                {operation: l.applyDotEnv, name: PassDotEnv.String()},
		PassEnv:                   {operation: l.applyEnvVars, name: PassEnv.String()},
		PassFlags:                 {operation: l.parseFlags, name: PassFlags.String()},
		PassResolvers:             {operation: l.resolvePlaceholders, name: PassResolvers.String()},
		PassProgrammaticOverrides: {operation: l.applyProgrammaticOverrides, name: PassProgrammaticOverrides.String()},
		PassValidation:            {operation: l.validateConfig, name: PassValidation.String()},
		PassProgrammatic:          {operation: l.applyProgrammaticDefaults, name: PassProgrammatic.String()},
	}

	order := l.opts.PassOrder
	if len(order) == 0 {
		order = []Pass{
			PassDefaults,
			PassProgrammatic,
			PassFiles,
			PassDotEnv,
			PassEnv,
			PassFlags,
			PassResolvers,
			PassProgrammaticOverrides,
			PassValidation,
		}
	}

	execPasses := make([]passDefinition, 0, len(order))
	for _, p := range order {
		definition, ok := passMap[p]
		if !ok {
			return nil, fmt.Errorf("invalid or unknown pass specified in PassOrder: %d", p)
		}
		execPasses = append(execPasses, definition)
	}
	return execPasses, nil
}

// applyProgrammaticDefaults implements the programmatic
// defaults pass. It recursively merges the provided default
// struct into the target struct.
//
// Takes ptr (any) which is the target configuration struct to receive
// default values.
//
// Returns error when merging the default values into the target fails.
func (l *Loader) applyProgrammaticDefaults(ptr any, _ *LoadContext) error {
	if isNil(l.opts.ProgrammaticDefaults) {
		return nil
	}

	targetVal := reflect.Indirect(reflect.ValueOf(ptr))
	defaultsVal := reflect.Indirect(reflect.ValueOf(l.opts.ProgrammaticDefaults))

	if !targetVal.IsValid() || !defaultsVal.IsValid() {
		return nil
	}

	return mergeStructs(targetVal, defaultsVal)
}

// applyProgrammaticOverrides forcefully merges non-zero fields from the
// ProgrammaticOverrides struct onto the target, overwriting any values that
// were set by earlier passes (files, env vars, flags, resolvers). This ensures
// values set via functional options always take highest precedence.
//
// Takes ptr (any) which is the target struct to receive the overrides.
// Takes ctx (*LoadContext) which provides the loading context.
//
// Returns error when merging the override values fails.
func (l *Loader) applyProgrammaticOverrides(ptr any, ctx *LoadContext) error {
	if isNil(l.opts.ProgrammaticOverrides) {
		return nil
	}

	targetVal := reflect.Indirect(reflect.ValueOf(ptr))
	overridesVal := reflect.Indirect(reflect.ValueOf(l.opts.ProgrammaticOverrides))

	if !targetVal.IsValid() || !overridesVal.IsValid() {
		return nil
	}

	return overrideStructs(targetVal, overridesVal, "", ctx)
}

// validateConfig is the implementation for the validation pass.
//
// When no validator is configured, the pass is silently skipped.
//
// Takes ptr (any) which is the configuration struct to validate against
// its struct tag rules.
//
// Returns error when any field fails its validation constraint.
func (l *Loader) validateConfig(ptr any, _ *LoadContext) error {
	if l.validator == nil {
		return nil
	}
	if err := l.validator.Struct(ptr); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}

// Load creates a new Loader with default settings and orchestrates the entire
// configuration loading process.
//
// Takes ptr (any) which is the configuration struct to populate.
//
// Returns *LoadContext which contains metadata about the loaded configuration.
// Returns error when loading fails.
func Load(ctx context.Context, ptr any, opts LoaderOptions) (*LoadContext, error) {
	loader := NewLoader(opts, WithDefaultResolvers())
	defer loader.Close()
	return loader.Load(ctx, ptr)
}

// WithDefaultResolvers returns an option that adds all built-in resolvers
// which have no external dependencies.
//
// Returns LoaderOption which sets up the default resolvers on a Loader.
func WithDefaultResolvers() LoaderOption {
	return func(l *Loader) {
		l.opts.Resolvers = append(l.opts.Resolvers,
			&EnvResolver{},
			&Base64Resolver{},
			NewFileResolver(nil),
		)
	}
}

// newResolverCircuitBreaker creates a circuit breaker for resolver operations.
//
// Returns *gobreaker.CircuitBreaker[any] configured with standard settings
// for config resolver operations.
func newResolverCircuitBreaker() *gobreaker.CircuitBreaker[any] {
	settings := gobreaker.Settings{
		Name:         "config-resolver",
		MaxRequests:  1,
		Interval:     0,
		Timeout:      circuitBreakerTimeout,
		BucketPeriod: circuitBreakerBucketPeriod,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= circuitBreakerConsecutiveFailures
		},
		IsExcluded: func(err error) bool {
			return errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded)
		},
	}
	return gobreaker.NewCircuitBreaker[any](settings)
}

// createResolverCache creates an otter cache with the specified TTL.
// Returns nil if TTL is zero or negative.
//
// Takes ttl (*time.Duration) which specifies the cache entry lifetime,
// or nil to use the default TTL.
//
// Returns *otter.Cache[string, string] which is the configured cache,
// or nil when the TTL is zero or negative.
func createResolverCache(ttl *time.Duration) *otter.Cache[string, string] {
	cacheTTL := defaultResolverCacheTTL
	if ttl != nil {
		cacheTTL = *ttl
	}
	if cacheTTL <= 0 {
		return nil
	}
	return otter.Must(&otter.Options[string, string]{
		MaximumSize:      defaultResolverCacheSize,
		ExpiryCalculator: otter.ExpiryCreating[string, string](cacheTTL),
	})
}

// getFileReader returns the given file reader or a default OS file reader.
//
// Takes fr (FileReader) which is the file reader to use, or nil for default.
//
// Returns FileReader which is the given reader, or osFileReader if fr is nil.
func getFileReader(fr FileReader) FileReader {
	if fr == nil {
		return osFileReader{}
	}
	return fr
}

// getFlagCoordinator returns the given coordinator or the global one.
//
// Takes fc (*FlagCoordinator) which is the coordinator to use, or nil to use
// the global one.
//
// Returns *FlagCoordinator which is either fc if provided, or the global
// coordinator if fc is nil.
func getFlagCoordinator(fc *FlagCoordinator) *FlagCoordinator {
	if fc == nil {
		return GetGlobalFlagCoordinator()
	}
	return fc
}

// getResolverRegistry returns the given registry or falls back to the global
// one.
//
// Takes rr (*ResolverRegistry) which is the registry to use, or nil to use the
// global registry.
//
// Returns *ResolverRegistry which is the given registry, or the global
// registry if rr is nil.
func getResolverRegistry(rr *ResolverRegistry) *ResolverRegistry {
	if rr == nil {
		return GetGlobalResolverRegistry()
	}
	return rr
}

// overrideStructs copies non-zero values from an overrides struct to a target
// struct unconditionally, overwriting existing values. Unlike mergeStructs
// (used for defaults), this does not check whether the target field already has
// a value - the override always wins.
//
// Takes target (reflect.Value) which is the struct to receive overridden
// values.
// Takes overrides (reflect.Value) which contains the values to apply.
// Takes prefix (string) which specifies the field path prefix for error
// messages.
// Takes ctx (*LoadContext) which provides the loading context.
//
// Returns error when a field override fails to apply.
func overrideStructs(target, overrides reflect.Value, prefix string, ctx *LoadContext) error {
	if target.Kind() != reflect.Struct || overrides.Kind() != reflect.Struct {
		return nil
	}

	for overrideFieldType, overrideField := range overrides.Fields() {
		if err := applyOverrideField(target, overrideFieldType, overrideField, prefix, ctx); err != nil {
			return fmt.Errorf("overriding field %q: %w", overrideFieldType.Name, err)
		}
	}
	return nil
}

// applyOverrideField copies a single non-zero field value from overrides to
// target unconditionally.
//
// Takes target (reflect.Value) which is the struct to copy the field into.
// Takes fieldType (reflect.StructField) which describes the field metadata.
// Takes overrideField (reflect.Value) which is the source value to copy.
// Takes prefix (string) which is the key prefix for field source tracking.
// Takes ctx (*LoadContext) which tracks where field values originate.
//
// Returns error when overriding nested struct fields fails.
func applyOverrideField(target reflect.Value, fieldType reflect.StructField, overrideField reflect.Value, prefix string, ctx *LoadContext) error {
	if !fieldType.IsExported() {
		return nil
	}

	targetField := target.FieldByName(fieldType.Name)
	if !targetField.IsValid() || !targetField.CanSet() {
		return nil
	}

	if overrideField.IsZero() {
		return nil
	}

	key := fieldType.Name
	if prefix != "" {
		key = prefix + "." + key
	}

	switch overrideField.Kind() {
	case reflect.Struct:
		return overrideStructs(targetField, overrideField, key, ctx)
	case reflect.Pointer:
		if overrideField.Elem().Kind() == reflect.Struct {
			if targetField.IsNil() {
				targetField.Set(reflect.New(targetField.Type().Elem()))
			}
			return overrideStructs(targetField.Elem(), overrideField.Elem(), key, ctx)
		}
		targetField.Set(overrideField)
	default:
		targetField.Set(overrideField)
	}

	if ctx != nil {
		ctx.FieldSources[key] = "programmatic_override"
	}
	return nil
}

// isNil checks if an interface is nil or holds a typed nil pointer.
//
// Takes i (any) which is the value to check.
//
// Returns bool which is true if i is nil or holds a typed nil pointer.
func isNil(i any) bool {
	if i == nil {
		return true
	}

	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// mergeStructs copies non-zero values from a defaults struct to a target struct
// by processing each field in turn.
//
// Takes target (reflect.Value) which is the struct to receive merged values.
// Takes defaults (reflect.Value) which provides the default values.
//
// Returns error when a field cannot be merged.
func mergeStructs(target, defaults reflect.Value) error {
	if target.Kind() != reflect.Struct || defaults.Kind() != reflect.Struct {
		return nil
	}

	for defaultFieldType, defaultField := range defaults.Fields() {
		if err := mergeField(target, defaultFieldType, defaultField); err != nil {
			return fmt.Errorf("merging field %q: %w", defaultFieldType.Name, err)
		}
	}
	return nil
}

// mergeField copies a single field value from defaults to target.
//
// Takes target (reflect.Value) which is the struct to merge values into.
// Takes defaultFieldType (reflect.StructField) which describes the field to
// merge.
// Takes defaultField (reflect.Value) which provides the default value.
//
// Returns error when the field value merge fails.
func mergeField(target reflect.Value, defaultFieldType reflect.StructField, defaultField reflect.Value) error {
	if !defaultFieldType.IsExported() {
		return nil
	}

	targetField := target.FieldByName(defaultFieldType.Name)
	if !targetField.IsValid() || !targetField.CanSet() {
		return nil
	}

	if defaultField.IsZero() {
		return nil
	}

	return mergeFieldValue(targetField, defaultField)
}

// mergeFieldValue merges a value into a target field based on the field kind.
//
// Takes targetField (reflect.Value) which is the field to merge into.
// Takes defaultField (reflect.Value) which provides the value to merge from.
//
// Returns error when merging nested structs or pointer fields fails.
func mergeFieldValue(targetField, defaultField reflect.Value) error {
	switch defaultField.Kind() {
	case reflect.Struct:
		return mergeStructs(targetField, defaultField)
	case reflect.Pointer:
		return mergePtrField(targetField, defaultField)
	default:
		targetField.Set(defaultField)
		return nil
	}
}

// mergePtrField merges pointer fields, including pointers to structs.
//
// When the default field does not point to a struct, it copies the default
// value to the target. When the target is nil, it creates a new value before
// merging.
//
// Takes targetField (reflect.Value) which is the pointer field to merge into.
// Takes defaultField (reflect.Value) which is the pointer field with defaults.
//
// Returns error when the struct merge fails.
func mergePtrField(targetField, defaultField reflect.Value) error {
	if defaultField.Elem().Kind() != reflect.Struct {
		targetField.Set(defaultField)
		return nil
	}

	if targetField.IsNil() {
		targetField.Set(reflect.New(targetField.Type().Elem()))
	}
	return mergeStructs(targetField.Elem(), defaultField.Elem())
}
