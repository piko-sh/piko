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
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"piko.sh/piko/internal/colour"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// bannerStar is the asterisk symbol used to mark exposed endpoints in the
	// banner.
	bannerStar = "*"

	// bannerSpace is a single space character used for padding in the banner.
	bannerSpace = " "

	// bannerColumnGap is the gap (in spaces) between the mascot and info columns.
	bannerColumnGap = 3
)

var (
	// ansiRegex matches ANSI escape sequences for stripping.
	ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

	colourCyan = colour.New(colour.FgCyan, colour.Bold).SprintFunc()

	colourGreen = colour.New(colour.FgGreen).SprintFunc()

	colourYellow = colour.New(colour.FgYellow).SprintFunc()

	colourDim = colour.New(colour.Faint).SprintFunc()

	colourDimItalic = colour.New(colour.Faint, colour.Italic).SprintFunc()

	// startupPhrases contains fun phrases shown randomly in the startup banner.
	startupPhrases = []string{
		"Ship it!",
		"Ready to build!",
		"Let's go!",
		"Making magic",
		"Here we go!",
		"Time to shine",
		"Building dreams",
		"Adventure awaits",
		"Let's make things",
		"Creating wonders",
		"Your move, dev",
		"Let's do this!",
		"Off we go!",
		"Ta-da!",
		"Ready, set, code!",
		"Woohoo!",
		"Hello, world!",
		"Let's roll!",
		"Game on!",
		"Hey there!",
		"Hi friend!",
		"Let's create!",
		"Make it so!",
		"Go time!",
		"Let's build!",
		"Happy coding!",
		"Onward!",
		"Away we go!",
		"Let's begin!",
		"Booting with love",
		"Fresh start!",
		"Rise and shine!",
		"Good vibes only",
		"Let's have fun!",
		"Ready when you are",
		"What shall we make?",
		"Ideas welcome!",
		"Dream big!",
		"Go forth!",
		"Onwards!",
		"Full speed ahead!",
		"Here for you",
		"Woo!",
		"Yay!",
		"Oh hello!",
		"Hiya!",
		"Let's get creative",
		"Build something cool",
		"Make something great",
		"The world awaits",
	}
)

// StartupBannerInfo holds the data needed to display the startup banner.
type StartupBannerInfo struct {
	// MonitoringURL is the URL of the gRPC monitoring service, e.g.,
	// "127.0.0.1:9091". Empty string indicates monitoring is disabled.
	MonitoringURL string

	// Mode is the run mode: "dev", "dev-i", or "prod".
	Mode string

	// ServerURL is the main server address, for example "http://localhost:8080".
	ServerURL string

	// HealthProbeURL is the URL of the health probe, e.g.,
	// "http://127.0.0.1:9090". Empty string indicates health probes are disabled.
	HealthProbeURL string

	// LivePath is the URL path for the liveness probe, such as "/live".
	LivePath string

	// ReadyPath is the URL path for the readiness probe, for example "/ready".
	ReadyPath string

	// ProfilingURL is the URL of the pprof server, e.g.,
	// "http://localhost:6060/debug/pprof/". Empty string indicates profiling
	// is disabled.
	ProfilingURL string

	// Version is the Piko version string, for example "0.1.0-alpha".
	Version string

	// ProfilingExposed indicates whether the profiling server is bound to
	// all interfaces (0.0.0.0).
	ProfilingExposed bool

	// AutoPort indicates whether the server may use a different port if
	// the configured one is busy.
	AutoPort bool

	// MonitoringExposed indicates whether the monitoring service is bound to
	// all interfaces (0.0.0.0).
	MonitoringExposed bool

	// HealthExposed indicates whether the health probe is bound to all
	// interfaces (0.0.0.0).
	HealthExposed bool

	// ServerExposed indicates whether the server is bound to all interfaces
	// (always true currently).
	ServerExposed bool

	// LargeMascot indicates whether the large pixel-art mascot should be
	// displayed in the banner instead of the small ASCII art.
	LargeMascot bool
}

// BuildStartupBannerInfo builds the startup banner details from the resolved
// daemon configuration and run mode.
//
// Takes config (DaemonConfig) which provides resolved network and health probe
// settings.
// Takes mode (string) which specifies the run mode (e.g. development).
// Takes version (string) which specifies the application version.
//
// Returns StartupBannerInfo which holds the formatted banner details.
func BuildStartupBannerInfo(config DaemonConfig, mode string, version string) StartupBannerInfo {
	info := StartupBannerInfo{
		Version:        version,
		Mode:           mode,
		ServerURL:      fmt.Sprintf("http://localhost:%s", config.NetworkPort),
		HealthProbeURL: "",
		LivePath:       "",
		ReadyPath:      "",
		AutoPort:       config.NetworkAutoNextPort,
		ServerExposed:  true,
		HealthExposed:  false,
		LargeMascot:    !config.IAmACatPerson,
	}

	if config.HealthEnabled {
		info.HealthProbeURL = fmt.Sprintf("http://%s:%s", config.HealthBindAddress, config.HealthPort)
		info.LivePath = config.HealthLivePath
		info.ReadyPath = config.HealthReadyPath
		info.HealthExposed = config.HealthBindAddress == "0.0.0.0"
	}

	return info
}

// PrintStartupBanner prints a formatted startup banner with server details.
// This bypasses the logging system to provide a clean, formatted output.
//
// When the banner is disabled, fallback NOTICE logs are printed instead.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes bannerEnabled (bool) which controls whether to print the banner or
// fallback logs.
// Takes info (StartupBannerInfo) which contains the server details to show.
func PrintStartupBanner(ctx context.Context, bannerEnabled bool, info StartupBannerInfo) {
	if !bannerEnabled {
		printFallbackLogs(ctx, info)
		return
	}

	lines := buildBannerLines(info)

	release := logger_domain.StderrWriter().HoldWrites()
	printBannerBox(os.Stderr, lines)
	release()
}

// buildBannerLines builds all content lines for the startup banner by placing
// the mascot art on the left and the info lines on the right.
//
// Takes info (StartupBannerInfo) which provides the server settings and
// version details to display.
//
// Returns []string which contains the formatted banner lines including the
// mascot, server URL, mode, and health details.
func buildBannerLines(info StartupBannerInfo) []string {
	var mascot []string
	if info.LargeMascot {
		mascot = mascotLargeLines()
	} else {
		mascot = mascotSmallLines()
	}
	return combineSideBySide(mascot, buildInfoLines(info), bannerColumnGap)
}

// buildInfoLines builds the right-side content lines (version, phrase, mode,
// server, health, footnote) independently of any mascot.
//
// Takes info (StartupBannerInfo) which provides server settings, version, and
// health probe details.
//
// Returns []string which contains the formatted info lines.
func buildInfoLines(info StartupBannerInfo) []string {
	serverURL := buildServerURL(info)
	phrase := startupPhrases[rand.IntN(len(startupPhrases))] //nolint:gosec // decorative, not security

	lines := []string{
		fmt.Sprintf("%s %s", colourCyan("Piko Website Development Kit"), colourDim(info.Version)),
		colourDimItalic(phrase),
		"",
		fmt.Sprintf("%s %s", colourDim("Mode:"), formatMode(info.Mode)),
		buildServerLine(serverURL, info.ServerExposed),
	}

	lines = appendHealthLines(lines, info)
	lines = appendMonitoringLines(lines, info)
	lines = appendProfilingLines(lines, info)
	lines = appendExposedFootnote(lines, info)

	return lines
}

// buildServerURL formats the server URL with an optional auto-port marker.
//
// Takes info (StartupBannerInfo) which provides the server URL and port
// settings.
//
// Returns string which is the formatted URL, with a dimmed suffix if the port
// is selected automatically.
func buildServerURL(info StartupBannerInfo) string {
	serverURL := colourGreen(info.ServerURL)
	if info.AutoPort {
		serverURL += bannerSpace + colourDim("(or next available)")
	}
	return serverURL
}

// buildServerLine creates the server info line with an optional exposed marker.
//
// Takes serverURL (string) which is the server address to display.
// Takes exposed (bool) which indicates whether to add the exposed marker.
//
// Returns string which is the formatted server line for the banner.
func buildServerLine(serverURL string, exposed bool) string {
	line := fmt.Sprintf("%s %s", colourDim("Server:"), serverURL)
	if exposed {
		line += bannerSpace + colourYellow(bannerStar)
	}
	return line
}

// appendHealthLines adds health probe details to the banner lines if enabled.
//
// Takes lines ([]string) which contains the existing banner lines.
// Takes info (StartupBannerInfo) which provides health probe settings.
//
// Returns []string which contains the original lines with health probe details
// added. Returns the original lines unchanged if health probes are not set.
func appendHealthLines(lines []string, info StartupBannerInfo) []string {
	if info.HealthProbeURL == "" {
		return lines
	}

	exposedMarker := ""
	if info.HealthExposed {
		exposedMarker = bannerSpace + colourYellow(bannerStar)
	}

	return append(lines,
		"",
		fmt.Sprintf("%s %s%s", colourDim("Health:"), colourYellow(info.HealthProbeURL), exposedMarker),
		fmt.Sprintf("  %s %s%s%s", colourDim("├──"), info.HealthProbeURL, info.LivePath, exposedMarker),
		fmt.Sprintf("  %s %s%s%s", colourDim("└──"), info.HealthProbeURL, info.ReadyPath, exposedMarker),
	)
}

// appendMonitoringLines adds monitoring gRPC service details to the banner
// lines if monitoring is enabled.
//
// Takes lines ([]string) which contains the existing banner lines.
// Takes info (StartupBannerInfo) which provides monitoring settings.
//
// Returns []string which contains the original lines with monitoring details
// added. Returns the original lines unchanged if monitoring is not enabled.
func appendMonitoringLines(lines []string, info StartupBannerInfo) []string {
	if info.MonitoringURL == "" {
		return lines
	}

	exposedMarker := ""
	if info.MonitoringExposed {
		exposedMarker = bannerSpace + colourYellow(bannerStar)
	}

	return append(lines,
		"",
		fmt.Sprintf("%s %s%s", colourDim("Monitoring:"), colourGreen("grpc://"+info.MonitoringURL), exposedMarker),
	)
}

// appendProfilingLines adds pprof server details to the banner lines if
// profiling is enabled.
//
// Takes lines ([]string) which contains the existing banner lines.
// Takes info (StartupBannerInfo) which provides profiling settings.
//
// Returns []string which contains the original lines with profiling details
// added. Returns the original lines unchanged if profiling is not enabled.
func appendProfilingLines(lines []string, info StartupBannerInfo) []string {
	if info.ProfilingURL == "" {
		return lines
	}

	exposedMarker := ""
	if info.ProfilingExposed {
		exposedMarker = bannerSpace + colourYellow(bannerStar)
	}

	return append(lines,
		"",
		fmt.Sprintf("%s %s%s", colourDim("Profiling:"), colourGreen(info.ProfilingURL), exposedMarker),
	)
}

// appendExposedFootnote adds the exposed ports footnote if needed.
//
// Takes lines ([]string) which is the current banner lines to add to.
// Takes info (StartupBannerInfo) which holds the exposure state flags.
//
// Returns []string which is the lines with the footnote added, or unchanged
// if neither the server nor health endpoints are exposed.
func appendExposedFootnote(lines []string, info StartupBannerInfo) []string {
	if !info.ServerExposed && !info.HealthExposed && !info.MonitoringExposed && !info.ProfilingExposed {
		return lines
	}
	return append(lines, "", colourYellow(bannerStar+" exposed to all network interfaces"))
}

// printBannerBox draws the given text lines inside a bordered box.
//
// Takes w (io.Writer) which receives the formatted output.
// Takes lines ([]string) which contains the text to display in the box.
func printBannerBox(w io.Writer, lines []string) {
	maxWidth := calculateMaxWidth(lines) + 2
	border := strings.Repeat("─", maxWidth)

	var buf strings.Builder
	_, _ = fmt.Fprintf(&buf, "\n  %s\n", colourDim("┌"+border+"┐"))
	for _, line := range lines {
		padding := maxWidth - utf8.RuneCountInString(stripANSI(line))
		_, _ = fmt.Fprintf(&buf, "  %s %s%s%s\n", colourDim("│"), line, strings.Repeat(bannerSpace, padding-1), colourDim("│"))
	}
	_, _ = fmt.Fprintf(&buf, "  %s\n\n", colourDim("└"+border+"┘"))
	_, _ = io.WriteString(w, buf.String())
}

// calculateMaxWidth finds the longest display width among the given lines.
//
// Takes lines ([]string) which contains the banner lines to measure.
//
// Returns int which is the width of the longest line in runes, excluding
// ANSI escape codes.
func calculateMaxWidth(lines []string) int {
	maxWidth := 0
	for _, line := range lines {
		if width := utf8.RuneCountInString(stripANSI(line)); width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth
}

// formatMode converts a mode string to a readable display name.
//
// Takes mode (string) which is the mode name to convert.
//
// Returns string which is the display name for the mode.
func formatMode(mode string) string {
	switch mode {
	case "dev":
		return "Development"
	case "dev-i":
		return "Development (Interpreted)"
	case "prod":
		return "Production"
	default:
		return mode
	}
}

// stripANSI removes ANSI escape codes from a string for accurate width
// calculation.
//
// Takes s (string) which is the input that may contain ANSI escape codes.
//
// Returns string which is the input with all ANSI escape codes removed.
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// printFallbackLogs writes NOTICE-level logs when the startup banner is off.
// This means key server details are still shown in the logs.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes info (StartupBannerInfo) which holds the server details to log.
func printFallbackLogs(ctx context.Context, info StartupBannerInfo) {
	_, l := logger_domain.From(ctx, log)
	l = l.With(logger_domain.String(logger_domain.FieldStrContext, "startup"))

	serverURL := info.ServerURL
	if info.AutoPort {
		serverURL += " (or next available)"
	}

	l.Notice("Piko server starting",
		logger_domain.String("version", info.Version),
		logger_domain.String("mode", formatMode(info.Mode)),
		logger_domain.String("url", serverURL),
	)

	if info.HealthProbeURL != "" {
		l.Notice("Health probe enabled",
			logger_domain.String("url", info.HealthProbeURL),
			logger_domain.String("live_path", info.LivePath),
			logger_domain.String("ready_path", info.ReadyPath),
		)
	}

	if info.MonitoringURL != "" {
		l.Notice("Monitoring gRPC service enabled",
			logger_domain.String("address", "grpc://"+info.MonitoringURL),
		)
	}

	if info.ProfilingURL != "" {
		l.Notice("Profiling pprof server enabled",
			logger_domain.String("url", info.ProfilingURL),
		)
	}
}
