package actions

import (
	"context"
	stream_f7037a81 "testcase_178_action_sse_get_alias/actions/stream"

	"piko.sh/piko"
	pikobinder "piko.sh/piko/wdk/binder"
)

func invokeStreamEvents(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input stream_f7037a81.EventsInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*stream_f7037a81.EventsAction)
	return a.Call(input)
}
func bindStreamEvents(ctx context.Context, action any, argsMap map[string]any) error {
	var input stream_f7037a81.EventsInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return err
	}
	piko.SetActionInput(action, input)
	return nil
}
