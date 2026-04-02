package actions

import (
	"context"
	"testcase_127_action_bad_signature_error/actions/broken"

	"piko.sh/piko"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"broken.BadSignature": {Name: "broken.BadSignature", Method: "POST", Create: func() any {
		return &broken.BadSignatureAction{}
	}, Invoke: invokeBrokenBadSignature, HasSSE: false}})
}

type ActionHandler struct {
	Name   string
	Method string
	Create func() any
	Invoke func(ctx context.Context, action any, args map[string]any) (any, error)
	HasSSE bool
}

func Registry() map[string]ActionHandler {
	return map[string]ActionHandler{"broken.BadSignature": {Name: "broken.BadSignature", Method: "POST", Create: func() any {
		return &broken.BadSignatureAction{}
	}, Invoke: invokeBrokenBadSignature, HasSSE: false}}
}
