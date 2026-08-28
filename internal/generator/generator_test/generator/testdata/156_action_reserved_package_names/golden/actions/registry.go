package actions

import (
	"context"
	"reflect"
	context_6a771c37 "testcase_156_action_reserved_package_names/actions/context"
	log_107e3be5 "testcase_156_action_reserved_package_names/actions/log"
	multipart_e28a457e "testcase_156_action_reserved_package_names/actions/multipart"
	pikobinder_7d00b3bf "testcase_156_action_reserved_package_names/actions/pikobinder"
	reflect_5a35ffe5 "testcase_156_action_reserved_package_names/actions/reflect"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"context.Read": {Name: "context.Read", Method: "POST", Create: func() any {
		return &context_6a771c37.ReadAction{}
	}, Invoke: invokeContextRead, HasSSE: false}, "log.Write": {Name: "log.Write", Method: "POST", Create: func() any {
		return &log_107e3be5.WriteAction{}
	}, Invoke: invokeLogWrite, HasSSE: false}, "multipart.Send": {Name: "multipart.Send", Method: "POST", Create: func() any {
		return &multipart_e28a457e.SendAction{}
	}, Invoke: invokeMultipartSend, HasSSE: false}, "pikobinder.Bind": {Name: "pikobinder.Bind", Method: "POST", Create: func() any {
		return &pikobinder_7d00b3bf.BindAction{}
	}, Invoke: invokePikobinderBind, HasSSE: false}, "reflect.Scan": {Name: "reflect.Scan", Method: "POST", Create: func() any {
		return &reflect_5a35ffe5.ScanAction{}
	}, Invoke: invokeReflectScan, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[context_6a771c37.ReadOutput](), reflect.TypeFor[log_107e3be5.WriteInput](), reflect.TypeFor[log_107e3be5.WriteOutput](), reflect.TypeFor[multipart_e28a457e.SendOutput](), reflect.TypeFor[pikobinder_7d00b3bf.BindInput](), reflect.TypeFor[pikobinder_7d00b3bf.BindOutput](), reflect.TypeFor[reflect_5a35ffe5.ScanInput](), reflect.TypeFor[reflect_5a35ffe5.ScanOutput]()}
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
	return map[string]ActionHandler{"context.Read": {Name: "context.Read", Method: "POST", Create: func() any {
		return &context_6a771c37.ReadAction{}
	}, Invoke: invokeContextRead, HasSSE: false}, "log.Write": {Name: "log.Write", Method: "POST", Create: func() any {
		return &log_107e3be5.WriteAction{}
	}, Invoke: invokeLogWrite, HasSSE: false}, "multipart.Send": {Name: "multipart.Send", Method: "POST", Create: func() any {
		return &multipart_e28a457e.SendAction{}
	}, Invoke: invokeMultipartSend, HasSSE: false}, "pikobinder.Bind": {Name: "pikobinder.Bind", Method: "POST", Create: func() any {
		return &pikobinder_7d00b3bf.BindAction{}
	}, Invoke: invokePikobinderBind, HasSSE: false}, "reflect.Scan": {Name: "reflect.Scan", Method: "POST", Create: func() any {
		return &reflect_5a35ffe5.ScanAction{}
	}, Invoke: invokeReflectScan, HasSSE: false}}
}
