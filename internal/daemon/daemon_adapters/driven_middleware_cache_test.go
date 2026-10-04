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

package daemon_adapters

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/templater/templater_domain"
)

type persistedArtefact struct {
	cause      error
	carrier    *daemon_dto.PikoRequestCtx
	artefactID string
	body       string
}

func newTestCacheMiddleware(
	registry registry_domain.RegistryService,
	capabilities capabilities_domain.CapabilityService,
) *CacheMiddleware {
	return NewCacheMiddleware(CacheMiddlewareConfig{}, &templater_domain.MockManifestStoreView{}, registry, capabilities, "")
}

func newCarrierRequest(t *testing.T, acceptEncoding string) (*http.Request, *daemon_dto.PikoRequestCtx) {
	t.Helper()

	pctx := daemon_dto.AcquirePikoRequestCtx()
	pctx.ClientIP = "10.0.0.1"
	request := httptest.NewRequest(http.MethodGet, "/blog/post", nil)
	request.Header.Set("Accept-Encoding", acceptEncoding)

	return request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), pctx)), pctx
}

func TestGenerateAndCacheResponse_UpstreamFinishesBeforeReturning(t *testing.T) {
	compressionFailure := errors.New("compressor unavailable")

	testCases := []struct {
		capabilities   capabilities_domain.CapabilityService
		upstream       func(finished chan<- struct{}) http.HandlerFunc
		wantErr        error
		name           string
		acceptEncoding string
	}{
		{
			name:           "compression fails while the upstream is still writing",
			acceptEncoding: "gzip",
			capabilities: &capabilities_domain.MockCapabilityService{
				ExecuteFunc: func(context.Context, string, io.Reader, capabilities_domain.CapabilityParams) (io.Reader, error) {
					return nil, compressionFailure
				},
			},
			upstream: func(finished chan<- struct{}) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					defer close(finished)
					chunk := bytes.Repeat([]byte("x"), 4096)
					for range 64 {
						if _, err := w.Write(chunk); err != nil {
							return
						}
					}
				}
			},
			wantErr: compressionFailure,
		},
		{
			name:           "upstream panics",
			acceptEncoding: "",
			upstream: func(finished chan<- struct{}) http.HandlerFunc {
				return func(http.ResponseWriter, *http.Request) {
					defer close(finished)
					panic("render failed")
				}
			},
			wantErr: errHandlerNonSuccess,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			upstreamFinished := make(chan struct{})
			middleware := newTestCacheMiddleware(&registry_domain.MockRegistryService{}, tc.capabilities)
			request, pctx := newCarrierRequest(t, tc.acceptEncoding)
			defer daemon_dto.ReleasePikoRequestCtx(pctx)

			result, err := middleware.generateAndCacheResponse(request, tc.upstream(upstreamFinished), "artefact-1")

			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, result)
			select {
			case <-upstreamFinished:
			default:
				assert.Fail(t, "the upstream handler was still running after the response was abandoned")
			}
		})
	}
}

func TestGenerateAndCacheResponse_PersistsUnderADetachedContext(t *testing.T) {
	persisted := make(chan persistedArtefact, 1)
	requestFinished := make(chan struct{})
	registry := &registry_domain.MockRegistryService{
		UpsertArtefactFunc: func(
			ctx context.Context, artefactID, _ string, sourceData io.Reader, _ string, _ []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			<-requestFinished
			body, err := io.ReadAll(sourceData)
			if err != nil {
				return nil, err
			}
			persisted <- persistedArtefact{
				cause:      context.Cause(ctx),
				carrier:    daemon_dto.PikoRequestCtxFromContext(ctx),
				artefactID: artefactID,
				body:       string(body),
			}
			return &registry_dto.ArtefactMeta{}, nil
		},
	}
	middleware := newTestCacheMiddleware(registry, nil)

	request, pctx := newCarrierRequest(t, "")
	requestCtx, cancelRequest := context.WithCancelCause(request.Context())
	request = request.WithContext(requestCtx)
	upstream := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<p>rendered</p>"))
	})

	result, err := middleware.generateAndCacheResponse(request, upstream, "artefact-1")
	require.NoError(t, err)
	assert.Equal(t, "<p>rendered</p>", string(result.Content))

	cancelRequest(errors.New("client went away"))
	daemon_dto.ReleasePikoRequestCtx(pctx)
	next := daemon_dto.AcquirePikoRequestCtx()
	next.ClientIP = "192.168.9.9"
	defer daemon_dto.ReleasePikoRequestCtx(next)
	close(requestFinished)

	got := <-persisted
	assert.Equal(t, "artefact-1", got.artefactID)
	assert.Equal(t, "<p>rendered</p>", got.body)
	require.NoError(t, got.cause, "persistence must outlive the request")
	require.NotNil(t, got.carrier)
	assert.NotSame(t, pctx, got.carrier, "persistence must not read the pooled carrier")
	assert.Equal(t, "10.0.0.1", got.carrier.ClientIP)
}
