package actions

import (
	"context"
	"testcase_127_action_bad_signature_error/actions/broken"

	"piko.sh/piko/wdk/logger"
)

var log = logger.GetLogger("piko/actions")

func invokeBrokenBadSignature(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	a := action.(*broken.BadSignatureAction)
	result := a.Call()
	return result, nil
}
