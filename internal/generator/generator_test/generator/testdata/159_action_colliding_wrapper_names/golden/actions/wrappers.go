package actions

import (
	"context"
	zza_c833380c "testcase_159_action_colliding_wrapper_names/actions/zza"
	zzab_f1a412af "testcase_159_action_colliding_wrapper_names/actions/zzaB"

	"piko.sh/piko/wdk/logger"
)

var pikoActionLog = logger.GetLogger("piko/actions")

func invokeZzaBC(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	a := action.(*zza_c833380c.BCAction)
	return a.Call()
}
func invokeZzaBC2(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	a := action.(*zzab_f1a412af.CAction)
	return a.Call()
}
