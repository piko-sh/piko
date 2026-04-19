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

package logger_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logger/logger_dto"
)

func setupPrettyTestLogger(t *testing.T, loggerConfig logger_dto.Config) (logger_domain.Logger, func() string) {
	t.Helper()

	buffer := new(bytes.Buffer)
	originalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	var wg sync.WaitGroup
	wg.Go(func() {
		_, _ = io.Copy(buffer, r)
	})

	slogLogger, shutdown, err := Initialise(context.Background(), loggerConfig)
	require.NoError(t, err)

	InitDefaultFactory(slogLogger)
	log := logger_domain.GetLogger("pretty-test-" + t.Name())

	getOutput := func() string {
		_ = shutdown(context.Background())
		_ = w.Close()
		wg.Wait()
		os.Stderr = originalStderr
		return buffer.String()
	}

	return log, getOutput
}

func stripAnsi(str string) string {
	const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PR-TZcf-ntqry=><~]))"
	var re = regexp.MustCompile(ansi)
	return re.ReplaceAllString(str, "")
}

func TestPrettyOutput_FormatAndFields(t *testing.T) {
	loggerConfig := loadConfigFromYAML(t, "testdata/pretty_output.yaml")
	log, getOutput := setupPrettyTestLogger(t, loggerConfig)

	log.Info("user logged in successfully",
		logger_domain.Field(logger_domain.KeyContext, "auth-service"),
		logger_domain.Field(logger_domain.KeyMethod, "/api/v1/login"),
		logger_domain.String("user_id", "usr_123"),
		logger_domain.Int("tenant_id", 99),
	)

	rawOutput := getOutput()
	t.Logf("Captured Raw (Coloured) Output:\n%q", rawOutput)

	output := stripAnsi(rawOutput)
	t.Logf("Captured Stripped (Plain Text) Output:\n%s", output)

	assert.Contains(t, output, "INFO", "Log level should be present")
	assert.Contains(t, output, "user logged in successfully", "The log message should be present")

	assert.Contains(t, output, "user_id=usr_123", "Extra string attribute should be formatted correctly")
	assert.Contains(t, output, "tenant_id=99", "Extra int attribute should be formatted correctly")

	ctxValue := "auth-service"
	ctxPadding := 28
	expectedCtxSubstring := "| " + ctxValue + strings.Repeat(" ", ctxPadding-len(ctxValue)) + " |"
	assert.Contains(t, output, expectedCtxSubstring, "The 'ctx' field should be correctly padded")

}

func TestPrettyOutput_ColorControl(t *testing.T) {
	t.Run("WithColorByDefault", func(t *testing.T) {
		loggerConfig := loadConfigFromYAML(t, "testdata/pretty_output.yaml")
		log, getOutput := setupPrettyTestLogger(t, loggerConfig)

		log.Error("this is a critical error")

		output := getOutput()
		assert.Contains(t, output, "\x1b[", "Output should contain ANSI colour codes by default")
	})

	t.Run("WithNoColourEnabled", func(t *testing.T) {
		loggerConfig := logger_dto.Config{
			Outputs: []logger_dto.OutputConfig{{
				Type: "stderr", Level: "debug", Format: "pretty", NoColour: true,
			}},
		}
		log, getOutput := setupPrettyTestLogger(t, loggerConfig)

		log.Warn("this is a simple warning")

		output := getOutput()
		assert.NotContains(t, output, "\x1b[", "Output should NOT contain ANSI colour codes when NoColour is true")
	})
}

func TestPrettyOutput_WithSource(t *testing.T) {
	loggerConfig := logger_dto.Config{
		AddSource: true,
		Outputs: []logger_dto.OutputConfig{{
			Type: "stderr", Level: "debug", Format: "pretty", NoColour: true,
		}},
	}
	log, getOutput := setupPrettyTestLogger(t, loggerConfig)

	log.Info("testing source location")

	output := getOutput()
	t.Logf("Captured Pretty Output:\n%s", output)

	sourceRegex := regexp.MustCompile(`\|\s*logger_pretty_test\.go:\d+`)
	assert.Regexp(t, sourceRegex, output, "Output should contain the source file and line number at the end")
}

func TestPrettyOutput_WithAttrs(t *testing.T) {
	loggerConfig := logger_dto.Config{
		Outputs: []logger_dto.OutputConfig{{
			Type: "stderr", Level: "debug", Format: "pretty", NoColour: true,
		}},
	}
	log, getOutput := setupPrettyTestLogger(t, loggerConfig)

	contextualLog := log.With(
		logger_domain.String("service", "billing"),
		logger_domain.String("request_id", "request-abc-123"),
	)

	contextualLog.Info("invoice generated", logger_domain.Int("invoice_id", 456))

	output := getOutput()
	t.Logf("Captured Pretty Output:\n%s", output)

	assert.Contains(t, output, "service=billing")
	assert.Contains(t, output, "request_id=request-abc-123")
	assert.Contains(t, output, "invoice_id=456")
}

func TestPrettyOutput_GroupedAttrs(t *testing.T) {
	loggerConfig := logger_dto.Config{
		Outputs: []logger_dto.OutputConfig{{
			Type: "stderr", Level: "debug", Format: "pretty", NoColour: true,
		}},
	}
	log, getOutput := setupPrettyTestLogger(t, loggerConfig)

	log.Warn("permission check failed",
		logger_domain.Field("user.id", 789),
		logger_domain.Field("resource.type", "document"),
	)

	output := getOutput()
	t.Logf("Captured Pretty Output:\n%s", output)

	assert.Contains(t, output, "user.id=789")
	assert.Contains(t, output, "resource.type=document")
}

func TestPrettyOutput_MultilineMessage(t *testing.T) {
	loggerConfig := logger_dto.Config{
		Outputs: []logger_dto.OutputConfig{{
			Type: "stderr", Level: "debug", Format: "pretty", NoColour: true,
		}},
	}
	log, getOutput := setupPrettyTestLogger(t, loggerConfig)

	log.Error("an error occurred with a stack trace:\n  at DoThing (file.go:123)\n  at main.main (main.go:45)")

	output := getOutput()
	assert.Contains(t, output, "an error occurred with a stack trace:\n  at DoThing (file.go:123)\n  at main.main (main.go:45)")
}

func TestPrettyOutput_ConcurrencySafety(t *testing.T) {
	loggerConfig := logger_dto.Config{
		Outputs: []logger_dto.OutputConfig{{
			Type: "stderr", Level: "debug", Format: "pretty", NoColour: true,
		}},
	}
	log, getOutput := setupPrettyTestLogger(t, loggerConfig)

	numGoroutines := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			log.Info("log from concurrent goroutine", logger_domain.Int("goroutine_id", id))
		}(i)
	}

	wg.Wait()

	output := getOutput()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(t, lines, numGoroutines, "Should have exactly one uncorrupted log line per goroutine")

	assert.Contains(t, lines[0], "log from concurrent goroutine")
	assert.Contains(t, lines[0], "goroutine_id=")
	assert.Contains(t, lines[len(lines)-1], "log from concurrent goroutine")
	assert.Contains(t, lines[len(lines)-1], "goroutine_id=")
}

func TestPrettyOutput_CustomLevels(t *testing.T) {
	t.Run("Notice level displays as NOTIC", func(t *testing.T) {
		loggerConfig := logger_dto.Config{
			Outputs: []logger_dto.OutputConfig{{
				Type: "stderr", Level: "trace", Format: "pretty", NoColour: true,
			}},
		}
		log, getOutput := setupPrettyTestLogger(t, loggerConfig)

		log.Notice("this is a notice message")

		rawOutput := getOutput()
		output := stripAnsi(rawOutput)
		t.Logf("Captured output:\n%s", output)

		assert.Contains(t, output, "NOTICE", "NOTICE level should display correctly")
		assert.Contains(t, output, "this is a notice message", "The log message should be present")

		assert.NotContains(t, output, "INFO+2", "Should NOT show INFO+2 for NOTICE level")
	})

	t.Run("Trace level displays as TRACE", func(t *testing.T) {
		loggerConfig := logger_dto.Config{
			Outputs: []logger_dto.OutputConfig{{
				Type: "stderr", Level: "trace", Format: "pretty", NoColour: true,
			}},
		}
		log, getOutput := setupPrettyTestLogger(t, loggerConfig)

		log.Trace("this is a trace message")

		rawOutput := getOutput()
		output := stripAnsi(rawOutput)
		t.Logf("Captured output:\n%s", output)

		assert.Contains(t, output, "TRACE", "TRACE level should display correctly")
		assert.Contains(t, output, "this is a trace message", "The log message should be present")

		assert.NotContains(t, output, "DEBUG-4", "Should NOT show DEBUG-4 for TRACE level")
	})

	t.Run("Internal level displays as INTERNAL", func(t *testing.T) {
		loggerConfig := logger_dto.Config{
			Outputs: []logger_dto.OutputConfig{{
				Type: "stderr", Level: "trace", Format: "pretty", NoColour: true,
			}},
		}
		log, getOutput := setupPrettyTestLogger(t, loggerConfig)

		log.Internal("this is an internal message")

		rawOutput := getOutput()
		output := stripAnsi(rawOutput)
		t.Logf("Captured output:\n%s", output)

		assert.Contains(t, output, "INTERNAL", "INTERNAL level should display correctly")
		assert.Contains(t, output, "this is an internal message", "The log message should be present")

		assert.NotContains(t, output, "DEBUG-2", "Should NOT show DEBUG-2 for INTERNAL level")
	})
}
