package actions

import (
	"context"
	broken_e68d7c38 "testcase_127_action_bad_signature_error/actions/broken"

	"piko.sh/piko"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"broken.BadSignature": {Name: "broken.BadSignature", Method: "POST", Create: func() any {
		return &broken_e68d7c38.BadSignatureAction{}
	}, Invoke: invokeBrokenBadSignature, HasSSE: false}})
}

type ActionHandler struct {
	Name   string
	Method string
	Create func() any
	Invoke func(ctx context.Context, action any, args map[string]any) (any, error)
	Bind   func(ctx context.Context, action any, args map[string]any) error
	HasSSE bool
}

func Registry() map[string]ActionHandler {
	return map[string]ActionHandler{"broken.BadSignature": {Name: "broken.BadSignature", Method: "POST", Create: func() any {
		return &broken_e68d7c38.BadSignatureAction{}
	}, Invoke: invokeBrokenBadSignature, HasSSE: false}}
}
