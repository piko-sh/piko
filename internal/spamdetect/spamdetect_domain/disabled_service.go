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

package spamdetect_domain

import (
	"context"

	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

// DisabledSpamDetectService implements SpamDetectServicePort but returns
// errors for analysis operations. Used when no spam detection detector is
// configured.
type DisabledSpamDetectService struct{}

var _ SpamDetectServicePort = (*DisabledSpamDetectService)(nil)

// NewDisabledSpamDetectService creates a disabled spam detection service.
//
// Returns *DisabledSpamDetectService which is the disabled service.
func NewDisabledSpamDetectService() *DisabledSpamDetectService {
	return &DisabledSpamDetectService{}
}

// Analyse returns ErrSpamDetectDisabled.
//
// Returns *spamdetect_dto.AnalysisResult which is always nil.
// Returns error which is always ErrSpamDetectDisabled.
func (*DisabledSpamDetectService) Analyse(_ context.Context, _ *spamdetect_dto.Submission, _ *spamdetect_dto.Schema) (*spamdetect_dto.AnalysisResult, error) {
	return nil, spamdetect_dto.ErrSpamDetectDisabled
}

// IsEnabled returns false.
//
// Returns bool which is always false.
func (*DisabledSpamDetectService) IsEnabled() bool { return false }

// RegisterDetector returns ErrSpamDetectDisabled.
//
// Returns error which is always ErrSpamDetectDisabled.
func (*DisabledSpamDetectService) RegisterDetector(_ context.Context, _ string, _ Detector) error {
	return spamdetect_dto.ErrSpamDetectDisabled
}

// GetDetectors returns an empty list.
//
// Returns []string which is always empty.
func (*DisabledSpamDetectService) GetDetectors(_ context.Context) []string { return []string{} }

// HasDetector returns false.
//
// Returns bool which is always false.
func (*DisabledSpamDetectService) HasDetector(_ string) bool { return false }

// ListDetectors returns an empty list.
//
// Returns []provider_domain.ProviderInfo which is always empty.
func (*DisabledSpamDetectService) ListDetectors(_ context.Context) []provider_domain.ProviderInfo {
	return []provider_domain.ProviderInfo{}
}

// SetFeedbackStore is a no-op.
func (*DisabledSpamDetectService) SetFeedbackStore(_ FeedbackStore) {}

// ReportSpam returns ErrSpamDetectDisabled.
//
// Returns error which is always ErrSpamDetectDisabled.
func (*DisabledSpamDetectService) ReportSpam(_ context.Context, _ string) error {
	return spamdetect_dto.ErrSpamDetectDisabled
}

// ReportHam returns ErrSpamDetectDisabled.
//
// Returns error which is always ErrSpamDetectDisabled.
func (*DisabledSpamDetectService) ReportHam(_ context.Context, _ string) error {
	return spamdetect_dto.ErrSpamDetectDisabled
}

// HealthCheck returns nil.
//
// Returns error which is always nil.
func (*DisabledSpamDetectService) HealthCheck(_ context.Context) error { return nil }

// Close is a no-op.
//
// Returns error which is always nil.
func (*DisabledSpamDetectService) Close(_ context.Context) error { return nil }
