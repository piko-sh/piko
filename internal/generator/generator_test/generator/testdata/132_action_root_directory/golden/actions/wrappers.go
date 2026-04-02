package actions

import (
	"context"
	"testcase_132_action_root_directory/actions"

	"piko.sh/piko/wdk/logger"
)

var log = logger.GetLogger("piko/actions")

func invokeActionsPing(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	a := action.(*actions.PingAction)
	return a.Call()
}
