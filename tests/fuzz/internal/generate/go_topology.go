// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package generate

import (
	"fmt"
	"strconv"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// topologyStepChoices bounds the drawn ledger adjustment step.
	topologyStepChoices = 3

	// topologyFactoryOffsetStep is the offset added by each successive callback factory.
	topologyFactoryOffsetStep = 10

	// releasePageTemplate formats the embedded release page from its heading label.
	releasePageTemplate = `<template><main><h1 data-fuzz-element="release">%s</h1><a data-fuzz-element="back" href="/main">Home</a></main></template>`

	// topologyStateSource holds the shared state package, before its selection constant.
	topologyStateSource = `package state
import ("errors";"fmt";"sync")
type Revision int
type Number = Revision
type Ref[T any] struct { Value T }
func(r Ref[T]) Get() T {return r.Value}
type Problem struct{ Message string }
func(p Problem) Error()string{return p.Message}
var mu sync.Mutex
var value Revision
var hooks = map[string]int{}
func Register(k string){mu.Lock();defer mu.Unlock();hooks[k]++}
func Membership()string{mu.Lock();defer mu.Unlock();return fmt.Sprintf("%d/%d/%d",hooks["base"],hooks["left"],hooks["right"])}
func Snapshot()int {mu.Lock();defer mu.Unlock();return int(Ref[Number]{Value:value}.Get())}
func Set(v int){mu.Lock();defer mu.Unlock();value=Revision(v)}
func Change(n int) (int,error){if n<=0{return Snapshot(),Problem{"positive delta required"}};mu.Lock();defer mu.Unlock();value+=Revision(n);return int(value),nil}
func ErrorContract()string{_,err:=Change(0);var typed Problem;if errors.As(err,&typed){return typed.Message};return "missing error"}
func Capture()func()int{initial:=Snapshot();return func()int{return initial}}
`

	// topologyServiceSource holds the shared service package.
	topologyServiceSource = `package service
import ("sync"; a "` + casefile.TopologyModule + `/left"; b "` + casefile.TopologyModule + `/right"; "` + casefile.TopologyModule + `/state")
type Entry = state.Ref[state.Revision]
var Old = state.Capture()
var callbackMutex sync.Mutex
var retained=map[string]func()int{}
func Save(name string, cb func()int){callbackMutex.Lock();defer callbackMutex.Unlock();retained[name]=cb}
func Recall(name string)int{callbackMutex.Lock();cb:=retained[name];callbackMutex.Unlock();if cb==nil{return -1};return cb()}
func Current()Entry{return Entry{Value:state.Revision(state.Snapshot())}}
func TryZero()(int,error){return state.Change(0)}
func Read()int{return state.Snapshot()}
func Previous()int{return Old()}
func NewCallback()int {next:=state.Capture();return next()}
func Summary()string{return a.Name()+"/"+b.Name()+"/"+state.Membership()+"/"+state.Selected+"/"+state.ErrorContract()}
func Increment(n int)(int,error){return state.Change(n)}
`

	// topologyRecordBase holds the record package's live and previous readers.
	topologyRecordBase = `package record
import model "` + casefile.TopologyModule + `/service"
func Live()int{entry:=model.Current();return int(entry.Get())}
func Before()int{return model.Previous()}
`

	// topologyRecordMetadata holds the record package's callback and origin readers.
	topologyRecordMetadata = `package record
import model "` + casefile.TopologyModule + `/service"
func Fresh()int{return model.NewCallback()}
func Origin()string{_,err:=model.TryZero();if err==nil{return "missing error"};return model.Summary()+"/"+err.Error()}
`

	// topologyLedgerPrefix holds the ledger page source before its page name.
	topologyLedgerPrefix = `<template><main id="fuzz-program"><output data-fuzz-output="page">`

	// topologyLedgerSuffix holds the ledger page source after its page name.
	topologyLedgerSuffix = `</output><output data-fuzz-output="value">{{ state.Value }}</output>` +
		`<output data-fuzz-output="old">{{ state.Old }}</output><output data-fuzz-output="new">{{ state.New }}</output>` +
		`<output data-fuzz-output="origin">{{ state.Origin }}</output></main></template>
<script type="application/x-go">
package main
import ("piko.sh/piko";records "example.org/pikofuzzcase/internal/record")
type Response struct{Value, Old, New int;Origin string}
func Render(r *piko.RequestData, props piko.NoProps)(Response,piko.Metadata,error){` +
		`return Response{Value:records.Live(), Old:records.Before(), New:records.Fresh(), Origin:records.Origin()},piko.Metadata{},nil}
</script>`

	// topologyFactoryTemplate formats a callback factory package from its offset.
	topologyFactoryTemplate = `package factory
var offset = %d
func Capture(v int)func()int{return func()int{return v+offset}}
`

	// topologyHooksPage holds the page that installs and recalls guest callbacks.
	topologyHooksPage = `<template><main id="fuzz-program"><output data-fuzz-output="past">{{ state.Past }}</output>` +
		`<output data-fuzz-output="present">{{ state.Present }}</output></main></template>
<script type="application/x-go">
package main
import ("piko.sh/piko";past "example.org/pikofuzzcase/internal/callback/past";present "example.org/pikofuzzcase/internal/callback/present";` +
		`records "example.org/pikofuzzcase/internal/record";svc "` + casefile.TopologyModule + `/service")
type Response struct{Past,Present int}
func Render(r *piko.RequestData, props piko.NoProps)(Response,piko.Metadata,error){
	switch r.QueryParam("install") {case "past":svc.Save("past",past.Capture(records.Live()));` +
		`case "present":svc.Save("present",present.Capture(records.Live()))}
	return Response{Past:svc.Recall("past"),Present:svc.Recall("present")},piko.Metadata{},nil
}
</script>`

	// topologyAdjustAction holds the ledger adjustment action source.
	topologyAdjustAction = `package ledger
import ("piko.sh/piko";svc "` + casefile.TopologyModule + `/service")
type AdjustInput struct{ Delta int ` + "\x60json:\"delta\"\x60" + ` }
type AdjustResult struct{Revision int ` + "\x60json:\"revision\"\x60" + ` }
type AdjustAction struct{piko.ActionMetadata}
func(a *AdjustAction)Call(in AdjustInput)(AdjustResult,error){v,err:=svc.Increment(in.Delta);return AdjustResult{Revision:v},err}
`
)

// addEmbeddedDeployment adds source and expected observations for embedded deployment.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addEmbeddedDeployment(c *casefile.Case, p Project) {
	label := fmt.Sprintf("Release %d.%d", p.Count, c.Index)
	c.Files["pages/release.pk"] = fmt.Sprintf(releasePageTemplate, label)
	probe := semanticProbe("/release", responseContract(nil, catalogueElement("release", "h1", label), catalogueElement("back", "a", "Home", "href", "/main")))
	probe.Stage = "binary-only"
	c.Probes = append(c.Probes, probe)
	c.Features = append(c.Features, "compiled-embedded-deployment", "public-cli-build", "binary-only-release")
}

// addGoTopology adds source and expected observations for Go package topology.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addGoTopology(c *casefile.Case, p Project) {
	initial := 1 + p.Count
	c.Expected.Topology = &casefile.GoTopology{Initial: initial, Step: 2 + int(c.Index%topologyStepChoices), GuestCallbacks: true}
	for _, directory := range []string{"module", "workspace"} {
		addTopologyModule(c, "topology-modules/"+directory+"/", directory, initial)
	}
	c.Files["internal/record/base.go"] = topologyRecordBase
	c.Files["internal/record/metadata.go"] = topologyRecordMetadata
	for _, page := range []string{"ledger", "ledger-detail"} {
		c.Files["pages/"+page+".pk"] = topologyLedgerPrefix + page + topologyLedgerSuffix
	}
	for position, name := range []string{"past", "present"} {
		c.Files["internal/callback/"+name+"/factory.go"] = fmt.Sprintf(topologyFactoryTemplate, (position+1)*topologyFactoryOffsetStep)
	}
	c.Files["pages/hooks.pk"] = topologyHooksPage
	c.Files["actions/ledger/adjust.go"] = topologyAdjustAction
	c.Features = append(c.Features, "go-workspace-precedence", "go-registration-diamond", "go-internal-files", "go-generic-dto", "go-typed-error", "native-service-callbacks")
}

// addTopologyModule adds one copy of the shared topology module.
//
// Takes c (*casefile.Case) which receives the module sources.
// Takes path (string) which is the module's directory prefix, ending in a slash.
// Takes directory (string) which names the copy and is recorded as its selection.
// Takes initial (int) which is the revision the base package registers.
func addTopologyModule(c *casefile.Case, path, directory string, initial int) {
	const module = casefile.TopologyModule
	c.Files[path+"go.mod"] = "module " + module + "\n\ngo 1.27.0\n"
	c.Files[path+"state/value.go"] = topologyStateSource + "\nconst Selected=" + strconv.Quote(directory) + "\n"
	c.Files[path+"base/registration.go"] = `package base
import "` + module + `/state"
func init(){state.Register("base");state.Set(` + strconv.Itoa(initial) + `)}
`
	for _, side := range []string{"left", "right"} {
		c.Files[path+side+"/register.go"] = `package branch
import (_ "` + module + `/base";"` + module + `/state")
func init(){state.Register("` + side + `")}
func Name()string{return "` + side + `"}
`
	}
	c.Files[path+"service/service.go"] = topologyServiceSource
}

// addFileEditHistory adds source and expected observations for file-edit histories.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addFileEditHistory(c *casefile.Case, p Project) {
	base := fmt.Sprintf("edition-%d-%d", p.Count, c.Index)
	c.Files["fuzz-history/marker.txt"] = base
	c.Files["fuzz-history/page.pk"] = `<template><main id="fuzz-program"><output data-fuzz-output="edition">` + base + `</output></main></template>`
	c.Files["fuzz-history/repaired.pk"] = strings.Replace(c.Files["fuzz-history/page.pk"], base, base+"-repaired", 1)
	c.Files["fuzz-history/invalid.pk"] = `<template><main id="fuzz-program"><output>{{state.Value}}</output></main></template>
<script type="application/x-go">
package main
import "piko.sh/piko"
type Response struct{Value string}
func Render(r *piko.RequestData, props piko.NoProps)(Response,piko.Metadata,error){return Response{Value:FuzzMissingEdition},piko.Metadata{},nil}
</script>`
	c.Features = append(c.Features, "warm-file-create-rename-delete", "warm-invalid-diagnostic-repair", "cold-route-equivalence")
}
