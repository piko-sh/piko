package actions

import (
	"context"
	"reflect"
	email_317401ce "testcase_119_action_basic/actions/email"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"email.Contact": {Name: "email.Contact", Method: "POST", Create: func() any {
		return &email_317401ce.ContactAction{}
	}, Invoke: invokeEmailContact, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[email_317401ce.ContactInput](), reflect.TypeFor[email_317401ce.ContactOutput]()}
	for _, t := range pretouchTypes {
		_ = pikojson.Pretouch(t)
	}
}

type ActionHandler struct {
	Name        string
	Method      string
	Create      func() any
	Invoke      func(ctx context.Context, action any, args map[string]any) (any, error)
	Bind        func(ctx context.Context, action any, args map[string]any) error
	HasSSE      bool
	SSEGetAlias bool
}

func Registry() map[string]ActionHandler {
	return map[string]ActionHandler{"email.Contact": {Name: "email.Contact", Method: "POST", Create: func() any {
		return &email_317401ce.ContactAction{}
	}, Invoke: invokeEmailContact, HasSSE: false}}
}
