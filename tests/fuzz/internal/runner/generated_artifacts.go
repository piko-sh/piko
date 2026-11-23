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

package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/interp/interp_piko_symbols"
)

const (
	// actionClientName is the generated TypeScript action client, relative to the
	// application directory.
	actionClientName = "dist/ts/actions.gen.ts"
)

// GeneratedArtifactReceipt lists verified generated artefacts.
type GeneratedArtifactReceipt struct {
	// Checks holds completed checks.
	Checks []string
}

// artefactMismatch identifies a generated artefact mismatch.
type artefactMismatch struct {
	// field identifies the mismatched generated artefact property.
	field string

	// detail describes the mismatch.
	detail string
}

// artefactSession holds the inputs shared by the generated-artefact phase checks.
type artefactSession struct {
	// want maps the artefact and unaffected routes to their saved titles.
	want map[string]string

	// receipt records the completed checks.
	receipt *GeneratedArtifactReceipt

	// app is the private generated application directory.
	app string

	// work is the invocation-private working directory.
	work string

	// generator is the private generator executable.
	generator string

	// original is the saved artefact page source.
	original string

	// env supplies the private process environment.
	env []string

	// baseline holds the action client's bytes before any edit.
	baseline []byte
}

// Error formats the recorded failure.
//
// Returns string which is the formatted failure message.
func (e artefactMismatch) Error() string { return "generated-artifact: " + e.field + " " + e.detail }

// FailureSubject identifies the failed property independently of generated values.
//
// Returns string which is the stable failed-property identity.
func (e artefactMismatch) FailureSubject() string { return e.field }

// checkPhase regenerates the manifest for one edit phase and checks the generated Go
// packages and the unaffected action client.
//
// Takes phase (string) which is "initial", "edit" or "undo".
//
// Returns error which reports a regeneration, manifest, render or client failure.
func (a *artefactSession) checkPhase(ctx context.Context, phase string) error {
	label := a.want["/artifact"]
	if phase == "edit" {
		label += " revised"
	}
	if phase != "initial" {
		source := filepath.Join(a.app, "pages", "artifact.pk")
		if err := os.WriteFile(source, []byte(strings.ReplaceAll(a.original, a.want["/artifact"], label)), privateFileMode); err != nil {
			return err
		}
		if _, err := command(ctx, a.app, a.env, "generate-manifest", a.generator, "--manifest"); err != nil {
			return err
		}
	}
	packages, err := a.manifestPackages(ctx, phase)
	if err != nil {
		return err
	}
	a.receipt.Checks = append(a.receipt.Checks, phase+"/manifest")
	if err := a.checkGoRender(ctx, phase, label, packages); err != nil {
		return err
	}
	a.receipt.Checks = append(a.receipt.Checks, phase+"/go-load-and-render")
	current, err := readBoundedFile(a.app, actionClientName, stagedFileMax)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, a.baseline) {
		return artefactMismatch{field: phase + "/action-client", detail: "unrelated output changed"}
	}
	a.receipt.Checks = append(a.receipt.Checks, phase+"/unaffected-client")
	return nil
}

// manifestPackages reads the generated manifest and returns the two pages' packages.
//
// Takes phase (string) which names the edit phase in failure subjects.
//
// Returns []string which holds the artefact and unaffected page packages.
// Returns error which reports an unreadable manifest or a missing public entry.
func (a *artefactSession) manifestPackages(ctx context.Context, phase string) ([]string, error) {
	data, err := readBoundedFile(a.app, filepath.Join("dist", "manifest.bin"), stagedFileMax)
	if err != nil {
		return nil, err
	}
	manifest, err := generator_adapters.NewFlatBufferManifestProviderFromBytes(data).Load(ctx)
	if err != nil {
		return nil, err
	}
	var packages []string
	for _, route := range []string{"/artifact", "/unaffected"} {
		entry, ok := manifest.Pages["pages"+route+".pk"]
		if !ok || !strings.HasPrefix(entry.PackagePath, casefile.Module+"/dist/") {
			return nil, artefactMismatch{field: phase + route + "/manifest", detail: fmt.Sprintf("missing public package entry %q", entry.PackagePath)}
		}
		packages = append(packages, entry.PackagePath)
	}
	return packages, nil
}

// checkGoRender loads both generated page packages in a fresh program and checks their
// rendered titles.
//
// Takes phase (string) which names the edit phase in failure subjects.
// Takes label (string) which is the artefact page's expected title.
// Takes packages ([]string) which holds the artefact and unaffected page packages.
//
// Returns error which reports a build, run or title mismatch.
func (a *artefactSession) checkGoRender(ctx context.Context, phase, label string, packages []string) error {
	verifier := fmt.Sprintf(`package main
import("fmt"; "piko.sh/piko"; changed %q; stable %q; _ %q)
func main(){
 _, a, err:=changed.Render(nil,piko.NoProps{});if err!=nil{panic(err)}
 _, b,err:=stable.Render(nil,piko.NoProps{});if err!=nil{panic(err)}
 fmt.Printf("%%s\n%%s",a.Title,b.Title)
}
`, packages[0], packages[1], casefile.Module+"/dist")
	path := filepath.Join(a.work, "artifact_verify.go")
	if err := os.WriteFile(path, []byte(verifier), privateFileMode); err != nil {
		return err
	}
	output, err := command(ctx, a.app, a.env, "generated-go-load", "go", "run", path)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(strings.TrimSpace(output), label+"\n"+a.want["/unaffected"]) {
		return artefactMismatch{field: phase + "/go/metadata", detail: fmt.Sprintf("got %q", output)}
	}
	return nil
}

// exerciseGeneratedArtifacts executes the generated action clients and checks their
// public output.
//
// Takes app (string) which is the private generated application directory.
// Takes work (string) which is the invocation-private working directory.
// Takes generator (string) which is the private generator executable.
// Takes env ([]string) which supplies the private process environment.
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
//
// Returns *GeneratedArtifactReceipt which lists the generated artefacts whose public
// behaviour passed.
// Returns error which reports generated-client execution or public-output failure, or nil
// on success.
func exerciseGeneratedArtifacts(ctx context.Context, app, work, generator string, env []string, c *casefile.Case) (*GeneratedArtifactReceipt, error) {
	receipt := &GeneratedArtifactReceipt{Checks: nil}
	want := generatedPageModels(c)
	if len(want) != 2 {
		return receipt, errors.New("generated-artifact: missing saved page models")
	}
	baseline, err := readBoundedFile(app, actionClientName, stagedFileMax)
	if err != nil {
		return receipt, err
	}
	if err := checkTypeScriptAction(ctx, work, env, baseline, want["/artifact"]); err != nil {
		return receipt, err
	}
	receipt.Checks = append(receipt.Checks, "typescript/compile", "typescript/load-and-call")
	if err := checkCoerceSymbol(c.Expected.Count); err != nil {
		return receipt, err
	}
	receipt.Checks = append(receipt.Checks, "symbols/load-and-call")
	session := &artefactSession{
		want:      want,
		receipt:   receipt,
		app:       app,
		work:      work,
		generator: generator,
		original:  c.Files["pages/artifact.pk"],
		env:       env,
		baseline:  baseline,
	}
	for _, phase := range []string{"initial", "edit", "undo"} {
		if err := session.checkPhase(ctx, phase); err != nil {
			return receipt, err
		}
	}
	return receipt, nil
}

// generatedPageModels maps the artefact and unaffected routes to their saved titles.
//
// Takes c (*casefile.Case) which holds the saved probes.
//
// Returns map[string]string which maps each route with one output to its title.
func generatedPageModels(c *casefile.Case) map[string]string {
	want := map[string]string{}
	for index := range c.Probes {
		probe := &c.Probes[index]
		if (probe.Path == "/artifact" || probe.Path == "/unaffected") && len(probe.Output) == 1 {
			want[probe.Path] = probe.Output[0].Text
		}
	}
	return want
}

// checkCoerceSymbol calls the exported CoerceInt runtime symbol through the interpreter
// symbol provider.
//
// Takes count (int) which is the value to round-trip.
//
// Returns error which reports a missing symbol, wrong signature or wrong value.
func checkCoerceSymbol(count int) error {
	symbol := interp_piko_symbols.NewProvider().Exports()["piko.sh/piko/wdk/runtime"]["CoerceInt"]
	if !symbol.IsValid() {
		return artefactMismatch{field: "symbols/coerce-int", detail: "missing"}
	}
	coerce, ok := reflect.TypeAssert[func(any) (int, bool)](symbol)
	if !ok {
		return artefactMismatch{field: "symbols/coerce-int", detail: "wrong signature"}
	}
	if coerced, ok := coerce(int64(count)); !ok || coerced != count {
		return artefactMismatch{field: "symbols/coerce-int", detail: "wrong value"}
	}
	return nil
}

// checkTypeScriptAction runs the generated TypeScript action client against its fixture
// host.
//
// Takes work (string) which is the invocation-private working directory.
// Takes env ([]string) which supplies the private process environment.
// Takes source ([]byte) which holds generated TypeScript action-client source.
// Takes label (string) which is the generated action label.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkTypeScriptAction(ctx context.Context, work string, env []string, source []byte, label string) error {
	directory := filepath.Join(work, "artifact-typescript")
	if err := os.Mkdir(directory, privateDirectoryMode); err != nil {
		return err
	}

	const runtimePath = "/_piko/dist/ppframework.core.es.js"
	if bytes.Count(source, []byte(runtimePath)) != 1 {
		return artefactMismatch{field: "typescript/transport", detail: "expected one public runtime import"}
	}
	files := map[string]string{
		"action.ts": strings.Replace(string(source), runtimePath, "./transport.js", 1),
		"transport.d.ts": `export interface ActionBuilder<T>{call():Promise<T>}
export declare function createActionBuilder<T>(name:string, input:unknown):ActionBuilder<T>;
export declare function registerActionFunction<I,T>(name:string, factory:(input:I)=>ActionBuilder<T>):void;
export declare function registered(name:string):unknown;`,
		"transport.js": `const registry=new Map();
export function registerActionFunction(n,f){registry.set(n,f)};
export function registered(n){return registry.get(n)};
export function createActionBuilder(name,input){if(name!=="contract.Echo")throw Error(name);return {async call(){return input}};}`,
		"package.json": `{"type":"module"}`,
		"check.ts": fmt.Sprintf(`import {action} from "./action.js";
import {registered} from "./transport.js";
if(registered("contract.Echo")!==action.contract.Echo)throw Error("wrong public action registration");
const response: {label: string}=await action.contract.Echo({label:%q}).call();
if(response.label!==%q)throw Error("wrong generated action result");
if(false){
// @ts-expect-error the public typed client must reject a numeric label
action.contract.Echo({label:123});
}
`, label, label),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(content), privateFileMode); err != nil {
			return err
		}
	}
	compile := []string{"--target", "es2022", "--module", "es2022", "--moduleResolution", "bundler", "--strict", "--skipLibCheck", "action.ts", "check.ts"}
	if _, err := command(ctx, directory, env, "typescript-compile", "tsc", compile...); err != nil {
		return err
	}
	_, err := command(ctx, directory, env, "typescript-load", "node", "check.js")
	return err
}
