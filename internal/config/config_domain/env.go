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
	"fmt"
	"os"
	"reflect"
)

// lookuper defines a way to look up values by key.
type lookuper interface {
	// Lookup retrieves the value for the given key.
	//
	// Takes key (string) which identifies the value to retrieve.
	//
	// Returns string which is the value if found.
	// Returns bool which is true if the key exists, false otherwise.
	Lookup(key string) (string, bool)
}

// osLookuper implements lookuper using the operating system environment.
type osLookuper struct{}

// Lookup retrieves an environment variable by name.
//
// Takes key (string) which specifies the environment variable name.
//
// Returns string which contains the value of the environment variable.
// Returns bool which indicates whether the variable was found.
func (osLookuper) Lookup(key string) (string, bool) { return os.LookupEnv(key) }

// applyEnvVars applies environment variable values to the struct at ptr.
//
// Takes ptr (any) which is a pointer to the struct to populate.
// Takes ctx (*LoadContext) which tracks the loading state and errors.
//
// Returns error when walking the struct fields fails.
func (l *Loader) applyEnvVars(ptr any, ctx *LoadContext) error {
	lookuper := osLookuper{}
	processor := func(field *reflect.StructField, value reflect.Value, _ string, _ string) error {
		return processEnv(field, value, l.opts.EnvPrefix, lookuper)
	}
	state := &walkState{
		processor: processor,
		ctx:       ctx,
		keyPrefix: "",
		source:    sourceEnv,
	}
	return l.walk(reflect.ValueOf(ptr), state)
}

// processEnv sets a struct field value from an environment variable.
//
// Takes field (*reflect.StructField) which provides the struct tag metadata.
// Takes value (reflect.Value) which is the field to set.
// Takes envPrefix (string) which is added before the environment key.
// Takes lookuper (lookuper) which looks up environment variable values.
//
// Returns error when the field cannot be set from the environment value.
func processEnv(field *reflect.StructField, value reflect.Value, envPrefix string, lookuper lookuper) error {
	envKey, ok := field.Tag.Lookup("env")
	if !ok {
		return nil
	}
	envValue, present := lookuper.Lookup(envPrefix + envKey)
	if !present {
		return nil
	}
	if err := setField(value, envValue, field.Tag); err != nil {
		return fmt.Errorf("env var %q: %w", envKey, err)
	}
	return nil
}
