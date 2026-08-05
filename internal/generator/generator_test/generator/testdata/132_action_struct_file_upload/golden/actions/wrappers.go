package actions

import (
	"context"
	"mime/multipart"
	"testcase_132_action_struct_file_upload/actions/files"

	"piko.sh/piko"
	pikobinder "piko.sh/piko/wdk/binder"
	"piko.sh/piko/wdk/logger"
)

var log = logger.GetLogger("piko/actions")

func invokeFilesUpload(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, log)
	a := action.(*files.UploadAction)
	var input files.UploadInput
	if fh, ok := argsMap["avatar"].(*multipart.FileHeader); ok {
		input.Avatar = piko.NewFileUpload(fh)
	}
	delete(argsMap, "avatar")
	if err := pikobinder.BindMap(ctx, &input, argsMap, pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter from flat argsMap", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	return a.Call(input)
}
