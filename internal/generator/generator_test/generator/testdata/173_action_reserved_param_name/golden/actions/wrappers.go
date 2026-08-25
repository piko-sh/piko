package actions

import (
	"context"
	echo_9e9b592e "testcase_173_action_reserved_param_name/actions/echo"

	pikobinder "piko.sh/piko/wdk/binder"
	"piko.sh/piko/wdk/logger"
)

var pikoActionLog = logger.GetLogger("piko/actions")

func invokeEchoRun(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, pikoActionLog)
	var in echo_9e9b592e.RunInput
	if err := pikobinder.BindMap(ctx, &in, pikobinder.ActionInputSource(argsMap, "in"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "in"), logger.Error(err))
		return nil, err
	}
	a := action.(*echo_9e9b592e.RunAction)
	return a.Call(in)
}
