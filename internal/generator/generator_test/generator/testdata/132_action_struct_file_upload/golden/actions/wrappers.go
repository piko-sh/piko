package actions

import (
	"context"
	"mime/multipart"
	files_203cc8ad "testcase_132_action_struct_file_upload/actions/files"

	"piko.sh/piko"
	pikobinder "piko.sh/piko/wdk/binder"
)

func invokeFilesUpload(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input files_203cc8ad.UploadInput
	if fh, ok := argsMap["avatar"].(*multipart.FileHeader); ok {
		input.Avatar = piko.NewFileUpload(fh)
	}
	delete(argsMap, "avatar")
	if err := pikobinder.BindMap(ctx, &input, argsMap, pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*files_203cc8ad.UploadAction)
	return a.Call(input)
}
