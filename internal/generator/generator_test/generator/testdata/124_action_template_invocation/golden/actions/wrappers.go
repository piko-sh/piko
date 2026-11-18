package actions

import (
	"context"
	contact_c540eef8 "testcase_124_action_template_invocation/actions/contact"

	pikobinder "piko.sh/piko/wdk/binder"
)

func invokeContactSend(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input contact_c540eef8.SendInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*contact_c540eef8.SendAction)
	return a.Call(input)
}
