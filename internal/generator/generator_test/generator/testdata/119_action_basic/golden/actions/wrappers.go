package actions

import (
	"context"
	email_317401ce "testcase_119_action_basic/actions/email"

	pikobinder "piko.sh/piko/wdk/binder"
)

func invokeEmailContact(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input email_317401ce.ContactInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*email_317401ce.ContactAction)
	return a.Call(input)
}
