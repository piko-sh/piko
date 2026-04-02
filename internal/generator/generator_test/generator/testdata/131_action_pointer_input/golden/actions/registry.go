package actions

import (
	"context"
	"reflect"
	"testcase_131_action_pointer_input/actions/data"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"data.Process": {Name: "data.Process", Method: "POST", Create: func() any {
		return &data.ProcessAction{}
	}, Invoke: invokeDataProcess, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[data.ProcessInput](), reflect.TypeFor[data.ProcessOutput]()}
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
	return map[string]ActionHandler{"data.Process": {Name: "data.Process", Method: "POST", Create: func() any {
		return &data.ProcessAction{}
	}, Invoke: invokeDataProcess, HasSSE: false}}
}
