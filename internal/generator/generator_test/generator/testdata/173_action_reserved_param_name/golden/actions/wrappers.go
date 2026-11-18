package actions

import (
	"context"
	echo_9e9b592e "testcase_173_action_reserved_param_name/actions/echo"

	pikobinder "piko.sh/piko/wdk/binder"
)

func invokeEchoRun(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var in echo_9e9b592e.RunInput
	if err := pikobinder.BindMap(ctx, &in, pikobinder.ActionInputSource(argsMap, "in"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*echo_9e9b592e.RunAction)
	return a.Call(in)
}
