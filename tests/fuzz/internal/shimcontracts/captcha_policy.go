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

package shimcontracts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/wdk/captcha"
)

const (
	// policyChoiceOffset is the first choice byte the policy checks draw.
	policyChoiceOffset = 24

	// configuredScoreBase is the lowest configured default threshold in score steps.
	configuredScoreBase = 32

	// explicitScoreBase is the lowest explicit threshold in score steps.
	explicitScoreBase = 42

	// scoreJitter bounds the drawn threshold increment in score steps.
	scoreJitter = 8

	// explicitScoreDrop lowers the explicit threshold below the configured default.
	explicitScoreDrop = .375

	// baseVerifyQuota is the smallest configured verification quota.
	baseVerifyQuota = 3

	// quotaDrawIndex is the policy draw that selects the verification quota.
	quotaDrawIndex = 3

	// verifyQuotaJitter bounds the drawn verification quota increment.
	verifyQuotaJitter = 4

	// scoreReplyEpoch is the Unix second of the first score check's provider reply.
	scoreReplyEpoch = 100

	// admissionScore is the provider score returned during quota checks.
	admissionScore = .875

	// fixtureAddress is the remote address of policy verification requests.
	fixtureAddress = "192.0.2.88"

	// admissionAction is the form action of quota checks.
	admissionAction = "admission"
)

var (
	// errOfflineQuotaStore is the controlled quota-store failure.
	errOfflineQuotaStore = errors.New("offline quota store")

	// scoreRelations names the score's position relative to the threshold.
	scoreRelations = []string{"below", "equal", "above"}
)

// captchaAdmission records a CAPTCHA rate-limit request.
type captchaAdmission struct {
	// key is the rate-limit bucket key.
	key string

	// limit is the allowed request count per window.
	limit int

	// window is the rate-limit window duration.
	window time.Duration
}

// captchaLimiter records rate-limit calls and returns a controlled decision.
type captchaLimiter struct {
	// fault is the controlled rate-limit failure.
	fault error

	// calls holds rate-limit requests in call order.
	calls []captchaAdmission

	// allowed is the controlled admission result.
	allowed bool
}

// IsAllowed records the admission request and returns its controlled decision.
//
// Takes key (string) which is the verification rate-limit bucket key.
// Takes limit (int) which is the permitted verification count per window.
// Takes window (time.Duration) which is the rate-limit window duration.
//
// Returns bool which is the controlled admission decision.
// Returns error which is the controlled service error or the request context's
// cancellation error.
func (l *captchaLimiter) IsAllowed(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	l.calls = append(l.calls, captchaAdmission{key: key, limit: limit, window: window})
	return l.allowed, l.fault
}

// exerciseCaptchaPolicy checks score thresholds, quotas and disabled verification.
//
// Takes choices (*[captchaChoices]byte) which selects bounded provider and policy
// transitions.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func exerciseCaptchaPolicy(ctx context.Context, choices *[captchaChoices]byte) (err error) {
	if err := checkDisabledCaptcha(ctx, captcha_domain.NewDisabledCaptchaService()); err != nil {
		return err
	}
	draws := choices[policyChoiceOffset:]
	configured := float64(configuredScoreBase+draws[0]%scoreJitter) / scoreSteps
	explicit := float64(explicitScoreBase+draws[1]%scoreJitter) / scoreSteps
	if draws[2]&1 != 0 {
		explicit -= explicitScoreDrop
	}
	configuration := captcha_dto.DefaultServiceConfig()
	configuration.DefaultScoreThreshold = configured
	configuration.VerifyRateLimit = baseVerifyQuota + int(draws[quotaDrawIndex]%verifyQuotaJitter)
	limiter := &captchaLimiter{fault: nil, calls: nil, allowed: true}
	service, err := captcha_domain.NewCaptchaService(configuration, captcha_domain.WithRateLimiter(limiter))
	if err != nil {
		return fmt.Errorf("construct scored captcha service: %w", err)
	}
	defer func() {
		if closeErr := service.Close(context.WithoutCancel(ctx)); closeErr != nil {
			err = errors.Join(err, captchaFailure("close", closeErr))
		}
	}()
	var calls []captchaCall
	provider := &captchaFixture{failure: nil, calls: &calls, cancel: nil, name: "scores", kind: captcha.ProviderTypeRecaptchaV3, reply: captcha.VerifyResponse{}}
	if err := service.RegisterProvider(ctx, provider.name, provider); err != nil {
		return fmt.Errorf("register scored provider: %w", err)
	}
	if err := service.SetDefaultProvider(provider.name); err != nil {
		return fmt.Errorf("select scored provider: %w", err)
	}
	if err := checkCaptchaScores(ctx, service, provider, configured, explicit); err != nil {
		return err
	}
	return checkCaptchaLimiter(ctx, service, provider, limiter, configuration.VerifyRateLimit)
}

// checkDisabledCaptcha checks the disabled service's public responses.
//
// Takes service (captcha.ServicePort) which is the production CAPTCHA service.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkDisabledCaptcha(ctx context.Context, service captcha.ServicePort) error {
	if service.IsEnabled() || service.SiteKey() != "" || service.ScriptURL() != "" {
		return captchaDifference("disabled-metadata", "expected inactive service")
	}
	const ticket, action = "fixture-disabled", "signup"
	if err := service.Verify(ctx, ticket, fixtureAddress, action); !errors.Is(err, captcha.ErrCaptchaDisabled) {
		return captchaDifference("disabled-default", "got %v", err)
	}
	response, err := service.VerifyWithScore(ctx, ticket, fixtureAddress, action, minimumScore)
	if response != nil || !errors.Is(err, captcha.ErrCaptchaDisabled) {
		return captchaDifference("disabled-score", "got %v, %v", response, err)
	}
	response, err = service.VerifyWithProvider(ctx, "scores", ticket, fixtureAddress, action, minimumScore)
	if response != nil || !errors.Is(err, captcha.ErrCaptchaDisabled) {
		return captchaDifference("disabled-named", "got %v, %v", response, err)
	}
	return nil
}

// checkCaptchaScores checks default and explicit score thresholds at their boundaries.
//
// Takes service (captcha.ServicePort) which is the production CAPTCHA service.
// Takes provider (*captchaFixture) which is the controlled provider fixture.
// Takes configured (float64) which is the default score threshold.
// Takes explicit (float64) which is the per-request score threshold.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkCaptchaScores(ctx context.Context, service captcha.ServicePort, provider *captchaFixture, configured, explicit float64) error {
	for _, mode := range []string{operationDefault, "score", operationNamed} {
		threshold := explicit
		if mode == operationDefault {
			threshold = configured
		}
		for index := range scoreRelations {
			if cause := context.Cause(ctx); cause != nil {
				return cause
			}
			if err := checkCaptchaScore(ctx, service, provider, mode, index, threshold); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkCaptchaScore checks one score just below, at or just above a threshold.
//
// Takes service (captcha.ServicePort) which is the production CAPTCHA service.
// Takes provider (*captchaFixture) which is the controlled provider fixture.
// Takes mode (string) which selects the service entry point.
// Takes index (int) which selects the score's relation to the threshold.
// Takes threshold (float64) which is the effective score threshold.
//
// Returns error which reports a model mismatch.
func checkCaptchaScore(ctx context.Context, service captcha.ServicePort, provider *captchaFixture, mode string, index int, threshold float64) error {
	relation := scoreRelations[index]
	score := threshold + float64(index-1)/scoreSteps
	action := "score-" + relation
	ticket := "fixture-" + mode + "-" + relation
	before := len(*provider.calls)
	want := captcha.VerifyResponse{
		Timestamp:  time.Unix(scoreReplyEpoch+int64(index), 0).UTC(),
		Score:      &score,
		Action:     action,
		Hostname:   "scores.example.invalid",
		ErrorCodes: nil,
		Success:    true,
	}
	provider.reply = want
	var got *captcha.VerifyResponse
	var err error
	switch mode {
	case operationDefault:
		err = service.Verify(ctx, ticket, fixtureAddress, action)
	case operationNamed:
		got, err = service.VerifyWithProvider(ctx, provider.name, ticket, fixtureAddress, action, threshold)
	default:
		got, err = service.VerifyWithScore(ctx, ticket, fixtureAddress, action, threshold)
	}
	var wantErr error
	if index == 0 {
		wantErr = captcha.ErrScoreBelowThreshold
	}
	field := "score-" + mode + "-" + relation
	if !errors.Is(err, wantErr) {
		return captchaDifference(field, "score %g, threshold %g: got %v, expected %v", score, threshold, err, wantErr)
	}
	if mode != operationDefault && !captchaResponseEqual(got, &want) {
		return captchaDifference(field+"-result", "provider verdict or score changed")
	}
	if len(*provider.calls) != before+1 {
		return captchaDifference(field+"-calls", "expected one provider evaluation")
	}
	return nil
}

// checkCaptchaLimiter checks rate-limit decisions and forwarded quota settings.
//
// Takes service (captcha.ServicePort) which is the production CAPTCHA service.
// Takes provider (*captchaFixture) which is the controlled provider fixture.
// Takes limiter (*captchaLimiter) which records rate-limit calls and controlled
// decisions.
// Takes limit (int) which is the configured verification quota.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkCaptchaLimiter(ctx context.Context, service captcha.ServicePort, provider *captchaFixture, limiter *captchaLimiter, limit int) error {
	for index, outcome := range []string{"allow", "deny", "error", "recover"} {
		if cause := context.Cause(ctx); cause != nil {
			return cause
		}
		before, prior := len(limiter.calls), len(*provider.calls)
		limiter.allowed = outcome != "deny"
		limiter.fault = nil
		if outcome == "error" {
			limiter.fault = errOfflineQuotaStore
		}
		ticket := fmt.Sprintf("fixture-admission-%d", index)
		provider.reply = captcha.VerifyResponse{
			Timestamp:  time.Time{},
			Score:      new(admissionScore),
			Action:     admissionAction,
			Hostname:   "",
			ErrorCodes: nil,
			Success:    true,
		}
		got, err := service.VerifyWithProvider(ctx, provider.name, ticket, fixtureAddress, admissionAction, minimumScore)
		consulted := captchaAdmission{key: "captcha:verify:" + fixtureAddress, limit: limit, window: time.Minute}
		if len(limiter.calls) != before+1 || limiter.calls[before] != consulted {
			return captchaDifference("limiter-consult-"+outcome, "configured quota request differed")
		}
		if err := checkAdmission(outcome, got, err, provider, prior); err != nil {
			return err
		}
	}
	return nil
}

// checkAdmission compares a quota decision's effect with its expected outcome.
//
// Takes outcome (string) which is the controlled quota decision.
// Takes got (*captcha.VerifyResponse) which is the service's response.
// Takes err (error) which is the service's verification error.
// Takes provider (*captchaFixture) which is the controlled provider fixture.
// Takes prior (int) which is the number of provider calls before the request.
//
// Returns error which reports a refusal that reached the provider or an admitted request
// that did not complete.
func checkAdmission(outcome string, got *captcha.VerifyResponse, err error, provider *captchaFixture, prior int) error {
	if outcome == "deny" || outcome == "error" {
		if !errors.Is(err, captcha_dto.ErrRateLimited) || got != nil || len(*provider.calls) != prior {
			return captchaDifference("limiter-"+outcome, "expected quota refusal before the provider ran")
		}
		return nil
	}
	if err != nil || !captchaResponseEqual(got, &provider.reply) || len(*provider.calls) != prior+1 {
		return captchaDifference("limiter-"+outcome, "expected one completed provider evaluation, got %v", err)
	}
	return nil
}
