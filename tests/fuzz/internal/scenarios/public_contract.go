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

package scenarios

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/process"
	"piko.sh/piko/tests/fuzz/internal/publiccontracts"
)

var (
	// errNoProviderContracts reports an empty provider-contract catalogue.
	errNoProviderContracts = errors.New("infrastructure: no provider contracts")
)

// RunPublic runs the selected public-consumer contract.
//
// Takes repository (string) which is the repository source directory.
// Takes work (string) which is the invocation-private working directory.
// Takes fuzzCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns publiccontracts.Receipt which holds selected public-provider assertion
// evidence, including partial evidence on failure.
// Returns error which reports contract setup, execution or a missing positive assertion
// receipt, or nil on success.
func RunPublic(ctx context.Context, repository, work string, fuzzCase *casefile.Case) (publiccontracts.Receipt, error) {
	if fuzzCase.Target == "example-contracts" {
		return runExample(ctx, repository, work, fuzzCase)
	}
	contracts, err := publiccontracts.Providers(os.DirFS(repository))
	if err != nil {
		return publiccontracts.Receipt{}, fmt.Errorf("infrastructure: list provider contracts: %w", err)
	}
	if len(contracts) == 0 {
		return publiccontracts.Receipt{}, errNoProviderContracts
	}
	provider := contracts[fuzzCase.Index%uint64(len(contracts))]
	var source []byte
	if provider.Execution == publiccontracts.OverlayTest {
		source, err = renderSuite(repository, provider.Source, fuzzCase.Data)
	} else {
		source, err = readFileIn(repository, provider.Source)
	}
	if err != nil {
		return publiccontracts.Receipt{}, fmt.Errorf("infrastructure: read provider/%s source: %w", provider.Family, err)
	}
	receipt := publiccontracts.Receipt{
		Contract:           "provider/" + provider.Family + "/" + provider.Test,
		SourceSHA256:       fmt.Sprintf("%x", sha256.Sum256(source)),
		Steps:              0,
		Assertions:         0,
		HTTPStatusVerified: false,
		BrowserVisited:     false,
	}
	if provider.Execution == publiccontracts.OverlayTest {
		root, absErr := filepath.Abs(repository)
		if absErr != nil {
			return receipt, fmt.Errorf("infrastructure: resolve repository: %w", absErr)
		}
		test := SuiteTest{Package: provider.Package, Source: provider.Source, TestName: provider.Test, Files: nil}
		err = runRenderedSuite(ctx, root, work, fuzzCase, test, source)
	} else {
		err = runPackageContract(ctx, repository, &provider)
	}
	if err == nil {
		receipt.Steps = 1
		receipt.Assertions = 1
	}
	return receipt, err
}

// runPackageContract runs a provider contract test in its own package unchanged.
//
// Takes repository (string) which is the repository source directory.
// Takes provider (*publiccontracts.Provider) which is the selected contract.
//
// Returns error which reports a contract failure, a preparation failure or a missing
// positive test result.
func runPackageContract(ctx context.Context, repository string, provider *publiccontracts.Provider) error {
	output, err := process.Run(ctx, filepath.Join(repository, provider.Package), nil,
		"go", "test", "-mod=readonly", "-p=1", "-parallel=1", "-count=1", "-run=^"+regexp.QuoteMeta(provider.Test)+"$", "-v", ".")
	if err != nil {
		if !strings.Contains(output, "--- FAIL: "+provider.Test+" (") {
			return fmt.Errorf("infrastructure: preparing provider/%s/%s: %w\n%s", provider.Family, provider.Test, err, output)
		}
		return fmt.Errorf("provider/%s/%s: %w\n%s", provider.Family, provider.Test, err, output)
	}
	if !strings.Contains(output, "--- PASS: "+provider.Test+" (") {
		return fmt.Errorf("infrastructure: provider/%s missing pass for %s\n%s", provider.Family, provider.Test, output)
	}
	return nil
}
