package actions

import (
	"context"
	users_56a457b7 "testcase_123_action_nested_directories/actions/admin/users"

	pikobinder "piko.sh/piko/wdk/binder"
)

func invokeUsersDelete(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input users_56a457b7.DeleteInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*users_56a457b7.DeleteAction)
	return a.Call(input)
}
func invokeUsersUpdate(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input users_56a457b7.UpdateInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*users_56a457b7.UpdateAction)
	return a.Call(input)
}
