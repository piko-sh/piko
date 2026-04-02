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

package config

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/config/config_domain"
)

// fileSystem provides file reading and metadata operations.
// It allows mock versions to be used during testing.
type fileSystem interface {
	// ReadFile reads the contents of the named file.
	//
	// Takes name (string) which is the path to the file to read.
	//
	// Returns []byte which contains the file contents.
	// Returns error when the file cannot be read.
	ReadFile(name string) ([]byte, error)

	// Stat returns file information for the named file.
	//
	// Takes name (string) which is the path to the file.
	//
	// Returns os.FileInfo which contains the file metadata.
	// Returns error when the file does not exist or cannot be accessed.
	Stat(name string) (os.FileInfo, error)
}

// defaultFileSystem implements the fileSystem interface using the standard os
// package.
type defaultFileSystem struct{}

// ReadFile reads the file at the given path and returns its contents.
// Paths come from environment variables and working directory, not from
// untrusted user input.
//
// Takes name (string) which specifies the file path to read.
//
// Returns []byte which contains the file contents.
// Returns error when the file cannot be read.
func (defaultFileSystem) ReadFile(name string) ([]byte, error) {
	//nolint:gosec // trusted config path
	return os.ReadFile(name)
}

// Stat returns file information for the named file.
//
// Takes name (string) which specifies the path to the file.
//
// Returns os.FileInfo which contains the file's metadata.
// Returns error when the file does not exist or cannot be read.
func (defaultFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

// Provider holds the application configuration and is the main access
// point for configuration values throughout the system.
type Provider struct {
	// fs provides file system access for reading config files and checking paths.
	fs fileSystem

	// WebsiteConfig holds the parsed website settings used for theme building.
	WebsiteConfig WebsiteConfig

	// ServerConfig holds the loaded server settings.
	ServerConfig ServerConfig
}

// NewConfigProvider creates a new configuration provider with default
// dependencies.
//
// Returns *Provider which is ready to load and manage configuration.
func NewConfigProvider() *Provider {
	return &Provider{
		fs: defaultFileSystem{},
	}
}

// LoadConfig orchestrates the loading of the server-specific configuration.
// It discovers and merges environment-specific configuration files.
//
// Takes defaults (*ServerConfig) which provides the base configuration values
// (lowest precedence).
// Takes overrides (*ServerConfig) which provides programmatic overrides that
// always win over file/env/flag values (highest precedence). May be nil.
// Takes resolvers (...config_domain.Resolver) which handle value resolution.
//
// Returns *config_domain.LoadContext which contains the loaded configuration
// context.
// Returns error when file discovery, loading, or validation fails.
func (p *Provider) LoadConfig(defaults *ServerConfig, overrides *ServerConfig, resolvers ...config_domain.Resolver) (*config_domain.LoadContext, error) {
	env := determineEnvironment()

	filePaths, err := p.buildFileLoadList(env)
	if err != nil {
		return nil, fmt.Errorf("failed to determine config files: %w", err)
	}

	opts := config_domain.LoaderOptions{
		ProgrammaticDefaults:  defaults,
		ProgrammaticOverrides: overrides,
		FilePaths:             filePaths,
		StrictFile:            false,
		Resolvers:             resolvers,
		FlagPrefix:            "",
		UseGlobalResolvers:    true,
	}

	ctx, err := config_domain.Load(context.Background(), &p.ServerConfig, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load server configuration: %w", err)
	}

	if err = p.validateServerPaths(); err != nil {
		return nil, fmt.Errorf("validating server paths: %w", err)
	}

	return ctx, nil
}

// LoadWebsiteConfig loads website settings from config.json.
// This file is simpler and is not merged from other sources.
//
// Returns error when the file cannot be read or contains invalid JSON.
// A missing config.json is not treated as an error.
func (p *Provider) LoadWebsiteConfig() error {
	var baseDir string
	if p.ServerConfig.Paths.BaseDir != nil {
		baseDir = *p.ServerConfig.Paths.BaseDir
	}
	path := filepath.Join(baseDir, "config.json")
	data, err := p.fs.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to read site config at %s: %w", path, err)
	}

	var parsed WebsiteConfig
	if jErr := json.Unmarshal(data, &parsed); jErr != nil {
		return fmt.Errorf("failed to unmarshal site config JSON at %s: %w", path, jErr)
	}

	p.WebsiteConfig = parsed
	return nil
}

// buildFileLoadList builds the ordered list of config files to load.
//
// Takes env (string) which specifies the environment name for variant lookup.
//
// Returns []string which contains paths to existing config files in order of
// precedence (base, environment-specific, local).
// Returns error when a config file exists but cannot be accessed.
func (p *Provider) buildFileLoadList(env string) ([]string, error) {
	baseConfigFile := findFlagValue("configFile", "piko.yaml")

	directory := filepath.Dir(baseConfigFile)
	ext := filepath.Ext(baseConfigFile)
	base := strings.TrimSuffix(filepath.Base(baseConfigFile), ext)

	pathsToTry := []string{
		baseConfigFile,
		filepath.Join(directory, fmt.Sprintf("%s-%s%s", base, env, ext)),
		filepath.Join(directory, fmt.Sprintf("%s.local%s", base, ext)),
	}

	var existingPaths []string
	for _, path := range pathsToTry {
		if _, err := p.fs.Stat(path); err == nil {
			existingPaths = append(existingPaths, path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("cannot access config file %s: %w", path, err)
		}
	}

	return existingPaths, nil
}

// validateServerPaths runs custom validation logic for filesystem paths that
// cannot be expressed in simple struct tags (e.g., checking if a path is a
// directory).
//
// Returns error when the base directory does not exist, is not accessible,
// or is not a directory.
func (p *Provider) validateServerPaths() error {
	var baseDir string
	if p.ServerConfig.Paths.BaseDir != nil {
		baseDir = *p.ServerConfig.Paths.BaseDir
	}
	info, err := p.fs.Stat(baseDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("website base directory not found: %s", baseDir)
		}
		return fmt.Errorf("cannot access website base directory %s: %w", baseDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("baseDir is not a directory: %s", baseDir)
	}

	return nil
}

// newTestConfigProvider creates a configuration provider with mock
// dependencies for testing.
//
// Takes fs (fileSystem) which provides file system operations.
//
// Returns *Provider which is the configured provider ready for use.
func newTestConfigProvider(fs fileSystem) *Provider {
	return &Provider{
		fs: fs,
	}
}

// determineEnvironment finds the current environment name.
//
// It checks the --env flag first, then the PIKO_ENV environment variable.
// If neither is set, it returns "dev" as the default.
//
// Returns string which is the environment name.
func determineEnvironment() string {
	return cmp.Or(findFlagValue("env", ""), os.Getenv("PIKO_ENV"), "dev")
}

// findFlagValue scans os.Args for a given flag and returns its value.
// This is needed to get settings before the loader is set up, since a normal
// parse would fail on flags that are not yet defined.
//
// Takes flagName (string) which specifies the flag to search for.
// Takes defaultValue (string) which is returned if the flag is not found.
//
// Returns string which is the flag value, or the default if not found.
func findFlagValue(flagName, defaultValue string) string {
	for i, argument := range os.Args {
		if strings.HasPrefix(argument, "--"+flagName+"=") || strings.HasPrefix(argument, "-"+flagName+"=") {
			parts := strings.SplitN(argument, "=", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
		if (argument == "--"+flagName || argument == "-"+flagName) && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return defaultValue
}
