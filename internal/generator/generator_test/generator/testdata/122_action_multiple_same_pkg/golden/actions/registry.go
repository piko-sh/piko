package actions

import (
	"context"
	"reflect"
	"testcase_122_action_multiple_same_pkg/actions/email"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"email.Contact": {Name: "email.Contact", Method: "POST", Create: func() any {
		return &email.ContactAction{}
	}, Invoke: invokeEmailContact, HasSSE: false}, "email.Subscribe": {Name: "email.Subscribe", Method: "POST", Create: func() any {
		return &email.SubscribeAction{}
	}, Invoke: invokeEmailSubscribe, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[email.ContactInput](), reflect.TypeFor[email.ContactOutput](), reflect.TypeFor[email.SubscribeInput](), reflect.TypeFor[email.SubscribeOutput]()}
	for _, t := range pretouchTypes {
		_ = pikojson.Pretouch(t)
	}
}

type ActionHandler struct {
	Name   string
	Method string
	Create func() any
	Invoke func(ctx context.Context, action any, args map[string]any) (any, error)
	HasSSE bool
}

func Registry() map[string]ActionHandler {
	return map[string]ActionHandler{"email.Contact": {Name: "email.Contact", Method: "POST", Create: func() any {
		return &email.ContactAction{}
	}, Invoke: invokeEmailContact, HasSSE: false}, "email.Subscribe": {Name: "email.Subscribe", Method: "POST", Create: func() any {
		return &email.SubscribeAction{}
	}, Invoke: invokeEmailSubscribe, HasSSE: false}}
}
