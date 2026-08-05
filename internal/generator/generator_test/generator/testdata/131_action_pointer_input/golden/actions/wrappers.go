package actions

import (
	"context"
	"testcase_131_action_pointer_input/actions/data"

	pikobinder "piko.sh/piko/wdk/binder"
	"piko.sh/piko/wdk/logger"
)

var log = logger.GetLogger("piko/actions")

func invokeDataProcess(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, log)
	a := action.(*data.ProcessAction)
	var input data.ProcessInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	return a.Call(&input)
}
