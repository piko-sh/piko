package actions

import (
	"context"
	"reflect"
	users_56a457b7 "testcase_123_action_nested_directories/actions/admin/users"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"users.Delete": {Name: "users.Delete", Method: "POST", Create: func() any {
		return &users_56a457b7.DeleteAction{}
	}, Invoke: invokeUsersDelete, HasSSE: false}, "users.Update": {Name: "users.Update", Method: "POST", Create: func() any {
		return &users_56a457b7.UpdateAction{}
	}, Invoke: invokeUsersUpdate, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[users_56a457b7.DeleteInput](), reflect.TypeFor[users_56a457b7.DeleteOutput](), reflect.TypeFor[users_56a457b7.UpdateInput](), reflect.TypeFor[users_56a457b7.UpdateOutput]()}
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
	return map[string]ActionHandler{"users.Delete": {Name: "users.Delete", Method: "POST", Create: func() any {
		return &users_56a457b7.DeleteAction{}
	}, Invoke: invokeUsersDelete, HasSSE: false}, "users.Update": {Name: "users.Update", Method: "POST", Create: func() any {
		return &users_56a457b7.UpdateAction{}
	}, Invoke: invokeUsersUpdate, HasSSE: false}}
}
