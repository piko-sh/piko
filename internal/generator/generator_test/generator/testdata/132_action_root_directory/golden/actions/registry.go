package actions

import (
	"context"
	"reflect"
	"testcase_132_action_root_directory/actions"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"actions.Ping": {Name: "actions.Ping", Method: "POST", Create: func() any {
		return &actions.PingAction{}
	}, Invoke: invokeActionsPing, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[actions.PingOutput]()}
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
	return map[string]ActionHandler{"actions.Ping": {Name: "actions.Ping", Method: "POST", Create: func() any {
		return &actions.PingAction{}
	}, Invoke: invokeActionsPing, HasSSE: false}}
}
