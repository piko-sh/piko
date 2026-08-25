package actions

import (
	"context"
	"reflect"
	order_e0134290 "testcase_176_action_type_only_package/actions/order"
	report_e820dae8 "testcase_176_action_type_only_package/actions/report"
	contracts_b257f8b0 "testcase_176_action_type_only_package/pkg/contracts"
	results_b0cc1a7b "testcase_176_action_type_only_package/pkg/results"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"order.Create": {Name: "order.Create", Method: "POST", Create: func() any {
		return &order_e0134290.CreateAction{}
	}, Invoke: invokeOrderCreate, HasSSE: false}, "report.Fetch": {Name: "report.Fetch", Method: "POST", Create: func() any {
		return &report_e820dae8.FetchAction{}
	}, Invoke: invokeReportFetch, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[order_e0134290.CreateOutput](), reflect.TypeFor[contracts_b257f8b0.CreateInput](), reflect.TypeFor[results_b0cc1a7b.FetchOutput]()}
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
	return map[string]ActionHandler{"order.Create": {Name: "order.Create", Method: "POST", Create: func() any {
		return &order_e0134290.CreateAction{}
	}, Invoke: invokeOrderCreate, HasSSE: false}, "report.Fetch": {Name: "report.Fetch", Method: "POST", Create: func() any {
		return &report_e820dae8.FetchAction{}
	}, Invoke: invokeReportFetch, HasSSE: false}}
}
