package actions

import (
	"context"
	"reflect"
	stream_f7037a81 "testcase_178_action_sse_get_alias/actions/stream"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"stream.Events": {Name: "stream.Events", Method: "POST", Create: func() any {
		return &stream_f7037a81.EventsAction{}
	}, Invoke: invokeStreamEvents, Bind: bindStreamEvents, HasSSE: true, SSEGetAlias: true}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[stream_f7037a81.EventsInput](), reflect.TypeFor[stream_f7037a81.EventsOutput]()}
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
	return map[string]ActionHandler{"stream.Events": {Name: "stream.Events", Method: "POST", Create: func() any {
		return &stream_f7037a81.EventsAction{}
	}, Invoke: invokeStreamEvents, Bind: bindStreamEvents, HasSSE: true, SSEGetAlias: true}}
}
