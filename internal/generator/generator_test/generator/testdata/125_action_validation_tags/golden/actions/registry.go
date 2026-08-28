package actions

import (
	"context"
	"reflect"
	user_83ddb1f3 "testcase_125_action_validation_tags/actions/user"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"user.Register": {Name: "user.Register", Method: "POST", Create: func() any {
		return &user_83ddb1f3.RegisterAction{}
	}, Invoke: invokeUserRegister, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[user_83ddb1f3.RegisterInput](), reflect.TypeFor[user_83ddb1f3.RegisterOutput]()}
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
	return map[string]ActionHandler{"user.Register": {Name: "user.Register", Method: "POST", Create: func() any {
		return &user_83ddb1f3.RegisterAction{}
	}, Invoke: invokeUserRegister, HasSSE: false}}
}
