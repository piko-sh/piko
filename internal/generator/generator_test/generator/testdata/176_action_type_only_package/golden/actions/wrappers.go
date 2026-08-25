package actions

import (
	"context"
	order_e0134290 "testcase_176_action_type_only_package/actions/order"
	report_e820dae8 "testcase_176_action_type_only_package/actions/report"
	contracts_b257f8b0 "testcase_176_action_type_only_package/pkg/contracts"

	pikobinder "piko.sh/piko/wdk/binder"
	"piko.sh/piko/wdk/logger"
)

var pikoActionLog = logger.GetLogger("piko/actions")

func invokeOrderCreate(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, pikoActionLog)
	var input contracts_b257f8b0.CreateInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	a := action.(*order_e0134290.CreateAction)
	return a.Call(input)
}
func invokeReportFetch(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	a := action.(*report_e820dae8.FetchAction)
	return a.Call()
}
