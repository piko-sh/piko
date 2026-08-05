package actions

import (
	"context"
	"testcase_120_action_input_type/actions/user"

	pikobinder "piko.sh/piko/wdk/binder"
	"piko.sh/piko/wdk/logger"
)

var log = logger.GetLogger("piko/actions")

func invokeUserCreate(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, log)
	a := action.(*user.CreateAction)
	var input user.CreateInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	return a.Call(input)
}
