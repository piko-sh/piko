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
	"cmp"
	"context"
	"debug/buildinfo"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/process"
)

const (
	// deploymentGenerator holds the embedded deployment's generator entry point.
	deploymentGenerator = `package main
import ("context";"fmt";"os";"piko.sh/piko")
func main(){
 if len(os.Args)!=2 || os.Args[1]!="all" {panic("expected generation mode all")}
 server:=piko.New(piko.WithHealthEnabled(false),piko.WithCSSReset(piko.WithCSSResetComplete()))
 err:=server.Generate(context.Background(),piko.GenerateModeAll);server.Close()
 if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
}`

	// deploymentMain holds the compiled artefact's public server entry point.
	deploymentMain = `package main
import ("crypto/subtle";"fmt";"net/http";"os";"runtime/coverage";"strconv"; _ "example.org/pikofuzzcase/dist";"piko.sh/piko")
func main(){
 port,err:=strconv.Atoi(os.Getenv("PIKO_FUZZ_PORT"));if err!=nil{panic(err)}
 server:=piko.New(piko.WithPort(port),piko.WithBindAddress("127.0.0.1"),piko.WithAutoNextPort(false),piko.WithHealthEnabled(false),piko.WithCSSReset(piko.WithCSSResetComplete()))
 server.AppRouter.Get("/__piko_fuzz_ready",func(w http.ResponseWriter,r *http.Request){
  secret:=os.Getenv("PIKO_FUZZ_COVERAGE_SECRET")
  authorised:=secret!="" && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Piko-Fuzz-Coverage")),[]byte(secret))==1
  if dir:=os.Getenv("PIKO_FUZZ_GO_COVERAGE");dir!="" && authorised {
    if err:=coverage.WriteMetaDir(dir);err!=nil{http.Error(w,err.Error(),500);return}
    if err:=coverage.WriteCountersDir(dir);err!=nil{http.Error(w,err.Error(),500);return}
    w.WriteHeader(http.StatusNoContent);return
  }
  fmt.Fprint(w,os.Getenv("PIKO_FUZZ_TOKEN"))
 })
 if err:=server.Run(piko.RunModeProd);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
}`
)

// DeploymentReceipt records the linked artefact and source-detachment check.
type DeploymentReceipt struct {
	// Tags records the deployment build tags.
	Tags string

	// Files lists compiled artefact file names.
	Files []string

	// SourceDetached reports whether the artefact ran after source detachment.
	SourceDetached bool
}

// deploymentSource is one generated source file staged for the embedded deployment.
type deploymentSource struct {
	// name is the application-relative file path.
	name string

	// content is the file's Go source.
	content string
}

// checkDeployment checks that the compiled artefact ran after source detachment.
//
// Takes receipt (*DeploymentReceipt) which holds the saved receipt or campaign report.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkDeployment(receipt *DeploymentReceipt) error {
	if receipt == nil {
		return errors.New("deployment/receipt: missing runtime receipt")
	}
	if !slices.Equal(receipt.Files, []string{"app"}) {
		return fmt.Errorf("deployment/files: unexpected deployment files %q", receipt.Files)
	}
	if !slices.Contains(strings.FieldsFunc(receipt.Tags, func(character rune) bool { return character == ',' || character == ' ' }), "piko_embed") {
		return fmt.Errorf("deployment/tags: expected piko_embed in %q", receipt.Tags)
	}
	if !receipt.SourceDetached {
		return errors.New("deployment/source: build-time project still present")
	}
	return nil
}

// runDeployment links and runs the embedded application without its source tree.
//
// Takes run (*applicationRun) which holds the run inputs and the staged application.
//
// Returns Observation which holds the application evidence collected before completion or
// failure.
// Returns error which reports artefact generation, linking, source detachment or
// server-check failure, or nil on success.
func runDeployment(ctx context.Context, run *applicationRun) (Observation, error) {
	var o Observation
	linked := *run
	linked.workers = cmp.Or(run.workers, run.work)
	host, deployment, err := linkDeployment(ctx, &linked)
	if err != nil {
		return o, err
	}
	receipt, err := detachDeployment(linked.work, linked.app, deployment, host)
	if err != nil {
		return o, err
	}
	if err := checkDeployment(&receipt); err != nil {
		return o, err
	}
	browserBin := ""
	if linked.browser {
		browserBin, err = process.BuildWorker(ctx, linked.repo, linked.workers, "browser-worker", "")
		if err != nil {
			return o, err
		}
	}

	hosted := linked
	hosted.app, hosted.follow = deployment, nil
	o, err = runHost(ctx, &hosted, []string{"GOWORK=off"}, host, browserBin)
	o.Mode, o.ComponentCompiled, o.Deployment = modeCompiled, run.c.Expected.Tag != "", &receipt
	return o, err
}

// linkDeployment compiles the components and links the embedded deployment with the
// public CLI.
//
// Takes run (*applicationRun) which holds the run inputs, the staged application and a
// resolved worker directory.
//
// Returns string which is the linked deployment executable.
// Returns string which is the deployment directory holding it.
// Returns error which reports compilation, staging or link failure, or nil on success.
func linkDeployment(ctx context.Context, run *applicationRun) (host, deployment string, err error) {
	env := []string{"GOWORK=" + run.workspaceFile, "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local", "PIKO_FUZZ_TAG=" + run.c.Expected.Tag}
	checker, err := process.BuildWorker(ctx, run.repo, run.workers, "generate-worker", "")
	if err != nil {
		return "", "", err
	}
	if _, err := command(ctx, run.app, env, "pkc-compile", checker, "--compile-components"); err != nil {
		return "", "", err
	}
	if err := writeDeploymentSources(run.app); err != nil {
		return "", "", err
	}
	cli, err := buildCachedBinary(ctx, run.repo, env, "deploy/cli-link", filepath.Join(run.workers, "piko-cli"), "./cmd/piko")
	if err != nil {
		return "", "", err
	}
	deployment = filepath.Join(run.work, "deployment")
	if err := os.Mkdir(deployment, privateDirectoryMode); err != nil {
		return "", "", err
	}
	host = filepath.Join(deployment, "app")
	if run.coverageDirectory != "" {
		if err := os.MkdirAll(filepath.Join(run.coverageDirectory, "raw"), privateDirectoryMode); err != nil {
			return "", "", err
		}
		env = append(env, "GOFLAGS="+os.Getenv("GOFLAGS")+" -cover -covermode=atomic -coverpkg=piko.sh/piko/...,example.org/pikofuzzcase/...")
	}
	if _, err := command(ctx, run.app, env, "deploy/public-build", cli, "build", "--output", host); err != nil {
		return "", "", err
	}
	return host, deployment, nil
}

// writeDeploymentSources stages the deployment's entry points in the application.
//
// Takes app (string) which is the private generated application directory.
//
// Returns error which reports a directory or file write failure, or nil on success.
func writeDeploymentSources(app string) error {
	sources := []deploymentSource{
		{name: "cmd/main/main.go", content: deploymentMain},
		{name: "cmd/generator/main.go", content: deploymentGenerator},
		{name: "dist/generated.go", content: "package dist\n"},
	}
	for _, source := range sources {
		destination := filepath.Join(app, source.name)
		if err := os.MkdirAll(filepath.Dir(destination), privateDirectoryMode); err != nil {
			return err
		}
		if err := os.WriteFile(destination, []byte(source.content), privateFileMode); err != nil {
			return err
		}
	}
	return nil
}

// detachDeployment publishes the linked image, moves the source tree away and records
// what the deployment contains.
//
// The identical image is published to campaign provenance without another byte copy, and
// the build tree is frozen at a different absolute path.
//
// Takes work (string) which is the invocation-private working directory.
// Takes app (string) which is the private generated application directory.
// Takes deployment (string) which is the deployment directory.
// Takes host (string) which is the linked deployment executable.
//
// Returns DeploymentReceipt which records the deployment files, tags and detachment.
// Returns error which reports a link, rename, inspection or build-info failure.
func detachDeployment(work, app, deployment, host string) (DeploymentReceipt, error) {
	var receipt DeploymentReceipt
	if err := os.Link(host, filepath.Join(work, "native")); err != nil {
		return receipt, err
	}
	if err := os.Rename(app, filepath.Join(work, "detached-source")); err != nil {
		return receipt, err
	}
	_, err := os.Stat(app)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return receipt, fmt.Errorf("deployment/source: inspect detached source: %w", err)
	}
	receipt.SourceDetached = errors.Is(err, fs.ErrNotExist)
	entries, err := os.ReadDir(deployment)
	if err != nil {
		return receipt, err
	}
	for _, entry := range entries {
		receipt.Files = append(receipt.Files, entry.Name())
	}
	info, err := buildinfo.ReadFile(host)
	if err != nil {
		return receipt, err
	}
	for _, setting := range info.Settings {
		if setting.Key == "-tags" {
			receipt.Tags = setting.Value
		}
	}
	return receipt, nil
}
