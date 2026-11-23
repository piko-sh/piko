package formfixture

import (
	"context"
	"net"
	"sync"

	"piko.sh/piko"
	"piko.sh/piko/wdk/captcha"
	hmacchallenge "piko.sh/piko/wdk/captcha/captcha_provider_hmac_challenge"
)

const Quota = 5
const Score = 75
const ActionName = "intake"

var state struct {
	sync.Mutex
	checks, writes int
	events         []string
}

type localProvider struct{ captcha.Provider }

func (p localProvider) Verify(ctx context.Context, input *captcha.VerifyRequest) (*captcha.VerifyResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	state.Lock()
	defer state.Unlock()
	state.checks++
	host := input.RemoteIP
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	address := "other"
	if net.ParseIP(host).IsLoopback() {
		address = "loopback"
	}
	state.events = append(state.events, "verify:"+input.Token+":"+input.Action+":"+address)
	switch state.checks {
	case 2:
		return &captcha.VerifyResponse{Success: false, Action: input.Action, Score: new(float64(Score) / 100), ErrorCodes: []string{"fixture-declined"}}, nil
	case 3:
		return nil, captcha.ErrProviderUnavailable
	default:
		return &captcha.VerifyResponse{Success: true, Action: input.Action, Score: new(float64(Score) / 100)}, nil
	}
}

func Options() []piko.Option {
	adapter, err := hmacchallenge.NewProvider(hmacchallenge.Config{Secret: []byte("generated-local-form-fixture")})
	if err != nil {
		panic(err)
	}
	return []piko.Option{piko.WithCaptchaProvider("fixture", localProvider{adapter}), piko.WithDefaultCaptchaProvider("fixture"), piko.WithRateLimitEnabled(true)}
}

func Save(label string) int {
	state.Lock()
	defer state.Unlock()
	state.writes++
	state.events = append(state.events, "write:"+label)
	return state.writes
}

func Snapshot() (int, int, []string) {
	state.Lock()
	defer state.Unlock()
	return state.checks, state.writes, append([]string{}, state.events...)
}
