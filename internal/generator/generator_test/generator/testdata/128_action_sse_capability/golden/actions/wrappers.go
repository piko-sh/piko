package actions

import (
	"context"
	stream_c862ba1d "testcase_128_action_sse_capability/actions/stream"

	"piko.sh/piko"
	pikobinder "piko.sh/piko/wdk/binder"
	"piko.sh/piko/wdk/logger"
)

var pikoActionLog = logger.GetLogger("piko/actions")

func invokeStreamEvents(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, pikoActionLog)
	var input stream_c862ba1d.EventsInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	a := action.(*stream_c862ba1d.EventsAction)
	return a.Call(input)
}
func bindStreamEvents(ctx context.Context, action any, argsMap map[string]any) error {
	ctx, l := logger.From(ctx, pikoActionLog)
	var input stream_c862ba1d.EventsInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return err
	}
	piko.SetActionInput(action, input)
	return nil
}
