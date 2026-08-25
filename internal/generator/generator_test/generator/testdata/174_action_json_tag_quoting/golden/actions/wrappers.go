package actions

import (
	"context"
	record_dc37792e "testcase_174_action_json_tag_quoting/actions/record"

	pikobinder "piko.sh/piko/wdk/binder"
	"piko.sh/piko/wdk/logger"
)

var pikoActionLog = logger.GetLogger("piko/actions")

func invokeRecordSave(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, pikoActionLog)
	var input record_dc37792e.SaveInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	a := action.(*record_dc37792e.SaveAction)
	return a.Call(input)
}
