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

	"github.com/stretchr/testify/assert"
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

		assert.Equal(t, "1.0.0", info.Version, "expected version '1.0.0'")
		assert.Equal(t, "dev", info.Mode, "expected mode 'dev'")
		assert.Equal(t, "http://localhost:8080", info.ServerURL, "expected 'http://localhost:8080'")
		assert.Empty(t, info.HealthProbeURL, "expected empty health probe URL")
		assert.False(t, info.AutoPort, "expected AutoPort=false")
		assert.True(t, info.ServerExposed, "expected ServerExposed=true")
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

		assert.Empty(t, info.HealthProbeURL,
			"the URL is filled from the listener that bound, not from config, so a health probe "+
				"that never bound advertises nothing")
		assert.Equal(t, "/live", info.LivePath, "expected '/live'")
		assert.Equal(t, "/ready", info.ReadyPath, "expected '/ready'")
		assert.False(t, info.HealthExposed, "expected HealthExposed=false for 127.0.0.1")
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

		assert.Empty(t, info.HealthProbeURL, "expected empty health probe URL when disabled")
	})

	t.Run("AutoPort", func(t *testing.T) {
		t.Parallel()

		config := DaemonConfig{
			NetworkPort:         "8080",
			NetworkAutoNextPort: true,
			HealthEnabled:       false,
		}

		info := BuildStartupBannerInfo(config, "dev", "1.0.0")

		assert.True(t, info.AutoPort, "expected AutoPort=true")
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

		assert.True(t, info.HealthExposed, "expected HealthExposed=true for 0.0.0.0")
	})
}

func TestFormatMode(t *testing.T) {
	t.Run("Dev", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "Development", formatMode("dev"), "expected 'Development'")
	})

	t.Run("DevInterpreted", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "Development (Interpreted)", formatMode("dev-i"),
			"expected 'Development (Interpreted)'")
	})

	t.Run("Prod", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "Production", formatMode("prod"), "expected 'Production'")
	})

	t.Run("Unknown", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "custom", formatMode("custom"), "expected 'custom' passthrough")
	})
}

func TestStripANSI(t *testing.T) {
	t.Run("PlainText", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "hello world", stripANSI("hello world"), "expected 'hello world'")
	})

	t.Run("WithColourCodes", func(t *testing.T) {
		t.Parallel()
		input := "\x1b[31mred\x1b[0m text"
		got := stripANSI(input)
		assert.Equal(t, "red text", got, "expected 'red text'")
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

		assert.Contains(t, stripped, "http://localhost:8080", "expected URL in output")
		assert.NotContains(t, stripped, "or next available", "did not expect auto-port suffix")
	})

	t.Run("WithAutoPort", func(t *testing.T) {
		t.Parallel()

		info := StartupBannerInfo{
			ServerURL: "http://localhost:8080",
			AutoPort:  true,
		}

		result := buildServerURL(info)
		stripped := stripANSI(result)

		assert.Contains(t, stripped, "or next available", "expected auto-port suffix in output")
	})
}

func TestBuildServerLine(t *testing.T) {
	t.Run("NotExposed", func(t *testing.T) {
		t.Parallel()

		result := buildServerLine("http://localhost:8080", false)
		stripped := stripANSI(result)

		assert.Contains(t, stripped, "Server:", "expected 'Server:' in output")
	})

	t.Run("Exposed", func(t *testing.T) {
		t.Parallel()

		result := buildServerLine("http://localhost:8080", true)
		stripped := stripANSI(result)

		assert.Contains(t, stripped, "*", "expected star marker for exposed server")
	})
}

func TestAppendHealthLines(t *testing.T) {
	t.Run("NoHealthURL", func(t *testing.T) {
		t.Parallel()

		lines := []string{"line1"}
		info := StartupBannerInfo{HealthProbeURL: ""}

		result := appendHealthLines(lines, info)
		assert.Len(t, result, 1, "expected 1 line")
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
		assert.Len(t, result, 5, "expected 5 lines (1 original + separator + 3 health)")
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

		assert.Len(t, result, 5, "expected 5 lines")
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
		assert.Len(t, result, 1, "expected 1 line")
	})

	t.Run("ServerExposed", func(t *testing.T) {
		t.Parallel()

		lines := []string{"line1"}
		info := StartupBannerInfo{
			ServerExposed: true,
			HealthExposed: false,
		}

		result := appendExposedFootnote(lines, info)
		assert.Len(t, result, 3, "expected 3 lines (1 original + empty + footnote)")
	})

	t.Run("HealthExposed", func(t *testing.T) {
		t.Parallel()

		lines := []string{"line1"}
		info := StartupBannerInfo{
			ServerExposed: false,
			HealthExposed: true,
		}

		result := appendExposedFootnote(lines, info)
		assert.Len(t, result, 3, "expected 3 lines (1 original + empty + footnote)")
	})
}

func TestCalculateMaxWidth(t *testing.T) {
	t.Run("EmptySlice", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 0, calculateMaxWidth(nil), "expected 0")
	})

	t.Run("WithANSI", func(t *testing.T) {
		t.Parallel()

		lines := []string{
			"plain",
			"\x1b[31mlonger text\x1b[0m here",
		}

		got := calculateMaxWidth(lines)
		assert.Equal(t, 16, got, "expected 16 (ignoring ANSI)")
	})

	t.Run("MultiplePlainLines", func(t *testing.T) {
		t.Parallel()

		lines := []string{"abc", "abcdef", "ab"}
		got := calculateMaxWidth(lines)
		assert.Equal(t, 6, got, "expected 6")
	})
}

func TestPrintBannerBox(t *testing.T) {
	t.Run("WritesToWriter", func(t *testing.T) {
		t.Parallel()

		var buffer bytes.Buffer
		lines := []string{"Hello", "World"}

		printBannerBox(&buffer, lines)

		output := buffer.String()
		assert.Contains(t, output, "Hello", "expected 'Hello' in output")
		assert.Contains(t, output, "World", "expected 'World' in output")
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

		assert.GreaterOrEqual(t, len(lines), 6, "expected at least 6 lines")

		joined := stripANSI(strings.Join(lines, "\n"))
		assert.Contains(t, joined, "Piko Website Development Kit",
			"expected 'Piko Website Development Kit' in banner")
		assert.Contains(t, joined, "1.0.0", "expected version in banner")
		assert.Contains(t, joined, "Mode:", "expected 'Mode:' in banner")
		assert.Contains(t, joined, "Server:", "expected 'Server:' in banner")
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

		assert.GreaterOrEqual(t, len(lines), 9, "expected at least 9 lines with health")
	})
}

func TestIsExposedBindAddress(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		address  string
		expected bool
	}{
		{name: "Empty", address: "", expected: true},
		{name: "Whitespace", address: "  ", expected: true},
		{name: "IPv4Wildcard", address: "0.0.0.0", expected: true},
		{name: "IPv6Wildcard", address: "::", expected: true},
		{name: "IPv6WildcardBracketed", address: "[::]", expected: true},
		{name: "Loopback", address: "127.0.0.1", expected: false},
		{name: "IPv6Loopback", address: "::1", expected: false},
		{name: "Hostname", address: "internal.example.com", expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, IsExposedBindAddress(testCase.address))
		})
	}
}

func TestIsExposedHostPort(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		address  string
		expected bool
	}{
		{name: "IPv4Wildcard", address: "0.0.0.0:9090", expected: true},
		{name: "IPv6Wildcard", address: "[::]:9090", expected: true},
		{name: "PortOnly", address: ":9090", expected: true},
		{name: "Loopback", address: "127.0.0.1:9090", expected: false},
		{name: "IPv6Loopback", address: "[::1]:9090", expected: false},
		{name: "BareHostNoPort", address: "0.0.0.0", expected: true},
		{name: "BareLoopbackNoPort", address: "127.0.0.1", expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, IsExposedHostPort(testCase.address))
		})
	}
}

func TestBannerScheme(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "https", BannerScheme(true))
	assert.Equal(t, "http", BannerScheme(false))
}

func TestIsExposedBindAddressCoversEverySpelling(t *testing.T) {
	t.Parallel()

	exposed := []string{"", "0.0.0.0", "::", "[::]", "::0", "[::0]", "0:0:0:0:0:0:0:0", "::ffff:0.0.0.0"}
	for _, address := range exposed {
		assert.True(t, IsExposedBindAddress(address), "address %q publishes on every interface", address)
	}

	private := []string{"127.0.0.1", "::1", "[::1]", "10.0.0.5", "192.168.1.1", "internal.example.com"}
	for _, address := range private {
		assert.False(t, IsExposedBindAddress(address), "address %q does not publish everywhere", address)
	}
}
