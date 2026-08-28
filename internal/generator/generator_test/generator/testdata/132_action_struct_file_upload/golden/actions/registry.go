package actions

import (
	"context"
	"reflect"
	files_203cc8ad "testcase_132_action_struct_file_upload/actions/files"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"files.Upload": {Name: "files.Upload", Method: "POST", Create: func() any {
		return &files_203cc8ad.UploadAction{}
	}, Invoke: invokeFilesUpload, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[files_203cc8ad.UploadInput](), reflect.TypeFor[files_203cc8ad.UploadOutput]()}
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
	return map[string]ActionHandler{"files.Upload": {Name: "files.Upload", Method: "POST", Create: func() any {
		return &files_203cc8ad.UploadAction{}
	}, Invoke: invokeFilesUpload, HasSSE: false}}
}
