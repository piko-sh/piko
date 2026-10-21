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

package lifecycle_domain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_domain"
)

type recordingReconciler struct {
	checked    [][]string
	dispatched int
	err        error
}

func (r *recordingReconciler) ReconcileArtefacts(_ context.Context, artefactIDs []string) (int, error) {
	r.checked = append(r.checked, artefactIDs)
	return r.dispatched, r.err
}

func (*recordingReconciler) UnbuiltProfiles(context.Context, []string) (map[string][]string, error) {
	return nil, nil
}

func listingRegistry(ids []string, err error) *registry_domain.MockRegistryService {
	return &registry_domain.MockRegistryService{
		ListAllArtefactIDsFunc: func(context.Context) ([]string, error) { return ids, err },
	}
}

func TestReconcileUnbuiltProfiles_ChecksEveryArtefact(t *testing.T) {
	t.Parallel()

	reconciler := &recordingReconciler{dispatched: 2}
	dispatched, err := ReconcileUnbuiltProfiles(t.Context(), listingRegistry([]string{"a.css", "b.svg"}, nil), reconciler)

	require.NoError(t, err)
	assert.Equal(t, 2, dispatched)
	assert.Equal(t, [][]string{{"a.css", "b.svg"}}, reconciler.checked)
}

func TestReconcileUnbuiltProfiles_EmptyRegistryChecksNothing(t *testing.T) {
	t.Parallel()

	reconciler := &recordingReconciler{}
	dispatched, err := ReconcileUnbuiltProfiles(t.Context(), listingRegistry(nil, nil), reconciler)

	require.NoError(t, err)
	assert.Zero(t, dispatched)
	assert.Empty(t, reconciler.checked)
}

func TestReconcileUnbuiltProfiles_ReportsAListingFailure(t *testing.T) {
	t.Parallel()

	reconciler := &recordingReconciler{}
	_, err := ReconcileUnbuiltProfiles(t.Context(), listingRegistry(nil, errors.New("registry unavailable")), reconciler)

	require.Error(t, err)
	assert.Empty(t, reconciler.checked)
}

func TestStartupReconcile_RunsOnlyWhenTheRegistryCanBeWritten(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		readOnly    bool
		reconciler  bool
		wantChecked bool
	}{
		{name: "writable registry with a reconciler", reconciler: true, wantChecked: true},
		{name: "read-only embedded base", readOnly: true, reconciler: true, wantChecked: false},
		{name: "no reconciler wired", reconciler: false, wantChecked: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reconciler := &recordingReconciler{}
			ls := &lifecycleService{
				registryService:       listingRegistry([]string{"theme.css"}, nil),
				registryBlobsReadOnly: tc.readOnly,
			}
			if tc.reconciler {
				ls.artefactReconciler = reconciler
			}

			ls.reconcileUnbuiltProfiles(t.Context())

			assert.Equal(t, tc.wantChecked, len(reconciler.checked) == 1)
		})
	}
}

func TestStartupReconcile_FailureIsNotFatal(t *testing.T) {
	t.Parallel()

	reconciler := &recordingReconciler{err: errors.New("dispatcher stopped")}
	ls := &lifecycleService{
		registryService:    listingRegistry([]string{"theme.css"}, nil),
		artefactReconciler: reconciler,
	}

	assert.NotPanics(t, func() { ls.reconcileUnbuiltProfiles(t.Context()) },
		"a failed re-check is logged, and the initial tasks go on")
	assert.Len(t, reconciler.checked, 1)
}
