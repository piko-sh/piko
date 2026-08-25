package actions

import (
	"context"
	"reflect"
	email_57f46b27 "testcase_122_action_multiple_same_pkg/actions/email"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"email.Contact": {Name: "email.Contact", Method: "POST", Create: func() any {
		return &email_57f46b27.ContactAction{}
	}, Invoke: invokeEmailContact, HasSSE: false}, "email.Subscribe": {Name: "email.Subscribe", Method: "POST", Create: func() any {
		return &email_57f46b27.SubscribeAction{}
	}, Invoke: invokeEmailSubscribe, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[email_57f46b27.ContactInput](), reflect.TypeFor[email_57f46b27.ContactOutput](), reflect.TypeFor[email_57f46b27.SubscribeInput](), reflect.TypeFor[email_57f46b27.SubscribeOutput]()}
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
	return map[string]ActionHandler{"email.Contact": {Name: "email.Contact", Method: "POST", Create: func() any {
		return &email_57f46b27.ContactAction{}
	}, Invoke: invokeEmailContact, HasSSE: false}, "email.Subscribe": {Name: "email.Subscribe", Method: "POST", Create: func() any {
		return &email_57f46b27.SubscribeAction{}
	}, Invoke: invokeEmailSubscribe, HasSSE: false}}
}
