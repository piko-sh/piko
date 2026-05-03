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

package config_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/config"
)

type defaultsConfig struct {
	Port int    `default:"8080"`
	Host string `default:"localhost"`
}

func TestNewLoader_BuildsLoader(t *testing.T) {
	t.Parallel()

	loader := config.NewLoader(config.LoaderOptions{}, config.WithDefaultResolvers())

	require.NotNil(t, loader)
}

func TestLoad_AppliesDefaultsFromStructTags(t *testing.T) {
	t.Parallel()

	target := &defaultsConfig{}
	loadCtx, err := config.Load(context.Background(), target, config.LoaderOptions{
		PassOrder: []config.Pass{config.PassDefaults},
	})

	require.NoError(t, err)
	require.NotNil(t, loadCtx)
	require.Equal(t, 8080, target.Port)
	require.Equal(t, "localhost", target.Host)
}

func TestPassConstants_HaveDistinctValues(t *testing.T) {
	t.Parallel()

	all := []config.Pass{
		config.PassProgrammatic,
		config.PassDefaults,
		config.PassFiles,
		config.PassDotEnv,
		config.PassEnv,
		config.PassFlags,
		config.PassResolvers,
		config.PassValidation,
	}

	seen := make(map[config.Pass]struct{}, len(all))
	for _, p := range all {
		_, dup := seen[p]
		require.Falsef(t, dup, "duplicate Pass value: %v", p)
		seen[p] = struct{}{}
	}
}

func TestRegisterFlags_AcceptsValidPointer(t *testing.T) {
	t.Parallel()

	type testCfg struct {
		Port int `flag:"port"`
	}

	err := config.RegisterFlags(&testCfg{}, "uniquetestprefix.somesection")

	require.NoError(t, err)
}

func TestNewFileResolver_ReturnsResolver(t *testing.T) {
	t.Parallel()

	resolver := config.NewFileResolver(nil)

	require.NotNil(t, resolver)
}
