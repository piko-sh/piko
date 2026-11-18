package actions

import (
	"context"
	user_84be158b "testcase_120_action_input_type/actions/user"

	pikobinder "piko.sh/piko/wdk/binder"
)

func invokeUserCreate(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input user_84be158b.CreateInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*user_84be158b.CreateAction)
	return a.Call(input)
}
