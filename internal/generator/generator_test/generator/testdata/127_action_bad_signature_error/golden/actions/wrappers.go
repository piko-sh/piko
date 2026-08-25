package actions

import (
	"context"
	broken_e68d7c38 "testcase_127_action_bad_signature_error/actions/broken"

	"piko.sh/piko/wdk/logger"
)

var pikoActionLog = logger.GetLogger("piko/actions")

func invokeBrokenBadSignature(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	a := action.(*broken_e68d7c38.BadSignatureAction)
	result := a.Call()
	return result, nil
}
