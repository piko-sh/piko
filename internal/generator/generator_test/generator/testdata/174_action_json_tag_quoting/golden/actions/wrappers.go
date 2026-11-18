package actions

import (
	"context"
	record_dc37792e "testcase_174_action_json_tag_quoting/actions/record"

	pikobinder "piko.sh/piko/wdk/binder"
)

func invokeRecordSave(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input record_dc37792e.SaveInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*record_dc37792e.SaveAction)
	return a.Call(input)
}
