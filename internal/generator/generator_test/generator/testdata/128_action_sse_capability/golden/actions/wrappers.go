package actions

import (
	"context"
	stream_c862ba1d "testcase_128_action_sse_capability/actions/stream"

	"piko.sh/piko"
	pikobinder "piko.sh/piko/wdk/binder"
)

func invokeStreamEvents(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input stream_c862ba1d.EventsInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*stream_c862ba1d.EventsAction)
	return a.Call(input)
}
func bindStreamEvents(ctx context.Context, action any, argsMap map[string]any) error {
	var input stream_c862ba1d.EventsInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return err
	}
	piko.SetActionInput(action, input)
	return nil
}
