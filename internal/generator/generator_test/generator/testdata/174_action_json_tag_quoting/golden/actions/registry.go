package actions

import (
	"context"
	"reflect"
	record_dc37792e "testcase_174_action_json_tag_quoting/actions/record"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"record.Save": {Name: "record.Save", Method: "POST", Create: func() any {
		return &record_dc37792e.SaveAction{}
	}, Invoke: invokeRecordSave, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[record_dc37792e.SaveInput](), reflect.TypeFor[record_dc37792e.SaveOutput]()}
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
	return map[string]ActionHandler{"record.Save": {Name: "record.Save", Method: "POST", Create: func() any {
		return &record_dc37792e.SaveAction{}
	}, Invoke: invokeRecordSave, HasSSE: false}}
}
