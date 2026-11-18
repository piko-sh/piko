package actions

import (
	"context"
	data_8063f3ae "testcase_131_action_pointer_input/actions/data"

	pikobinder "piko.sh/piko/wdk/binder"
)

func invokeDataProcess(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input data_8063f3ae.ProcessInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*data_8063f3ae.ProcessAction)
	return a.Call(&input)
}
