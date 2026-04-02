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
	if raw, ok := argsMap["input"]; ok {
		if rawMap, ok := raw.(map[string]any); ok {
			if err := pikobinder.BindMap(ctx, &input, rawMap, pikobinder.IgnoreUnknownKeys(true)); err != nil {
				l.Error("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
				return nil, err
			}
		}
	} else if len(argsMap) > 0 {
		if err := pikobinder.BindMap(ctx, &input, argsMap, pikobinder.IgnoreUnknownKeys(true)); err != nil {
			l.Error("Failed to bind action parameter from flat argsMap", logger.String("param", "input"), logger.Error(err))
			return nil, err
		}
	}
	return a.Call(&input)
}
