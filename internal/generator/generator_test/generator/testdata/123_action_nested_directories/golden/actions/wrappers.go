package actions

import (
	"context"
	"testcase_123_action_nested_directories/actions/admin/users"

	pikobinder "piko.sh/piko/wdk/binder"
	"piko.sh/piko/wdk/logger"
)

var log = logger.GetLogger("piko/actions")

func invokeUsersDelete(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, log)
	a := action.(*users.DeleteAction)
	var input users.DeleteInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	return a.Call(input)
}
func invokeUsersUpdate(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, log)
	a := action.(*users.UpdateAction)
	var input users.UpdateInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	return a.Call(input)
}
