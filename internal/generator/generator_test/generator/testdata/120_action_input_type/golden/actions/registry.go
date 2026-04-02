package actions

import (
	"context"
	"reflect"
	"testcase_120_action_input_type/actions/user"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"user.Create": {Name: "user.Create", Method: "POST", Create: func() any {
		return &user.CreateAction{}
	}, Invoke: invokeUserCreate, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[user.CreateInput](), reflect.TypeFor[user.CreateOutput]()}
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
	return map[string]ActionHandler{"user.Create": {Name: "user.Create", Method: "POST", Create: func() any {
		return &user.CreateAction{}
	}, Invoke: invokeUserCreate, HasSSE: false}}
}
