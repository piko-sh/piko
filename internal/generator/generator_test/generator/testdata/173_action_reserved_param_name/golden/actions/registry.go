package actions

import (
	"context"
	"reflect"
	echo_9e9b592e "testcase_173_action_reserved_param_name/actions/echo"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"echo.Run": {Name: "echo.Run", Method: "POST", Create: func() any {
		return &echo_9e9b592e.RunAction{}
	}, Invoke: invokeEchoRun, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[echo_9e9b592e.RunInput](), reflect.TypeFor[echo_9e9b592e.RunOutput]()}
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
	return map[string]ActionHandler{"echo.Run": {Name: "echo.Run", Method: "POST", Create: func() any {
		return &echo_9e9b592e.RunAction{}
	}, Invoke: invokeEchoRun, HasSSE: false}}
}
