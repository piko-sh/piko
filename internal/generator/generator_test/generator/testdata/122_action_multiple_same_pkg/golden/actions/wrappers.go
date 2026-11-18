package actions

import (
	"context"
	email_57f46b27 "testcase_122_action_multiple_same_pkg/actions/email"

	pikobinder "piko.sh/piko/wdk/binder"
)

func invokeEmailContact(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input email_57f46b27.ContactInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*email_57f46b27.ContactAction)
	return a.Call(input)
}
func invokeEmailSubscribe(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input email_57f46b27.SubscribeInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*email_57f46b27.SubscribeAction)
	return a.Call(input)
}
