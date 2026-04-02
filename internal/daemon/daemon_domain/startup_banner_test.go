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

package daemon_domain

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestBuildStartupBannerInfo(t *testing.T) {
	t.Run("BasicConfig", func(t *testing.T) {
		t.Parallel()

		config := DaemonConfig{
			NetworkPort:         "8080",
			NetworkAutoNextPort: false,
			HealthEnabled:       false,
		}

		info := BuildStartupBannerInfo(config, "dev", "1.0.0")

		if info.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got %q", info.Version)
		}
		if info.Mode != "dev" {
			t.Errorf("expected mode 'dev', got %q", info.Mode)
		}
		if info.ServerURL != "http://localhost:8080" {
			t.Errorf("expected 'http://localhost:8080', got %q", info.ServerURL)
		}
		if info.HealthProbeURL != "" {
			t.Errorf("expected empty health probe URL, got %q", info.HealthProbeURL)
		}
		if info.AutoPort {
			t.Error("expected AutoPort=false")
		}
		if !info.ServerExposed {
			t.Error("expected ServerExposed=true")
		}
	})

	t.Run("WithHealthProbe", func(t *testing.T) {
		t.Parallel()

		config := DaemonConfig{
			NetworkPort:       "8080",
			HealthEnabled:     true,
			HealthBindAddress: "127.0.0.1",
			HealthPort:        "9090",
			HealthLivePath:    "/live",
			HealthReadyPath:   "/ready",
		}

		info := BuildStartupBannerInfo(config, "prod", "2.0.0")

		if info.HealthProbeURL != "http://127.0.0.1:9090" {
			t.Errorf("expected 'http://127.0.0.1:9090', got %q", info.HealthProbeURL)
		}
		if info.LivePath != "/live" {
			t.Errorf("expected '/live', got %q", info.LivePath)
		}
		if info.ReadyPath != "/ready" {
			t.Errorf("expected '/ready', got %q", info.ReadyPath)
		}
		if info.HealthExposed {
			t.Error("expected HealthExposed=false for 127.0.0.1")
		}
	})

	t.Run("HealthProbeDisabled", func(t *testing.T) {
		t.Parallel()

		config := DaemonConfig{
			NetworkPort:       "8080",
			HealthEnabled:     false,
			HealthBindAddress: "0.0.0.0",
			HealthPort:        "9090",
		}

		info := BuildStartupBannerInfo(config, "dev", "1.0.0")

		if info.HealthProbeURL != "" {
			t.Errorf("expected empty health probe URL when disabled, got %q", info.HealthProbeURL)
		}
	})

	t.Run("AutoPort", func(t *testing.T) {
		t.Parallel()

		config := DaemonConfig{
			NetworkPort:         "8080",
			NetworkAutoNextPort: true,
			HealthEnabled:       false,
		}

		info := BuildStartupBannerInfo(config, "dev", "1.0.0")

		if !info.AutoPort {
			t.Error("expected AutoPort=true")
		}
	})

	t.Run("HealthExposed", func(t *testing.T) {
		t.Parallel()

		config := DaemonConfig{
			NetworkPort:       "8080",
			HealthEnabled:     true,
			HealthBindAddress: "0.0.0.0",
			HealthPort:        "9090",
			HealthLivePath:    "/live",
			HealthReadyPath:   "/ready",
		}

		info := BuildStartupBannerInfo(config, "dev", "1.0.0")

		if !info.HealthExposed {
			t.Error("expected HealthExposed=true for 0.0.0.0")
		}
	})
}

func TestFormatMode(t *testing.T) {
	t.Run("Dev", func(t *testing.T) {
		t.Parallel()
		if got := formatMode("dev"); got != "Development" {
			t.Errorf("expected 'Development', got %q", got)
		}
	})

	t.Run("DevInterpreted", func(t *testing.T) {
		t.Parallel()
		if got := formatMode("dev-i"); got != "Development (Interpreted)" {
			t.Errorf("expected 'Development (Interpreted)', got %q", got)
		}
	})

	t.Run("Prod", func(t *testing.T) {
		t.Parallel()
		if got := formatMode("prod"); got != "Production" {
			t.Errorf("expected 'Production', got %q", got)
		}
	})

	t.Run("Unknown", func(t *testing.T) {
		t.Parallel()
		if got := formatMode("custom"); got != "custom" {
			t.Errorf("expected 'custom' passthrough, got %q", got)
		}
	})
}

func TestStripANSI(t *testing.T) {
	t.Run("PlainText", func(t *testing.T) {
		t.Parallel()
		if got := stripANSI("hello world"); got != "hello world" {
			t.Errorf("expected 'hello world', got %q", got)
		}
	})

	t.Run("WithColourCodes", func(t *testing.T) {
		t.Parallel()
		input := "\x1b[31mred\x1b[0m text"
		got := stripANSI(input)
		if got != "red text" {
			t.Errorf("expected 'red text', got %q", got)
		}
	})
}

func TestBuildServerURL(t *testing.T) {
	t.Run("NoAutoPort", func(t *testing.T) {
		t.Parallel()

		info := StartupBannerInfo{
			ServerURL: "http://localhost:8080",
			AutoPort:  false,
		}

		result := buildServerURL(info)
		stripped := stripANSI(result)

		if !strings.Contains(stripped, "http://localhost:8080") {
			t.Errorf("expected URL in output, got %q", stripped)
		}
		if strings.Contains(stripped, "or next available") {
			t.Error("did not expect auto-port suffix")
		}
	})

	t.Run("WithAutoPort", func(t *testing.T) {
		t.Parallel()

		info := StartupBannerInfo{
			ServerURL: "http://localhost:8080",
			AutoPort:  true,
		}

		result := buildServerURL(info)
		stripped := stripANSI(result)

		if !strings.Contains(stripped, "or next available") {
			t.Errorf("expected auto-port suffix in output, got %q", stripped)
		}
	})
}

func TestBuildServerLine(t *testing.T) {
	t.Run("NotExposed", func(t *testing.T) {
		t.Parallel()

		result := buildServerLine("http://localhost:8080", false)
		stripped := stripANSI(result)

		if !strings.Contains(stripped, "Server:") {
			t.Errorf("expected 'Server:' in output, got %q", stripped)
		}
	})

	t.Run("Exposed", func(t *testing.T) {
		t.Parallel()

		result := buildServerLine("http://localhost:8080", true)
		stripped := stripANSI(result)

		if !strings.Contains(stripped, "*") {
			t.Errorf("expected star marker for exposed server, got %q", stripped)
		}
	})
}

func TestAppendHealthLines(t *testing.T) {
	t.Run("NoHealthURL", func(t *testing.T) {
		t.Parallel()

		lines := []string{"line1"}
		info := StartupBannerInfo{HealthProbeURL: ""}

		result := appendHealthLines(lines, info)
		if len(result) != 1 {
			t.Errorf("expected 1 line, got %d", len(result))
		}
	})

	t.Run("WithHealth", func(t *testing.T) {
		t.Parallel()

		lines := []string{"line1"}
		info := StartupBannerInfo{
			HealthProbeURL: "http://127.0.0.1:9090",
			LivePath:       "/live",
			ReadyPath:      "/ready",
			HealthExposed:  false,
		}

		result := appendHealthLines(lines, info)
		if len(result) != 5 {
			t.Errorf("expected 5 lines (1 original + separator + 3 health), got %d", len(result))
		}
	})

	t.Run("WithHealthExposed", func(t *testing.T) {
		t.Parallel()

		lines := []string{"line1"}
		info := StartupBannerInfo{
			HealthProbeURL: "http://0.0.0.0:9090",
			LivePath:       "/live",
			ReadyPath:      "/ready",
			HealthExposed:  true,
		}

		result := appendHealthLines(lines, info)

		if len(result) != 5 {
			t.Errorf("expected 5 lines, got %d", len(result))
		}
	})
}

func TestAppendExposedFootnote(t *testing.T) {
	t.Run("NeitherExposed", func(t *testing.T) {
		t.Parallel()

		lines := []string{"line1"}
		info := StartupBannerInfo{
			ServerExposed: false,
			HealthExposed: false,
		}

		result := appendExposedFootnote(lines, info)
		if len(result) != 1 {
			t.Errorf("expected 1 line, got %d", len(result))
		}
	})

	t.Run("ServerExposed", func(t *testing.T) {
		t.Parallel()

		lines := []string{"line1"}
		info := StartupBannerInfo{
			ServerExposed: true,
			HealthExposed: false,
		}

		result := appendExposedFootnote(lines, info)
		if len(result) != 3 {
			t.Errorf("expected 3 lines (1 original + empty + footnote), got %d", len(result))
		}
	})

	t.Run("HealthExposed", func(t *testing.T) {
		t.Parallel()

		lines := []string{"line1"}
		info := StartupBannerInfo{
			ServerExposed: false,
			HealthExposed: true,
		}

		result := appendExposedFootnote(lines, info)
		if len(result) != 3 {
			t.Errorf("expected 3 lines (1 original + empty + footnote), got %d", len(result))
		}
	})
}

func TestCalculateMaxWidth(t *testing.T) {
	t.Run("EmptySlice", func(t *testing.T) {
		t.Parallel()

		if got := calculateMaxWidth(nil); got != 0 {
			t.Errorf("expected 0, got %d", got)
		}
	})

	t.Run("WithANSI", func(t *testing.T) {
		t.Parallel()

		lines := []string{
			"plain",
			"\x1b[31mlonger text\x1b[0m here",
		}

		got := calculateMaxWidth(lines)
		if got != 16 {
			t.Errorf("expected 16 (ignoring ANSI), got %d", got)
		}
	})

	t.Run("MultiplePlainLines", func(t *testing.T) {
		t.Parallel()

		lines := []string{"abc", "abcdef", "ab"}
		got := calculateMaxWidth(lines)
		if got != 6 {
			t.Errorf("expected 6, got %d", got)
		}
	})
}

func TestPrintBannerBox(t *testing.T) {
	t.Run("WritesToWriter", func(t *testing.T) {
		t.Parallel()

		var buffer bytes.Buffer
		lines := []string{"Hello", "World"}

		printBannerBox(&buffer, lines)

		output := buffer.String()
		if !strings.Contains(output, "Hello") {
			t.Error("expected 'Hello' in output")
		}
		if !strings.Contains(output, "World") {
			t.Error("expected 'World' in output")
		}
	})
}

func TestPrintStartupBanner(t *testing.T) {
	t.Run("Disabled", func(t *testing.T) {
		t.Parallel()

		info := StartupBannerInfo{
			Version:   "1.0.0",
			Mode:      "dev",
			ServerURL: "http://localhost:8080",
		}

		PrintStartupBanner(context.Background(), false, info)
	})

	t.Run("Enabled", func(t *testing.T) {
		t.Parallel()

		info := StartupBannerInfo{
			Version:       "1.0.0",
			Mode:          "dev",
			ServerURL:     "http://localhost:8080",
			ServerExposed: true,
		}

		PrintStartupBanner(context.Background(), true, info)
	})
}

func TestPrintFallbackLogs(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		info := StartupBannerInfo{
			Version:   "1.0.0",
			Mode:      "dev",
			ServerURL: "http://localhost:8080",
		}

		printFallbackLogs(context.Background(), info)
	})

	t.Run("WithHealth", func(t *testing.T) {
		t.Parallel()

		info := StartupBannerInfo{
			Version:        "1.0.0",
			Mode:           "prod",
			ServerURL:      "http://localhost:8080",
			HealthProbeURL: "http://127.0.0.1:9090",
			LivePath:       "/live",
			ReadyPath:      "/ready",
		}

		printFallbackLogs(context.Background(), info)
	})

	t.Run("WithAutoPort", func(t *testing.T) {
		t.Parallel()

		info := StartupBannerInfo{
			Version:   "1.0.0",
			Mode:      "dev",
			ServerURL: "http://localhost:8080",
			AutoPort:  true,
		}

		printFallbackLogs(context.Background(), info)
	})
}

func TestBuildBannerLines(t *testing.T) {
	t.Run("BasicStructure", func(t *testing.T) {
		t.Parallel()

		info := StartupBannerInfo{
			Version:       "1.0.0",
			Mode:          "dev",
			ServerURL:     "http://localhost:8080",
			ServerExposed: true,
		}

		lines := buildBannerLines(info)

		if len(lines) < 6 {
			t.Errorf("expected at least 6 lines, got %d", len(lines))
		}

		joined := stripANSI(strings.Join(lines, "\n"))
		if !strings.Contains(joined, "Piko Website Development Kit") {
			t.Error("expected 'Piko Website Development Kit' in banner")
		}
		if !strings.Contains(joined, "1.0.0") {
			t.Error("expected version in banner")
		}
		if !strings.Contains(joined, "Mode:") {
			t.Error("expected 'Mode:' in banner")
		}
		if !strings.Contains(joined, "Server:") {
			t.Error("expected 'Server:' in banner")
		}
	})

	t.Run("WithHealthProbe", func(t *testing.T) {
		t.Parallel()

		info := StartupBannerInfo{
			Version:        "1.0.0",
			Mode:           "prod",
			ServerURL:      "http://localhost:8080",
			HealthProbeURL: "http://127.0.0.1:9090",
			LivePath:       "/live",
			ReadyPath:      "/ready",
			ServerExposed:  true,
		}

		lines := buildBannerLines(info)

		if len(lines) < 9 {
			t.Errorf("expected at least 9 lines with health, got %d", len(lines))
		}
	})
}
