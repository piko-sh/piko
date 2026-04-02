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
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
)

// prefixSeparator is the character placed between a flag prefix and its name.
const prefixSeparator = "."

// FlagCoordinator manages a shared FlagSet that allows multiple config structs
// to register flags without conflicts. It solves the problem where Go's flag
// package can only parse os.Args once.
type FlagCoordinator struct {
	// registrations maps prefixes to their flag registration data.
	registrations map[string]*flagRegistration

	// flagSet holds the FlagSet used to parse command-line flags.
	flagSet *flag.FlagSet

	// mu guards the parsed and registrations fields.
	mu sync.Mutex

	// parsed indicates whether flags have been parsed.
	parsed bool
}

// flagRegistration holds the state needed to register a flag with its loader.
type flagRegistration struct {
	// ptr holds the pointer to the flag value storage.
	ptr any

	// loader is the configuration loader that registered this flag.
	loader *Loader

	// prefix is added to the start of all flag names when adding them to the flag set.
	prefix string
}

var (
	globalCoordinator *FlagCoordinator

	globalCoordinatorOnce sync.Once
)

// RegisterStruct registers a struct's flags with an optional prefix.
//
// The prefix is prepended to all flag names. For example, prefix "app" makes
// flag "dbUrl" into "app.dbUrl". If prefix is empty, flags are registered
// without a prefix. This must be called before Parse is called.
//
// Takes ptr (any) which is a pointer to the struct containing flag fields.
// Takes prefix (string) which is prepended to all flag names.
// Takes loader (*Loader) which provides the flag loading configuration.
//
// Returns error when flags have already been parsed or flag definition fails.
//
// Safe for concurrent use. If the prefix is already registered, the call is
// a no-op and returns nil.
func (fc *FlagCoordinator) RegisterStruct(ptr any, prefix string, loader *Loader) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if fc.parsed {
		return errors.New("cannot register struct after flags have been parsed")
	}

	if prefix != "" && !strings.HasSuffix(prefix, prefixSeparator) {
		prefix = prefix + prefixSeparator
	}

	if _, exists := fc.registrations[prefix]; exists {
		return nil
	}

	reg := &flagRegistration{
		prefix: prefix,
		ptr:    ptr,
		loader: loader,
	}
	fc.registrations[prefix] = reg

	if err := fc.defineAllFlagsWithPrefix(ptr, prefix, loader); err != nil {
		delete(fc.registrations, prefix)
		return fmt.Errorf("failed to define flags for prefix %q: %w", prefix, err)
	}

	return nil
}

// Parse parses os.Args using the shared FlagSet.
// This method can only be called once; subsequent calls return immediately.
//
// Unknown flags (those not registered with the coordinator) are silently
// ignored, so commands can define their own flags using the standard flag
// package without conflicting with the config system.
//
// Returns error when flag parsing fails for a known flag.
//
// Safe for concurrent use. Uses a mutex to ensure only one parse occurs.
func (fc *FlagCoordinator) Parse() error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if fc.parsed {
		return nil
	}

	filteredArgs := fc.filterKnownFlags(filterTestFlags(os.Args[1:]))

	if err := fc.flagSet.Parse(filteredArgs); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	fc.parsed = true
	return nil
}

// GetVisitedFlags returns a map of flag names that were visited (set by user)
// for the given prefix. Flag names are returned without the prefix.
//
// Takes prefix (string) which filters flags by their prefix. Use an empty
// string to get flags without any prefix (those without dots in their names).
//
// Returns map[string]*flag.Flag which contains the visited flags keyed by
// their names with the prefix stripped.
//
// Safe for concurrent use.
func (fc *FlagCoordinator) GetVisitedFlags(prefix string) map[string]*flag.Flag {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if prefix != "" && !strings.HasSuffix(prefix, prefixSeparator) {
		prefix = prefix + prefixSeparator
	}

	visited := make(map[string]*flag.Flag)
	fc.flagSet.Visit(func(f *flag.Flag) {
		if prefix == "" {
			if !strings.Contains(f.Name, prefixSeparator) {
				visited[f.Name] = f
			}
		} else {
			if nameWithoutPrefix, ok := strings.CutPrefix(f.Name, prefix); ok {
				visited[nameWithoutPrefix] = f
			}
		}
	})

	return visited
}

// IsParsed returns true if Parse has been called.
//
// Returns bool which indicates whether parsing has completed.
//
// Safe for concurrent use.
func (fc *FlagCoordinator) IsParsed() bool {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.parsed
}

// Reset resets the coordinator to allow re-registration. This is primarily
// for testing purposes.
//
// Safe for concurrent use.
func (fc *FlagCoordinator) Reset() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.flagSet = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fc.parsed = false
	fc.registrations = make(map[string]*flagRegistration)
}

// filterKnownFlags returns only the arguments that correspond to flags
// registered with this coordinator's FlagSet.
//
// Takes arguments ([]string) which contains the command line arguments to filter.
//
// Returns []string which contains only the arguments for known flags.
func (fc *FlagCoordinator) filterKnownFlags(arguments []string) []string {
	knownFlags := make(map[string]bool)
	fc.flagSet.VisitAll(func(f *flag.Flag) {
		knownFlags[f.Name] = true
	})

	filtered := make([]string, 0, len(arguments))
	skipNext := false

	for i, argument := range arguments {
		if skipNext {
			skipNext = false
			continue
		}

		if argument == "--" {
			filtered = append(filtered, arguments[i:]...)
			break
		}

		if !strings.HasPrefix(argument, "-") {
			filtered = append(filtered, argument)
			continue
		}

		flagName, hasValue := extractFlagName(argument)

		if knownFlags[flagName] {
			filtered = append(filtered, argument)
			continue
		}

		if !hasValue && i+1 < len(arguments) && !strings.HasPrefix(arguments[i+1], "-") {
			skipNext = true
		}
	}

	return filtered
}

// defineAllFlagsWithPrefix registers command-line flags for all tagged fields
// in a struct, applying the given prefix to each flag name.
//
// Takes ptr (any) which is a pointer to the struct to define flags for.
// Takes prefix (string) which is prepended to each flag name.
// Takes loader (*Loader) which provides the struct traversal logic.
//
// Returns error when flag definition fails for any field.
func (fc *FlagCoordinator) defineAllFlagsWithPrefix(ptr any, prefix string, loader *Loader) error {
	state := &walkState{
		processor: func(field *reflect.StructField, value reflect.Value, _, _ string) error {
			return fc.defineFlagWithPrefix(field, value, prefix)
		},
		ctx:       nil,
		keyPrefix: "",
		source:    "",
	}
	return loader.walk(reflect.ValueOf(ptr), state)
}

// defineFlagWithPrefix registers a flag for a struct field with a name prefix.
//
// Takes field (*reflect.StructField) which provides the field metadata and tag.
// Takes value (reflect.Value) which is the field value to bind to the flag.
// Takes prefix (string) which is added before the flag name from the tag.
//
// Returns error when flag registration fails.
func (fc *FlagCoordinator) defineFlagWithPrefix(field *reflect.StructField, value reflect.Value, prefix string) error {
	flagName, ok := field.Tag.Lookup("flag")
	if !ok || flagName == "" {
		return nil
	}
	if !value.CanAddr() {
		return nil
	}

	fullFlagName := prefix + flagName
	usage := buildFlagUsage(field.Tag)

	flagValue := value
	if flagValue.Kind() == reflect.Pointer && flagValue.Type().Elem().Kind() != reflect.Struct {
		if flagValue.IsNil() {
			flagValue.Set(reflect.New(flagValue.Type().Elem()))
		}
		flagValue = flagValue.Elem()
	}

	ptr := flagValue.Addr().Interface()

	return defineFlagOnFlagSet(fc.flagSet, fullFlagName, ptr, flagValue, field.Tag, usage)
}

// GetGlobalFlagCoordinator returns the singleton flag coordinator.
//
// Returns *FlagCoordinator which is the shared coordinator instance,
// created on the first call.
func GetGlobalFlagCoordinator() *FlagCoordinator {
	globalCoordinatorOnce.Do(func() {
		globalCoordinator = &FlagCoordinator{
			registrations: make(map[string]*flagRegistration),
			flagSet:       flag.NewFlagSet(os.Args[0], flag.ContinueOnError),
			mu:            sync.Mutex{},
			parsed:        false,
		}
	})
	return globalCoordinator
}

// ResetGlobalFlagCoordinator resets the global flag coordinator singleton.
// This is used for testing to ensure each test runs in isolation.
func ResetGlobalFlagCoordinator() {
	globalCoordinatorOnce = sync.Once{}
	globalCoordinator = nil
}

// newFlagCoordinator creates a new, isolated flag coordinator.
// Use it in tests that need a coordinator that does not share state with
// other tests.
//
// Returns *FlagCoordinator which is the newly created coordinator.
func newFlagCoordinator() *FlagCoordinator {
	return &FlagCoordinator{
		registrations: make(map[string]*flagRegistration),
		flagSet:       flag.NewFlagSet("test", flag.ContinueOnError),
		mu:            sync.Mutex{},
		parsed:        false,
	}
}

// extractFlagName extracts the flag name from a command-line argument.
//
// Takes argument (string) which is the command-line argument to parse.
//
// Returns name (string) which is the flag name with leading dashes removed.
// Returns hasValue (bool) which indicates whether the argument contains "=".
func extractFlagName(argument string) (name string, hasValue bool) {
	name = strings.TrimLeft(argument, "-")

	if before, _, found := strings.Cut(name, "="); found {
		return before, true
	}

	return name, false
}

// defineFlagOnFlagSet defines a flag on a given FlagSet based on the field
// type.
//
// Takes fs (*flag.FlagSet) which is the flag set to define the flag on.
// Takes flagName (string) which is the name of the flag to define.
// Takes ptr (any) which is a pointer to the variable that stores the flag
// value.
// Takes value (reflect.Value) which provides the default value for the flag.
// Takes tags (reflect.StructTag) which contains struct tags for configuration.
// Takes usage (string) which describes the flag in help text.
//
// Returns error when the flag type is not supported, though currently returns
// nil for unsupported types.
func defineFlagOnFlagSet(fs *flag.FlagSet, flagName string, ptr any, value reflect.Value, tags reflect.StructTag, usage string) error {
	switch typedPtr := ptr.(type) {
	case *string:
		fs.StringVar(typedPtr, flagName, value.String(), usage)
	case *int:
		fs.IntVar(typedPtr, flagName, int(value.Int()), usage)
	case *int64:
		fs.Int64Var(typedPtr, flagName, value.Int(), usage)
	case *uint:
		fs.UintVar(typedPtr, flagName, uint(value.Uint()), usage)
	case *uint64:
		fs.Uint64Var(typedPtr, flagName, value.Uint(), usage)
	case *bool:
		fs.BoolVar(typedPtr, flagName, value.Bool(), usage)
	case *time.Duration:
		fs.DurationVar(typedPtr, flagName, time.Duration(value.Int()), usage)
	case *[]string:
		fs.Var(&stringSliceValue{slice: typedPtr}, flagName, usage)
	case *map[string]string:
		fs.Var(&stringMapValue{sMap: typedPtr, tags: tags}, flagName, usage)
	default:
		return nil
	}
	return nil
}
