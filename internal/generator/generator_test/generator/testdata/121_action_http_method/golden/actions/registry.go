package actions

import (
	"context"
	"reflect"
	"testcase_121_action_http_method/actions/resource"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"resource.Fetch": {Name: "resource.Fetch", Method: "POST", Create: func() any {
		return &resource.FetchAction{}
	}, Invoke: invokeResourceFetch, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[resource.FetchInput](), reflect.TypeFor[resource.FetchOutput]()}
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
	return map[string]ActionHandler{"resource.Fetch": {Name: "resource.Fetch", Method: "POST", Create: func() any {
		return &resource.FetchAction{}
	}, Invoke: invokeResourceFetch, HasSSE: false}}
}
