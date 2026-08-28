package actions

import (
	"context"
	stream_f7037a81 "testcase_178_action_sse_get_alias/actions/stream"

	"piko.sh/piko"
	pikobinder "piko.sh/piko/wdk/binder"
	"piko.sh/piko/wdk/logger"
)

var pikoActionLog = logger.GetLogger("piko/actions")

func invokeStreamEvents(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, pikoActionLog)
	var input stream_f7037a81.EventsInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	a := action.(*stream_f7037a81.EventsAction)
	return a.Call(input)
}
func bindStreamEvents(ctx context.Context, action any, argsMap map[string]any) error {
	ctx, l := logger.From(ctx, pikoActionLog)
	var input stream_f7037a81.EventsInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return err
	}
	piko.SetActionInput(action, input)
	return nil
}
