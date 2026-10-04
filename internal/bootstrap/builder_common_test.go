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
package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/fonts"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/templater/templater_domain"
)

type initialTasksLifecycle struct {
	lifecycle_domain.LifecycleService
	err  error
	done chan struct{}
}

func (l *initialTasksLifecycle) RunInitialTasks(context.Context) error {
	defer close(l.done)
	return l.err
}

func TestNewDefaultPdfFontEntries(t *testing.T) {
	entries := newDefaultPdfFontEntries()

	require.Len(t, entries, 2)
	testCases := []struct {
		name       string
		data       []byte
		index      int
		wantWeight int
	}{
		{name: "regular", index: 0, wantWeight: fontWeightNormal, data: fonts.NotoSansRegularTTF},
		{name: "bold", index: 1, wantWeight: fontWeightBold, data: fonts.NotoSansBoldTTF},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := entries[tc.index]
			assert.Equal(t, fonts.NotoSansFamilyName, entry.Family)
			assert.Equal(t, tc.wantWeight, entry.Weight)
			assert.Equal(t, int(layouter_domain.FontStyleNormal), entry.Style)
			assert.Equal(t, tc.data, entry.Data)
			assert.NotEmpty(t, entry.Data)
			assert.False(t, entry.IsVariable)
		})
	}
}

func TestSetupPdfWriterService_InstallsTheService(t *testing.T) {
	c := NewContainer()
	c.renderRegistryOverride = &stubRenderRegistryPort{}
	require.Nil(t, c.GetPdfWriterService())

	require.NoError(t, setupPdfWriterService(c, &templater_domain.MockManifestRunnerPort{}, "test mode"))

	assert.NotNil(t, c.GetPdfWriterService())
}

func TestRunInitialTasksInBackground(t *testing.T) {
	testCases := []struct {
		err    error
		name   string
		cancel bool
	}{
		{name: "tasks succeed"},
		{name: "tasks fail while the application runs", err: errors.New("seeding failed")},
		{name: "tasks fail after shutdown began", err: context.Canceled, cancel: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(errors.New("test finished"))
			if tc.cancel {
				cancel(errors.New("application shutting down"))
			}
			lifecycle := &initialTasksLifecycle{err: tc.err, done: make(chan struct{})}

			runInitialTasksInBackground(ctx, lifecycle)

			<-lifecycle.done
		})
	}
}
