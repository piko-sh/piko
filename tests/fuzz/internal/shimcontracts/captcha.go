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
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/wdk/captcha"
	hmacchallenge "piko.sh/piko/wdk/captcha/captcha_provider_hmac_challenge"
	"piko.sh/piko/wdk/clock"
)

const (
	// localChallengeTTL is the validity of a local challenge in the fixture.
	localChallengeTTL = time.Minute

	// captchaChoices is the number of choice bytes a CAPTCHA run draws.
	captchaChoices = 32

	// minimumScore is the score threshold requested from the service.
	minimumScore = 0.5

	// baseScore is the lowest score a fixture provider returns.
	baseScore = 0.75

	// scoreSteps divides the drawn score increment.
	scoreSteps = 64

	// variableStages bounds the drawn routing stages appended to the fixed sequence.
	variableStages = 5

	// stageDrawOffset is the first choice byte that varies stage requests.
	stageDrawOffset = 7

	// operationNamed verifies through an explicitly named provider.
	operationNamed = "named"

	// operationDefault verifies through the default provider with a score threshold.
	operationDefault = "default"

	// operationSimple verifies through the default provider without returning a response.
	operationSimple = "simple"

	// outcomeAccept is a successful verification.
	outcomeAccept = "accept"

	// outcomeDecline is a provider refusal.
	outcomeDecline = "decline"

	// outcomeMismatch is a successful provider reply for another action.
	outcomeMismatch = "mismatch"

	// outcomeCancel is a verification whose request context is cancelled.
	outcomeCancel = "cancel"

	// outcomeUnavailable is a provider outage.
	outcomeUnavailable = "unavailable"

	// commentAction is the form action used by fixture challenges.
	commentAction = "comment"
)

var (
	// errFixtureMissedCancellation reports a fixture provider that did not see its own
	// cancellation.
	errFixtureMissedCancellation = errors.New("fixture did not observe cancellation")

	// fixedCaptchaStages is the routing sequence every run performs before its drawn stages.
	fixedCaptchaStages = []captchaStage{
		{operation: operationDefault, outcome: outcomeAccept, changeDefault: false},
		{operation: operationNamed, outcome: outcomeAccept, changeDefault: false},
		{operation: operationNamed, outcome: outcomeDecline, changeDefault: false},
		{operation: operationDefault, outcome: outcomeUnavailable, changeDefault: false},
		{operation: operationDefault, outcome: outcomeAccept, changeDefault: false},
		{operation: operationNamed, outcome: outcomeCancel, changeDefault: false},
		{operation: operationNamed, outcome: outcomeAccept, changeDefault: false},
		{operation: operationSimple, outcome: outcomeAccept, changeDefault: true},
		{operation: operationNamed, outcome: outcomeAccept, changeDefault: false},
		{operation: operationDefault, outcome: outcomeAccept, changeDefault: false},
		{operation: operationDefault, outcome: outcomeMismatch, changeDefault: false},
		{operation: operationSimple, outcome: outcomeMismatch, changeDefault: false},
	}
)

// captchaContextKey marks fixture CAPTCHA context values.
type captchaContextKey struct{}

// captchaCall records the provider request and context marker.
type captchaCall struct {
	// provider identifies the selected CAPTCHA provider.
	provider string

	// request holds the provider request.
	request captcha.VerifyRequest

	// marker is the forwarded request-context marker.
	marker int
}

// captchaFixture supplies controlled provider replies and failures.
type captchaFixture struct {
	// failure is the controlled provider error.
	failure error

	// calls receives the observed provider requests.
	calls *[]captchaCall

	// cancel cancels the fixture request during verification.
	cancel context.CancelFunc

	// name identifies the fixture or operation.
	name string

	// kind selects the node or provider kind.
	kind captcha.ProviderType

	// reply is the controlled provider response.
	reply captcha.VerifyResponse
}

// captchaStage selects one routing transition and expected outcome.
type captchaStage struct {
	// operation selects the service entry point.
	operation string

	// outcome is the expected verification outcome.
	outcome string

	// changeDefault changes the default provider before the call.
	changeDefault bool
}

// captchaRun holds the providers and recorded calls shared by a routing sequence.
type captchaRun struct {
	// at is the independently fixed verification time.
	at time.Time

	// service is the production service under test.
	service captcha.ServicePort

	// calls records provider requests.
	calls *[]captchaCall

	// selected is the current default provider.
	selected *captchaFixture

	// other is the named provider.
	other *captchaFixture
}

// captchaMismatch identifies a CAPTCHA field mismatch.
type captchaMismatch struct {
	// err carries the mismatch detail and any wrapped cause.
	err error

	// field identifies the mismatched property.
	field string
}

// localChallengeIssuer issues challenges from the built-in provider.
type localChallengeIssuer interface {
	// GenerateChallenge issues a signed challenge for an action.
	//
	// Takes action (string) which names the action to bind to the challenge.
	//
	// Returns string which is the signed challenge.
	// Returns error which reports a failure to issue the challenge.
	GenerateChallenge(action string) (string, error)
}

// Type returns the fixture provider kind.
//
// Returns captcha.ProviderType which is the configured fixture provider kind.
func (p *captchaFixture) Type() captcha.ProviderType { return p.kind }

// SiteKey returns the fixture provider's public site key.
//
// Returns string which is the fixture provider's public site key.
func (p *captchaFixture) SiteKey() string { return "site-" + p.name }

// ScriptURL returns the fixture provider's browser script URL.
//
// Returns string which is the fixture provider's browser script URL.
func (p *captchaFixture) ScriptURL() string { return "/local-fixtures/" + p.name + ".js" }

// RenderRequirements returns the fixture provider's rendering requirements.
//
// Returns *captcha_dto.RenderRequirements which holds the fixture provider's rendering
// requirements.
func (*captchaFixture) RenderRequirements() *captcha_dto.RenderRequirements {
	return &captcha_dto.RenderRequirements{}
}

// HealthCheck reports the fixture provider's health.
//
// Returns error which is the request context's cancellation error, or nil while it
// remains active.
func (*captchaFixture) HealthCheck(ctx context.Context) error { return ctx.Err() }

// Verify records the provider request and returns its controlled outcome.
//
// Takes request (*captcha.VerifyRequest) which holds the CAPTCHA verification input.
//
// Returns *captcha.VerifyResponse which holds a clone of the configured response, or is
// nil on controlled failure.
// Returns error which reports the fixture provider's controlled failure or request
// cancellation, or nil on success.
func (p *captchaFixture) Verify(ctx context.Context, request *captcha.VerifyRequest) (*captcha.VerifyResponse, error) {
	marker, ok := ctx.Value(captchaContextKey{}).(int)
	if !ok {
		marker = 0
	}
	*p.calls = append(*p.calls, captchaCall{provider: p.name, request: *request, marker: marker})
	if p.cancel != nil {
		p.cancel()
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return nil, errFixtureMissedCancellation
	}
	if p.failure != nil {
		return nil, p.failure
	}
	reply := p.reply
	reply.ErrorCodes = slices.Clone(p.reply.ErrorCodes)
	if p.reply.Score != nil {
		reply.Score = new(*p.reply.Score)
	}
	return &reply, nil
}

// Error formats the recorded failure.
//
// Returns string which is the formatted failure message.
func (e captchaMismatch) Error() string { return "captcha-" + e.field + ": " + e.err.Error() }

// Unwrap exposes the wrapped cause of the mismatch.
//
// Returns error which is the mismatch detail and any cause it wraps.
func (e captchaMismatch) Unwrap() error { return e.err }

// FailureSubject identifies the failed property independently of generated values.
//
// Returns string which is the stable failed-property identity.
func (e captchaMismatch) FailureSubject() string { return "captcha-" + e.field }

// captchaDifference creates a CAPTCHA failure with a stable field identity.
//
// Takes field (string) which identifies the mismatched field.
// Takes format (string) which is the failure-message format, which may use %w.
// Takes arguments (...any) which supplies formatted mismatch values.
//
// Returns error which is the non-nil CAPTCHA mismatch for the selected field.
func captchaDifference(field string, format string, arguments ...any) error {
	return captchaMismatch{field: field, err: fmt.Errorf(format, arguments...)}
}

// captchaFailure attributes an error to a CAPTCHA field.
//
// Takes field (string) which identifies the failed field.
// Takes err (error) which is the failure.
//
// Returns error which is the CAPTCHA mismatch wrapping err.
func captchaFailure(field string, err error) error {
	return captchaMismatch{field: field, err: err}
}

// RunCaptcha checks provider routing, score thresholds and verification quotas.
//
// Takes data ([]byte) which holds the replayable choice bytes.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func RunCaptcha(ctx context.Context, data []byte) (err error) {
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}
	at := time.Unix(1_700_000_000, 0).UTC()
	service, err := captcha_domain.NewCaptchaService(nil, captcha_domain.WithClock(clock.NewMockClock(at)))
	if err != nil {
		return captchaFailure("construction", err)
	}
	defer func() {
		if closeErr := service.Close(context.WithoutCancel(ctx)); closeErr != nil {
			err = errors.Join(err, captchaFailure("close", closeErr))
		}
	}()
	var choices [captchaChoices]byte
	copy(choices[:], data)
	err = exerciseCaptchaService(ctx, service, &choices, at)
	if err == nil {
		err = exerciseCaptchaPolicy(ctx, &choices)
	}
	if err == nil {
		err = exerciseLocalCaptchaAdapter(ctx, service, at)
	}
	if err != nil && context.Cause(ctx) == nil {
		if _, ok := errors.AsType[captchaMismatch](err); !ok {
			return captchaFailure("configuration", err)
		}
	}
	return err
}

// stage performs one routing stage and checks the observed provider call.
//
// Takes stage (captchaStage) which selects the operation and expected outcome.
// Takes index (int) which is the stage's position.
// Takes draw (byte) which varies the request.
//
// Returns error which reports a model mismatch, or nil when the stage passes.
func (r *captchaRun) stage(ctx context.Context, stage captchaStage, index int, draw byte) error {
	if r.service.SiteKey() != r.selected.SiteKey() || r.service.ScriptURL() != r.selected.ScriptURL() {
		return captchaDifference("default-metadata", "step %d expected %s", index, r.selected.name)
	}
	who := r.selected
	if stage.operation == operationNamed {
		who = r.other
	}
	request, want := r.expectation(stage, who, index, draw)
	who.reply = want
	who.failure = nil
	if stage.outcome == outcomeUnavailable {
		who.failure = captcha.ErrProviderUnavailable
	}
	callCtx, cancel := context.WithCancel(context.WithValue(ctx, captchaContextKey{}, index+1))
	if stage.outcome == outcomeCancel {
		who.cancel = cancel
	}
	start := len(*r.calls)
	got, err := r.invoke(callCtx, stage.operation, &request)
	cancel()
	who.cancel = nil
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}
	if callErr := r.checkCall(start, who, &request, index); callErr != nil {
		return callErr
	}
	return checkCaptchaOutcome(stage, who, got, &want, err, index)
}

// expectation builds a stage's provider request and the reply the provider returns.
//
// Takes stage (captchaStage) which selects the operation and expected outcome.
// Takes who (*captchaFixture) which is the provider the stage must reach.
// Takes index (int) which is the stage's position.
// Takes draw (byte) which varies the request.
//
// Returns captcha.VerifyRequest which is the expected provider request.
// Returns captcha.VerifyResponse which is the provider's controlled reply.
func (r *captchaRun) expectation(stage captchaStage, who *captchaFixture, index int, draw byte) (captcha.VerifyRequest, captcha.VerifyResponse) {
	actions := []string{commentAction, "checkout", "signup"}
	request := captcha.VerifyRequest{
		Token:    fmt.Sprintf("fixture-receipt-%d-%d", index, draw),
		RemoteIP: fmt.Sprintf("192.0.2.%d", 1+int(draw)%254),
		Action:   actions[int(draw)%len(actions)],
	}
	score := baseScore + float64(draw%16)/scoreSteps
	want := captcha.VerifyResponse{
		Timestamp:  r.at.Add(time.Duration(index) * time.Second),
		Score:      &score,
		Action:     request.Action,
		Hostname:   who.name + ".example.invalid",
		ErrorCodes: nil,
		Success:    stage.outcome != outcomeDecline,
	}
	if !want.Success {
		want.ErrorCodes = []string{"fixture-declined"}
	}
	if stage.outcome == outcomeMismatch {
		want.Action = "fixture-other-form"
	}
	return request, want
}

// invoke calls the service entry point a stage selects.
//
// Takes operation (string) which selects the service entry point.
// Takes request (*captcha.VerifyRequest) which is the verification input.
//
// Returns *captcha.VerifyResponse which is the service's response, or nil for the simple
// entry point.
// Returns error which is the service's verification error.
func (r *captchaRun) invoke(ctx context.Context, operation string, request *captcha.VerifyRequest) (*captcha.VerifyResponse, error) {
	switch operation {
	case operationNamed:
		return r.service.VerifyWithProvider(ctx, r.other.name, request.Token, request.RemoteIP, request.Action, minimumScore)
	case operationDefault:
		return r.service.VerifyWithScore(ctx, request.Token, request.RemoteIP, request.Action, minimumScore)
	default:
		return nil, r.service.Verify(ctx, request.Token, request.RemoteIP, request.Action)
	}
}

// checkCall checks that a stage made exactly one provider call with the expected routing,
// request and context.
//
// Takes start (int) which is the number of calls before the stage.
// Takes who (*captchaFixture) which is the provider the stage must reach.
// Takes request (*captcha.VerifyRequest) which is the expected provider request.
// Takes index (int) which is the stage's position.
//
// Returns error which reports a missing, misrouted or altered provider call.
func (r *captchaRun) checkCall(start int, who *captchaFixture, request *captcha.VerifyRequest, index int) error {
	if len(*r.calls) != start+1 {
		return captchaDifference("call-count", "step %d added %d calls", index, len(*r.calls)-start)
	}
	observed := (*r.calls)[start]
	if observed.provider != who.name {
		return captchaDifference("routing", "step %d expected %s, saw %s", index, who.name, observed.provider)
	}
	if observed.request != *request {
		return captchaDifference("request", "step %d got %+v, want %+v", index, observed.request, *request)
	}
	if observed.marker != index+1 {
		return captchaDifference("context", "step %d lost caller context", index)
	}
	return nil
}

// exerciseCaptchaService checks provider routing, recovery and request forwarding.
//
// Takes service (captcha.ServicePort) which is the production service under test.
// Takes choices ([captchaChoices]byte) which selects bounded provider and policy
// transitions.
// Takes at (time.Time) which is the independently fixed verification time.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func exerciseCaptchaService(ctx context.Context, service captcha.ServicePort, choices *[captchaChoices]byte, at time.Time) error {
	var calls []captchaCall
	amber := &captchaFixture{failure: nil, calls: &calls, cancel: nil, name: "amber", kind: captcha.ProviderTypeHCaptcha, reply: captcha.VerifyResponse{}}
	indigo := &captchaFixture{failure: nil, calls: &calls, cancel: nil, name: "indigo", kind: captcha.ProviderTypeTurnstile, reply: captcha.VerifyResponse{}}
	if err := service.Verify(ctx, "fixture-startup", "192.0.2.1", "startup"); !errors.Is(err, captcha.ErrProviderUnavailable) {
		return captchaDifference("missing-provider", "got %v", err)
	}
	if err := registerCaptchaFixtures(ctx, service, amber, indigo); err != nil {
		return err
	}
	run := &captchaRun{at: at, service: service, calls: &calls, selected: amber, other: indigo}
	if choices[0]&1 != 0 {
		run.selected, run.other = indigo, amber
	}
	if err := service.SetDefaultProvider(run.selected.name); err != nil {
		return fmt.Errorf("select default provider: %w", err)
	}
	for index, stage := range captchaStages(choices) {
		if cause := context.Cause(ctx); cause != nil {
			return cause
		}
		if stage.changeDefault {
			run.selected, run.other = run.other, run.selected
			if err := service.SetDefaultProvider(run.selected.name); err != nil {
				return fmt.Errorf("switch default provider: %w", err)
			}
		}
		if err := run.stage(ctx, stage, index, choices[(stageDrawOffset+index)%captchaChoices]); err != nil {
			return err
		}
	}
	return nil
}

// registerCaptchaFixtures registers fixture providers and checks the registered names.
//
// Takes service (captcha.ServicePort) which is the production service under test.
// Takes fixtures (...*captchaFixture) which lists the providers in name order.
//
// Returns error which reports a registration failure or a differing name list.
func registerCaptchaFixtures(ctx context.Context, service captcha.ServicePort, fixtures ...*captchaFixture) error {
	names := make([]string, 0, len(fixtures))
	for _, provider := range fixtures {
		if err := service.RegisterProvider(ctx, provider.name, provider); err != nil {
			return fmt.Errorf("register %s: %w", provider.name, err)
		}
		names = append(names, provider.name)
	}
	if registered := service.GetProviders(ctx); !slices.Equal(registered, names) {
		return captchaDifference("registered-names", "got %v", registered)
	}
	return nil
}

// captchaStages lists the fixed routing sequence followed by the drawn stages.
//
// Takes choices (*[captchaChoices]byte) which selects the drawn stages.
//
// Returns []captchaStage which lists the stages in order.
func captchaStages(choices *[captchaChoices]byte) []captchaStage {
	stages := make([]captchaStage, 0, len(fixedCaptchaStages)+variableStages)
	stages = append(stages, fixedCaptchaStages...)
	operations := []string{operationNamed, operationDefault, operationSimple}
	outcomes := []string{outcomeAccept, outcomeDecline, outcomeAccept, outcomeMismatch}
	for index := range int(choices[1] % variableStages) {
		draw := int(choices[2+index])
		stages = append(stages, captchaStage{
			operation:     operations[draw%len(operations)],
			outcome:       outcomes[(draw/len(operations))%len(outcomes)],
			changeDefault: draw&32 != 0,
		})
	}
	return stages
}

// checkCaptchaOutcome compares a stage's service result with its expected outcome.
//
// Takes stage (captchaStage) which selects the operation and expected outcome.
// Takes who (*captchaFixture) which is the provider the stage reached.
// Takes got (*captcha.VerifyResponse) which is the service's response.
// Takes want (*captcha.VerifyResponse) which is the provider's controlled reply.
// Takes err (error) which is the service's verification error.
// Takes index (int) which is the stage's position.
//
// Returns error which reports a wrong status, provider attribution or response.
func checkCaptchaOutcome(stage captchaStage, who *captchaFixture, got, want *captcha.VerifyResponse, err error, index int) error {
	cause := expectedCaptchaCause(stage.outcome)
	if !errors.Is(err, cause) {
		return captchaDifference("status-"+stage.outcome, "step %d got %v, want %v", index, err, cause)
	}
	if stage.outcome == outcomeCancel || stage.outcome == outcomeUnavailable {
		wrapped, ok := errors.AsType[*captcha.CaptchaError](err)
		if got != nil || !ok || wrapped.Provider != string(who.kind) {
			return captchaDifference("provider-error", "step %d invalid result or provider attribution: %v", index, err)
		}
		return nil
	}
	if stage.operation != operationSimple && !captchaResponseEqual(got, want) {
		return captchaDifference("response", "step %d expected preserved provider response", index)
	}
	return nil
}

// expectedCaptchaCause maps a stage outcome to the error the service must report.
//
// Takes outcome (string) which is the expected verification outcome.
//
// Returns error which is the expected cause, or nil for an accepted verification.
func expectedCaptchaCause(outcome string) error {
	switch outcome {
	case outcomeDecline, outcomeMismatch:
		return captcha.ErrVerificationFailed
	case outcomeUnavailable:
		return captcha.ErrProviderUnavailable
	case outcomeCancel:
		return context.Canceled
	default:
		return nil
	}
}

// captchaResponseEqual compares public CAPTCHA verification fields.
//
// Takes got (*captcha.VerifyResponse) which holds the observed result.
// Takes want (*captcha.VerifyResponse) which holds the independently expected result.
//
// Returns bool which reports whether all public verification fields agree.
func captchaResponseEqual(got, want *captcha.VerifyResponse) bool {
	return got != nil && got.Score != nil && want.Score != nil && *got.Score == *want.Score &&
		got.Success == want.Success && got.Action == want.Action && got.Hostname == want.Hostname &&
		got.Timestamp.Equal(want.Timestamp) && slices.Equal(got.ErrorCodes, want.ErrorCodes)
}

// exerciseLocalCaptchaAdapter checks the built-in provider's local verification
// lifecycle.
//
// Takes service (captcha.ServicePort) which is the production service under test.
// Takes at (time.Time) which is the independently fixed verification time.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func exerciseLocalCaptchaAdapter(ctx context.Context, service captcha.ServicePort, at time.Time) error {
	localClock := clock.NewMockClock(at)
	local, err := hmacchallenge.NewProvider(hmacchallenge.Config{Clock: localClock, Secret: []byte("ordinary-local-provider-fixture"), TTL: localChallengeTTL})
	if err != nil {
		return fmt.Errorf("construct local provider: %w", err)
	}
	const name = "local"
	if err := service.RegisterProvider(ctx, name, local); err != nil {
		return fmt.Errorf("register local provider: %w", err)
	}
	if err := service.SetDefaultProvider(name); err != nil {
		return fmt.Errorf("select local provider: %w", err)
	}
	defaultProvider, err := service.GetDefaultProvider(ctx)
	if err != nil {
		return fmt.Errorf("read default provider: %w", err)
	}
	requirements := defaultProvider.RenderRequirements()
	challenger, ok := defaultProvider.(interface{ ChallengeHandler() http.Handler })
	if defaultProvider.Type() != captcha.ProviderTypeHMACChallenge || requirements == nil || !requirements.ServerSideToken ||
		len(requirements.ScriptURLs) != 0 || service.ScriptURL() != "" || service.SiteKey() != "hmac-challenge" ||
		!ok || challenger.ChallengeHandler() == nil {
		return captchaDifference("local-adapter", "unexpected built-in renderer or callback registration")
	}
	if err := service.HealthCheck(ctx); err != nil {
		return captchaFailure("local-self-check", err)
	}
	issuer, ok := defaultProvider.(localChallengeIssuer)
	if !ok {
		return captchaDifference("local-adapter", "built-in provider cannot issue challenges")
	}
	return exerciseLocalChallengeTokens(ctx, service, issuer, localClock)
}

// exerciseLocalChallengeTokens checks that the built-in provider accepts a fresh
// challenge once and refuses replayed, forged, mismatched and expired challenges.
//
// Takes service (captcha.ServicePort) which routes to the built-in provider.
// Takes issuer (localChallengeIssuer) which issues challenges.
// Takes localClock (*clock.MockClock) which drives challenge expiry.
//
// Returns error which reports an accepted invalid challenge or a refused valid one.
func exerciseLocalChallengeTokens(ctx context.Context, service captcha.ServicePort, issuer localChallengeIssuer, localClock *clock.MockClock) error {
	issue := func() (string, error) {
		token, err := issuer.GenerateChallenge(commentAction)
		if err != nil {
			return "", captchaFailure("local-issue", err)
		}
		return token, nil
	}
	verify := func(token, address, action string) (*captcha.VerifyResponse, error) {
		return service.VerifyWithScore(ctx, token, address, action, minimumScore)
	}
	refused := func(check, token, address, action string) error {
		if _, err := verify(token, address, action); !errors.Is(err, captcha.ErrVerificationFailed) {
			return captchaDifference("local-"+check, "got %v, want verification failure", err)
		}
		return nil
	}

	token, err := issue()
	if err != nil {
		return err
	}
	got, err := verify(token, "192.0.2.21", commentAction)
	if err != nil || got == nil || !got.Success || got.Score == nil || *got.Score != 1 {
		return captchaDifference("local-accept", "fresh challenge refused: %v", err)
	}
	if err := refused("replay", token, "192.0.2.22", commentAction); err != nil {
		return err
	}

	token, err = issue()
	if err != nil {
		return err
	}
	forged, err := forgeLocalChallenge(token)
	if err != nil {
		return err
	}
	if err := refused("forgery", forged, "192.0.2.23", commentAction); err != nil {
		return err
	}
	if err := refused("action", token, "192.0.2.24", "checkout"); err != nil {
		return err
	}

	token, err = issue()
	if err != nil {
		return err
	}
	localClock.Advance(localChallengeTTL + time.Second)
	return refused("expiry", token, "192.0.2.25", commentAction)
}

// forgeLocalChallenge alters the last signature byte of a local challenge while keeping
// its encoding valid.
//
// Takes token (string) which is a genuine challenge.
//
// Returns string which is the forged challenge.
// Returns error when the challenge is not valid base64.
func forgeLocalChallenge(token string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", captchaDifference("local-forgery", "challenge is not base64: %w", err)
	}
	if len(raw) == 0 {
		return "", captchaDifference("local-forgery", "challenge is empty")
	}
	last := len(raw) - 1
	if raw[last] == 'A' {
		raw[last] = 'B'
	} else {
		raw[last] = 'A'
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
