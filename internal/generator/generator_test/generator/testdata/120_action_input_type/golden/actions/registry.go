package actions

import (
	"context"
	"reflect"
	user_84be158b "testcase_120_action_input_type/actions/user"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"user.Create": {Name: "user.Create", Method: "POST", Create: func() any {
		return &user_84be158b.CreateAction{}
	}, Invoke: invokeUserCreate, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[user_84be158b.CreateInput](), reflect.TypeFor[user_84be158b.CreateOutput]()}
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
	return map[string]ActionHandler{"user.Create": {Name: "user.Create", Method: "POST", Create: func() any {
		return &user_84be158b.CreateAction{}
	}, Invoke: invokeUserCreate, HasSSE: false}}
}
