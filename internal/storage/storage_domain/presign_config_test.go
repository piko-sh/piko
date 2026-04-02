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

package storage_domain

import (
	"context"
	"testing"
	"time"
)

func TestDefaultPresignConfig(t *testing.T) {
	config := DefaultPresignConfig()

	if config.Secret != nil {
		t.Error("default secret should be nil")
	}
	if config.DefaultExpiry != DefaultPresignExpiry {
		t.Errorf("expected DefaultExpiry %v, got %v", DefaultPresignExpiry, config.DefaultExpiry)
	}
	if config.MaxExpiry != MaxPresignExpiry {
		t.Errorf("expected MaxExpiry %v, got %v", MaxPresignExpiry, config.MaxExpiry)
	}
	if config.DefaultMaxSize != DefaultPresignMaxSize {
		t.Errorf("expected DefaultMaxSize %d, got %d", DefaultPresignMaxSize, config.DefaultMaxSize)
	}
	if config.MaxMaxSize != MaxPresignMaxSize {
		t.Errorf("expected MaxMaxSize %d, got %d", MaxPresignMaxSize, config.MaxMaxSize)
	}
	if config.RateLimitPerMinute != DefaultPresignRateLimit {
		t.Errorf("expected RateLimitPerMinute %d, got %d", DefaultPresignRateLimit, config.RateLimitPerMinute)
	}
}

func TestPresignConfig_Validate(t *testing.T) {
	tests := []struct {
		checkFunc func(t *testing.T, config *PresignConfig)
		name      string
		config    PresignConfig
		wantErr   bool
	}{
		{
			name:    "valid config passes",
			config:  DefaultPresignConfig(),
			wantErr: false,
		},
		{
			name: "short secret returns error",
			config: PresignConfig{
				Secret: []byte("short"),
			},
			wantErr: true,
		},
		{
			name: "valid secret passes",
			config: PresignConfig{
				Secret: []byte("test-secret-key-32-bytes-long!!!"),
			},
			wantErr: false,
		},
		{
			name: "zero DefaultExpiry gets default",
			config: PresignConfig{
				DefaultExpiry: 0,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, config *PresignConfig) {
				if config.DefaultExpiry != DefaultPresignExpiry {
					t.Errorf("expected DefaultExpiry %v, got %v", DefaultPresignExpiry, config.DefaultExpiry)
				}
			},
		},
		{
			name: "negative DefaultExpiry gets default",
			config: PresignConfig{
				DefaultExpiry: -5 * time.Minute,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, config *PresignConfig) {
				if config.DefaultExpiry != DefaultPresignExpiry {
					t.Errorf("expected DefaultExpiry %v, got %v", DefaultPresignExpiry, config.DefaultExpiry)
				}
			},
		},
		{
			name: "zero MaxExpiry gets default",
			config: PresignConfig{
				MaxExpiry: 0,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, config *PresignConfig) {
				if config.MaxExpiry != MaxPresignExpiry {
					t.Errorf("expected MaxExpiry %v, got %v", MaxPresignExpiry, config.MaxExpiry)
				}
			},
		},
		{
			name: "DefaultExpiry greater than MaxExpiry is capped",
			config: PresignConfig{
				DefaultExpiry: 2 * time.Hour,
				MaxExpiry:     30 * time.Minute,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, config *PresignConfig) {
				if config.DefaultExpiry != config.MaxExpiry {
					t.Errorf("expected DefaultExpiry to be capped to MaxExpiry %v, got %v", config.MaxExpiry, config.DefaultExpiry)
				}
			},
		},
		{
			name: "zero DefaultMaxSize gets default",
			config: PresignConfig{
				DefaultMaxSize: 0,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, config *PresignConfig) {
				if config.DefaultMaxSize != DefaultPresignMaxSize {
					t.Errorf("expected DefaultMaxSize %d, got %d", DefaultPresignMaxSize, config.DefaultMaxSize)
				}
			},
		},
		{
			name: "zero MaxMaxSize gets default",
			config: PresignConfig{
				MaxMaxSize: 0,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, config *PresignConfig) {
				if config.MaxMaxSize != MaxPresignMaxSize {
					t.Errorf("expected MaxMaxSize %d, got %d", MaxPresignMaxSize, config.MaxMaxSize)
				}
			},
		},
		{
			name: "DefaultMaxSize greater than MaxMaxSize is capped",
			config: PresignConfig{
				DefaultMaxSize: 2 * 1024 * 1024 * 1024,
				MaxMaxSize:     500 * 1024 * 1024,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, config *PresignConfig) {
				if config.DefaultMaxSize != config.MaxMaxSize {
					t.Errorf("expected DefaultMaxSize to be capped to MaxMaxSize %d, got %d", config.MaxMaxSize, config.DefaultMaxSize)
				}
			},
		},
		{
			name: "negative RateLimitPerMinute is set to zero",
			config: PresignConfig{
				RateLimitPerMinute: -10,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, config *PresignConfig) {
				if config.RateLimitPerMinute != 0 {
					t.Errorf("expected RateLimitPerMinute 0, got %d", config.RateLimitPerMinute)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := tc.config
			err := config.Validate()

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tc.checkFunc != nil {
				tc.checkFunc(t, &config)
			}
		})
	}
}

func TestPresignConfig_EnsureSecret(t *testing.T) {
	t.Run("generates secret when nil", func(t *testing.T) {
		config := PresignConfig{
			Secret: nil,
		}

		err := config.EnsureSecret()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(config.Secret) < presignSecretMinLength {
			t.Errorf("expected secret length >= %d, got %d", presignSecretMinLength, len(config.Secret))
		}
	})

	t.Run("generates secret when too short", func(t *testing.T) {
		config := PresignConfig{
			Secret: []byte("short"),
		}

		err := config.EnsureSecret()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(config.Secret) < presignSecretMinLength {
			t.Errorf("expected secret length >= %d, got %d", presignSecretMinLength, len(config.Secret))
		}
	})

	t.Run("preserves valid secret", func(t *testing.T) {
		originalSecret := []byte("test-secret-key-32-bytes-long!!!")
		config := PresignConfig{
			Secret: originalSecret,
		}

		err := config.EnsureSecret()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if string(config.Secret) != string(originalSecret) {
			t.Error("valid secret should be preserved")
		}
	})

	t.Run("generates unique secrets", func(t *testing.T) {
		secrets := make(map[string]bool)
		iterations := 10

		for range iterations {
			config := PresignConfig{}
			_ = config.EnsureSecret()

			secretString := string(config.Secret)
			if secrets[secretString] {
				t.Error("duplicate secret generated")
			}
			secrets[secretString] = true
		}
	})
}

func TestPresignConfig_ClampExpiry(t *testing.T) {
	config := PresignConfig{
		DefaultExpiry: 15 * time.Minute,
		MaxExpiry:     1 * time.Hour,
	}

	tests := []struct {
		name   string
		expiry time.Duration
		want   time.Duration
	}{
		{
			name:   "zero returns default",
			expiry: 0,
			want:   config.DefaultExpiry,
		},
		{
			name:   "negative returns default",
			expiry: -5 * time.Minute,
			want:   config.DefaultExpiry,
		},
		{
			name:   "within bounds returns unchanged",
			expiry: 30 * time.Minute,
			want:   30 * time.Minute,
		},
		{
			name:   "exceeds max returns max",
			expiry: 2 * time.Hour,
			want:   config.MaxExpiry,
		},
		{
			name:   "equals max returns max",
			expiry: 1 * time.Hour,
			want:   1 * time.Hour,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := config.ClampExpiry(tc.expiry)
			if got != tc.want {
				t.Errorf("ClampExpiry(%v) = %v, want %v", tc.expiry, got, tc.want)
			}
		})
	}
}

func TestPresignConfig_ClampMaxSize(t *testing.T) {
	config := PresignConfig{
		DefaultMaxSize: 100 * 1024 * 1024,
		MaxMaxSize:     1024 * 1024 * 1024,
	}

	tests := []struct {
		name    string
		maxSize int64
		want    int64
	}{
		{
			name:    "zero returns default",
			maxSize: 0,
			want:    config.DefaultMaxSize,
		},
		{
			name:    "negative returns default",
			maxSize: -1024,
			want:    config.DefaultMaxSize,
		},
		{
			name:    "within bounds returns unchanged",
			maxSize: 500 * 1024 * 1024,
			want:    500 * 1024 * 1024,
		},
		{
			name:    "exceeds max returns max",
			maxSize: 2 * 1024 * 1024 * 1024,
			want:    config.MaxMaxSize,
		},
		{
			name:    "equals max returns max",
			maxSize: 1024 * 1024 * 1024,
			want:    1024 * 1024 * 1024,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := config.ClampMaxSize(tc.maxSize)
			if got != tc.want {
				t.Errorf("ClampMaxSize(%d) = %d, want %d", tc.maxSize, got, tc.want)
			}
		})
	}
}

func TestPresignConfig_EnsureRIDCache(t *testing.T) {
	t.Run("creates cache when nil", func(t *testing.T) {
		config := PresignConfig{}

		if config.RIDCache != nil {
			t.Error("RIDCache should be nil initially")
		}

		config.EnsureRIDCache(context.Background(), 1*time.Minute)

		if config.RIDCache == nil {
			t.Error("RIDCache should not be nil after EnsureRIDCache")
		}

		config.RIDCache.Stop()
	})

	t.Run("preserves existing cache", func(t *testing.T) {
		existingCache := NewPresignRIDCache(context.Background(), 1*time.Minute)
		defer existingCache.Stop()

		config := PresignConfig{
			RIDCache: existingCache,
		}

		config.EnsureRIDCache(context.Background(), 1*time.Minute)

		if config.RIDCache != existingCache {
			t.Error("existing cache should be preserved")
		}
	})
}
