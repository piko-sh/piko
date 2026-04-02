package actions

import (
	"context"
	"reflect"
	"testcase_119_action_basic/actions/email"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"email.Contact": {Name: "email.Contact", Method: "POST", Create: func() any {
		return &email.ContactAction{}
	}, Invoke: invokeEmailContact, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[email.ContactInput](), reflect.TypeFor[email.ContactOutput]()}
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
	}, Invoke: invokeEmailContact, HasSSE: false}}
}
