package intake

import (
	"example.org/pikofuzzcase/internal/formfixture"
	"math"
	"piko.sh/piko"
)

type SubmitAction struct{ piko.ActionMetadata }
type SubmitInput struct {
	Label string `json:"label"`
}
type SubmitOutput struct {
	Label        string `json:"label"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	Revision     int    `json:"revision"`
	ScorePercent int    `json:"scorePercent"`
}

func (*SubmitAction) CaptchaConfig() *piko.CaptchaConfig {
	return &piko.CaptchaConfig{Provider: "fixture", Action: formfixture.ActionName, ScoreThreshold: 0.5}
}
func (*SubmitAction) RateLimit() *piko.RateLimit {
	return &piko.RateLimit{KeyFunc: piko.RateLimitByIP, RequestsPerMinute: formfixture.Quota, BurstSize: formfixture.Quota}
}
func (a *SubmitAction) Call(input SubmitInput) (SubmitOutput, error) {
	r := a.Request()
	score := -1
	if r.CaptchaScore != nil {
		score = int(math.Round(100 * *r.CaptchaScore))
	}
	return SubmitOutput{Label: input.Label, Method: r.Method, Path: r.Path, Revision: formfixture.Save(input.Label), ScorePercent: score}, nil
}

type ReadAction struct{ piko.ActionMetadata }
type ReadInput struct{}
type ReadOutput struct {
	Checks int      `json:"checks"`
	Writes int      `json:"writes"`
	Events []string `json:"events"`
}

func (*ReadAction) Call(_ ReadInput) (ReadOutput, error) {
	checks, writes, events := formfixture.Snapshot()
	return ReadOutput{Checks: checks, Writes: writes, Events: events}, nil
}
