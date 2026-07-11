package actions

import (
	"context"
	"reflect"
	"testcase_132_action_struct_file_upload/actions/files"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"files.Upload": {Name: "files.Upload", Method: "POST", Create: func() any {
		return &files.UploadAction{}
	}, Invoke: invokeFilesUpload, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[files.UploadInput](), reflect.TypeFor[files.UploadOutput]()}
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
	return map[string]ActionHandler{"files.Upload": {Name: "files.Upload", Method: "POST", Create: func() any {
		return &files.UploadAction{}
	}, Invoke: invokeFilesUpload, HasSSE: false}}
}
