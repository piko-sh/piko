package actions

import (
	"context"
	actions_0e28fc6f "testcase_132_action_root_directory/actions"

	"piko.sh/piko/wdk/logger"
)

var pikoActionLog = logger.GetLogger("piko/actions")

func invokeActionsPing(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	a := action.(*actions_0e28fc6f.PingAction)
	return a.Call()
}
