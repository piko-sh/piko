package actions

import (
	"context"
	"reflect"
	zza_c833380c "testcase_159_action_colliding_wrapper_names/actions/zza"
	zzab_f1a412af "testcase_159_action_colliding_wrapper_names/actions/zzaB"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"zza.BC": {Name: "zza.BC", Method: "POST", Create: func() any {
		return &zza_c833380c.BCAction{}
	}, Invoke: invokeZzaBC, HasSSE: false}, "zzaB.C": {Name: "zzaB.C", Method: "POST", Create: func() any {
		return &zzab_f1a412af.CAction{}
	}, Invoke: invokeZzaBC2, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[zza_c833380c.BCOutput](), reflect.TypeFor[zzab_f1a412af.COutput]()}
	for _, t := range pretouchTypes {
		_ = pikojson.Pretouch(t)
	}
}

type ActionHandler struct {
	Name   string
	Method string
	Create func() any
	Invoke func(ctx context.Context, action any, args map[string]any) (any, error)
	Bind   func(ctx context.Context, action any, args map[string]any) error
	HasSSE bool
}

func Registry() map[string]ActionHandler {
	return map[string]ActionHandler{"zza.BC": {Name: "zza.BC", Method: "POST", Create: func() any {
		return &zza_c833380c.BCAction{}
	}, Invoke: invokeZzaBC, HasSSE: false}, "zzaB.C": {Name: "zzaB.C", Method: "POST", Create: func() any {
		return &zzab_f1a412af.CAction{}
	}, Invoke: invokeZzaBC2, HasSSE: false}}
}
