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

package llm_provider_ollama

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
	"piko.sh/piko/wdk/logger"
)

// logKeyModel is the logger attribute key for the Ollama model name.
const logKeyModel = "model"

// resolveModel returns the model name and reference to verify against.
// If reqModel is non-empty it takes priority as a per-request override;
// otherwise the default reference is used.
//
// Takes reqModel (string) which specifies the per-request model override.
// Takes defaultRef (ModelRef) which provides the fallback model reference.
//
// Returns string which is the model name to use.
// Returns ModelRef which is the reference to verify against; per-request
// models have no digest and are unverified.
func (*ollamaProvider) resolveModel(reqModel string, defaultRef ModelRef) (string, ModelRef) {
	if reqModel != "" {
		return reqModel, ModelRef{Name: reqModel}
	}
	return defaultRef.Name, defaultRef
}

// ensureModel checks that a model is available locally and pulls it if
// AutoPull is enabled and the model is not found. When ref carries a Digest,
// the locally installed model's digest is verified after pull/discovery.
//
// Takes model (string) which is the model name to ensure.
// Takes ref (ModelRef) which optionally carries a digest for verification.
//
// Returns error when the model cannot be found, pulled, or verified.
func (p *ollamaProvider) ensureModel(ctx context.Context, model string, ref ModelRef) error {
	ctx, l := logger.From(ctx, log)
	_, err := p.client.Show(ctx, &api.ShowRequest{Model: model})
	if err == nil {
		return p.verifyModelDigest(ctx, model, ref)
	}

	if !*p.config.AutoPull {
		return fmt.Errorf(
			"ollama model %q not found locally and AutoPull is disabled - run: ollama pull %s",
			model, model,
		)
	}

	if err := p.pullModel(ctx, l, model); err != nil {
		return err
	}

	return p.verifyModelDigest(ctx, model, ref)
}

// pullModel downloads a model via the Ollama Pull API, logging
// layer progress as each new layer is discovered.
//
// Takes l (logger.Logger) which receives progress log entries.
// Takes model (string) which is the model name to pull.
//
// Returns error when the pull request fails.
func (p *ollamaProvider) pullModel(ctx context.Context, l logger.Logger, model string) error {
	l.Info("Pulling Ollama model",
		logger.String(logKeyModel, model),
	)

	modelPullCount.Add(ctx, 1)
	start := time.Now()

	tracker := newPullTracker(l, model)

	pullErr := p.client.Pull(ctx, &api.PullRequest{Model: model}, tracker.onProgress)

	modelPullDuration.Record(ctx, float64(time.Since(start).Milliseconds()))

	if pullErr != nil {
		return fmt.Errorf("pulling model %q: %w", model, wrapError(pullErr))
	}

	l.Info("Model pull complete",
		logger.String(logKeyModel, model),
		logger.Duration("duration", time.Since(start)),
	)

	return nil
}

// pullTracker tracks layer download progress during a model pull.
type pullTracker struct {
	// l is the logger that receives progress messages.
	l logger.Logger

	// layers maps digest strings to their download progress.
	layers map[string]*layerProgress

	// model is the name of the model being pulled.
	model string

	// lastLogStatus is the most recently logged status string,
	// used to avoid duplicate log entries.
	lastLogStatus string

	// layerOrder records digests in discovery order so that
	// layer indices are stable.
	layerOrder []string
}

// layerProgress tracks the byte progress of a single layer
// download.
type layerProgress struct {
	// total is the total number of bytes for this layer.
	total int64

	// completed is the number of bytes downloaded so far.
	completed int64
}

// onProgress handles a single progress callback from the Ollama
// Pull API.
//
// Takes response (api.ProgressResponse) which contains the current
// pull progress data.
//
// Returns error which is always nil.
func (t *pullTracker) onProgress(response api.ProgressResponse) error {
	if response.Digest != "" && response.Total > 0 {
		lp, seen := t.layers[response.Digest]
		if !seen {
			lp = &layerProgress{}
			t.layers[response.Digest] = lp
			t.layerOrder = append(t.layerOrder, response.Digest)
		}
		lp.total = response.Total
		lp.completed = response.Completed

		if !seen {
			index := len(t.layerOrder)
			t.l.Info("Downloading layer",
				logger.String(logKeyModel, t.model),
				logger.String("layer", fmt.Sprintf("%d", index)),
				logger.String("digest", shortDigest(response.Digest)),
				logger.String("size", formatBytes(response.Total)),
			)
		}
		return nil
	}

	if response.Status != t.lastLogStatus {
		t.l.Info("Pulling model",
			logger.String(logKeyModel, t.model),
			logger.String("status", response.Status),
		)
		t.lastLogStatus = response.Status
	}
	return nil
}

// verifyModelDigest checks the local model's digest against ref.Digest when
// a digest was specified. It queries the Ollama List API to obtain the
// installed model's digest.
//
// Takes model (string) which specifies the model name to verify.
// Takes ref (ModelRef) which contains the expected digest to check against.
//
// Returns error when the model is not found or the digest does not match.
func (p *ollamaProvider) verifyModelDigest(ctx context.Context, model string, ref ModelRef) error {
	if ref.Digest == "" {
		return nil
	}

	listResp, err := p.client.List(ctx)
	if err != nil {
		return fmt.Errorf("listing models for digest verification: %w", err)
	}

	normalised := model
	if !strings.Contains(normalised, ":") {
		normalised = normalised + ":latest"
	}

	for i := range listResp.Models {
		m := &listResp.Models[i]
		if m.Name == model || m.Model == model || m.Name == normalised || m.Model == normalised {
			return ref.verifyDigest(m.Digest)
		}
	}

	return fmt.Errorf("model %q not found in model list after ensure (cannot verify digest)", model)
}

// newPullTracker creates a pullTracker for the given model.
//
// Takes l (logger.Logger) which receives progress messages.
// Takes model (string) which is the model being pulled.
//
// Returns *pullTracker which is the initialised tracker.
func newPullTracker(l logger.Logger, model string) *pullTracker {
	return &pullTracker{
		l:      l,
		model:  model,
		layers: make(map[string]*layerProgress),
	}
}

// shortDigest returns the first 12 hex characters of a digest string,
// stripping any "sha256:" prefix.
//
// Takes digest (string) which is the digest value to shorten.
//
// Returns string which is the shortened digest, at most 12 characters.
func shortDigest(digest string) string {
	const prefix = "sha256:"
	const shortDigestLen = 12

	if len(digest) > len(prefix) && digest[:len(prefix)] == prefix {
		digest = digest[len(prefix):]
	}
	if len(digest) > shortDigestLen {
		return digest[:shortDigestLen]
	}
	return digest
}

// formatBytes formats a byte count as a human-readable string.
//
// Takes b (int64) which specifies the number of bytes to format.
//
// Returns string which is the formatted size with appropriate units.
func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
