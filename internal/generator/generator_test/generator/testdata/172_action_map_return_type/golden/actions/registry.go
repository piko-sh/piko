package actions

import (
	"context"
	"reflect"
	report_e21cbd2e "testcase_172_action_map_return_type/actions/report"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"report.Fetch": {Name: "report.Fetch", Method: "POST", Create: func() any {
		return &report_e21cbd2e.FetchAction{}
	}, Invoke: invokeReportFetch, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[report_e21cbd2e.FetchInput]()}
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
	return map[string]ActionHandler{"report.Fetch": {Name: "report.Fetch", Method: "POST", Create: func() any {
		return &report_e21cbd2e.FetchAction{}
	}, Invoke: invokeReportFetch, HasSSE: false}}
}
