package actions

import (
	"context"
	"testcase_122_action_multiple_same_pkg/actions/email"

	pikobinder "piko.sh/piko/wdk/binder"
	"piko.sh/piko/wdk/logger"
)

var log = logger.GetLogger("piko/actions")

func invokeEmailContact(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, log)
	a := action.(*email.ContactAction)
	var input email.ContactInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	return a.Call(input)
}
func invokeEmailSubscribe(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, log)
	a := action.(*email.SubscribeAction)
	var input email.SubscribeInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	return a.Call(input)
}
