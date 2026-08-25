package actions

import (
	"context"
	"mime/multipart"
	context_6a771c37 "testcase_156_action_reserved_package_names/actions/context"
	log_107e3be5 "testcase_156_action_reserved_package_names/actions/log"
	multipart_e28a457e "testcase_156_action_reserved_package_names/actions/multipart"
	pikobinder_7d00b3bf "testcase_156_action_reserved_package_names/actions/pikobinder"
	reflect_5a35ffe5 "testcase_156_action_reserved_package_names/actions/reflect"

	"piko.sh/piko"
	pikobinder "piko.sh/piko/wdk/binder"
	"piko.sh/piko/wdk/logger"
)

var pikoActionLog = logger.GetLogger("piko/actions")

func invokeContextRead(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	key, _ := argsMap["key"].(string)
	a := action.(*context_6a771c37.ReadAction)
	return a.Call(key)
}
func invokeLogWrite(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, pikoActionLog)
	var input log_107e3be5.WriteInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	a := action.(*log_107e3be5.WriteAction)
	return a.Call(input)
}
func invokeMultipartSend(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var attachment piko.FileUpload
	if fh, ok := argsMap["attachment"].(*multipart.FileHeader); ok {
		attachment = piko.NewFileUpload(fh)
	}
	a := action.(*multipart_e28a457e.SendAction)
	return a.Call(attachment)
}
func invokePikobinderBind(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, pikoActionLog)
	var input pikobinder_7d00b3bf.BindInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	a := action.(*pikobinder_7d00b3bf.BindAction)
	return a.Call(input)
}
func invokeReflectScan(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, pikoActionLog)
	var input reflect_5a35ffe5.ScanInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	a := action.(*reflect_5a35ffe5.ScanAction)
	return a.Call(input)
}
