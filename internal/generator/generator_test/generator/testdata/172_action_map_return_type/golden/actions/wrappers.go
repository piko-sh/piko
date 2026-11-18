package actions

import (
	"context"
	report_e21cbd2e "testcase_172_action_map_return_type/actions/report"

	pikobinder "piko.sh/piko/wdk/binder"
)

func invokeReportFetch(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input report_e21cbd2e.FetchInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*report_e21cbd2e.FetchAction)
	return a.Call(input)
}
