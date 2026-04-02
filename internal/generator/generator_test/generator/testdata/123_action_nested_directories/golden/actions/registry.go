package actions

import (
	"context"
	"reflect"
	"testcase_123_action_nested_directories/actions/admin/users"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"users.Delete": {Name: "users.Delete", Method: "POST", Create: func() any {
		return &users.DeleteAction{}
	}, Invoke: invokeUsersDelete, HasSSE: false}, "users.Update": {Name: "users.Update", Method: "POST", Create: func() any {
		return &users.UpdateAction{}
	}, Invoke: invokeUsersUpdate, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[users.DeleteInput](), reflect.TypeFor[users.DeleteOutput](), reflect.TypeFor[users.UpdateInput](), reflect.TypeFor[users.UpdateOutput]()}
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
	return map[string]ActionHandler{"users.Delete": {Name: "users.Delete", Method: "POST", Create: func() any {
		return &users.DeleteAction{}
	}, Invoke: invokeUsersDelete, HasSSE: false}, "users.Update": {Name: "users.Update", Method: "POST", Create: func() any {
		return &users.UpdateAction{}
	}, Invoke: invokeUsersUpdate, HasSSE: false}}
}
