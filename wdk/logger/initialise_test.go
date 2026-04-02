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

package logger

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveAddSource_OutputSpecificTrue(t *testing.T) {
	assert.True(t, resolveAddSource(new(true), false))
}

func TestResolveAddSource_OutputSpecificFalse(t *testing.T) {
	assert.False(t, resolveAddSource(new(false), true))
}

func TestResolveAddSource_NilUsesGlobalTrue(t *testing.T) {
	assert.True(t, resolveAddSource(nil, true))
}

func TestResolveAddSource_NilUsesGlobalFalse(t *testing.T) {
	assert.False(t, resolveAddSource(nil, false))
}

func TestBuildCoreHandler_SingleHandler(t *testing.T) {
	handler := slog.NewTextHandler(io.Discard, nil)
	result := buildCoreHandler([]slog.Handler{handler})
	assert.NotNil(t, result)
	assert.Equal(t, handler, result)
}

func TestBuildCoreHandler_MultipleHandlers(t *testing.T) {
	h1 := slog.NewTextHandler(io.Discard, nil)
	h2 := slog.NewJSONHandler(io.Discard, nil)
	result := buildCoreHandler([]slog.Handler{h1, h2})
	assert.NotNil(t, result)

	assert.IsType(t, &slog.MultiHandler{}, result)
}

func TestWithLevel_SetsLevel(t *testing.T) {
	config := &outputConfig{}
	opt := WithLevel(slog.LevelWarn)
	opt(config)

	assert.NotNil(t, config.level)
	assert.Equal(t, slog.LevelWarn, *config.level)
}

func TestWithJSON_SetsFlag(t *testing.T) {
	config := &outputConfig{}
	opt := WithJSON()
	opt(config)

	assert.True(t, config.asJSON)
}

func TestWithNoColour_SetsFlag(t *testing.T) {
	config := &outputConfig{}
	opt := WithNoColour()
	opt(config)

	assert.True(t, config.noColour)
}

func TestResolveLevel_ExplicitLevel(t *testing.T) {
	config := &outputConfig{level: new(slog.LevelError)}
	assert.Equal(t, slog.LevelError, config.resolveLevel())
}

func TestResolveLevel_NilLevel_DefaultsToInfo(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	config := &outputConfig{}
	result := config.resolveLevel()
	assert.Equal(t, slog.LevelInfo, result)
}

func TestResolveLevel_NilLevel_ReadsEnvVar(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	config := &outputConfig{}
	result := config.resolveLevel()
	assert.Equal(t, slog.LevelDebug, result)
}
