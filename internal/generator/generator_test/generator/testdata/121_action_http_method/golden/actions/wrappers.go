package actions

import (
	"context"
	resource_13254f2c "testcase_121_action_http_method/actions/resource"

	pikobinder "piko.sh/piko/wdk/binder"
)

func invokeResourceFetch(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input resource_13254f2c.FetchInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*resource_13254f2c.FetchAction)
	return a.Call(input)
}
