package actions

import (
	"context"
	"reflect"
	"testcase_128_action_sse_capability/actions/stream"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"stream.Events": {Name: "stream.Events", Method: "POST", Create: func() any {
		return &stream.EventsAction{}
	}, Invoke: invokeStreamEvents, HasSSE: true}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[stream.EventsInput](), reflect.TypeFor[stream.EventsOutput]()}
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
	return map[string]ActionHandler{"stream.Events": {Name: "stream.Events", Method: "POST", Create: func() any {
		return &stream.EventsAction{}
	}, Invoke: invokeStreamEvents, HasSSE: true}}
}
