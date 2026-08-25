package actions

import (
	"context"
	"reflect"
	contact_c540eef8 "testcase_124_action_template_invocation/actions/contact"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"contact.Send": {Name: "contact.Send", Method: "POST", Create: func() any {
		return &contact_c540eef8.SendAction{}
	}, Invoke: invokeContactSend, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[contact_c540eef8.SendInput](), reflect.TypeFor[contact_c540eef8.SendOutput]()}
	for _, t := range pretouchTypes {
		_ = pikojson.Pretouch(t)
	}
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
	return map[string]ActionHandler{"contact.Send": {Name: "contact.Send", Method: "POST", Create: func() any {
		return &contact_c540eef8.SendAction{}
	}, Invoke: invokeContactSend, HasSSE: false}}
}
